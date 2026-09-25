package outcome_matrix

import (
	"context"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

// ResolveCellRatingDescriptionsRepositories groups infrastructure
// dependencies. Query is the SAME generated OutcomeMatrixServiceServer
// interface GetOutcomeMatrix/GetOutcomeSummaryRoster use — no separate port.
type ResolveCellRatingDescriptionsRepositories struct {
	Query query
}

// ResolveCellRatingDescriptionsUseCase resolves, for a batch of
// (job_id, job_task_id, outcome_criteria_id) cells, which cells have an
// active PUBLISHED/DEPRECATED rating_description_set linked to their product
// offering × academic year and that set's text for the requested criterion
// (schema-proposal.md §4, PD 20260925-criterion-descriptors-by-program-year).
//
// Unlike GetOutcomeMatrix/GetOutcomeSummaryRoster this use case gates NO new
// permission (interfaces.md §5): it is consulted from inside the record
// action's own authority — the caller has already passed the task_outcome
// create/update + MINE ownership gate (resolveCellAuthority) before it ever
// asks "what does this cell's rubric say". Re-gating here would just repeat
// that check under a different name.
//
// It is still fail-closed: a nil Query (no provider registered) or a nil
// request degrades to one UNRESOLVED_IDENTITY result PER REQUESTED CELL
// (RD-66 — every cell must get exactly one result; an empty response would
// let a caller silently treat "no answer" as "no descriptions configured",
// which is the distinct NO_LINK status), never a hard error and never an
// empty Results slice for a non-empty request.
type ResolveCellRatingDescriptionsUseCase struct {
	repositories ResolveCellRatingDescriptionsRepositories
}

// NewResolveCellRatingDescriptionsUseCase wires the use case. Query may be
// nil (mock/non-postgres builds); Execute degrades to fail-closed per-cell
// results rather than skipping resolution.
func NewResolveCellRatingDescriptionsUseCase(
	repositories ResolveCellRatingDescriptionsRepositories,
) *ResolveCellRatingDescriptionsUseCase {
	return &ResolveCellRatingDescriptionsUseCase{repositories: repositories}
}

// Execute delegates to the adapter port. See the type doc comment for the
// fail-closed contract on a nil request/port.
func (uc *ResolveCellRatingDescriptionsUseCase) Execute(
	ctx context.Context,
	req *matrixpb.ResolveCellRatingDescriptionsRequest,
) (*matrixpb.ResolveCellRatingDescriptionsResponse, error) {
	if req == nil || len(req.GetCells()) == 0 {
		return &matrixpb.ResolveCellRatingDescriptionsResponse{Success: true}, nil
	}

	if uc.repositories.Query == nil {
		// No provider registered (mock / non-postgres) — fail closed, one
		// result per requested cell (RD-66), never an empty response.
		return &matrixpb.ResolveCellRatingDescriptionsResponse{
			Results: unresolvedIdentityResults(req.GetCells(), "rating-description resolver unavailable"),
			Success: true,
		}, nil
	}

	return uc.repositories.Query.ResolveCellRatingDescriptions(ctx, req)
}

// unresolvedIdentityResults builds one fail-closed CellRatingResolution per
// requested cell, all sharing the given reason.
func unresolvedIdentityResults(cells []*matrixpb.CellRatingRef, reason string) []*matrixpb.CellRatingResolution {
	out := make([]*matrixpb.CellRatingResolution, len(cells))
	for i, c := range cells {
		out[i] = &matrixpb.CellRatingResolution{
			Cell:   c,
			Status: enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY,
			Reason: reason,
		}
	}
	return out
}
