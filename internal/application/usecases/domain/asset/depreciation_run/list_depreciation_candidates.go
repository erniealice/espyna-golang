package depreciation_run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
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
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAssetDepreciationRun,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New("list_depreciation_candidates: request is required")
	}

	// Phase 1.6 — 2026-05-10 — codex C1.5 (tenancy bypass close):
	// Reject cross-tenant attempts before resolving the workspace.
	ctxWorkspaceID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	reqWorkspaceID := strings.TrimSpace(req.GetWorkspaceId())
	if ctxWorkspaceID != "" && reqWorkspaceID != "" && ctxWorkspaceID != reqWorkspaceID {
		// TODO: translate via Translator (Phase 7.3/8.2 owns lyngua wiring).
		return nil, fmt.Errorf("list_depreciation_candidates: workspace context and request do not match")
	}
	workspaceID := reqWorkspaceID
	if workspaceID == "" {
		workspaceID = ctxWorkspaceID
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

	// Enumerate pending periods. Per codex C3 (2026-05-10) the schedule-based
	// pre-check no longer pre-filters posted periods; the dry-run candidate
	// view uses the same enumeration as the writer and tolerates already-posted
	// periods being shown in the candidate list (UI annotates them).
	periods := enumerateElapsedPeriods(asset, asOfDate)
	if len(periods) == 0 {
		candidate.ProjectedBookValue = asset.GetBookValue()
		return candidate
	}

	// Compute period amounts
	runningAccumulated := asset.GetAccumulatedDepreciation()
	runningBookValue := asset.GetBookValue()

	// Build a mutable copy of asset params for running balance simulation.
	// Use proto.Clone to safely copy the proto message without copying its
	// embedded sync.Mutex (protoimpl.MessageState).
	simulatedAsset := proto.Clone(asset).(*assetpb.Asset)

	for _, pd := range periods {
		amount, err := computeAmountForMethod(simulatedAsset, pd, runningAccumulated)
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
//
// Workspace tenancy (Phase 1 — 2026-05-10): explicit workspace_id filter is
// applied to every list query as defense-in-depth, in addition to the
// WorkspaceAwareOperations decorator's auto-injection. Empty workspace_id is
// rejected for any scope (including ASSET) because cross-tenant lookups are
// never permitted in the asset graph (codex C2).
func (uc *ListDepreciationCandidatesUseCase) resolveAssets(
	ctx context.Context,
	req *deprunpb.ListDepreciationCandidatesRequest,
) ([]*assetpb.Asset, error) {
	workspaceID := strings.TrimSpace(req.GetWorkspaceId())
	if workspaceID == "" {
		workspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}
	if workspaceID == "" {
		// TODO: translate via Translator (Fix #4 deferred — codex L1).
		// Suggested key: asset.assetDetail.depreciationRun.errors.workspaceRequired
		return nil, errors.New("list_depreciation_candidates: Workspace context required")
	}

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
		// Defense-in-depth: verify the asset belongs to the requested workspace.
		// WorkspaceAwareOperations.Read already enforces this, but a stale row
		// without a workspace_id should still be rejected here.
		if got := strings.TrimSpace(resp.GetData()[0].GetWorkspaceId()); got != workspaceID {
			return nil, fmt.Errorf("asset %q does not belong to workspace %q", scopeID, workspaceID)
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
					stringFilter("workspace_id", workspaceID),
				},
			},
		})
		if err != nil || resp == nil {
			return nil, err
		}
		return resp.GetData(), nil

	case deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_WORKSPACE,
		deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_UNSPECIFIED:
		resp, err := uc.repositories.Asset.ListAssets(ctx, &assetpb.ListAssetsRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					stringFilter("workspace_id", workspaceID),
				},
			},
		})
		if err != nil || resp == nil {
			return nil, err
		}
		return resp.GetData(), nil

	default:
		return nil, fmt.Errorf("unsupported scope_kind: %v", req.GetScopeKind())
	}
}
