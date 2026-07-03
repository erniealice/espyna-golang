// Package outcome_matrix hosts the service-driven outcome-matrix read use case
// (proto: service/operation/outcome_matrix). It is the generic, cross-vertical
// replacement for the education-specific grade_sheet: a principal-scoped
// outcome-score matrix (rows = client × job_template, columns = the template's
// phase→task→criterion tree, cells = task_outcome) served through the typed
// stack with authcheck (task_outcome:list + fail-closed scope=ALL→MINE).
//
// Per Q-PROTO-MODE (service{rpc}) the port IS the GENERATED
// operationv1.OutcomeMatrixServiceServer interface — there is no hand-written
// port. The postgres adapter (contrib/postgres/.../operation/outcome_matrix_query.go)
// implements it and self-registers via the outcome-matrix registry factory; the
// composition initializer resolves that factory into this aggregate. On
// mock/non-postgres builds the port is nil and Execute degrades to an empty,
// successful response.
//
// Apps reach it via uc.Service.OutcomeMatrix.GetOutcomeMatrix.Execute(ctx, req).
package outcome_matrix

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
)

// UseCases aggregates every outcome-matrix service use case.
type UseCases struct {
	GetOutcomeMatrix *GetOutcomeMatrixUseCase
}

// Repositories groups infrastructure dependencies. Query may be nil when no
// provider is registered — the use cases degrade gracefully (empty response).
type Repositories struct {
	Query query
}

// Services groups application services. ActionGatekeeper is REQUIRED for the
// base task_outcome:list gate + the workspace:list scope=ALL widen check.
type Services struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires every outcome-matrix use case from shared dependencies.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		GetOutcomeMatrix: NewGetOutcomeMatrixUseCase(
			GetOutcomeMatrixRepositories{Query: repositories.Query},
			GetOutcomeMatrixServices{
				Translator:       services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
