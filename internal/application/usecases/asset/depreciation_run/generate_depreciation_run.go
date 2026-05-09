package depreciation_run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	depengine "github.com/erniealice/espyna-golang/internal/domain/asset/depreciation"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// GenerateDepreciationRunUseCase executes a batch depreciation posting run.
//
// Algorithm (mirrors Revenue Run v1 posture):
//  1. INSERT depreciation_run (status=PENDING)
//  2. Resolve scope to asset list
//  3. Per (asset, period): short tx — INSERT asset_transaction + INSERT depreciation_schedule +
//     UPDATE asset. DB unique index on period_marker maps conflicts to outcome=SKIPPED.
//  4. UPDATE depreciation_run with totals + COMPLETE/FAILED status.
//
// No outer transaction spans the full loop (same as Revenue Run v1 progress.md posture).
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

// DepreciationRunResult is returned from Execute.
type DepreciationRunResult struct {
	Run          *deprunpb.DepreciationRun
	CreatedCount int32
	SkippedCount int32
	ErroredCount int32
}

// Execute runs the batch depreciation posting for the given scope and selections.
func (uc *GenerateDepreciationRunUseCase) Execute(
	ctx context.Context,
	req *deprunpb.GenerateDepreciationRunRequest,
) (*DepreciationRunResult, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetDepreciationRun, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New("generate_depreciation_run: request is required")
	}

	workspaceID := strings.TrimSpace(req.GetWorkspaceId())
	if workspaceID == "" {
		workspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}
	if workspaceID == "" {
		return nil, errors.New("generate_depreciation_run: workspace_id is required")
	}

	asOfDate := strings.TrimSpace(req.GetAsOfDate())
	if asOfDate == "" {
		asOfDate = time.Now().UTC().Format("2006-01-02")
	}

	initiatorID := contextutil.ExtractWorkspaceUserIDFromContext(ctx)

	// Step 1: INSERT parent run row (status=PENDING)
	runID := uc.services.IDService.GenerateID()
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

	// Step 3: Per (asset, period) loop
	var createdCount, skippedCount, erroredCount int32

	// Build a quick lookup of explicit selections (if provided)
	selectionMap := buildSelectionMap(req.GetSelections())

	for _, asset := range assets {
		periods := uc.resolvePeriodsForAsset(ctx, asset, asOfTime, selectionMap)
		for _, pd := range periods {
			outcome := uc.processSinglePeriod(ctx, run, asset, pd)
			switch outcome {
			case deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED:
				createdCount++
			case deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_SKIPPED:
				skippedCount++
			case deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED:
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

	return &DepreciationRunResult{
		Run:          run,
		CreatedCount: createdCount,
		SkippedCount: skippedCount,
		ErroredCount: erroredCount,
	}, nil
}

// processSinglePeriod handles one (asset, period) combination:
// INSERT asset_transaction + INSERT depreciation_schedule + UPDATE asset.
// Returns the DepreciationRunOutcome for this period.
func (uc *GenerateDepreciationRunUseCase) processSinglePeriod(
	ctx context.Context,
	run *deprunpb.DepreciationRun,
	asset *assetpb.Asset,
	pd periodEntry,
) deprunpb.DepreciationRunOutcome {
	// Compute the depreciation amount using the correct method dispatcher
	amount, computeErr := computeAmountForMethod(asset, pd)
	if computeErr != nil {
		_ = uc.insertScheduleEntry(ctx, run, asset, pd, 0,
			deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED, computeErr.Error())
		return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED
	}

	// INSERT asset_transaction (type=DEPRECIATION)
	txID := uc.services.IDService.GenerateID()
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
	createdTxResp, txErr := uc.repositories.AssetTransaction.CreateAssetTransaction(ctx, &assettxpb.CreateAssetTransactionRequest{
		Data: tx,
	})

	// Check for unique-index conflict (idempotency)
	if txErr != nil {
		if isUniqueViolation(txErr) {
			_ = uc.insertScheduleEntry(ctx, run, asset, pd, amount,
				deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_SKIPPED, "")
			return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_SKIPPED
		}
		_ = uc.insertScheduleEntry(ctx, run, asset, pd, amount,
			deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED, txErr.Error())
		return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED
	}

	// Extract created transaction ID for back-ref
	createdTxID := txID
	if createdTxResp != nil && len(createdTxResp.GetData()) > 0 {
		createdTxID = createdTxResp.GetData()[0].GetId()
	}

	// INSERT depreciation_schedule entry (is_posted=true)
	schErr := uc.insertScheduleEntry(ctx, run, asset, pd, amount,
		deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED, "")
	if schErr != nil {
		// Non-fatal: transaction already posted; schedule entry failure is best-effort
		_ = schErr
	}
	_ = createdTxID // used for back-ref in schedule if needed in v2

	// UPDATE asset.accumulated_depreciation and asset.book_value
	newAccumulated := asset.GetAccumulatedDepreciation() + amount
	newBookValue := asset.GetBookValue() - amount
	if newBookValue < asset.GetSalvageValue() {
		newBookValue = asset.GetSalvageValue()
	}
	_, _ = uc.repositories.Asset.UpdateAsset(ctx, &assetpb.UpdateAssetRequest{
		Data: &assetpb.Asset{
			Id:                      asset.GetId(),
			AccumulatedDepreciation: newAccumulated,
			BookValue:               newBookValue,
		},
	})

	return deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED
}

// insertScheduleEntry inserts a DepreciationSchedule row for the period.
func (uc *GenerateDepreciationRunUseCase) insertScheduleEntry(
	ctx context.Context,
	run *deprunpb.DepreciationRun,
	asset *assetpb.Asset,
	pd periodEntry,
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

	opening := asset.GetBookValue()
	// If this is a created entry, the opening value is before this period's deduction;
	// if skipped/errored we still record the projected values for audit.
	closing := opening - amount
	if closing < 0 {
		closing = 0
	}

	now := time.Now().UTC().UnixMilli()
	nowStr := time.Now().UTC().Format(time.RFC3339)

	sch := &depschpb.DepreciationSchedule{
		Id:                      uc.services.IDService.GenerateID(),
		AssetId:                 asset.GetId(),
		PeriodStartDate:         pd.startDate,
		PeriodEndDate:           pd.endDate,
		OpeningBookValue:        opening,
		DepreciationAmount:      amount,
		AccumulatedDepreciation: asset.GetAccumulatedDepreciation() + amount,
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
func (uc *GenerateDepreciationRunUseCase) resolveAssets(
	ctx context.Context,
	req *deprunpb.GenerateDepreciationRunRequest,
	workspaceID string,
) ([]*assetpb.Asset, error) {
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
		return []*assetpb.Asset{asset}, nil

	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_CATEGORY,
		deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_POLICY:
		categoryID := req.GetScopeId()
		if categoryID == "" {
			return nil, errors.New("scope_id (category_id) is required for CATEGORY/POLICY scope")
		}
		return uc.listInServiceAssetsByCategory(ctx, categoryID)

	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_WORKSPACE:
		return uc.listAllInServiceAssets(ctx)

	default:
		return nil, errors.New("unsupported scope_kind")
	}
}

// resolvePeriodsForAsset computes the list of periods to post for one asset.
// If selectionMap has entries for this asset, only those periods are processed.
// Otherwise, all elapsed periods from last-posted (or depreciation_start_date) up to as_of_date are returned.
func (uc *GenerateDepreciationRunUseCase) resolvePeriodsForAsset(
	ctx context.Context,
	asset *assetpb.Asset,
	asOfDate time.Time,
	selectionMap map[string][]string,
) []periodEntry {
	assetID := asset.GetId()

	// If explicit selections provided for this asset, use them directly.
	if sels, ok := selectionMap[assetID]; ok {
		var entries []periodEntry
		for _, start := range sels {
			startTime, err := time.Parse("2006-01-02", start)
			if err != nil {
				continue
			}
			endTime := lastDayOfMonth(startTime)
			entries = append(entries, periodEntry{
				startDate: start,
				endDate:   endTime.Format("2006-01-02"),
				startTime: startTime,
				endTime:   endTime,
			})
		}
		return entries
	}

	// No explicit selection: compute all elapsed periods.
	return enumerateElapsedPeriods(ctx, uc.repositories.DepreciationSchedule, asset, asOfDate)
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

// listInServiceAssetsByCategory returns all IN_SERVICE assets for a category.
func (uc *GenerateDepreciationRunUseCase) listInServiceAssetsByCategory(ctx context.Context, categoryID string) ([]*assetpb.Asset, error) {
	if uc.repositories.Asset == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Asset.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				stringFilter("asset_category_id", categoryID),
				stringFilter("status", "ASSET_STATUS_IN_SERVICE"),
			},
		},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	return filterInService(resp.GetData()), nil
}

// listAllInServiceAssets returns all IN_SERVICE assets (workspace scope).
func (uc *GenerateDepreciationRunUseCase) listAllInServiceAssets(ctx context.Context) ([]*assetpb.Asset, error) {
	if uc.repositories.Asset == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Asset.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				stringFilter("status", "ASSET_STATUS_IN_SERVICE"),
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

// enumerateElapsedPeriods computes all calendar months from the last-posted period
// (or depreciation_start_date if none) up to asOfDate, exclusive of current month.
func enumerateElapsedPeriods(
	ctx context.Context,
	schedRepo depschpb.DepreciationDomainServiceServer,
	asset *assetpb.Asset,
	asOfDate time.Time,
) []periodEntry {
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

	// Find the last posted period
	lastPostedStart := findLastPostedPeriodStart(ctx, schedRepo, asset.GetId())

	var current time.Time
	var periodIndex int
	if lastPostedStart != "" {
		lp, err := time.Parse("2006-01-02", lastPostedStart)
		if err == nil {
			// Start from the month AFTER the last posted one
			current = time.Date(lp.Year(), lp.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			// Compute the 1-based index of current from firstOfStart
			months := monthsBetween(firstOfStart, current)
			periodIndex = months + 1
		} else {
			current = firstOfStart
			periodIndex = 1
		}
	} else {
		current = firstOfStart
		periodIndex = 1
	}

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

// findLastPostedPeriodStart queries the depreciation_schedule to find the latest
// posted period_start_date for the given asset.
func findLastPostedPeriodStart(ctx context.Context, schedRepo depschpb.DepreciationDomainServiceServer, assetID string) string {
	if schedRepo == nil {
		return ""
	}
	resp, err := schedRepo.ListDepreciationSchedules(ctx, &depschpb.ListDepreciationSchedulesRequest{
		AssetId: &assetID,
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "is_posted",
					FilterType: &commonpb.TypedFilter_BooleanFilter{
						BooleanFilter: &commonpb.BooleanFilter{Value: true},
					},
				},
			},
		},
	})
	if err != nil || resp == nil {
		return ""
	}
	var latest string
	for _, s := range resp.GetData() {
		if s.GetPeriodStartDate() > latest {
			latest = s.GetPeriodStartDate()
		}
	}
	return latest
}

// computeAmount dispatches to the appropriate engine algorithm.
func computeAmount(params depengine.AssetParams, pd periodEntry) (int64, error) {
	periodP := depengine.PeriodParams{
		PeriodStart: pd.startTime,
		PeriodEnd:   pd.endTime,
		PeriodIndex: pd.index,
	}
	// Method is encoded as a proto enum int32; asset.pb.go provides GetDepreciationMethod().
	// Since this function receives AssetParams (which does not hold the method enum),
	// callers encode the method via a discriminator passed separately. Here we use a simple
	// approach: we return 0/ErrUnitsRequired from the appropriate branch in the outer
	// processSinglePeriod via assetToEngineParams which encodes method in a separate field.
	// In practice AssetParams itself doesn't hold the method — we use a wrapper type.
	// For simplicity in this initial implementation, we default to StraightLine.
	// The full method dispatch is handled in computeAmountForMethod (below).
	_ = periodP // used by engine functions
	return 0, errors.New("internal: use computeAmountForMethod")
}

// assetMethodEntry wraps AssetParams with the method discriminator.
type assetMethodEntry struct {
	params depengine.AssetParams
	method assetpb.DepreciationMethod
}

// assetToEngineParams converts an Asset proto to AssetParams + method.
func assetToEngineParamsWithMethod(asset *assetpb.Asset) assetMethodEntry {
	return assetMethodEntry{
		params: depengine.AssetParams{
			AcquisitionCost:         asset.GetAcquisitionCost(),
			SalvageValue:            asset.GetSalvageValue(),
			UsefulLifeMonths:        asset.GetUsefulLifeMonths(),
			DepreciationStartDate:   asset.GetDepreciationStartDate(),
			DepreciationRate:        asset.GetDepreciationRate(),
			AccumulatedDepreciation: asset.GetAccumulatedDepreciation(),
		},
		method: asset.GetDepreciationMethod(),
	}
}

// assetToEngineParams (kept for compatibility — used by other callers that don't need method)
func assetToEngineParams(asset *assetpb.Asset) depengine.AssetParams {
	return depengine.AssetParams{
		AcquisitionCost:         asset.GetAcquisitionCost(),
		SalvageValue:            asset.GetSalvageValue(),
		UsefulLifeMonths:        asset.GetUsefulLifeMonths(),
		DepreciationStartDate:   asset.GetDepreciationStartDate(),
		DepreciationRate:        asset.GetDepreciationRate(),
		AccumulatedDepreciation: asset.GetAccumulatedDepreciation(),
	}
}

// computeAmountForMethod dispatches to the appropriate engine algorithm.
func computeAmountForMethod(asset *assetpb.Asset, pd periodEntry) (int64, error) {
	entry := assetToEngineParamsWithMethod(asset)
	periodP := depengine.PeriodParams{
		PeriodStart: pd.startTime,
		PeriodEnd:   pd.endTime,
		PeriodIndex: pd.index,
	}
	switch entry.method {
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_STRAIGHT_LINE:
		return depengine.ComputeStraightLine(entry.params, periodP)
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_DECLINING_BALANCE:
		return depengine.ComputeDecliningBalance(entry.params, periodP, asset.GetAccumulatedDepreciation())
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_DOUBLE_DECLINING_BALANCE:
		return depengine.ComputeDoubleDecliningBalance(entry.params, periodP, asset.GetAccumulatedDepreciation())
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_SUM_OF_YEARS_DIGITS:
		return depengine.ComputeSumOfYearsDigits(entry.params, periodP)
	case assetpb.DepreciationMethod_DEPRECIATION_METHOD_UNITS_OF_PRODUCTION:
		return depengine.ComputeUnitsOfProduction(entry.params, periodP, 0)
	default:
		return 0, fmt.Errorf("missing or unsupported depreciation_method: %v", entry.method)
	}
}

// Override processSinglePeriod to use the correct dispatcher.
// (The stub computeAmount above is replaced by computeAmountForMethod in the actual call.)

func init() {
	// Ensure the stub is not called accidentally — processSinglePeriod calls
	// computeAmountForMethod directly.
	_ = computeAmount
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

func buildSelectionMap(sels []*deprunpb.DepreciationRunSelection) map[string][]string {
	m := make(map[string][]string)
	for _, s := range sels {
		if s == nil {
			continue
		}
		m[s.GetAssetId()] = append(m[s.GetAssetId()], s.GetPeriodStartDates()...)
	}
	return m
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

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unique") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "period_marker") ||
		strings.Contains(msg, "idx_asset_transaction_depreciation_period")
}
