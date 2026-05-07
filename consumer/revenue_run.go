package consumer

import (
	"context"

	revenueUC "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue"
	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
)

// Re-export types so view packages can use them without importing espyna internals.

// RevenueRunScope is the public scope type for revenue run operations.
// Views build this directly and pass it to ListRevenueRunCandidates or GenerateRevenueRun.
type RevenueRunScope = revenueUC.RevenueRunScope

// RevenueRunCandidate is the public candidate type returned by ListRevenueRunCandidates.
type RevenueRunCandidate = revenueUC.RevenueRunCandidate

// SelectedRevenueRunCandidate is one confirmed selection for GenerateRevenueRun.
type SelectedRevenueRunCandidate = revenueUC.SelectedRevenueRunCandidate

// RevenueRunSelections carries the operator's picks for GenerateRevenueRun.
type RevenueRunSelections = revenueUC.RevenueRunSelections

// RevenueRunResult is the output of GenerateRevenueRun.
type RevenueRunResult = revenueUC.RevenueRunResult

// ListRevenueRunCandidates enumerates un-invoiced billing periods for the given
// scope. The returned slice is empty (not nil) when there are no candidates.
//
// View packages call this function directly via a callback injected by block.go.
// No espyna internal types are exposed — scope and candidates are plain Go structs.
func ListRevenueRunCandidates(
	useCases *UseCases,
	ctx context.Context,
	scope RevenueRunScope,
) ([]RevenueRunCandidate, string, error) {
	if useCases == nil || useCases.Revenue == nil || useCases.Revenue.Revenue == nil {
		return nil, "", nil
	}
	uc := useCases.Revenue.Revenue.ListRevenueRunCandidates
	if uc == nil {
		return nil, "", nil
	}
	candidates, nextCursor, err := uc.Execute(ctx, scope)
	if candidates == nil {
		candidates = []RevenueRunCandidate{}
	}
	return candidates, nextCursor, err
}

// GenerateRevenueRun executes a batch revenue generation run for the given scope
// and selections. Returns the created RevenueRun + all attempt records.
func GenerateRevenueRun(
	useCases *UseCases,
	ctx context.Context,
	scope RevenueRunScope,
	selections RevenueRunSelections,
) (*RevenueRunResult, error) {
	if useCases == nil || useCases.Revenue == nil || useCases.Revenue.Revenue == nil {
		return nil, nil
	}
	uc := useCases.Revenue.Revenue.GenerateRevenueRun
	if uc == nil {
		return nil, nil
	}
	initiator := GetWorkspaceUserIDFromContext(ctx)
	return uc.Execute(ctx, scope, selections, initiator)
}

// ListRevenueRuns is a proto pass-through to the RevenueRunDomainService.
// View packages use this to render the run history list page (Surface D).
func ListRevenueRuns(
	useCases *UseCases,
	ctx context.Context,
	req *revenuerunpb.ListRevenueRunsRequest,
) (*revenuerunpb.ListRevenueRunsResponse, error) {
	repo := revenueRunRepo(useCases)
	if repo == nil {
		return &revenuerunpb.ListRevenueRunsResponse{Success: true}, nil
	}
	return repo.ListRevenueRuns(ctx, req)
}

// ReadRevenueRun is a proto pass-through to the RevenueRunDomainService.
// View packages use this to render the run detail page (Surface D).
func ReadRevenueRun(
	useCases *UseCases,
	ctx context.Context,
	req *revenuerunpb.ReadRevenueRunRequest,
) (*revenuerunpb.ReadRevenueRunResponse, error) {
	repo := revenueRunRepo(useCases)
	if repo == nil {
		return &revenuerunpb.ReadRevenueRunResponse{}, nil
	}
	return repo.ReadRevenueRun(ctx, req)
}

// ListRevenueRunAttempts is a proto pass-through to the RevenueRunDomainService.
// View packages use this to render the attempt list on a run detail page.
func ListRevenueRunAttempts(
	useCases *UseCases,
	ctx context.Context,
	req *revenuerunpb.ListRevenueRunAttemptsRequest,
) (*revenuerunpb.ListRevenueRunAttemptsResponse, error) {
	repo := revenueRunRepo(useCases)
	if repo == nil {
		return &revenuerunpb.ListRevenueRunAttemptsResponse{}, nil
	}
	return repo.ListRevenueRunAttempts(ctx, req)
}

// revenueRunRepo is a nil-safe helper that drills into the use-case aggregate
// to retrieve the RevenueRun repository. Returns nil when the repo is not wired
// (e.g. mock_db composition that hasn't registered a revenue_run adapter).
func revenueRunRepo(useCases *UseCases) revenuerunpb.RevenueRunDomainServiceServer {
	if useCases == nil || useCases.Revenue == nil || useCases.Revenue.Revenue == nil {
		return nil
	}
	uc := useCases.Revenue.Revenue.GenerateRevenueRun
	if uc == nil {
		return nil
	}
	return uc.RevenueRunRepo()
}
