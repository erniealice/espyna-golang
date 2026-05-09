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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListDepreciationCandidatesRepositories groups all repository dependencies.
type ListDepreciationCandidatesRepositories struct {
	Asset                assetpb.AssetDomainServiceServer
	AssetCategory        assetcategorypb.AssetCategoryDomainServiceServer
	DepreciationSchedule depschpb.DepreciationDomainServiceServer
}

// ListDepreciationCandidatesServices groups all business service dependencies.
type ListDepreciationCandidatesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListDepreciationCandidatesUseCase is the dry-run (no writes) engine.
// Returns per-asset candidate info with pending periods, projected amounts, and blockers.
type ListDepreciationCandidatesUseCase struct {
	repositories ListDepreciationCandidatesRepositories
	services     ListDepreciationCandidatesServices
}

// NewListDepreciationCandidatesUseCase wires the use case.
func NewListDepreciationCandidatesUseCase(
	repositories ListDepreciationCandidatesRepositories,
	services ListDepreciationCandidatesServices,
) *ListDepreciationCandidatesUseCase {
	return &ListDepreciationCandidatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute returns depreciation candidates for the given scope.
// No DB writes are performed — this is a pure read/compute call.
func (uc *ListDepreciationCandidatesUseCase) Execute(
	ctx context.Context,
	req *deprunpb.ListDepreciationCandidatesRequest,
) (*deprunpb.ListDepreciationCandidatesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetDepreciationRun, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New("list_depreciation_candidates: request is required")
	}

	workspaceID := strings.TrimSpace(req.GetWorkspaceId())
	if workspaceID == "" {
		workspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}

	asOfDate := strings.TrimSpace(req.GetAsOfDate())
	if asOfDate == "" {
		asOfDate = time.Now().UTC().Format("2006-01-02")
	}
	asOfTime, err := time.Parse("2006-01-02", asOfDate)
	if err != nil {
		return nil, fmt.Errorf("list_depreciation_candidates: invalid as_of_date %q: %w", asOfDate, err)
	}

	// Resolve scope to asset list
	assets, err := uc.resolveAssets(ctx, req)
	if err != nil {
		return nil, err
	}

	var candidates []*deprunpb.DepreciationCandidate
	for _, asset := range assets {
		candidate := uc.buildCandidate(ctx, asset, asOfTime)
		candidates = append(candidates, candidate)
	}

	return &deprunpb.ListDepreciationCandidatesResponse{
		Data:    candidates,
		Success: true,
	}, nil
}

// buildCandidate computes one candidate row for an asset.
func (uc *ListDepreciationCandidatesUseCase) buildCandidate(
	ctx context.Context,
	asset *assetpb.Asset,
	asOfDate time.Time,
) *deprunpb.DepreciationCandidate {
	candidate := &deprunpb.DepreciationCandidate{
		AssetId:          asset.GetId(),
		AssetName:        asset.GetName(),
		Currency:         asset.GetCurrency(),
		CurrentBookValue: asset.GetBookValue(),
	}

	// Check for blockers first
	blockers := detectBlockers(asset)
	if len(blockers) > 0 {
		candidate.Blockers = blockers
		candidate.ProjectedBookValue = asset.GetBookValue()
		return candidate
	}

	// Enumerate pending periods
	periods := enumerateElapsedPeriods(ctx, uc.repositories.DepreciationSchedule, asset, asOfDate)
	if len(periods) == 0 {
		candidate.ProjectedBookValue = asset.GetBookValue()
		return candidate
	}

	// Compute period amounts
	runningAccumulated := asset.GetAccumulatedDepreciation()
	runningBookValue := asset.GetBookValue()

	// Build a mutable copy of asset params for running balance simulation
	simulatedAsset := *asset

	for _, pd := range periods {
		amount, err := computeAmountForMethod(&simulatedAsset, pd)
		if err != nil {
			if err == depengine.ErrUnitsRequired {
				// UoP asset — return UNITS_REQUIRED blocker instead of periods
				candidate.Blockers = []*deprunpb.DepreciationCandidateBlocker{
					{
						Kind:  deprunpb.DepreciationCandidateBlocker_DEPRECIATION_CANDIDATE_BLOCKER_KIND_UNITS_REQUIRED,
						Label: "UNITS_REQUIRED",
					},
				}
				candidate.Periods = nil
				candidate.ProjectedBookValue = asset.GetBookValue()
				return candidate
			}
			continue // skip errored periods in dry-run
		}

		runningAccumulated += amount
		runningBookValue -= amount
		if runningBookValue < asset.GetSalvageValue() {
			runningBookValue = asset.GetSalvageValue()
		}

		// Update simulated asset for next period's accumulation
		simulatedAsset.AccumulatedDepreciation = runningAccumulated
		simulatedAsset.BookValue = runningBookValue

		candidate.Periods = append(candidate.Periods, &deprunpb.DepreciationCandidatePeriod{
			PeriodStartDate:    pd.startDate,
			PeriodEndDate:      pd.endDate,
			Amount:             amount,
			RunningAccumulated: runningAccumulated,
			RunningBookValue:   runningBookValue,
		})
	}

	candidate.ProjectedBookValue = runningBookValue
	return candidate
}

// detectBlockers checks an asset for conditions that prevent depreciation.
func detectBlockers(asset *assetpb.Asset) []*deprunpb.DepreciationCandidateBlocker {
	var blockers []*deprunpb.DepreciationCandidateBlocker

	// Not in service
	if asset.GetStatus() != assetpb.AssetStatus_ASSET_STATUS_IN_SERVICE {
		blockers = append(blockers, &deprunpb.DepreciationCandidateBlocker{
			Kind:  deprunpb.DepreciationCandidateBlocker_DEPRECIATION_CANDIDATE_BLOCKER_KIND_NOT_IN_SERVICE,
			Label: "NOT_IN_SERVICE",
		})
	}

	// Missing depreciation start date
	if strings.TrimSpace(asset.GetDepreciationStartDate()) == "" {
		blockers = append(blockers, &deprunpb.DepreciationCandidateBlocker{
			Kind:  deprunpb.DepreciationCandidateBlocker_DEPRECIATION_CANDIDATE_BLOCKER_KIND_NO_START_DATE,
			Label: "NO_START_DATE",
		})
	}

	// Missing method
	if asset.GetDepreciationMethod() == assetpb.DepreciationMethod_DEPRECIATION_METHOD_UNSPECIFIED {
		blockers = append(blockers, &deprunpb.DepreciationCandidateBlocker{
			Kind:  deprunpb.DepreciationCandidateBlocker_DEPRECIATION_CANDIDATE_BLOCKER_KIND_MISSING_METHOD,
			Label: "MISSING_METHOD",
		})
	}

	// Units of production — surface as blocker (not an error, but requires extra data)
	if asset.GetDepreciationMethod() == assetpb.DepreciationMethod_DEPRECIATION_METHOD_UNITS_OF_PRODUCTION {
		blockers = append(blockers, &deprunpb.DepreciationCandidateBlocker{
			Kind:  deprunpb.DepreciationCandidateBlocker_DEPRECIATION_CANDIDATE_BLOCKER_KIND_UNITS_REQUIRED,
			Label: "UNITS_REQUIRED",
		})
	}

	// Zero useful life (for SL / SoYD)
	if asset.GetUsefulLifeMonths() <= 0 {
		m := asset.GetDepreciationMethod()
		if m == assetpb.DepreciationMethod_DEPRECIATION_METHOD_STRAIGHT_LINE ||
			m == assetpb.DepreciationMethod_DEPRECIATION_METHOD_SUM_OF_YEARS_DIGITS ||
			m == assetpb.DepreciationMethod_DEPRECIATION_METHOD_DOUBLE_DECLINING_BALANCE {
			blockers = append(blockers, &deprunpb.DepreciationCandidateBlocker{
				Kind:  deprunpb.DepreciationCandidateBlocker_DEPRECIATION_CANDIDATE_BLOCKER_KIND_ZERO_USEFUL_LIFE,
				Label: "ZERO_USEFUL_LIFE",
			})
		}
	}

	// Fully depreciated
	depBase := depengine.DepreciableBase(asset.GetAcquisitionCost(), asset.GetSalvageValue())
	if depBase > 0 && asset.GetAccumulatedDepreciation() >= depBase {
		blockers = append(blockers, &deprunpb.DepreciationCandidateBlocker{
			Kind:  deprunpb.DepreciationCandidateBlocker_DEPRECIATION_CANDIDATE_BLOCKER_KIND_FULLY_DEPRECIATED,
			Label: "FULLY_DEPRECIATED",
		})
	}

	return blockers
}

// resolveAssets resolves the scope to an asset list for the candidates call.
func (uc *ListDepreciationCandidatesUseCase) resolveAssets(
	ctx context.Context,
	req *deprunpb.ListDepreciationCandidatesRequest,
) ([]*assetpb.Asset, error) {
	switch req.GetScopeKind() {
	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_ASSET:
		scopeID := req.GetScopeId()
		if scopeID == "" {
			return nil, errors.New("scope_id required for ASSET scope")
		}
		resp, err := uc.repositories.Asset.ReadAsset(ctx, &assetpb.ReadAssetRequest{
			Data: &assetpb.Asset{Id: scopeID},
		})
		if err != nil || resp == nil || len(resp.GetData()) == 0 {
			return nil, err
		}
		return resp.GetData(), nil

	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_CATEGORY,
		deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_POLICY:
		scopeID := req.GetScopeId()
		if scopeID == "" {
			return nil, errors.New("scope_id required for CATEGORY/POLICY scope")
		}
		resp, err := uc.repositories.Asset.ListAssets(ctx, &assetpb.ListAssetsRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					stringFilter("asset_category_id", scopeID),
				},
			},
		})
		if err != nil || resp == nil {
			return nil, err
		}
		return resp.GetData(), nil

	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_WORKSPACE,
		deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_UNSPECIFIED:
		resp, err := uc.repositories.Asset.ListAssets(ctx, &assetpb.ListAssetsRequest{})
		if err != nil || resp == nil {
			return nil, err
		}
		return resp.GetData(), nil

	default:
		return nil, fmt.Errorf("unsupported scope_kind: %v", req.GetScopeKind())
	}
}
