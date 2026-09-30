package expenserecognition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
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
	// Optional (S1 shared source claim, 20260927-usage-and-pass-through-charges): when set (and a
	// transactor is available) the recognition runs in one transaction that locks the
	// expenditure's cost_source_component rows, refuses source_claimed_by_allocation and marks
	// the RECOGNITION claim. Expenditures without components behave exactly as before.
	CostSourceComponent costsourcecomponentpb.CostSourceComponentDomainServiceServer
}

// RecognizeFromExpenditureServices groups service dependencies.
type RecognizeFromExpenditureServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
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
//
// Blast radius (S1, 20260927-usage-and-pass-through-charges): every expenditure of every vertical
// goes through this use case, so the transactional claim path is taken ONLY for an expenditure that
// has cost_source_component rows. A plain (non-locking, workspace-scoped) existence read decides;
// with no components, or with no S1 collaborator wired, the legacy non-transactional path runs
// byte-for-byte as before, including its old semantics for a failed line insert. With components
// the whole operation runs in the injected transaction (component locks + claim + recognition rows
// commit or roll back together) and a failed line insert aborts it. A repository without the locker
// capability or a failing existence read refuses (fail closed, C4). Limitation: a component created
// after the existence read and before commit is not seen; components are authored before the
// expenditure is recognised.
func (uc *RecognizeFromExpenditureUseCase) Execute(ctx context.Context, req *expenserecognitionpb.RecognizeFromExpenditureRequest) (*expenserecognitionpb.RecognizeFromExpenditureResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenseRecognition,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if uc.repositories.CostSourceComponent != nil && req != nil && req.GetExpenditureId() != "" {
		has, err := uc.expenditureHasSourceComponents(ctx, req.GetExpenditureId())
		if err != nil {
			return nil, err
		}
		if has {
			if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
				return nil, uc.refuseClaim(ctx, codeTransactionRequired)
			}
			var resp *expenserecognitionpb.RecognizeFromExpenditureResponse
			err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
				r, e := uc.execute(txCtx, req, true)
				resp = r
				return e
			})
			if err != nil {
				return nil, err
			}
			return resp, nil
		}
	}
	return uc.execute(ctx, req, false)
}

// execute is the recognition body; claimSource selects the S1 component-claim behaviour.
func (uc *RecognizeFromExpenditureUseCase) execute(ctx context.Context, req *expenserecognitionpb.RecognizeFromExpenditureRequest, claimSource bool) (*expenserecognitionpb.RecognizeFromExpenditureResponse, error) {
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

	// S1 shared source claim: lock the expenditure's components (id ascending) BEFORE any write and
	// refuse when an allocation owns the source. No components => nothing changes.
	var claimComponents []*costsourcecomponentpb.CostSourceComponent
	if claimSource {
		var err error
		if claimComponents, err = uc.lockSourceComponents(ctx, req.GetExpenditureId()); err != nil {
			return nil, err
		}
	}

	// Read the source Expenditure to capture FK fields added in the buying/selling
	// parity epic (supplier_subscription_id, field 34).
	var supplierSubscriptionID string
	var sourceName, sourceCurrency string
	var sourceTotal int64
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
			sourceName = expResp.Data[0].GetName()
			sourceCurrency = expResp.Data[0].GetCurrency()
			sourceTotal = expResp.Data[0].GetTotalAmount()
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
		// NOT NULL header columns on the postgres schema (internal_id UNIQUE, name,
		// recognition_date): populated like the sibling recognizers/creators do
		// (supplier subscription recognizer names + dates the header; create_supplier.go:208
		// generates internal_id from the ID generator).
		InternalId:      uc.services.IDGenerator.GenerateID(),
		Name:            recognitionName(sourceName, expenditureID, period),
		RecognitionDate: timestamppb.New(recognitionDateFor(period, now)),
		Currency:        sourceCurrency,
		TotalAmount:     sourceTotal,
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
				_, lineErr := uc.repositories.ExpenseRecognitionLine.CreateExpenseRecognitionLine(ctx, &expenserecognitionlinepb.CreateExpenseRecognitionLineRequest{
					Data: lineData,
				})
				if lineErr != nil && claimSource {
					// Inside the claim transaction a failed insert aborts it (postgres), so it must
					// surface instead of being swallowed. The legacy path keeps its old semantics.
					return nil, fmt.Errorf("expense_recognition: create recognition line: %w", lineErr)
				}
			}
		}
	}

	if claimSource && data != nil {
		if err := uc.markRecognitionClaim(ctx, claimComponents, data.GetId()); err != nil {
			return nil, err
		}
	}

	return &expenserecognitionpb.RecognizeFromExpenditureResponse{Success: true, Data: data}, nil
}

// recognitionName is the header name: "<expenditure name> — <period>" (the supplier-subscription
// recognizer's "<name> — <period>" shape), falling back to the expenditure id.
func recognitionName(expenditureName, expenditureID, period string) string {
	if expenditureName == "" {
		expenditureName = expenditureID
	}
	return fmt.Sprintf("%s — %s", expenditureName, period)
}

// recognitionDateFor is the last day of the recognition period ("YYYY-MM"), or now when the caller
// supplied a period that is not a month.
func recognitionDateFor(period string, now time.Time) time.Time {
	if first, err := time.Parse("2006-01", period); err == nil {
		return first.AddDate(0, 1, -1)
	}
	return now.UTC()
}
