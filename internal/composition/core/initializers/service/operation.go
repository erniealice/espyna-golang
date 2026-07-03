package service

import (
	"database/sql"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	outcomematrixusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/operation/outcome_matrix"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// initServiceOperation wires the service-layer Operation sub-aggregate
// (service/operation/outcome_matrix). Unlike initServiceSecurity it threads the
// ActionGatekeeper through, because the outcome-matrix read gates on
// task_outcome:list + a fail-closed workspace:list scope=ALL widen check.
func initServiceOperation(db *sql.DB, i18nSvc ports.Translator, actionGate *actiongate.ActionGatekeeper) *outcomematrixusecases.UseCases {
	query := outcomeMatrixQueryFromDB(db)
	return outcomematrixusecases.NewUseCases(
		outcomematrixusecases.Repositories{Query: query},
		outcomematrixusecases.Services{Translator: i18nSvc, ActionGatekeeper: actionGate},
	)
}

// outcomeMatrixQueryFromDB returns the registered outcome-matrix query port
// backed by the provided raw connection, or nil when no provider has been
// registered (e.g. non-postgres / non-mock builds). The factory takes `any` to
// dodge the cyclic import — see registry/outcome_matrix.go.
func outcomeMatrixQueryFromDB(db *sql.DB) matrixpb.OutcomeMatrixServiceServer {
	factory, ok := internalregistry.GetOutcomeMatrixFactory()
	if !ok || factory == nil {
		return nil
	}
	result := factory(db)
	if result == nil {
		return nil
	}
	if q, ok := result.(matrixpb.OutcomeMatrixServiceServer); ok {
		return q
	}
	return nil
}
