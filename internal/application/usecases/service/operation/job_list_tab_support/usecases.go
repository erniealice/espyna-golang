// Package job_list_tab_support hosts the service-driven job-list tabstrip
// support read (the 20260718 courses-list-perf Rank-1 counter-plan). It is the
// generic, workspace-scoped read that collapses the job-list "/classes" tabstrip's
// category + active-template needs into ONE UNION-ALL statement — replacing the
// former 12 generic-List statements (2 category calls + ~4 paged template calls,
// each COUNT+SELECT).
//
// Proto-less internal-service shape (mirroring the engine-identity-bridge
// WorkflowAssigneeQueryService): the port is a HAND-WRITTEN interface
// (ports.JobListTabSupportQueryService) with plain Go request/response DTOs — no
// generated proto ServiceServer, so no new esqyma service contract. The postgres
// adapter (contrib/postgres/.../operation/job_list_tab_support_query.go)
// implements it and self-registers via the job-list-tab-support registry factory;
// the composition initializer resolves that factory into this aggregate. On
// mock/non-postgres builds the port is nil and Execute degrades to an empty
// response.
//
// Apps reach it via uc.Service.JobListTabSupport.ListJobListTabSupport.Execute(ctx).
package job_list_tab_support

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
)

// query is the hand-written port the tab-support use case reads from.
type query = ports.JobListTabSupportQueryService

// UseCases aggregates every job-list tab-support service use case.
type UseCases struct {
	ListJobListTabSupport *ListJobListTabSupportUseCase
}

// Repositories groups infrastructure dependencies. Query may be nil when no
// provider is registered — the use case degrades gracefully (empty response).
type Repositories struct {
	Query query
}

// Services groups application services. ActionGatekeeper is REQUIRED for the two
// independent per-kind read gates (job_category:list, job_template:list).
type Services struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires every job-list tab-support use case from shared dependencies.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		ListJobListTabSupport: NewListJobListTabSupportUseCase(
			ListJobListTabSupportRepositories{Query: repositories.Query},
			ListJobListTabSupportServices{
				Translator:       services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
