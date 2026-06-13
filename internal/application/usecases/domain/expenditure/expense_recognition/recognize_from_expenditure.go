package expenserecognition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// RecognizeFromExpenditureRepositories groups repository dependencies.
type RecognizeFromExpenditureRepositories struct {
	ExpenseRecognition     expenserecognitionpb.ExpenseRecognitionDomainServiceServer
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
	Expenditure            expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem    expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	// Optional: when set, cross-workspace ownership of SupplierSubscription is validated.
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

// RecognizeFromExpenditureServices groups service dependencies.
type RecognizeFromExpenditureServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// RecognizeFromExpenditureUseCase converts a posted Expenditure into one or more
// ExpenseRecognition rows. Routine pattern: derive idempotency_key, build the
// recognition row, persist via the underlying CRUD adapter. Multi-period
// amortization (e.g. annual prepayment recognized monthly) is driven by the
// caller emitting multiple calls with distinct recognition_period values.
//
// Buying/selling parity (2026-05-09): when the source Expenditure carries a
// supplier_subscription_id, that FK is threaded through to both the
// ExpenseRecognition header (field 60) and every ExpenseRecognitionLine (field 15).
// supplier_product_cost_plan_id is copied from each ExpenditureLineItem (field 28)
// to the corresponding ExpenseRecognitionLine (field 14).
type RecognizeFromExpenditureUseCase struct {
	repositories RecognizeFromExpenditureRepositories
	services     RecognizeFromExpenditureServices
}

// NewRecognizeFromExpenditureUseCase creates a use case with grouped dependencies.
func NewRecognizeFromExpenditureUseCase(
	repositories RecognizeFromExpenditureRepositories,
	services RecognizeFromExpenditureServices,
) *RecognizeFromExpenditureUseCase {
	return &RecognizeFromExpenditureUseCase{repositories: repositories, services: services}
}

// Execute performs the recognize-from-expenditure operation.
func (uc *RecognizeFromExpenditureUseCase) Execute(ctx context.Context, req *expenserecognitionpb.RecognizeFromExpenditureRequest) (*expenserecognitionpb.RecognizeFromExpenditureResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenseRecognition,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.GetExpenditureId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"expense_recognition.validation.expenditure_id_required", "Expenditure ID is required [DEFAULT]"))
	}

	period := req.GetRecognitionPeriod()
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}

	// Derive idempotency_key per HIGH-2 amendment when the caller hasn't provided one.
	idempotencyKey := req.GetIdempotencyKey()
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("EXPENDITURE:%s:%s", req.GetExpenditureId(), period)
	}

	// Read the source Expenditure to capture FK fields added in the buying/selling
	// parity epic (supplier_subscription_id, field 34).
	var supplierSubscriptionID string
	if uc.repositories.Expenditure != nil {
		expenditureID := req.GetExpenditureId()
		expResp, err := uc.repositories.Expenditure.ReadExpenditure(ctx, &expenditurepb.ReadExpenditureRequest{
			Data: &expenditurepb.Expenditure{Id: expenditureID},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to read source expenditure: %w", err)
		}
		if expResp != nil && len(expResp.Data) > 0 {
			supplierSubscriptionID = expResp.Data[0].GetSupplierSubscriptionId()
		}
	}

	// Cross-workspace consistency check: when the recognition carries a
	// supplier_subscription_id, verify the referenced SupplierSubscription belongs
	// to the same workspace as the current request context. This mirrors the
	// pattern established in CreateSupplierSubscription (currency hard-block)
	// and generalised by the 20260506 P2.3 cross-workspace FK validation policy.
	if supplierSubscriptionID != "" && uc.repositories.SupplierSubscription != nil {
		wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
		if wsID != "" {
			subResp, subErr := uc.repositories.SupplierSubscription.ReadSupplierSubscription(ctx, &suppliersubscriptionpb.ReadSupplierSubscriptionRequest{
				Data: &suppliersubscriptionpb.SupplierSubscription{Id: supplierSubscriptionID},
			})
			if subErr == nil && subResp != nil && len(subResp.Data) > 0 {
				if subResp.Data[0].GetWorkspaceId() != wsID {
					return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
						"expense_recognition.errors.supplier_subscription_workspace_mismatch",
						"supplier subscription does not belong to the current workspace"))
				}
			}
		}
	}

	now := time.Now()
	id := uc.services.IDGenerator.GenerateID()
	expenditureID := req.GetExpenditureId()
	createData := &expenserecognitionpb.ExpenseRecognition{
		Id:                 id,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true,
		Status:             expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_DRAFT,
		ExpenditureId:      &expenditureID,
		IdempotencyKey:     idempotencyKey,
	}
	// Thread supplier_subscription_id from the source Expenditure (field 60).
	if supplierSubscriptionID != "" {
		createData.SupplierSubscriptionId = &supplierSubscriptionID
	}

	createResp, err := uc.repositories.ExpenseRecognition.CreateExpenseRecognition(ctx, &expenserecognitionpb.CreateExpenseRecognitionRequest{
		Data: createData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create recognition from expenditure: %w", err)
	}
	var data *expenserecognitionpb.ExpenseRecognition
	if len(createResp.Data) > 0 {
		data = createResp.Data[0]
	}

	// Create ExpenseRecognitionLine rows mirroring the source ExpenditureLineItems.
	// Threads supplier_product_cost_plan_id (field 14) from ExpenditureLineItem.field28
	// and supplier_subscription_id (field 15) from the parent recognition.
	if uc.repositories.ExpenditureLineItem != nil && uc.repositories.ExpenseRecognitionLine != nil && data != nil {
		listResp, listErr := uc.repositories.ExpenditureLineItem.ListExpenditureLineItems(ctx, &expenditurelineitempb.ListExpenditureLineItemsRequest{
			ExpenditureId: &expenditureID,
		})
		if listErr == nil && listResp != nil {
			for _, eli := range listResp.Data {
				lineID := uc.services.IDGenerator.GenerateID()
				eliID := eli.GetId()
				lineData := &expenserecognitionlinepb.ExpenseRecognitionLine{
					Id:                    lineID,
					DateCreated:           &[]int64{now.UnixMilli()}[0],
					DateCreatedString:     &[]string{now.Format(time.RFC3339)}[0],
					DateModified:          &[]int64{now.UnixMilli()}[0],
					DateModifiedString:    &[]string{now.Format(time.RFC3339)}[0],
					Active:                true,
					ExpenseRecognitionId:  data.GetId(),
					ExpenditureLineItemId: &eliID,
					Description:           eli.GetDescription(),
					Quantity:              eli.GetQuantity(),
					UnitAmount:            eli.GetUnitPrice(),
					Amount:                eli.GetTotalPrice(),
				}
				if pid := eli.GetProductId(); pid != "" {
					lineData.ProductId = &pid
				}
				// Copy supplier_product_cost_plan_id from ExpenditureLineItem.field28.
				if v := eli.GetSupplierProductCostPlanId(); v != "" {
					lineData.SupplierProductCostPlanId = &v
				}
				// Propagate supplier_subscription_id from the recognition header.
				if supplierSubscriptionID != "" {
					lineData.SupplierSubscriptionId = &supplierSubscriptionID
				}
				_, _ = uc.repositories.ExpenseRecognitionLine.CreateExpenseRecognitionLine(ctx, &expenserecognitionlinepb.CreateExpenseRecognitionLineRequest{
					Data: lineData,
				})
			}
		}
	}

	return &expenserecognitionpb.RecognizeFromExpenditureResponse{Success: true, Data: data}, nil
}
