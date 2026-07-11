// Package job_template_summary hosts the service-driven job-template-summary
// read use case (proto: service/operation/job_template_summary). It is the
// generic, cross-vertical server-side aggregate that collapses the education-
// tier class-list (and any deliverable-template roster summary) into ONE
// GROUP-BY read: one JobTemplateSummary row per job_template with >=1
// resolver-scoped job for the requested status.
//
// Per Q-PROTO-MODE (service{rpc}) the port IS the GENERATED
// operationv1.JobTemplateSummaryServiceServer interface — there is no hand-
// written port. The postgres adapter
// (contrib/postgres/.../operation/job_template_summary_query.go) implements it
// and self-registers via the job-template-summary registry factory; the
// composition initializer resolves that factory into this aggregate. On
// mock/non-postgres builds the port is nil and Execute degrades to an empty,
// successful response.
//
// Apps reach it via uc.Service.JobTemplateSummary.ListJobTemplateSummaries.Execute(ctx, req).
package job_template_summary

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
)

// UseCases aggregates every job-template-summary service use case.
type UseCases struct {
	ListJobTemplateSummaries *ListJobTemplateSummariesUseCase
}

// Repositories groups infrastructure dependencies. Query may be nil when no
// provider is registered — the use cases degrade gracefully (empty response).
type Repositories struct {
	Query query
}

// Services groups application services. ActionGatekeeper is REQUIRED for the
// base job:list read gate.
type Services struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires every job-template-summary use case from shared dependencies.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		ListJobTemplateSummaries: NewListJobTemplateSummariesUseCase(
			ListJobTemplateSummariesRepositories{Query: repositories.Query},
			ListJobTemplateSummariesServices{
				Translator:       services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
