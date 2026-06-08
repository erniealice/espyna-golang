package expenserecognition

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// RecognizeExpenseOutcome enumerates the terminal states for the run-engine.
type RecognizeExpenseOutcome string

const (
	RecognizeExpenseOutcomeCreated RecognizeExpenseOutcome = "CREATED"
	RecognizeExpenseOutcomeSkipped RecognizeExpenseOutcome = "SKIPPED"
	RecognizeExpenseOutcomeErrored RecognizeExpenseOutcome = "ERRORED"
)

// RecognizeExpenseFromSupplierSubscriptionInput is the request shape used by
// the ExpenseRecognitionRun engine (and any manual buying-side recognition
// trigger).
type RecognizeExpenseFromSupplierSubscriptionInput struct {
	SupplierSubscriptionID string
	PeriodStart            string // YYYY-MM-DD
	PeriodEnd              string // YYYY-MM-DD
	PeriodMarker           string // canonical marker; defaults to derived
	WorkspaceID            string
	RunID                  string // optional; set by the run engine
	IdempotencyKey         string // optional; default derived from workspace+sub+period_start
	ActorID                string
}

// RecognizeExpenseFromSupplierSubscriptionOutput captures the result.
type RecognizeExpenseFromSupplierSubscriptionOutput struct {
	Outcome                         RecognizeExpenseOutcome
	ExpenseRecognitionID            string
	ExpenditureID                   string
	ConflictingExpenseRecognitionID string
	Currency                        string
	TotalAmount                     int64
	Error                           error
}

// RecognizeExpenseFromSupplierSubscriptionRepositories groups deps.
//
// Notes on adapter ports:
//   - SupplierProductCostPlan is OPTIONAL — when nil, the use case falls back
//     to a single "header-only" Expenditure with no line items derived from
//     the cost plan rate card. The buying-side mirror still records a single
//     ExpenseRecognitionLine echoing the total amount in that case.
type RecognizeExpenseFromSupplierSubscriptionRepositories struct {
	ExpenseRecognition      expenserecognitionpb.ExpenseRecognitionDomainServiceServer
	ExpenseRecognitionLine  expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
	Expenditure             expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem     expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	SupplierSubscription    suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	CostPlan                costplanpb.CostPlanDomainServiceServer
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

// RecognizeExpenseFromSupplierSubscriptionServices groups infra deps.
type RecognizeExpenseFromSupplierSubscriptionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// RecognizeExpenseFromSupplierSubscriptionUseCase is the buying-side mirror
// of revenue.RecognizeRevenueFromSubscription. It composes Plan A's run
// engine — DO NOT modify the existing recognize_from_expenditure flow.
//
// Hard rule (per Plan A codex P0): idempotency check FIRST. The use case
// must NOT insert a draft Expenditure before checking for an existing
// recognition with the same idempotency_key — otherwise SKIPPED outcomes
// leak orphan draft Expenditure rows.
type RecognizeExpenseFromSupplierSubscriptionUseCase struct {
	repositories RecognizeExpenseFromSupplierSubscriptionRepositories
	services     RecognizeExpenseFromSupplierSubscriptionServices
}

// NewRecognizeExpenseFromSupplierSubscriptionUseCase wires the use case.
func NewRecognizeExpenseFromSupplierSubscriptionUseCase(
	repos RecognizeExpenseFromSupplierSubscriptionRepositories,
	svcs RecognizeExpenseFromSupplierSubscriptionServices,
) *RecognizeExpenseFromSupplierSubscriptionUseCase {
	return &RecognizeExpenseFromSupplierSubscriptionUseCase{repositories: repos, services: svcs}
}

// Execute recognizes one cycle of a supplier subscription as a draft
// Expenditure + ExpenseRecognition + ExpenseRecognitionLine(s).
//
// Flow:
//  1. authcheck (expense_recognition:create + supplier_subscription:read).
//  2. Compute the canonical idempotency_key.
//  3. Idempotency check FIRST — if a recognition with this key exists,
//     return SKIPPED. NO draft Expenditure is inserted.
//  4. Read SupplierSubscription + CostPlan; derive Currency, total amount,
//     and ExpenseRecognitionLine specs.
//  5. INSERT draft Expenditure (status="draft", source="expense_run",
//     supplier_subscription_id=…, run_id=…).
//  6. INSERT ExpenseRecognition (status=DRAFT, supplier_subscription_id=…,
//     expenditure_id=…, run_id=…, idempotency_key=…).
//  7. INSERT ExpenseRecognitionLine(s).
func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) Execute(
	ctx context.Context,
	input RecognizeExpenseFromSupplierSubscriptionInput,
) (*RecognizeExpenseFromSupplierSubscriptionOutput, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenseRecognition, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"supplier_subscription", entityid.ActionRead); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.SupplierSubscriptionID) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"expense_recognition.validation.supplier_subscription_id_required",
			"supplier_subscription_id is required [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var out *RecognizeExpenseFromSupplierSubscriptionOutput
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, execErr := uc.executeCore(txCtx, input)
			if execErr != nil {
				return execErr
			}
			out = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return uc.executeCore(ctx, input)
}

func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) executeCore(
	ctx context.Context,
	input RecognizeExpenseFromSupplierSubscriptionInput,
) (*RecognizeExpenseFromSupplierSubscriptionOutput, error) {
	// 1. Derive idempotency_key BEFORE any reads/inserts.
	idempotencyKey := input.IdempotencyKey
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = BuildSubscriptionIdempotencyKey(input.WorkspaceID, input.SupplierSubscriptionID, input.PeriodStart)
	}

	// 2. Idempotency check FIRST — fast path bails before any draft insert.
	if conflictID, found, err := uc.findExistingRecognition(ctx, idempotencyKey, input.SupplierSubscriptionID, input.PeriodStart); err != nil {
		return erroredExp(err), err
	} else if found {
		return &RecognizeExpenseFromSupplierSubscriptionOutput{
			Outcome:                         RecognizeExpenseOutcomeSkipped,
			ConflictingExpenseRecognitionID: conflictID,
		}, nil
	}

	// 3. Read the SupplierSubscription to derive amount / currency / supplier_id.
	sub, err := uc.readSubscription(ctx, input.SupplierSubscriptionID)
	if err != nil {
		return erroredExp(err), err
	}
	if sub == nil {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"supplier_subscription.errors.not_found",
			"supplier_subscription not found [DEFAULT]",
		))
		return erroredExp(err), err
	}
	if !sub.GetActive() {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"supplier_subscription.errors.inactive",
			"supplier subscription is inactive [DEFAULT]",
		))
		return erroredExp(err), err
	}

	// 4. Read the CostPlan for amount/currency.
	plan, err := uc.readCostPlan(ctx, sub.GetCostPlanId())
	if err != nil {
		return erroredExp(err), err
	}
	if plan == nil {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"cost_plan.errors.not_found",
			"cost_plan not found for supplier subscription [DEFAULT]",
		))
		return erroredExp(err), err
	}

	// 5. Resolve workspace / supplier IDs.
	workspaceID := input.WorkspaceID
	if workspaceID == "" {
		workspaceID = sub.GetWorkspaceId()
	}
	if workspaceID == "" {
		workspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}
	supplierID := sub.GetSupplierId()

	// 6. Build line specs and total.
	lineSpecs, totalAmount, currency, err := uc.buildLineSpecs(ctx, sub, plan)
	if err != nil {
		return erroredExp(err), err
	}

	// 7. INSERT draft Expenditure.
	expID, err := uc.insertDraftExpenditure(ctx, sub, supplierID, currency, totalAmount, input)
	if err != nil {
		return erroredExp(err), err
	}

	// 8. INSERT ExpenseRecognition row.
	recID, err := uc.insertRecognition(ctx, sub, supplierID, workspaceID, currency, totalAmount, expID, idempotencyKey, input)
	if err != nil {
		return erroredExp(err), err
	}

	// 9. INSERT ExpenseRecognitionLine rows.
	if err := uc.insertLines(ctx, recID, workspaceID, sub, lineSpecs); err != nil {
		return erroredExp(err), err
	}

	return &RecognizeExpenseFromSupplierSubscriptionOutput{
		Outcome:              RecognizeExpenseOutcomeCreated,
		ExpenseRecognitionID: recID,
		ExpenditureID:        expID,
		Currency:             currency,
		TotalAmount:          totalAmount,
	}, nil
}

// findExistingRecognition checks the canonical idempotency_key AND a
// secondary (supplier_subscription_id + period_start) heuristic for
// hand-rolled keys. Mirrors the selling-side period_marker check.
func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) findExistingRecognition(
	ctx context.Context,
	key, subID, periodStart string,
) (string, bool, error) {
	if uc.repositories.ExpenseRecognition == nil {
		return "", false, nil
	}
	// Primary: exact idempotency_key hit.
	resp, err := uc.repositories.ExpenseRecognition.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "idempotency_key",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    key,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return "", false, err
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0].GetId(), true, nil
	}

	// Secondary: (supplier_subscription_id) filter + in-memory period_start match.
	resp2, err := uc.repositories.ExpenseRecognition.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "supplier_subscription_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    subID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return "", false, err
	}
	if resp2 == nil {
		return "", false, nil
	}
	for _, r := range resp2.GetData() {
		ps := r.GetPeriodStart()
		if ps == nil {
			continue
		}
		if ps.AsTime().UTC().Format("2006-01-02") == periodStart {
			return r.GetId(), true, nil
		}
	}
	return "", false, nil
}

func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) readSubscription(
	ctx context.Context,
	id string,
) (*suppliersubscriptionpb.SupplierSubscription, error) {
	if uc.repositories.SupplierSubscription == nil {
		return nil, errors.New("supplier_subscription repo unavailable")
	}
	resp, err := uc.repositories.SupplierSubscription.ReadSupplierSubscription(ctx, &suppliersubscriptionpb.ReadSupplierSubscriptionRequest{
		Data: &suppliersubscriptionpb.SupplierSubscription{Id: id},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) readCostPlan(
	ctx context.Context,
	id string,
) (*costplanpb.CostPlan, error) {
	if id == "" || uc.repositories.CostPlan == nil {
		return nil, nil
	}
	resp, err := uc.repositories.CostPlan.ReadCostPlan(ctx, &costplanpb.ReadCostPlanRequest{
		Data: &costplanpb.CostPlan{Id: id},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

// lineSpec holds the to-be-inserted ExpenseRecognitionLine values pre-INSERT.
type lineSpec struct {
	Description               string
	UnitAmount                int64
	Amount                    int64
	Currency                  string
	SupplierProductCostPlanID string
}

// buildLineSpecs derives the ExpenseRecognitionLine inputs from the CostPlan +
// its SupplierProductCostPlan rows. When the plan has no per-line rate card
// (no SupplierProductCostPlan rows or repo unavailable), a single header-only
// line is created mirroring the plan's billing_amount.
func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) buildLineSpecs(
	ctx context.Context,
	sub *suppliersubscriptionpb.SupplierSubscription,
	plan *costplanpb.CostPlan,
) ([]lineSpec, int64, string, error) {
	currency := plan.GetBillingCurrency()

	// Per-line rate card path.
	if uc.repositories.SupplierProductCostPlan != nil {
		resp, err := uc.repositories.SupplierProductCostPlan.ListSupplierProductCostPlans(ctx, &supplierproductcostplanpb.ListSupplierProductCostPlansRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "cost_plan_id",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value:    plan.GetId(),
								Operator: commonpb.StringOperator_STRING_EQUALS,
							},
						},
					},
					{
						Field: "active",
						FilterType: &commonpb.TypedFilter_BooleanFilter{
							BooleanFilter: &commonpb.BooleanFilter{Value: true},
						},
					},
				},
			},
		})
		if err != nil {
			return nil, 0, "", err
		}
		if resp != nil && len(resp.GetData()) > 0 {
			specs := make([]lineSpec, 0, len(resp.GetData()))
			var total int64
			for _, line := range resp.GetData() {
				if !line.GetActive() {
					continue
				}
				amount := line.GetBillingAmount()
				lineCurrency := line.GetBillingCurrency()
				if lineCurrency == "" {
					lineCurrency = currency
				}
				specs = append(specs, lineSpec{
					Description:               sub.GetName(),
					UnitAmount:                amount,
					Amount:                    amount,
					Currency:                  lineCurrency,
					SupplierProductCostPlanID: line.GetId(),
				})
				total += amount
			}
			if total > 0 {
				return specs, total, currency, nil
			}
		}
	}

	// Header-only fallback path — single line equal to plan.billing_amount.
	total := plan.GetBillingAmount()
	if total == 0 {
		// Fallback to subscription name + zero-amount line so the recognition
		// still records something rather than silently producing 0-line draft.
		return []lineSpec{{
			Description: sub.GetName(),
			UnitAmount:  0,
			Amount:      0,
			Currency:    currency,
		}}, 0, currency, nil
	}
	return []lineSpec{{
		Description: sub.GetName(),
		UnitAmount:  total,
		Amount:      total,
		Currency:    currency,
	}}, total, currency, nil
}

func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) insertDraftExpenditure(
	ctx context.Context,
	sub *suppliersubscriptionpb.SupplierSubscription,
	supplierID, currency string,
	totalAmount int64,
	input RecognizeExpenseFromSupplierSubscriptionInput,
) (string, error) {
	id := uc.services.IDGenerator.GenerateID()
	now := time.Now()
	dc := now.UnixMilli()
	dcStr := now.Format(time.RFC3339)

	exp := &expenditurepb.Expenditure{
		Id:                 id,
		DateCreated:        &dc,
		DateCreatedString:  &dcStr,
		DateModified:       &dc,
		DateModifiedString: &dcStr,
		Active:             true,
		Name:               sub.GetName(),
		Currency:           currency,
		TotalAmount:        totalAmount,
		Status:             "draft",
	}
	source := "expense_run"
	exp.Source = &source
	if supplierID != "" {
		exp.SupplierId = &supplierID
	}
	subID := sub.GetId()
	exp.SupplierSubscriptionId = &subID
	if input.RunID != "" {
		runID := input.RunID
		exp.RunId = &runID
	}

	resp, err := uc.repositories.Expenditure.CreateExpenditure(ctx, &expenditurepb.CreateExpenditureRequest{
		Data: exp,
	})
	if err != nil {
		return "", fmt.Errorf("create draft expenditure: %w", err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0].GetId(), nil
	}
	return id, nil
}

func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) insertRecognition(
	ctx context.Context,
	sub *suppliersubscriptionpb.SupplierSubscription,
	supplierID, workspaceID, currency string,
	totalAmount int64,
	expenditureID, idempotencyKey string,
	input RecognizeExpenseFromSupplierSubscriptionInput,
) (string, error) {
	id := uc.services.IDGenerator.GenerateID()
	now := time.Now()
	dc := now.UnixMilli()
	dcStr := now.Format(time.RFC3339)

	ps, _ := time.Parse("2006-01-02", input.PeriodStart)
	pe, _ := time.Parse("2006-01-02", input.PeriodEnd)
	periodStart := timestamppb.New(ps)
	periodEnd := timestamppb.New(pe)
	recognitionDate := timestamppb.New(pe)

	rec := &expenserecognitionpb.ExpenseRecognition{
		Id:                 id,
		WorkspaceId:        workspaceID,
		DateCreated:        &dc,
		DateCreatedString:  &dcStr,
		DateModified:       &dc,
		DateModifiedString: &dcStr,
		Active:             true,
		Name:               fmt.Sprintf("%s — %s → %s", sub.GetName(), input.PeriodStart, input.PeriodEnd),
		RecognitionDate:    recognitionDate,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		Currency:           currency,
		TotalAmount:        totalAmount,
		Status:             expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_DRAFT,
		IdempotencyKey:     idempotencyKey,
	}
	subID := sub.GetId()
	rec.SupplierSubscriptionId = &subID
	if expenditureID != "" {
		expID := expenditureID
		rec.ExpenditureId = &expID
	}
	if supplierID != "" {
		rec.SupplierId = &supplierID
	}
	if input.RunID != "" {
		runID := input.RunID
		rec.RunId = &runID
	}
	notes := buildSubscriptionPeriodMarker(input.PeriodStart, input.PeriodEnd)
	if notes != "" {
		rec.Notes = &notes
	}

	resp, err := uc.repositories.ExpenseRecognition.CreateExpenseRecognition(ctx, &expenserecognitionpb.CreateExpenseRecognitionRequest{
		Data: rec,
	})
	if err != nil {
		return "", fmt.Errorf("create expense_recognition: %w", err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0].GetId(), nil
	}
	return id, nil
}

func (uc *RecognizeExpenseFromSupplierSubscriptionUseCase) insertLines(
	ctx context.Context,
	recognitionID, workspaceID string,
	sub *suppliersubscriptionpb.SupplierSubscription,
	specs []lineSpec,
) error {
	if uc.repositories.ExpenseRecognitionLine == nil {
		return nil
	}
	now := time.Now()
	dc := now.UnixMilli()
	dcStr := now.Format(time.RFC3339)
	subID := sub.GetId()

	for _, spec := range specs {
		lineID := uc.services.IDGenerator.GenerateID()
		line := &expenserecognitionlinepb.ExpenseRecognitionLine{
			Id:                   lineID,
			WorkspaceId:          workspaceID,
			DateCreated:          &dc,
			DateCreatedString:    &dcStr,
			DateModified:         &dc,
			DateModifiedString:   &dcStr,
			Active:               true,
			ExpenseRecognitionId: recognitionID,
			Description:          spec.Description,
			UnitAmount:           spec.UnitAmount,
			Amount:               spec.Amount,
			Currency:             spec.Currency,
		}
		line.SupplierSubscriptionId = &subID
		if spec.SupplierProductCostPlanID != "" {
			scpID := spec.SupplierProductCostPlanID
			line.SupplierProductCostPlanId = &scpID
		}
		if _, err := uc.repositories.ExpenseRecognitionLine.CreateExpenseRecognitionLine(ctx, &expenserecognitionlinepb.CreateExpenseRecognitionLineRequest{
			Data: line,
		}); err != nil {
			return fmt.Errorf("create expense_recognition_line: %w", err)
		}
	}
	return nil
}

// BuildSubscriptionIdempotencyKey is the canonical key shape:
//
//	{workspace_id}:SUBSCRIPTION:{supplier_subscription_id}:{period_start}
func BuildSubscriptionIdempotencyKey(workspaceID, subID, periodStart string) string {
	return fmt.Sprintf("%s:SUBSCRIPTION:%s:%s", workspaceID, subID, periodStart)
}

func buildSubscriptionPeriodMarker(start, end string) string {
	switch {
	case start != "" && end != "":
		return fmt.Sprintf("Period: %s → %s", start, end)
	case start != "":
		return fmt.Sprintf("Period: %s →", start)
	case end != "":
		return fmt.Sprintf("Period: → %s", end)
	default:
		return ""
	}
}

func erroredExp(err error) *RecognizeExpenseFromSupplierSubscriptionOutput {
	return &RecognizeExpenseFromSupplierSubscriptionOutput{
		Outcome: RecognizeExpenseOutcomeErrored,
		Error:   err,
	}
}
