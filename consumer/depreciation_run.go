package consumer

import (
	"context"

	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// Re-export types so view packages can use them without importing espyna internals.

// DepreciationRunResult is the output of GenerateDepreciationRun.
// Re-exported from the use case package for view-layer use.

// ListDepreciationCandidates returns per-asset depreciation candidate rows for the
// given scope. This is the dry-run (no writes) engine call.
// View packages call this directly via a callback injected by block.go.
func ListDepreciationCandidates(
	useCases *UseCases,
	ctx context.Context,
	req *deprunpb.ListDepreciationCandidatesRequest,
) (*deprunpb.ListDepreciationCandidatesResponse, error) {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.DepreciationRun == nil {
		return &deprunpb.ListDepreciationCandidatesResponse{Success: true}, nil
	}
	uc := useCases.Asset.DepreciationRun.ListDepreciationCandidates
	if uc == nil {
		return &deprunpb.ListDepreciationCandidatesResponse{Success: true}, nil
	}
	return uc.Execute(ctx, req)
}

// GenerateDepreciationRun executes a batch depreciation run for the given scope.
// Returns the created DepreciationRun + counts.
func GenerateDepreciationRun(
	useCases *UseCases,
	ctx context.Context,
	req *deprunpb.GenerateDepreciationRunRequest,
) (*deprunpb.GenerateDepreciationRunResponse, error) {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.DepreciationRun == nil {
		return &deprunpb.GenerateDepreciationRunResponse{Success: true}, nil
	}
	uc := useCases.Asset.DepreciationRun.GenerateDepreciationRun
	if uc == nil {
		return &deprunpb.GenerateDepreciationRunResponse{Success: true}, nil
	}
	result, err := uc.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &deprunpb.GenerateDepreciationRunResponse{Success: true}, nil
	}
	return &deprunpb.GenerateDepreciationRunResponse{
		Run:          result.Run,
		CreatedCount: result.CreatedCount,
		SkippedCount: result.SkippedCount,
		ErroredCount: result.ErroredCount,
		Success:      true,
	}, nil
}

// ListDepreciationRuns returns paginated depreciation run history.
func ListDepreciationRuns(
	useCases *UseCases,
	ctx context.Context,
	req *deprunpb.ListDepreciationRunsRequest,
) (*deprunpb.ListDepreciationRunsResponse, error) {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.DepreciationRun == nil {
		return &deprunpb.ListDepreciationRunsResponse{Success: true}, nil
	}
	uc := useCases.Asset.DepreciationRun.ListDepreciationRuns
	if uc == nil {
		return &deprunpb.ListDepreciationRunsResponse{Success: true}, nil
	}
	return uc.Execute(ctx, req)
}

// ReadDepreciationRun reads a single depreciation run by ID.
func ReadDepreciationRun(
	useCases *UseCases,
	ctx context.Context,
	req *deprunpb.ReadDepreciationRunRequest,
) (*deprunpb.ReadDepreciationRunResponse, error) {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.DepreciationRun == nil {
		return &deprunpb.ReadDepreciationRunResponse{}, nil
	}
	uc := useCases.Asset.DepreciationRun.ReadDepreciationRun
	if uc == nil {
		return &deprunpb.ReadDepreciationRunResponse{}, nil
	}
	return uc.Execute(ctx, req)
}

// ListDepreciationRunEntries lists DepreciationSchedule rows for a given run_id.
// Used by the Surface D run-detail page entries tab.
func ListDepreciationRunEntries(
	useCases *UseCases,
	ctx context.Context,
	runID string,
	pagination *commonpb.PaginationRequest,
) (*depschpb.ListDepreciationSchedulesResponse, error) {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.DepreciationRun == nil {
		return &depschpb.ListDepreciationSchedulesResponse{Success: true}, nil
	}
	uc := useCases.Asset.DepreciationRun.ListDepreciationRunEntries
	if uc == nil {
		return &depschpb.ListDepreciationSchedulesResponse{Success: true}, nil
	}
	return uc.Execute(ctx, runID, pagination)
}
