package service

import (
	"database/sql"

	summarypb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/job_template_summary"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	joblisttabsupportusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/operation/job_list_tab_support"
	jobtemplatesummaryusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/operation/job_template_summary"
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

// initServiceOperationJobTemplateSummary wires the service-layer
// job-template-summary read (service/operation/job_template_summary). Like
// initServiceOperation it threads the ActionGatekeeper through, because the
// read gates on job:list.
func initServiceOperationJobTemplateSummary(db *sql.DB, i18nSvc ports.Translator, actionGate *actiongate.ActionGatekeeper) *jobtemplatesummaryusecases.UseCases {
	query := jobTemplateSummaryQueryFromDB(db)
	return jobtemplatesummaryusecases.NewUseCases(
		jobtemplatesummaryusecases.Repositories{Query: query},
		jobtemplatesummaryusecases.Services{Translator: i18nSvc, ActionGatekeeper: actionGate},
	)
}

// jobTemplateSummaryQueryFromDB returns the registered job-template-summary
// query port backed by the provided raw connection, or nil when no provider has
// been registered (e.g. non-postgres / non-mock builds). The factory takes
// `any` to dodge the cyclic import — see registry/job_template_summary.go.
func jobTemplateSummaryQueryFromDB(db *sql.DB) summarypb.JobTemplateSummaryServiceServer {
	factory, ok := internalregistry.GetJobTemplateSummaryFactory()
	if !ok || factory == nil {
		return nil
	}
	result := factory(db)
	if result == nil {
		return nil
	}
	if q, ok := result.(summarypb.JobTemplateSummaryServiceServer); ok {
		return q
	}
	return nil
}

// initServiceOperationJobListTabSupport wires the service-layer job-list
// tabstrip support read (service/operation/job_list_tab_support — 20260718
// courses-list-perf Rank-1). Like the sibling summary read it threads the
// ActionGatekeeper through, because the read gates on job_category:list +
// job_template:list (two INDEPENDENT per-kind checks).
func initServiceOperationJobListTabSupport(db *sql.DB, i18nSvc ports.Translator, actionGate *actiongate.ActionGatekeeper) *joblisttabsupportusecases.UseCases {
	query := jobListTabSupportQueryFromDB(db)
	return joblisttabsupportusecases.NewUseCases(
		joblisttabsupportusecases.Repositories{Query: query},
		joblisttabsupportusecases.Services{Translator: i18nSvc, ActionGatekeeper: actionGate},
	)
}

// jobListTabSupportQueryFromDB returns the registered job-list tab-support query
// port backed by the provided raw connection, or nil when no provider has been
// registered (e.g. non-postgres / non-mock builds). The factory takes `any` to
// dodge the cyclic import — see registry/job_list_tab_support.go.
func jobListTabSupportQueryFromDB(db *sql.DB) ports.JobListTabSupportQueryService {
	factory, ok := internalregistry.GetJobListTabSupportFactory()
	if !ok || factory == nil {
		return nil
	}
	result := factory(db)
	if result == nil {
		return nil
	}
	if q, ok := result.(ports.JobListTabSupportQueryService); ok {
		return q
	}
	return nil
}
