package job_template_summary

import (
	"context"
	"errors"

	summarypb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/job_template_summary"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// query is the port the job-template-summary use case reads from. It is exactly
// the GENERATED operationv1.JobTemplateSummaryServiceServer interface
// (Q-PROTO-MODE: service{rpc}) — no hand-written port. The postgres adapter
// embeds UnimplementedJobTemplateSummaryServiceServer and satisfies this
// directly; on mock/non-postgres builds the port is nil and Execute degrades to
// an empty, successful response.
type query = summarypb.JobTemplateSummaryServiceServer

// ListJobTemplateSummariesRepositories groups infrastructure dependencies.
type ListJobTemplateSummariesRepositories struct {
	Query query
}

// ListJobTemplateSummariesServices groups application services.
type ListJobTemplateSummariesServices struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListJobTemplateSummariesUseCase serves the resolver-scoped, template-grain
// delivery summary. It gatekeeps the read on job:list, then delegates to the
// adapter port. Row scoping is entirely resolver-level inside the adapter
// (principalscope.StaffReachableJobClause narrows STAFF principals to their
// reachable jobs — the subscription_seat tier widens that to the class grain;
// non-staff principals see all). There is NO request-side staff filter and no
// MINE/ALL scope enum: unlike outcome_matrix, the summary carries no per-row
// edit semantics, so scope narrowing is unconditional in the adapter.
type ListJobTemplateSummariesUseCase struct {
	repositories ListJobTemplateSummariesRepositories
	services     ListJobTemplateSummariesServices
}

// NewListJobTemplateSummariesUseCase wires the use case. Any dep may be nil;
// Execute degrades to an empty, successful response when the Query port is
// missing.
func NewListJobTemplateSummariesUseCase(
	repositories ListJobTemplateSummariesRepositories,
	services ListJobTemplateSummariesServices,
) *ListJobTemplateSummariesUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &ListJobTemplateSummariesUseCase{repositories: repositories, services: services}
}

// Execute runs the summary read.
//
//	(a) ActionGatekeeper.Check(job, list) — the base read gate. Check has a
//	    nil-receiver guard that DENIES, so a mis-wired nil gatekeeper fails
//	    closed instead of skipping the gate.
//	(b) delegate to the adapter port (nil → empty, successful response).
func (uc *ListJobTemplateSummariesUseCase) Execute(
	ctx context.Context,
	req *summarypb.ListJobTemplateSummariesRequest,
) (*summarypb.ListJobTemplateSummariesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Job,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"job_template_summary.validation.request_required",
			"job template summary request is required"))
	}

	if uc.repositories.Query == nil {
		// No provider registered (mock / non-postgres) — empty, successful.
		return &summarypb.ListJobTemplateSummariesResponse{Success: true}, nil
	}

	return uc.repositories.Query.ListJobTemplateSummaries(ctx, req)
}
