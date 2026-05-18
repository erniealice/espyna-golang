package expenserecognitionrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_run"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"

	expenserecognition "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expense_recognition"
	treasurydisbursement "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement"
)

// SelectedExpenseRunCandidate is the internal Go-struct mirror of the proto.
type SelectedExpenseRunCandidate struct {
	SourceKind             expenserecognitionrunpb.ExpenseRecognitionRunSourceKind
	SupplierSubscriptionID string
	AdvanceDisbursementID  string
	PeriodStart            string
	PeriodEnd              string
	PeriodMarker           string
}

// runAttemptRecord holds the in-memory outcome for one selection.
type runAttemptRecord struct {
	outcome              expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome
	sourceKind           expenserecognitionrunpb.ExpenseRecognitionRunSourceKind
	supplierSubID        string
	advanceDisbID        string
	start                string
	end                  string
	marker               string
	expenseRecognitionID *string
	expenditureID        *string
	errCode              *string
	errMsg               *string
}

// ExpenseRecognitionRunAttemptWriter is the narrow port for inserting one
// ExpenseRecognitionRunAttempt row.
//
// REQUIRED ADAPTER METHOD (not yet on the proto service interface).
// The Phase 0 proto for ExpenseRecognitionRunDomainService omits the Attempt
// CRUD methods that the selling-side RevenueRunDomainService exposes; the
// adapter agent is expected to either:
//
//	(a) implement this narrow interface separately and wire a wrapper into
//	    GenerateExpenseRunRepositories.AttemptWriter, OR
//	(b) extend the proto service to expose CreateExpenseRecognitionRunAttempt
//	    (parity with revenue_run.proto:82) and regenerate bindings.
//
// Until (a) or (b) lands, the run engine writes the parent ExpenseRecognitionRun
// row but the per-selection Attempt rows are returned in-memory only — the
// view layer can render them from the response, but they won't survive a
// page reload.
type ExpenseRecognitionRunAttemptWriter interface {
	CreateExpenseRecognitionRunAttempt(
		ctx context.Context,
		req *expenserecognitionrunpb.CreateExpenseRecognitionRunAttemptRequest,
	) (*expenserecognitionrunpb.CreateExpenseRecognitionRunAttemptResponse, error)
}

// GenerateExpenseRunRepositories — composes both single-recognition use case
// repos plus the run repo itself.
type GenerateExpenseRunRepositories struct {
	ExpenseRecognition    expenserecognitionpb.ExpenseRecognitionDomainServiceServer
	ExpenseRecognitionRun expenserecognitionrunpb.ExpenseRecognitionRunDomainServiceServer
	// AttemptWriter is optional — see the interface comment for context.
	// When nil, Attempt rows are produced in-memory only (returned in the
	// GenerateExpenseRunResponse) and not persisted.
	AttemptWriter ExpenseRecognitionRunAttemptWriter
}

// GenerateExpenseRunServices groups infra services.
type GenerateExpenseRunServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// GenerateExpenseRunUseCase is the buying-side mirror of GenerateRevenueRun.
//
// Per the Plan A hard rules: COMPOSE, DON'T DUPLICATE. The run engine does
// NOT re-implement the INSERT logic — it dispatches each selection to one of:
//   - RecognizeExpenseFromSupplierSubscription (subscription cycles)
//   - AmortizeAdvanceDisbursement (advance disbursement tranches)
type GenerateExpenseRunUseCase struct {
	repositories                      GenerateExpenseRunRepositories
	services                          GenerateExpenseRunServices
	recognizeFromSupplierSubscription *expenserecognition.RecognizeExpenseFromSupplierSubscriptionUseCase
	amortizeAdvanceDisbursement       *treasurydisbursement.AmortizeAdvanceDisbursementUseCase
}

// NewGenerateExpenseRunUseCase wires the use case.
func NewGenerateExpenseRunUseCase(
	repos GenerateExpenseRunRepositories,
	svcs GenerateExpenseRunServices,
	recognizeFromSupplierSubscription *expenserecognition.RecognizeExpenseFromSupplierSubscriptionUseCase,
	amortizeAdvanceDisbursement *treasurydisbursement.AmortizeAdvanceDisbursementUseCase,
) *GenerateExpenseRunUseCase {
	return &GenerateExpenseRunUseCase{
		repositories:                      repos,
		services:                          svcs,
		recognizeFromSupplierSubscription: recognizeFromSupplierSubscription,
		amortizeAdvanceDisbursement:       amortizeAdvanceDisbursement,
	}
}

// ExpenseRecognitionRunRepo returns the run repository — used by view layers
// that need to pass through to ListExpenseRecognitionRuns etc. without
// exposing internal types.
func (uc *GenerateExpenseRunUseCase) ExpenseRecognitionRunRepo() expenserecognitionrunpb.ExpenseRecognitionRunDomainServiceServer {
	if uc == nil {
		return nil
	}
	return uc.repositories.ExpenseRecognitionRun
}

// Execute runs the batch expense generation process.
func (uc *GenerateExpenseRunUseCase) Execute(
	ctx context.Context,
	req *expenserecognitionrunpb.GenerateExpenseRunRequest,
) (*expenserecognitionrunpb.GenerateExpenseRunResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Translate scope + selections from proto to Go structs.
	scope := ExpenseRunScope{}
	if s := req.GetScope(); s != nil {
		scope.WorkspaceID = s.GetWorkspaceId()
		scope.SupplierID = s.GetSupplierId()
		scope.SupplierSubscriptionID = s.GetSupplierSubscriptionId()
		scope.AsOfDate = s.GetAsOfDate()
	}
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		scope.WorkspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}

	var selections []SelectedExpenseRunCandidate
	if s := req.GetSelections(); s != nil {
		if s.GetFilterToken() != "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.TranslationService,
				"expense_recognition.errors.filter_token_not_implemented",
				"filter_token is not implemented [DEFAULT]",
			))
		}
		for _, e := range s.GetExplicitList() {
			selections = append(selections, SelectedExpenseRunCandidate{
				SourceKind:             e.GetSourceKind(),
				SupplierSubscriptionID: e.GetSupplierSubscriptionId(),
				AdvanceDisbursementID:  e.GetAdvanceDisbursementId(),
				PeriodStart:            e.GetPeriodStart(),
				PeriodEnd:              e.GetPeriodEnd(),
				PeriodMarker:           e.GetPeriodMarker(),
			})
		}
	}
	if len(selections) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"expense_recognition.validation.no_selections",
			"At least one selection is required [DEFAULT]",
		))
	}

	asOfDate := strings.TrimSpace(scope.AsOfDate)
	if asOfDate == "" {
		asOfDate = time.Now().UTC().Format("2006-01-02")
	}

	initiator := contextutil.ExtractWorkspaceUserIDFromContext(ctx)
	runID := uc.services.IDService.GenerateID()
	now := time.Now().UTC().UnixMilli()

	run := &expenserecognitionrunpb.ExpenseRecognitionRun{
		Id:             runID,
		WorkspaceId:    scope.WorkspaceID,
		Scope:          uc.resolveScopeKind(scope),
		AsOfDate:       asOfDate,
		SelectionCount: int32(len(selections)),
		Status:         expenserecognitionrunpb.ExpenseRecognitionRunStatus_EXPENSE_RECOGNITION_RUN_STATUS_PENDING,
		InitiatedBy:    initiator,
		InitiatedAt:    &now,
		Active:         true,
	}
	if scope.SupplierID != "" {
		s := scope.SupplierID
		run.SupplierId = &s
	}
	if scope.SupplierSubscriptionID != "" {
		s := scope.SupplierSubscriptionID
		run.SupplierSubscriptionId = &s
	}

	createdRunResp, err := uc.repositories.ExpenseRecognitionRun.CreateExpenseRecognitionRun(ctx, &expenserecognitionrunpb.CreateExpenseRecognitionRunRequest{
		Data: run,
	})
	if err != nil {
		return nil, err
	}
	if createdRunResp == nil || len(createdRunResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"expense_recognition.errors.run_create_failed",
			"Failed to create expense recognition run record [DEFAULT]",
		))
	}
	run = createdRunResp.GetData()[0]

	var accumulator []runAttemptRecord
	var insertedAttempts []*expenserecognitionrunpb.ExpenseRecognitionRunAttempt

	for _, sel := range selections {
		acc := uc.processSelection(ctx, run, sel, scope)
		accumulator = append(accumulator, acc)

		// INSERT attempt row.
		attemptID := uc.services.IDService.GenerateID()
		if attemptID == "" {
			attemptID = "err-id-" + sel.SupplierSubscriptionID + sel.AdvanceDisbursementID
		}
		attemptTime := time.Now().UTC().UnixMilli()
		attempt := &expenserecognitionrunpb.ExpenseRecognitionRunAttempt{
			Id:                   attemptID,
			RunId:                run.GetId(),
			SourceKind:           acc.sourceKind,
			PeriodStart:          acc.start,
			PeriodEnd:            acc.end,
			PeriodMarker:         acc.marker,
			Outcome:              acc.outcome,
			ExpenseRecognitionId: acc.expenseRecognitionID,
			ExpenditureId:        acc.expenditureID,
			ErrorCode:            acc.errCode,
			ErrorMessage:         acc.errMsg,
			AttemptedAt:          &attemptTime,
			Active:               true,
		}
		if acc.supplierSubID != "" {
			s := acc.supplierSubID
			attempt.SupplierSubscriptionId = &s
		}
		if acc.advanceDisbID != "" {
			a := acc.advanceDisbID
			attempt.AdvanceDisbursementId = &a
		}
		if uc.repositories.AttemptWriter != nil {
			createdAttemptResp, insertErr := uc.repositories.AttemptWriter.CreateExpenseRecognitionRunAttempt(ctx, &expenserecognitionrunpb.CreateExpenseRecognitionRunAttemptRequest{
				Data: attempt,
			})
			if insertErr == nil && createdAttemptResp != nil && len(createdAttemptResp.GetData()) > 0 {
				attempt = createdAttemptResp.GetData()[0]
			}
		}
		insertedAttempts = append(insertedAttempts, attempt)
	}

	// Aggregate counts.
	var createdCount, skippedCount, erroredCount int32
	for _, a := range accumulator {
		switch a.outcome {
		case expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_CREATED:
			createdCount++
		case expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_SKIPPED:
			skippedCount++
		case expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED:
			erroredCount++
		}
	}
	finalStatus := expenserecognitionrunpb.ExpenseRecognitionRunStatus_EXPENSE_RECOGNITION_RUN_STATUS_COMPLETE
	if erroredCount > 0 {
		finalStatus = expenserecognitionrunpb.ExpenseRecognitionRunStatus_EXPENSE_RECOGNITION_RUN_STATUS_FAILED
	}

	completedAt := time.Now().UTC().UnixMilli()
	run.CreatedCount = createdCount
	run.SkippedCount = skippedCount
	run.ErroredCount = erroredCount
	run.Status = finalStatus
	run.CompletedAt = &completedAt

	_, _ = uc.repositories.ExpenseRecognitionRun.UpdateExpenseRecognitionRun(ctx, &expenserecognitionrunpb.UpdateExpenseRecognitionRunRequest{
		Data: run,
	})

	return &expenserecognitionrunpb.GenerateExpenseRunResponse{
		Success:  true,
		Run:      run,
		Attempts: insertedAttempts,
	}, nil
}

// processSelection dispatches to the right inner use case based on source_kind.
func (uc *GenerateExpenseRunUseCase) processSelection(
	ctx context.Context,
	run *expenserecognitionrunpb.ExpenseRecognitionRun,
	sel SelectedExpenseRunCandidate,
	scope ExpenseRunScope,
) runAttemptRecord {
	switch sel.SourceKind {
	case expenserecognitionrunpb.ExpenseRecognitionRunSourceKind_EXPENSE_RECOGNITION_RUN_SOURCE_KIND_SUBSCRIPTION_CYCLE:
		return uc.dispatchSubscriptionCycle(ctx, run, sel, scope)
	case expenserecognitionrunpb.ExpenseRecognitionRunSourceKind_EXPENSE_RECOGNITION_RUN_SOURCE_KIND_ADVANCE_DISBURSEMENT:
		return uc.dispatchAdvanceDisbursement(ctx, run, sel, scope)
	default:
		return runAttemptRecord{
			outcome:    expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED,
			sourceKind: sel.SourceKind,
			start:      sel.PeriodStart,
			end:        sel.PeriodEnd,
			marker:     sel.PeriodMarker,
			errCode:    strPtr("unsupported_source_kind"),
			errMsg:     strPtr(fmt.Sprintf("source_kind=%v is not supported by GenerateExpenseRun", sel.SourceKind)),
		}
	}
}

func (uc *GenerateExpenseRunUseCase) dispatchSubscriptionCycle(
	ctx context.Context,
	run *expenserecognitionrunpb.ExpenseRecognitionRun,
	sel SelectedExpenseRunCandidate,
	scope ExpenseRunScope,
) runAttemptRecord {
	if uc.recognizeFromSupplierSubscription == nil {
		return runAttemptRecord{
			outcome:       expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED,
			sourceKind:    sel.SourceKind,
			supplierSubID: sel.SupplierSubscriptionID,
			start:         sel.PeriodStart,
			end:           sel.PeriodEnd,
			marker:        sel.PeriodMarker,
			errCode:       strPtr("recognizer_unavailable"),
			errMsg:        strPtr("RecognizeExpenseFromSupplierSubscription is not wired"),
		}
	}

	out, err := uc.recognizeFromSupplierSubscription.Execute(ctx, expenserecognition.RecognizeExpenseFromSupplierSubscriptionInput{
		SupplierSubscriptionID: sel.SupplierSubscriptionID,
		PeriodStart:            sel.PeriodStart,
		PeriodEnd:              sel.PeriodEnd,
		PeriodMarker:           sel.PeriodMarker,
		WorkspaceID:            scope.WorkspaceID,
		RunID:                  run.GetId(),
	})
	if err != nil {
		return runAttemptRecord{
			outcome:       expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED,
			sourceKind:    sel.SourceKind,
			supplierSubID: sel.SupplierSubscriptionID,
			start:         sel.PeriodStart,
			end:           sel.PeriodEnd,
			marker:        sel.PeriodMarker,
			errCode:       strPtr("recognition_failed"),
			errMsg:        strPtr(err.Error()),
		}
	}
	switch out.Outcome {
	case expenserecognition.RecognizeExpenseOutcomeCreated:
		recID := out.ExpenseRecognitionID
		expID := out.ExpenditureID
		return runAttemptRecord{
			outcome:              expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_CREATED,
			sourceKind:           sel.SourceKind,
			supplierSubID:        sel.SupplierSubscriptionID,
			start:                sel.PeriodStart,
			end:                  sel.PeriodEnd,
			marker:               sel.PeriodMarker,
			expenseRecognitionID: &recID,
			expenditureID:        &expID,
		}
	case expenserecognition.RecognizeExpenseOutcomeSkipped:
		var conflict *string
		if out.ConflictingExpenseRecognitionID != "" {
			c := out.ConflictingExpenseRecognitionID
			conflict = &c
		}
		return runAttemptRecord{
			outcome:              expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_SKIPPED,
			sourceKind:           sel.SourceKind,
			supplierSubID:        sel.SupplierSubscriptionID,
			start:                sel.PeriodStart,
			end:                  sel.PeriodEnd,
			marker:               sel.PeriodMarker,
			expenseRecognitionID: conflict,
			errCode:              strPtr("period_already_recognized"),
		}
	default:
		errCode := "recognition_errored"
		errMsg := "subscription-cycle recognition returned errored outcome"
		if out.Error != nil {
			errMsg = out.Error.Error()
		}
		return runAttemptRecord{
			outcome:       expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED,
			sourceKind:    sel.SourceKind,
			supplierSubID: sel.SupplierSubscriptionID,
			start:         sel.PeriodStart,
			end:           sel.PeriodEnd,
			marker:        sel.PeriodMarker,
			errCode:       strPtr(errCode),
			errMsg:        strPtr(errMsg),
		}
	}
}

func (uc *GenerateExpenseRunUseCase) dispatchAdvanceDisbursement(
	ctx context.Context,
	run *expenserecognitionrunpb.ExpenseRecognitionRun,
	sel SelectedExpenseRunCandidate,
	scope ExpenseRunScope,
) runAttemptRecord {
	if uc.amortizeAdvanceDisbursement == nil {
		return runAttemptRecord{
			outcome:       expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED,
			sourceKind:    sel.SourceKind,
			advanceDisbID: sel.AdvanceDisbursementID,
			start:         sel.PeriodStart,
			end:           sel.PeriodEnd,
			marker:        sel.PeriodMarker,
			errCode:       strPtr("amortizer_unavailable"),
			errMsg:        strPtr("AmortizeAdvanceDisbursement is not wired"),
		}
	}

	runID := run.GetId()
	out, err := uc.amortizeAdvanceDisbursement.Execute(ctx, &disbursementpb.AmortizeAdvanceDisbursementRequest{
		TreasuryDisbursementId: sel.AdvanceDisbursementID,
		AsOfDate:               scope.AsOfDate,
		WorkspaceId:            scope.WorkspaceID,
		RunId:                  &runID,
	})
	if err != nil {
		return runAttemptRecord{
			outcome:       expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED,
			sourceKind:    sel.SourceKind,
			advanceDisbID: sel.AdvanceDisbursementID,
			start:         sel.PeriodStart,
			end:           sel.PeriodEnd,
			marker:        sel.PeriodMarker,
			errCode:       strPtr("amortize_failed"),
			errMsg:        strPtr(err.Error()),
		}
	}
	switch out.GetOutcome() {
	case advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_CREATED:
		recID := out.GetExpenseRecognitionId()
		return runAttemptRecord{
			outcome:              expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_CREATED,
			sourceKind:           sel.SourceKind,
			advanceDisbID:        sel.AdvanceDisbursementID,
			start:                sel.PeriodStart,
			end:                  sel.PeriodEnd,
			marker:               sel.PeriodMarker,
			expenseRecognitionID: &recID,
		}
	case advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED:
		var conflict *string
		if c := out.GetConflictingExpenseRecognitionId(); c != "" {
			cc := c
			conflict = &cc
		}
		return runAttemptRecord{
			outcome:              expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_SKIPPED,
			sourceKind:           sel.SourceKind,
			advanceDisbID:        sel.AdvanceDisbursementID,
			start:                sel.PeriodStart,
			end:                  sel.PeriodEnd,
			marker:               sel.PeriodMarker,
			expenseRecognitionID: conflict,
			errCode:              strPtr("period_already_recognized"),
		}
	default:
		errMsg := "advance amortization returned errored outcome"
		if e := out.GetError(); e != nil && e.GetMessage() != "" {
			errMsg = e.GetMessage()
		}
		return runAttemptRecord{
			outcome:       expenserecognitionrunpb.ExpenseRecognitionRunAttemptOutcome_EXPENSE_RECOGNITION_RUN_ATTEMPT_OUTCOME_ERRORED,
			sourceKind:    sel.SourceKind,
			advanceDisbID: sel.AdvanceDisbursementID,
			start:         sel.PeriodStart,
			end:           sel.PeriodEnd,
			marker:        sel.PeriodMarker,
			errCode:       strPtr("amortize_errored"),
			errMsg:        strPtr(errMsg),
		}
	}
}

func (uc *GenerateExpenseRunUseCase) resolveScopeKind(scope ExpenseRunScope) expenserecognitionrunpb.ExpenseRecognitionRunScope {
	if scope.SupplierSubscriptionID != "" {
		return expenserecognitionrunpb.ExpenseRecognitionRunScope_EXPENSE_RECOGNITION_RUN_SCOPE_SUBSCRIPTION
	}
	if scope.SupplierID != "" {
		return expenserecognitionrunpb.ExpenseRecognitionRunScope_EXPENSE_RECOGNITION_RUN_SCOPE_SUPPLIER
	}
	return expenserecognitionrunpb.ExpenseRecognitionRunScope_EXPENSE_RECOGNITION_RUN_SCOPE_WORKSPACE
}

func strPtr(s string) *string { return &s }
