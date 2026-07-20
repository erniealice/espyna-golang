package outcome_matrix

import (
	"context"
	"errors"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetOutcomeSummaryRosterRepositories groups infrastructure dependencies. The
// Query port is the SAME generated OutcomeMatrixServiceServer interface the
// matrix read uses (the roster RPC lives on that service) — no separate port.
type GetOutcomeSummaryRosterRepositories struct {
	Query query
}

// GetOutcomeSummaryRosterServices groups application services. ActionGatekeeper
// is REQUIRED for the job_outcome_summary:list read gate.
type GetOutcomeSummaryRosterServices struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetOutcomeSummaryRosterUseCase serves the roster-scoped composite read (per
// student: per-phase composite + stored year-final) for one job_template. It
// gates the read on job_outcome_summary:list FIRST (the summary family the read
// materializes — distinct from the matrix read's task_outcome:list), then
// delegates to the adapter port. The adapter scopes workspace + reads stored
// values verbatim (D8).
type GetOutcomeSummaryRosterUseCase struct {
	repositories GetOutcomeSummaryRosterRepositories
	services     GetOutcomeSummaryRosterServices
}

// NewGetOutcomeSummaryRosterUseCase wires the use case. Any dep may be nil;
// Execute degrades to an empty, successful response when the Query port is
// missing (mock / non-postgres builds).
func NewGetOutcomeSummaryRosterUseCase(
	repositories GetOutcomeSummaryRosterRepositories,
	services GetOutcomeSummaryRosterServices,
) *GetOutcomeSummaryRosterUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &GetOutcomeSummaryRosterUseCase{repositories: repositories, services: services}
}

// Execute runs the roster composite read.
//
//	(a) ActionGatekeeper.Check(job_outcome_summary, list) — the summary read gate
//	    (nil-receiver-safe: a mis-wired nil gatekeeper DENIES, fail-closed).
//	(b) delegate to the adapter port (nil port → empty, successful response).
func (uc *GetOutcomeSummaryRosterUseCase) Execute(
	ctx context.Context,
	req *matrixpb.GetOutcomeSummaryRosterRequest,
) (*matrixpb.GetOutcomeSummaryRosterResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobOutcomeSummary,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"outcome_matrix.validation.request_required",
			"outcome summary roster request is required"))
	}

	if uc.repositories.Query == nil {
		// No provider registered (mock / non-postgres) — empty, successful.
		return &matrixpb.GetOutcomeSummaryRosterResponse{
			JobTemplateId: req.GetJobTemplateId(),
			Success:       true,
		}, nil
	}

	return uc.repositories.Query.GetOutcomeSummaryRoster(ctx, req)
}
