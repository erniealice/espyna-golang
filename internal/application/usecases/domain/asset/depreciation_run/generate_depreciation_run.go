package depreciation_run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	depengine "github.com/erniealice/espyna-golang/internal/domain/asset/depreciation"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

const entityAssetDepreciationRun = "asset"

// GenerateDepreciationRunRepositories groups all repository dependencies.
type GenerateDepreciationRunRepositories struct {
	Asset                assetpb.AssetDomainServiceServer
	AssetCategory        assetcategorypb.AssetCategoryDomainServiceServer
	AssetTransaction     assettxpb.AssetTransactionDomainServiceServer
	DepreciationSchedule depschpb.DepreciationDomainServiceServer
	DepreciationRun      deprunpb.DepreciationRunDomainServiceServer
}

// GenerateDepreciationRunServices groups all business service dependencies.
type GenerateDepreciationRunServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// GenerateDepreciationRunUseCase executes a batch depreciation posting run.
//
// Algorithm (Phase 2 — 2026-05-10 — codex C1+C3+H1+H3 fixes):
//  1. INSERT depreciation_run (status=PENDING)
//  2. Resolve scope to asset list (workspace-tenancy enforced)
//  3. Per asset: maintain a running balance (BV + accumulated) that mutates after
//     every successful period. processSinglePeriod consumes & returns this state.
//     Each period's writes (INSERT asset_transaction + INSERT depreciation_schedule)
//     run inside one Transactor.ExecuteInTransaction call.
//     - On success: increment running balance and continue.
//     - On unique-violation (DB partial-unique on asset_id, period_marker WHERE
//     transaction_type='DEPRECIATION'): outcome=SKIPPED, running balance does
//     NOT advance, schedule row is best-effort recorded outside the failed tx.
//     - On any other error: rollback, outcome=ERRORED, running balance does NOT
//     advance, schedule row is best-effort recorded outside the failed tx.
//     The DB unique index is the SOLE idempotency decision point.
//     Pre-checking depreciation_schedule.is_posted is NOT done (codex C3).
//  4. After all periods for an asset: ONE UPDATE asset call with the final
//     running BV + accumulated_depreciation. Batched at end-of-asset to minimize
//     round trips and keep each per-period tx narrow (asset_transaction +
//     depreciation_schedule). If the asset update fails, that asset's run is
//     marked ERRORED for one synthetic count but the parent loop continues.
//  5. UPDATE depreciation_run with totals + COMPLETE/FAILED status.
type GenerateDepreciationRunUseCase struct {
	repositories GenerateDepreciationRunRepositories
	services     GenerateDepreciationRunServices
}

// DepreciationRunRepo exposes the underlying DepreciationRun repository for
// consumer-layer pass-through calls (ListDepreciationRuns, ReadDepreciationRun, etc.)
func (uc *GenerateDepreciationRunUseCase) DepreciationRunRepo() deprunpb.DepreciationRunDomainServiceServer {
	if uc == nil {
		return nil
	}
	return uc.repositories.DepreciationRun
}

// NewGenerateDepreciationRunUseCase wires the use case.
func NewGenerateDepreciationRunUseCase(
	repositories GenerateDepreciationRunRepositories,
	services GenerateDepreciationRunServices,
) *GenerateDepreciationRunUseCase {
	return &GenerateDepreciationRunUseCase{
		repositories: repositories,
		services:     services,
	}
}

// runningBalance tracks the in-memory state that mutates across an asset's
// period loop. opening_book_value / accumulated_depreciation / closing_book_value
// MUST come from this struct, never from the original asset proto (codex C1).
type runningBalance struct {
	BookValue               int64 // current opening book value for the next period
	AccumulatedDepreciation int64 // cumulative depreciation posted so far
}

// Execute runs the batch depreciation posting for the given scope and selections.
func (uc *GenerateDepreciationRunUseCase) Execute(
	ctx context.Context,
	req *deprunpb.GenerateDepreciationRunRequest,
) (*deprunpb.GenerateDepreciationRunResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAssetDepreciationRun, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New("generate_depreciation_run: request is required")
	}

	// Phase 1.6 — 2026-05-10 — codex C1.5 (tenancy bypass close):
	// Reject cross-tenant attempts before binding workspaceID into context.
	// A caller authenticated as ws-A must not be able to submit req.workspace_id=ws-B
	// and have the run execute under ws-B.
	ctxWorkspaceID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	reqWorkspaceID := strings.TrimSpace(req.GetWorkspaceId())
	if ctxWorkspaceID != "" && reqWorkspaceID != "" && ctxWorkspaceID != reqWorkspaceID {
		// TODO: translate via Translator (Phase 7.3/8.2 owns lyngua wiring).
		return nil, fmt.Errorf("generate_depreciation_run: workspace context and request do not match")
	}
	workspaceID := reqWorkspaceID
	if workspaceID == "" {
		workspaceID = ctxWorkspaceID
	}
	if workspaceID == "" {
		// TODO: translate via Translator (Fix #4 deferred — codex L1).
		// Suggested key: asset.assetDetail.depreciationRun.errors.workspaceRequired
		return nil, errors.New("generate_depreciation_run: Workspace context required")
	}

	// Fix #2 (Phase 1.5 — 2026-05-10 — codex C2): bind the resolved workspaceID
	// into the context so the WorkspaceAwareOperations decorator injects it on
	// every Create call (AssetTransaction, DepreciationSchedule). Without this,
	// request-only callers (no ctx workspace) would create NULL workspace_id rows
	// in the two child tables. Option A chosen (context binding) over Option B
	// (explicit field assignment) because it is non-invasive and consistent with
	// how Create injection works throughout the decorator.
	ctx = contextutil.WithWorkspaceID(ctx, workspaceID)

	asOfDate := strings.TrimSpace(req.GetAsOfDate())
	if asOfDate == "" {
		asOfDate = time.Now().UTC().Format("2006-01-02")
	}

	initiatorID := contextutil.ExtractWorkspaceUserIDFromContext(ctx)

	// Step 1: INSERT parent run row (status=PENDING)
	runID := uc.services.IDGenerator.GenerateID()
	now := time.Now().UTC().UnixMilli()
	scopeID := req.GetScopeId()
	run := &deprunpb.DepreciationRun{
		Id:          runID,
		WorkspaceId: workspaceID,
		ScopeKind:   req.GetScopeKind(),
		ScopeId:     &scopeID,
		AsOfDate:    asOfDate,
		InitiatorId: initiatorID,
		InitiatedAt: &now,
		Status:      deprunpb.DepreciationRunStatus_DEPRECIATION_RUN_STATUS_PENDING,
		Active:      true,
	}
	createdRunResp, err := uc.repositories.DepreciationRun.CreateDepreciationRun(ctx, &deprunpb.CreateDepreciationRunRequest{
		Data: run,
	})
	if err != nil {
		return nil, fmt.Errorf("generate_depreciation_run: failed to create run record: %w", err)
	}
	if createdRunResp == nil || len(createdRunResp.GetData()) == 0 {
		return nil, errors.New("generate_depreciation_run: run record creation returned empty response")
	}
	run = createdRunResp.GetData()[0]

	// Step 2: Resolve scope to asset list
	assets, err := uc.resolveAssets(ctx, req, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("generate_depreciation_run: scope resolution failed: %w", err)
	}

	asOfTime, err := time.Parse("2006-01-02", asOfDate)
	if err != nil {
		return nil, fmt.Errorf("generate_depreciation_run: invalid as_of_date %q: %w", asOfDate, err)
	}

	// Step 3: Per (asset, period) loop with per-asset running balance
	var createdCount, skippedCount, erroredCount int32

	// Build a quick lookup of explicit selections (if provided).
	// Empty period_start_dates for an asset key means "all elapsed periods" (codex H3).
	selectionMap, hasSelections := buildSelectionMap(req.GetSelections())

	for _, asset := range assets {
		// If selections were provided at all, only process assets that appear in
		// the selection map. Empty list (no selections) keeps the prior workspace-
		// scope behavior of processing every asset.
		if hasSelections {
			if _, ok := selectionMap[asset.GetId()]; !ok {
				continue
			}
		}

		periods := uc.resolvePeriodsForAsset(asset, asOfTime, selectionMap)
		if len(periods) == 0 {
			continue
		}

		// Initialize running balance from the asset's current state. It MUST mutate
		// only on CREATED outcomes; SKIPPED and ERRORED leave it untouched (codex C1).
		rb := runningBalance{
			BookValue:               asset.GetBookValue(),
			AccumulatedDepreciation: asset.GetAccumulatedDepreciation(),
		}

		anyCreated := false
		for _, pd := range periods {
			outcome, advance := uc.processSinglePeriod(ctx, run, asset, pd, rb)
			if outcome == deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED {
				rb = advance
				anyCreated = true
			}

			switch outcome {
			case deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED:
				createdCount++
			case deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_SKIPPED:
				skippedCount++
			case deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED:
				erroredCount++
			}
		}

		// After all periods for this asset complete: ONE UPDATE asset call writing
		// the FINAL running values (codex C1 + H1). Batched at end-of-asset to keep
		// per-period transactions narrow and minimize DB round-trips.
		// Skip if no period actually created — running balance unchanged.
		if anyCreated {
			if err := uc.persistAssetRunningBalance(ctx, asset.GetId(), rb); err != nil {
				// Asset state advance failed — record one synthetic ERRORED entry so
				// the parent run is FAILED. Continue with the remaining assets.
				erroredCount++
			}
		}
	}

	// Step 4: UPDATE run with final counts
	finalStatus := deprunpb.DepreciationRunStatus_DEPRECIATION_RUN_STATUS_COMPLETE
	if erroredCount > 0 {
		finalStatus = deprunpb.DepreciationRunStatus_DEPRECIATION_RUN_STATUS_FAILED
	}
	completedAt := time.Now().UTC().UnixMilli()
	run.CreatedCount = createdCount
	run.SkippedCount = skippedCount
	run.ErroredCount = erroredCount
	run.Status = finalStatus
	run.CompletedAt = &completedAt

	_, _ = uc.repositories.DepreciationRun.UpdateDepreciationRun(ctx, &deprunpb.UpdateDepreciationRunRequest{
		Data: run,
	})

	return &deprunpb.GenerateDepreciationRunResponse{
		Run:          run,
		CreatedCount: createdCount,
		SkippedCount: skippedCount,
		ErroredCount: erroredCount,
		Success:      true,
	}, nil
}

// processSinglePeriod handles one (asset, period) combination atomically.
//
// Returns (outcome, advancedBalance). When outcome == CREATED, advancedBalance
// holds the post-period running state and the caller MUST adopt it. For SKIPPED
// or ERRORED, the caller MUST keep its existing running balance (codex C1).
//
// Atomicity (codex H1): the asset_transaction insert and the depreciation_schedule
// insert run inside a single Transactor.ExecuteInTransaction. Any error
// in either rolls back the tx; the schedule audit row for SKIPPED/ERRORED is
// then written best-effort OUTSIDE the failed tx so the run history records
// the attempt.
//
// Idempotency (codex C3): a unique-index violation on
// (asset_id, period_marker) WHERE transaction_type='DEPRECIATION' is the SOLE
// idempotency decision point. The depreciation_schedule.is_posted column is NOT
// pre-checked.
func (uc *GenerateDepreciationRunUseCase) processSinglePeriod(
	ctx context.Context,
	run *deprunpb.DepreciationRun,
	asset *assetpb.Asset,
	pd periodEntry,
	rb runningBalance,
) (deprunpb.DepreciationRunOutcome, runningBalance) {
	// Compute the depreciation amount using the running accumulated total
	// (codex C1) — the engine MUST see the post-prior-period state, not the
	// original asset snapshot.
	amount, computeErr := computeAmountForMethod(asset, pd, rb.AccumulatedDepreciation)
	if computeErr != nil {
		_ = uc.insertScheduleEntry(ctx, run, asset, pd, rb, 0,
			deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED, computeErr.Error())
		return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED, rb
	}

	// Compute the projected post-period running state (used both for the schedule
	// row when CREATED and to advance the caller's running balance).
	advance := runningBalance{
		BookValue:               rb.BookValue - amount,
		AccumulatedDepreciation: rb.AccumulatedDepreciation + amount,
	}
	if advance.BookValue < asset.GetSalvageValue() {
		advance.BookValue = asset.GetSalvageValue()
	}

	// Run the two atomic writes inside one tx. The tx service's NoOp path runs
	// the closure inline (no real tx) — same code path for tests.
	var (
		txErr    error
		isUnique bool
	)
	innerErr := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// INSERT asset_transaction (type=DEPRECIATION)
		txID := uc.services.IDGenerator.GenerateID()
		txDate := time.Now().UTC().UnixMilli()
		txDateStr := time.Now().UTC().Format("2006-01-02")
		runID := run.GetId()
		periodStr := pd.startDate

		tx := &assettxpb.AssetTransaction{
			Id:                          txID,
			AssetId:                     asset.GetId(),
			TransactionType:             assettxpb.AssetTransactionType_ASSET_TRANSACTION_TYPE_DEPRECIATION,
			TransactionDate:             txDate,
			TransactionDateString:       txDateStr,
			Amount:                      amount,
			DepreciationRunId:           &runID,
			DepreciationPeriodStartDate: &periodStr,
			Active:                      true,
		}
		if _, err := uc.repositories.AssetTransaction.CreateAssetTransaction(txCtx, &assettxpb.CreateAssetTransactionRequest{
			Data: tx,
		}); err != nil {
			txErr = err
			isUnique = isUniqueViolation(err)
			return err
		}

		// INSERT depreciation_schedule entry (is_posted=true, outcome=CREATED).
		// Inside the tx so a schedule failure rolls back the transaction insert.
		if err := uc.insertScheduleEntry(txCtx, run, asset, pd, rb, amount,
			deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED, ""); err != nil {
			txErr = err
			return err
		}

		return nil
	})

	if innerErr != nil {
		// Tx rolled back. Decide whether SKIPPED (unique-violation on the tx insert)
		// or ERRORED (anything else). Record the audit row OUTSIDE the failed tx
		// so the run history reflects the attempt.
		if isUnique {
			_ = uc.insertScheduleEntry(ctx, run, asset, pd, rb, amount,
				deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_SKIPPED, "")
			return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_SKIPPED, rb
		}
		errMsg := innerErr.Error()
		if txErr != nil {
			errMsg = txErr.Error()
		}
		_ = uc.insertScheduleEntry(ctx, run, asset, pd, rb, amount,
			deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED, errMsg)
		return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED, rb
	}

	return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED, advance
}

// persistAssetRunningBalance writes the final running BV + accumulated values to
// the asset row in one tx. Called once per asset after its period loop completes
// (codex H1 + C1 batching decision). Returns the error so the caller can record
// an ERRORED count if it fails.
func (uc *GenerateDepreciationRunUseCase) persistAssetRunningBalance(
	ctx context.Context,
	assetID string,
	rb runningBalance,
) error {
	if uc.repositories.Asset == nil {
		return nil
	}
	return uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		_, err := uc.repositories.Asset.UpdateAsset(txCtx, &assetpb.UpdateAssetRequest{
			Data: &assetpb.Asset{
				Id:                      assetID,
				AccumulatedDepreciation: rb.AccumulatedDepreciation,
				BookValue:               rb.BookValue,
			},
		})
		return err
	})
}

// insertScheduleEntry inserts a DepreciationSchedule row for the period using
// the per-period running balance (codex C1). Opening = rb.BookValue; closing =
// opening - amount (clamped at salvage). Accumulated reflects the running total
// AFTER this period for CREATED entries; for SKIPPED/ERRORED it reflects what
// WOULD have been recorded so the audit trail is meaningful.
func (uc *GenerateDepreciationRunUseCase) insertScheduleEntry(
	ctx context.Context,
	run *deprunpb.DepreciationRun,
	asset *assetpb.Asset,
	pd periodEntry,
	rb runningBalance,
	amount int64,
	outcome deprunpb.DepreciationRunOutcome,
	errMsg string,
) error {
	if uc.repositories.DepreciationSchedule == nil {
		return nil
	}
	runID := run.GetId()
	outcomeStr := outcome.String()
	var errMsgPtr *string
	if errMsg != "" {
		errMsgPtr = &errMsg
	}

	isPosted := outcome == deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED

	opening := rb.BookValue
	closing := opening - amount
	if closing < asset.GetSalvageValue() {
		closing = asset.GetSalvageValue()
	}
	if closing < 0 {
		closing = 0
	}

	now := time.Now().UTC().UnixMilli()
	nowStr := time.Now().UTC().Format(time.RFC3339)

	sch := &depschpb.DepreciationSchedule{
		Id:                      uc.services.IDGenerator.GenerateID(),
		AssetId:                 asset.GetId(),
		PeriodStartDate:         pd.startDate,
		PeriodEndDate:           pd.endDate,
		OpeningBookValue:        opening,
		DepreciationAmount:      amount,
		AccumulatedDepreciation: rb.AccumulatedDepreciation + amount,
		ClosingBookValue:        closing,
		IsPosted:                isPosted,
		DepreciationRunId:       &runID,
		Outcome:                 &outcomeStr,
		ErrorMessage:            errMsgPtr,
		DateCreated:             &now,
		DateCreatedString:       &nowStr,
		Active:                  true,
	}
	_, err := uc.repositories.DepreciationSchedule.CreateDepreciationSchedule(ctx, &depschpb.CreateDepreciationScheduleRequest{
		Data: sch,
	})
	return err
}

// resolveAssets resolves the scope to the list of in-service assets to depreciate.
//
// Workspace tenancy (Phase 1 — 2026-05-10): explicit workspace_id filter is
// applied to every list query as defense-in-depth, in addition to the
// WorkspaceAwareOperations decorator's auto-injection. Empty workspace_id is
// rejected by Execute() before this is called. ASSET-scope reads verify the
// asset belongs to the requested workspace before returning (codex C2).
func (uc *GenerateDepreciationRunUseCase) resolveAssets(
	ctx context.Context,
	req *deprunpb.GenerateDepreciationRunRequest,
	workspaceID string,
) ([]*assetpb.Asset, error) {
	if strings.TrimSpace(workspaceID) == "" {
		// TODO: translate via Translator (Fix #4 deferred — codex L1).
		// Suggested key: asset.assetDetail.depreciationRun.errors.workspaceRequired
		return nil, errors.New("generate_depreciation_run: Workspace context required for scope resolution")
	}

	switch req.GetScopeKind() {
	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_ASSET:
		assetID := req.GetScopeId()
		if assetID == "" {
			return nil, errors.New("scope_id (asset_id) is required for ASSET scope")
		}
		asset, err := uc.readAsset(ctx, assetID)
		if err != nil || asset == nil {
			return nil, fmt.Errorf("asset %q not found: %w", assetID, err)
		}
		// Defense-in-depth: verify the asset belongs to the requested workspace.
		// WorkspaceAwareOperations.Read already enforces this, but a stale row
		// without a workspace_id should still be rejected here.
		if got := strings.TrimSpace(asset.GetWorkspaceId()); got != workspaceID {
			return nil, fmt.Errorf("asset %q does not belong to workspace %q", assetID, workspaceID)
		}
		return []*assetpb.Asset{asset}, nil

	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_CATEGORY,
		deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_POLICY:
		categoryID := req.GetScopeId()
		if categoryID == "" {
			return nil, errors.New("scope_id (category_id) is required for CATEGORY/POLICY scope")
		}
		return uc.listInServiceAssetsByCategory(ctx, categoryID, workspaceID)

	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_WORKSPACE:
		return uc.listAllInServiceAssets(ctx, workspaceID)

	default:
		return nil, errors.New("unsupported scope_kind")
	}
}

// resolvePeriodsForAsset computes the list of periods to post for one asset.
//
// Selection contract (codex H3 — fixed 2026-05-10):
//   - selectionMap == nil OR no entry for this asset_id     → enumerate all elapsed
//     periods from depreciation_start_date through as_of_date.
//   - entry exists with EMPTY period_start_dates list       → enumerate all elapsed
//     periods (Surface C/F drawer builds selections this way for category runs).
//   - entry exists with NON-EMPTY period_start_dates list   → use exactly those
//     periods, ignoring the elapsed-window enumeration.
//
// The previous behavior (treat empty list as "no periods") caused category/policy
// runs from the drawer to record zero entries — codex H3.
func (uc *GenerateDepreciationRunUseCase) resolvePeriodsForAsset(
	asset *assetpb.Asset,
	asOfDate time.Time,
	selectionMap map[string][]string,
) []periodEntry {
	assetID := asset.GetId()

	if sels, ok := selectionMap[assetID]; ok && len(sels) > 0 {
		// Explicit non-empty selection — use exactly these periods.
		var entries []periodEntry
		startDateStr := asset.GetDepreciationStartDate()
		var firstOfStart time.Time
		if startDateStr != "" {
			if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
				firstOfStart = time.Date(sd.Year(), sd.Month(), 1, 0, 0, 0, 0, time.UTC)
			}
		}
		for _, start := range sels {
			startTime, err := time.Parse("2006-01-02", start)
			if err != nil {
				continue
			}
			endTime := lastDayOfMonth(startTime)
			idx := 1
			if !firstOfStart.IsZero() {
				idx = monthsBetween(firstOfStart, startTime) + 1
			}
			entries = append(entries, periodEntry{
				startDate: start,
				endDate:   endTime.Format("2006-01-02"),
				startTime: startTime,
				endTime:   endTime,
				index:     idx,
			})
		}
		return entries
	}

	// No explicit selection (or empty list) — enumerate ALL elapsed periods from
	// depreciation_start_date through as_of_date. The DB unique index on
	// (asset_id, period_marker) WHERE transaction_type='DEPRECIATION' is the
	// idempotency safety net — already-posted periods will surface as SKIPPED,
	// not be pre-filtered (codex C3).
	return enumerateElapsedPeriods(asset, asOfDate)
}

// readAsset fetches a single asset by ID.
func (uc *GenerateDepreciationRunUseCase) readAsset(ctx context.Context, id string) (*assetpb.Asset, error) {
	if uc.repositories.Asset == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Asset.ReadAsset(ctx, &assetpb.ReadAssetRequest{
		Data: &assetpb.Asset{Id: id},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

// listInServiceAssetsByCategory returns all IN_SERVICE assets for a category
// within the given workspace (Phase 1 — 2026-05-10 — codex C2 fix).
func (uc *GenerateDepreciationRunUseCase) listInServiceAssetsByCategory(ctx context.Context, categoryID string, workspaceID string) ([]*assetpb.Asset, error) {
	if uc.repositories.Asset == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Asset.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				stringFilter("asset_category_id", categoryID),
				stringFilter("status", "ASSET_STATUS_IN_SERVICE"),
				stringFilter("workspace_id", workspaceID),
			},
		},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	return filterInService(resp.GetData()), nil
}

// listAllInServiceAssets returns all IN_SERVICE assets in the given workspace
// (Phase 1 — 2026-05-10 — codex C2 fix).
func (uc *GenerateDepreciationRunUseCase) listAllInServiceAssets(ctx context.Context, workspaceID string) ([]*assetpb.Asset, error) {
	if uc.repositories.Asset == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Asset.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				stringFilter("status", "ASSET_STATUS_IN_SERVICE"),
				stringFilter("workspace_id", workspaceID),
			},
		},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	return filterInService(resp.GetData()), nil
}

// periodEntry holds one computed depreciation period for an asset.
type periodEntry struct {
	startDate string
	endDate   string
	startTime time.Time
	endTime   time.Time
	index     int // 1-based ordinal from depreciation_start_date
}

// enumerateElapsedPeriods computes ALL calendar months from depreciation_start_date
// up to (but not including) the month that contains asOfDate. The DB unique index
// is the idempotency gate — already-posted periods become SKIPPED outcomes
// (codex C3 — drop schedule-based pre-check).
func enumerateElapsedPeriods(asset *assetpb.Asset, asOfDate time.Time) []periodEntry {
	startDateStr := asset.GetDepreciationStartDate()
	if startDateStr == "" {
		return nil
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return nil
	}
	// Normalize to first of month
	firstOfStart := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	current := firstOfStart
	periodIndex := 1

	// Truncate asOfDate to first of its month to exclude the current incomplete month
	asOfFirstOfMonth := time.Date(asOfDate.Year(), asOfDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	var entries []periodEntry
	for !current.After(asOfFirstOfMonth.AddDate(0, -1, 0)) {
		end := lastDayOfMonth(current)
		entries = append(entries, periodEntry{
			startDate: current.Format("2006-01-02"),
			endDate:   end.Format("2006-01-02"),
			startTime: current,
			endTime:   end,
			index:     periodIndex,
		})
		current = current.AddDate(0, 1, 0)
		periodIndex++
	}
	return entries
}

// computeAmountForMethod dispatches to the appropriate engine algorithm.
// runningAccumulated is the per-period running accumulated_depreciation (codex C1)
// — passed to declining-balance methods so each period sees the prior period's
// post-state, not the original asset snapshot.
func computeAmountForMethod(asset *assetpb.Asset, pd periodEntry, runningAccumulated int64) (int64, error) {
	params := depengine.AssetParams{
		AcquisitionCost:         asset.GetAcquisitionCost(),
		SalvageValue:            asset.GetSalvageValue(),
		UsefulLifeMonths:        asset.GetUsefulLifeMonths(),
		DepreciationStartDate:   asset.GetDepreciationStartDate(),
		DepreciationRate:        asset.GetDepreciationRate(),
		AccumulatedDepreciation: runningAccumulated,
	}
	periodP := depengine.PeriodParams{
		PeriodStart: pd.startTime,
		PeriodEnd:   pd.endTime,
		PeriodIndex: pd.index,
	}
	switch asset.GetDepreciationMethod() {
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_STRAIGHT_LINE:
		return depengine.ComputeStraightLine(params, periodP)
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_DECLINING_BALANCE:
		return depengine.ComputeDecliningBalance(params, periodP, runningAccumulated)
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_DOUBLE_DECLINING_BALANCE:
		return depengine.ComputeDoubleDecliningBalance(params, periodP, runningAccumulated)
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_SUM_OF_YEARS_DIGITS:
		return depengine.ComputeSumOfYearsDigits(params, periodP)
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_UNITS_OF_PRODUCTION:
		return depengine.ComputeUnitsOfProduction(params, periodP, 0)
	default:
		return 0, fmt.Errorf("missing or unsupported depreciation_method: %v", asset.GetDepreciationMethod())
	}
}

// helpers

func stringFilter(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    value,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}
}

func filterInService(assets []*assetpb.Asset) []*assetpb.Asset {
	var result []*assetpb.Asset
	for _, a := range assets {
		if a.GetStatus() == assetpb.AssetStatus_ASSET_STATUS_IN_SERVICE {
			result = append(result, a)
		}
	}
	return result
}

// buildSelectionMap returns (map, hasAnySelection). hasAnySelection is true
// if the request supplied at least one DepreciationRunSelection (regardless of
// whether its period_start_dates list is empty). Callers use the boolean to
// distinguish "no selections at all → process every resolved asset" from
// "explicit selections → process only those assets" (codex H3).
func buildSelectionMap(sels []*deprunpb.DepreciationRunSelection) (map[string][]string, bool) {
	m := make(map[string][]string)
	for _, s := range sels {
		if s == nil {
			continue
		}
		// Append nil for entries with no period_start_dates so the asset_id key
		// is still present — resolvePeriodsForAsset interprets the empty list as
		// "all elapsed periods" (codex H3).
		m[s.GetAssetId()] = append(m[s.GetAssetId()], s.GetPeriodStartDates()...)
	}
	return m, len(sels) > 0
}

func lastDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
}

func monthsBetween(from, to time.Time) int {
	months := (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
	if months < 0 {
		return 0
	}
	return months
}

// isUniqueViolation reports whether err is a Postgres unique-index violation,
// the SOLE idempotency decision point for depreciation runs (codex C3). The
// underlying driver may surface code 23505 either as a typed pgconn.PgError or
// as a wrapped string, so we inspect the message — the existing behavior
// across the espyna stack. The partial unique index
// idx_asset_transaction_depreciation_period on
// (asset_id, period_marker) WHERE transaction_type='DEPRECIATION' is the only
// constraint a re-run would trip; matching its name (or "23505" / "unique" /
// "duplicate") is sufficient.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "period_marker") ||
		strings.Contains(msg, "idx_asset_transaction_depreciation_period")
}
