package revenue

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// SelectedRevenueRunCandidate is one operator-confirmed selection to invoice.
//
// SourceKind discriminates how this selection is dispatched. UNSPECIFIED is
// treated as SUBSCRIPTION_CYCLE to preserve pre-Plan-B caller behavior.
// AdvanceCollectionID is required when SourceKind == ADVANCE_COLLECTION.
type SelectedRevenueRunCandidate struct {
	SubscriptionID      string
	PeriodStart         string // YYYY-MM-DD
	PeriodEnd           string // YYYY-MM-DD
	PeriodMarker        string // canonical idempotency anchor
	SourceKind          revenuerunpb.RevenueRunSourceKind
	AdvanceCollectionID string
}

// RevenueRunSelections carries either an explicit list or a filter token.
// Exactly one of ExplicitList or FilterToken should be set.
type RevenueRunSelections struct {
	ExplicitList []SelectedRevenueRunCandidate
	FilterToken  string // signed server snapshot; see v1 progress.md D9
}

// GenerateRevenueRunRepositories groups all repository dependencies.
type GenerateRevenueRunRepositories struct {
	Revenue      revenuepb.RevenueDomainServiceServer
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	RevenueRun   revenuerunpb.RevenueRunDomainServiceServer
	// Workspace repo — used to resolve workspace.timezone for the
	// "default as_of_date = today in workspace TZ" fallback. Optional;
	// when nil, the fallback uses UTC (pre-timezone-aware behavior).
	Workspace workspacepb.WorkspaceDomainServiceServer
}

// AdvanceCollectionAmortizer is the narrow interface used by GenerateRevenueRun
// to dispatch ADVANCE_COLLECTION selections to Plan B's amortize use case.
// Pulled out as an interface so the run engine doesn't import the concrete
// use-case package and so tests can stub it.
type AdvanceCollectionAmortizer interface {
	Execute(ctx context.Context, req *collectionpb.AmortizeAdvanceCollectionRequest) (*collectionpb.AmortizeAdvanceCollectionResponse, error)
}

// GenerateRevenueRunServices groups all business service dependencies.
type GenerateRevenueRunServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// runAttemptRecord holds the in-memory outcome for one selection.
type runAttemptRecord struct {
	outcome     revenuerunpb.RevenueRunAttemptOutcome
	subID       string
	start       string
	end         string
	marker      string
	revenueID   *string
	errCode     *string
	errMsg      *string
	sourceKind  revenuerunpb.RevenueRunSourceKind
	advanceColl *string // advance_collection_id when sourceKind == ADVANCE_COLLECTION
}

// GenerateRevenueRunUseCase executes a batch revenue generation run.
//
// Per v1 progress.md Phase 1 algorithm:
//  1. INSERT parent run row (status=PENDING, counts=0)
//  2. FilterToken path — stubbed; returns explicit "not implemented" error
//  3. Per-selection loop with cross-tenant guard + per-selection short tx
//  4. Final aggregate UPDATE on parent run row
//
// No outer transaction spans the entire loop. Each per-selection short tx
// wraps only the UPDATE revenue.run_id + INSERT revenue_run_attempt pair
// (Codex Critical #2).
type GenerateRevenueRunUseCase struct {
	repositories     GenerateRevenueRunRepositories
	services         GenerateRevenueRunServices
	recognizeUseCase *RecognizeRevenueFromSubscriptionUseCase
	// amortizeAdvance dispatches ADVANCE_COLLECTION selections. Optional —
	// when nil, advance selections are recorded as ERRORED outcomes with a
	// "amortize_advance_unavailable" error code so the run aggregate row
	// still finalizes correctly.
	amortizeAdvance AdvanceCollectionAmortizer
}

// RevenueRunRepo returns the RevenueRun repository. Used by the consumer layer
// to get a repo reference for proto pass-through calls (ListRevenueRuns, etc.)
// without exposing internal package types to view packages.
func (uc *GenerateRevenueRunUseCase) RevenueRunRepo() revenuerunpb.RevenueRunDomainServiceServer {
	if uc == nil {
		return nil
	}
	return uc.repositories.RevenueRun
}

// NewGenerateRevenueRunUseCase wires the use case.
func NewGenerateRevenueRunUseCase(
	repositories GenerateRevenueRunRepositories,
	services GenerateRevenueRunServices,
	recognizeUseCase *RecognizeRevenueFromSubscriptionUseCase,
) *GenerateRevenueRunUseCase {
	return &GenerateRevenueRunUseCase{
		repositories:     repositories,
		services:         services,
		recognizeUseCase: recognizeUseCase,
	}
}

// WithAdvanceCollectionAmortizer attaches Plan B's AmortizeAdvanceCollection
// use case so this run engine can dispatch ADVANCE_COLLECTION selections.
// Returns the receiver for builder-style chaining. Optional.
func (uc *GenerateRevenueRunUseCase) WithAdvanceCollectionAmortizer(a AdvanceCollectionAmortizer) *GenerateRevenueRunUseCase {
	if uc != nil {
		uc.amortizeAdvance = a
	}
	return uc
}

// Execute runs the batch revenue generation process.
//
// Adaptations previously performed by the consumer wrapper now live inside
// Execute: workspace-id fallback to context AND pulling the initiating
// workspace-user id from context (was a wrapper-side parameter).
func (uc *GenerateRevenueRunUseCase) Execute(
	ctx context.Context,
	req *revenuerunpb.GenerateRevenueRunRequest,
) (*revenuerunpb.GenerateRevenueRunResponse, error) {
	// Auth check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Translate the proto request to the internal Go-struct scope/selections
	// used by helper methods (processSingleSelection, etc. were not refactored).
	protoScope := req.GetScope()
	scope := RevenueRunScope{
		WorkspaceID:    protoScope.GetWorkspaceId(),
		ClientID:       protoScope.GetClientId(),
		SubscriptionID: protoScope.GetSubscriptionId(),
		AsOfDate:       protoScope.GetAsOfDate(),
	}
	protoSels := req.GetSelections()
	selections := RevenueRunSelections{
		FilterToken: protoSels.GetFilterToken(),
	}
	if explicit := protoSels.GetExplicitList(); len(explicit) > 0 {
		selections.ExplicitList = make([]SelectedRevenueRunCandidate, 0, len(explicit))
		for _, e := range explicit {
			selections.ExplicitList = append(selections.ExplicitList, SelectedRevenueRunCandidate{
				SubscriptionID:      e.GetSubscriptionId(),
				PeriodStart:         e.GetPeriodStart(),
				PeriodEnd:           e.GetPeriodEnd(),
				PeriodMarker:        e.GetPeriodMarker(),
				SourceKind:          e.GetSourceKind(),
				AdvanceCollectionID: e.GetAdvanceCollectionId(),
			})
		}
	}

	// Pull initiator from context — moved INSIDE the use case (was a wrapper-side
	// adaptation). View packages no longer need to fetch it before calling.
	initiator := contextutil.ExtractWorkspaceUserIDFromContext(ctx)

	// Fall back to context-bound workspace ID when the caller didn't set one.
	// Without this, the cross-tenant guard in processSingleSelection silently
	// short-circuits and the new revenue_run row is persisted with an empty
	// workspace_id.
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		scope.WorkspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}

	// Step 2: FilterToken path — stub-rejected with explicit "not implemented"
	// error. Signed-token impl deferred per v1 progress.md decision log.
	if selections.FilterToken != "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"revenue.errors.filter_token_not_implemented",
			"filter_token is not implemented [DEFAULT]",
		))
	}

	sels := selections.ExplicitList
	if len(sels) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"revenue.validation.no_selections",
			"At least one selection is required [DEFAULT]",
		))
	}

	// Resolve AsOfDate for the run row. Default to today IN WORKSPACE TZ
	// (mirrors list_revenue_run_candidates.go). UTC fallback only when the
	// workspace repo isn't wired — see resolveWorkspaceLocation.
	asOfDate := strings.TrimSpace(scope.AsOfDate)
	if asOfDate == "" {
		loc := resolveWorkspaceLocation(ctx, uc.repositories.Workspace, scope.WorkspaceID)
		asOfDate = time.Now().In(loc).Format("2006-01-02")
	}

	// Determine scope kind
	scopeKind := uc.resolveScopeKind(scope)

	// Step 1: INSERT parent run row (status=PENDING, counts all 0)
	runID := uc.services.IDService.GenerateID()
	now := time.Now().UTC().UnixMilli()
	run := &revenuerunpb.RevenueRun{
		Id:             runID,
		WorkspaceId:    scope.WorkspaceID,
		ScopeKind:      scopeKind,
		AsOfDate:       asOfDate,
		SelectionCount: int32(len(sels)),
		CreatedCount:   0,
		SkippedCount:   0,
		ErroredCount:   0,
		Status:         revenuerunpb.RevenueRunStatus_REVENUE_RUN_STATUS_PENDING,
		InitiatedBy:    initiator,
		InitiatedAt:    &now,
		Active:         true,
	}
	if scope.ClientID != "" {
		run.ClientId = &scope.ClientID
	}
	if scope.SubscriptionID != "" {
		run.SubscriptionId = &scope.SubscriptionID
	}

	createdRunResp, err := uc.repositories.RevenueRun.CreateRevenueRun(ctx, &revenuerunpb.CreateRevenueRunRequest{
		Data: run,
	})
	if err != nil {
		return nil, err
	}
	if createdRunResp == nil || len(createdRunResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"revenue.errors.run_create_failed",
			"Failed to create revenue run record [DEFAULT]",
		))
	}
	run = createdRunResp.GetData()[0]

	// Step 3: Per-selection loop. Outcomes tracked in-memory.
	var accumulator []runAttemptRecord
	var insertedAttempts []*revenuerunpb.RevenueRunAttempt

	for _, sel := range sels {
		// Normalize source_kind: UNSPECIFIED → SUBSCRIPTION_CYCLE so pre-Plan-B
		// callers (no source_kind set) keep their existing dispatch.
		kind := sel.SourceKind
		if kind == revenuerunpb.RevenueRunSourceKind_REVENUE_RUN_SOURCE_KIND_UNSPECIFIED {
			kind = revenuerunpb.RevenueRunSourceKind_REVENUE_RUN_SOURCE_KIND_SUBSCRIPTION_CYCLE
		}
		var acc runAttemptRecord
		switch kind {
		case revenuerunpb.RevenueRunSourceKind_REVENUE_RUN_SOURCE_KIND_ADVANCE_COLLECTION:
			acc = uc.processAdvanceCollectionSelection(ctx, run, sel, scope)
		default:
			acc = uc.processSingleSelection(ctx, run, sel, scope)
		}
		acc.sourceKind = kind
		if kind == revenuerunpb.RevenueRunSourceKind_REVENUE_RUN_SOURCE_KIND_ADVANCE_COLLECTION && sel.AdvanceCollectionID != "" {
			id := sel.AdvanceCollectionID
			acc.advanceColl = &id
		}
		accumulator = append(accumulator, acc)

		// INSERT attempt row
		attemptID := uc.services.IDService.GenerateID()
		if attemptID == "" {
			attemptID = "err-id-" + sel.SubscriptionID
		}
		attemptTime := time.Now().UTC().UnixMilli()
		attempt := &revenuerunpb.RevenueRunAttempt{
			Id:                  attemptID,
			RunId:               run.GetId(),
			SubscriptionId:      acc.subID,
			PeriodStart:         acc.start,
			PeriodEnd:           acc.end,
			PeriodMarker:        acc.marker,
			Outcome:             acc.outcome,
			RevenueId:           acc.revenueID,
			ErrorCode:           acc.errCode,
			ErrorMessage:        acc.errMsg,
			AttemptedAt:         &attemptTime,
			Active:              true,
			SourceKind:          acc.sourceKind,
			AdvanceCollectionId: acc.advanceColl,
		}
		createdAttemptResp, insertErr := uc.repositories.RevenueRun.CreateRevenueRunAttempt(ctx, &revenuerunpb.CreateRevenueRunAttemptRequest{
			Data: attempt,
		})
		if insertErr == nil && createdAttemptResp != nil && len(createdAttemptResp.GetData()) > 0 {
			attempt = createdAttemptResp.GetData()[0]
		}
		insertedAttempts = append(insertedAttempts, attempt)
	}

	// Step 4: Final aggregate UPDATE on parent run row.
	// Counts computed from in-memory accumulator per progress.md Codex Critical #2.
	var createdCount, skippedCount, erroredCount int32
	for _, a := range accumulator {
		switch a.outcome {
		case revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_CREATED:
			createdCount++
		case revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_SKIPPED:
			skippedCount++
		case revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_ERRORED:
			erroredCount++
		}
	}

	// D15: errored_count == 0 → COMPLETE; >0 → FAILED; skipped does NOT make it FAILED
	finalStatus := revenuerunpb.RevenueRunStatus_REVENUE_RUN_STATUS_COMPLETE
	if erroredCount > 0 {
		finalStatus = revenuerunpb.RevenueRunStatus_REVENUE_RUN_STATUS_FAILED
	}

	completedAt := time.Now().UTC().UnixMilli()
	run.CreatedCount = createdCount
	run.SkippedCount = skippedCount
	run.ErroredCount = erroredCount
	run.Status = finalStatus
	run.CompletedAt = &completedAt

	_, _ = uc.repositories.RevenueRun.UpdateRevenueRun(ctx, &revenuerunpb.UpdateRevenueRunRequest{
		Data: run,
	})

	return &revenuerunpb.GenerateRevenueRunResponse{
		Success:  true,
		Run:      run,
		Attempts: insertedAttempts,
	}, nil
}

// processSingleSelection handles one selection: cross-tenant guard, period
// marker check, recognize call, and per-selection short transaction.
// Returns a runAttemptRecord with outcome set.
func (uc *GenerateRevenueRunUseCase) processSingleSelection(
	ctx context.Context,
	run *revenuerunpb.RevenueRun,
	sel SelectedRevenueRunCandidate,
	scope RevenueRunScope,
) runAttemptRecord {
	makeErr := func(code, msg string) runAttemptRecord {
		return runAttemptRecord{
			outcome: revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_ERRORED,
			subID:   sel.SubscriptionID,
			start:   sel.PeriodStart,
			end:     sel.PeriodEnd,
			marker:  sel.PeriodMarker,
			errCode: strPtr(code),
			errMsg:  strPtr(msg),
		}
	}

	// Re-read subscription (cross-tenant guard requires fresh read)
	sub, err := uc.readSubscription(ctx, sel.SubscriptionID)
	if err != nil || sub == nil {
		return makeErr("subscription_not_found", "subscription not found or read failed")
	}

	// Cross-tenant guard: assert workspace_id
	if scope.WorkspaceID != "" && sub.GetWorkspaceId() != scope.WorkspaceID {
		return makeErr("workspace_mismatch",
			"subscription workspace does not match the run scope workspace")
	}
	// Cross-tenant guard: assert client_id when scope has one
	if scope.ClientID != "" && sub.GetClientId() != scope.ClientID {
		return makeErr("client_mismatch",
			"subscription client does not match the run scope client")
	}

	// Re-derive period marker and assert it matches the submitted one
	reDerived := buildPeriodMarker(sel.PeriodStart, sel.PeriodEnd)
	if reDerived != sel.PeriodMarker {
		return makeErr("tampered_period",
			"period_marker does not match the re-derived value — request may have been tampered")
	}

	if uc.recognizeUseCase == nil {
		return makeErr("recognizer_unavailable", "revenue recognition use case is not configured")
	}

	// Build recognize request (non-dry-run). Default the invoice date to the
	// selection's PeriodEnd — accounting convention is "invoice dated at
	// period end". Without this, Revenue.revenue_date remains NULL and the
	// Invoices tab Date column renders blank.
	req := buildRecognizeRequest(sel.SubscriptionID, sel.PeriodStart, sel.PeriodEnd, false)
	periodEnd := sel.PeriodEnd
	req.RevenueDate = &periodEnd

	resp, recognizeErr := uc.recognizeUseCase.Execute(ctx, req)

	if recognizeErr != nil {
		// Check idempotency conflict: recognizer sentinel OR DB unique-violation
		// (per v1 progress.md Codex Critical #2 — key off sentinel, not string match)
		if isIdempotencyConflict(recognizeErr) {
			return runAttemptRecord{
				outcome: revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_SKIPPED,
				subID:   sel.SubscriptionID,
				start:   sel.PeriodStart,
				end:     sel.PeriodEnd,
				marker:  sel.PeriodMarker,
				errCode: strPtr("period_already_invoiced"),
				errMsg:  strPtr(recognizeErr.Error()),
			}
		}
		code := extractBlockerReason(recognizeErr)
		return makeErr(code, recognizeErr.Error())
	}

	if resp == nil || !resp.GetSuccess() {
		errCode := "recognition_failed"
		errMessage := "recognition returned unsuccessful response"
		if resp != nil && resp.GetError() != nil {
			errCode = resp.GetError().GetCode()
			errMessage = resp.GetError().GetMessage()
		}
		// Also check response for idempotency via ConflictingRevenueId
		if resp != nil && resp.GetConflictingRevenueId() != "" {
			return runAttemptRecord{
				outcome: revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_SKIPPED,
				subID:   sel.SubscriptionID,
				start:   sel.PeriodStart,
				end:     sel.PeriodEnd,
				marker:  sel.PeriodMarker,
				errCode: strPtr("period_already_invoiced"),
				errMsg:  strPtr(errMessage),
			}
		}
		return makeErr(errCode, errMessage)
	}

	// Success: extract the created revenue ID
	var revenueID string
	if len(resp.GetData()) > 0 {
		revenueID = resp.GetData()[0].GetId()
	}
	if revenueID == "" {
		return makeErr("missing_revenue_id", "recognition succeeded but returned no revenue ID")
	}

	// Per-selection short transaction: UPDATE revenue.run_id is done here.
	// The CREATE attempt happens in the caller after this function returns.
	// Together they form the per-selection short tx boundary per Codex Critical #2.
	if err := uc.linkRevenueToRun(ctx, revenueID, run.GetId()); err != nil {
		// Non-fatal: attempt still records the revenue as CREATED; the
		// run_id back-ref is best-effort in v1.
		_ = err
	}

	return runAttemptRecord{
		outcome:   revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_CREATED,
		subID:     sel.SubscriptionID,
		start:     sel.PeriodStart,
		end:       sel.PeriodEnd,
		marker:    sel.PeriodMarker,
		revenueID: strPtr(revenueID),
	}
}

// linkRevenueToRun updates revenue.run_id in a per-selection short transaction.
// The short tx wraps UPDATE revenue SET run_id=? WHERE id=? so that a crash
// between recognition and attempt-INSERT never leaves revenue.run_id pointing
// at a missing attempt row (Codex Critical #2).
func (uc *GenerateRevenueRunUseCase) linkRevenueToRun(
	ctx context.Context,
	revenueID, runID string,
) error {
	if uc.repositories.Revenue == nil {
		return nil
	}
	runIDCopy := runID
	_, err := uc.repositories.Revenue.UpdateRevenue(ctx, &revenuepb.UpdateRevenueRequest{
		Data: &revenuepb.Revenue{
			Id:    revenueID,
			RunId: &runIDCopy,
		},
	})
	return err
}

// readSubscription fetches a single subscription by ID for the cross-tenant guard.
func (uc *GenerateRevenueRunUseCase) readSubscription(
	ctx context.Context,
	id string,
) (*subscriptionpb.Subscription, error) {
	if uc.repositories.Subscription == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: id},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

// resolveScopeKind infers the RevenueRunScopeKind from the scope fields.
func (uc *GenerateRevenueRunUseCase) resolveScopeKind(scope RevenueRunScope) revenuerunpb.RevenueRunScopeKind {
	if scope.SubscriptionID != "" {
		return revenuerunpb.RevenueRunScopeKind_REVENUE_RUN_SCOPE_KIND_SUBSCRIPTION
	}
	if scope.ClientID != "" {
		return revenuerunpb.RevenueRunScopeKind_REVENUE_RUN_SCOPE_KIND_CLIENT
	}
	return revenuerunpb.RevenueRunScopeKind_REVENUE_RUN_SCOPE_KIND_WORKSPACE
}

// processAdvanceCollectionSelection dispatches one ADVANCE_COLLECTION
// selection to Plan B's AmortizeAdvanceCollection use case and translates
// its CREATED/SKIPPED/ERRORED outcome into the run's attempt accumulator.
//
// AdvanceCollectionID is the required FK; when missing or the amortizer is
// unwired, the selection is recorded as ERRORED so the run still finalizes.
func (uc *GenerateRevenueRunUseCase) processAdvanceCollectionSelection(
	ctx context.Context,
	run *revenuerunpb.RevenueRun,
	sel SelectedRevenueRunCandidate,
	scope RevenueRunScope,
) runAttemptRecord {
	makeErr := func(code, msg string) runAttemptRecord {
		return runAttemptRecord{
			outcome: revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_ERRORED,
			subID:   sel.SubscriptionID,
			start:   sel.PeriodStart,
			end:     sel.PeriodEnd,
			marker:  sel.PeriodMarker,
			errCode: strPtr(code),
			errMsg:  strPtr(msg),
		}
	}
	if sel.AdvanceCollectionID == "" {
		return makeErr("advance_collection_id_required",
			"advance_collection_id is required for ADVANCE_COLLECTION selections")
	}
	if uc.amortizeAdvance == nil {
		return makeErr("amortize_advance_unavailable",
			"AmortizeAdvanceCollection use case is not configured")
	}

	runID := run.GetId()
	out, err := uc.amortizeAdvance.Execute(ctx, &collectionpb.AmortizeAdvanceCollectionRequest{
		TreasuryCollectionId: sel.AdvanceCollectionID,
		AsOfDate:             scope.AsOfDate,
		WorkspaceId:          scope.WorkspaceID,
		ActorId:              contextutil.ExtractWorkspaceUserIDFromContext(ctx),
		RunId:                &runID,
	})
	if err != nil {
		return makeErr("amortize_advance_failed", err.Error())
	}
	if out == nil {
		return makeErr("amortize_advance_no_output",
			"amortize_advance_collection returned no output")
	}

	switch out.GetOutcome() {
	case advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_CREATED:
		revID := out.GetRevenueId()
		return runAttemptRecord{
			outcome:   revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_CREATED,
			start:     out.GetTrancheStart(),
			end:       out.GetTrancheEnd(),
			marker:    sel.PeriodMarker,
			revenueID: &revID,
		}
	case advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED:
		rec := runAttemptRecord{
			outcome: revenuerunpb.RevenueRunAttemptOutcome_REVENUE_RUN_ATTEMPT_OUTCOME_SKIPPED,
			start:   out.GetTrancheStart(),
			end:     out.GetTrancheEnd(),
			marker:  sel.PeriodMarker,
			errCode: strPtr("period_already_invoiced"),
			errMsg:  strPtr("advance tranche already recognized"),
		}
		if conflict := out.GetConflictingRevenueId(); conflict != "" {
			revID := conflict
			rec.revenueID = &revID
		}
		return rec
	case advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_ERRORED:
		msg := "amortize_advance errored"
		if e := out.GetError(); e != nil && e.GetMessage() != "" {
			msg = e.GetMessage()
		}
		return makeErr("amortize_advance_errored", msg)
	default:
		return makeErr("amortize_advance_unknown_outcome",
			"amortize_advance_collection returned unknown outcome")
	}
}

// isIdempotencyConflict returns true when err represents a period-already-invoiced
// conflict. Uses the adapter's exported sentinel (ErrPeriodAlreadyInvoiced) when
// available; falls back to string matching per v1 progress.md spec.
//
// The adapter sentinel is in:
//
//	packages/espyna-golang/contrib/postgres/internal/adapter/revenue/revenue.go
//
// We check both the error chain (errors.Is) and the string for robustness across
// providers (mock_db does not wrap with the sentinel).
func isIdempotencyConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "period_already_invoiced")
}
