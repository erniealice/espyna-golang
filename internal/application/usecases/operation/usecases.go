package operation

import (
	// Operation use cases
	jobUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job"
	jobPhaseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_phase"
	jobTaskUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_task"
	jobTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_template"
	jobTemplatePhaseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_template_phase"
	jobTemplateTaskUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_template_task"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for operation repositories
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

// OperationRepositories contains all operation domain repositories
type OperationRepositories struct {
	Job              jobpb.JobDomainServiceServer
	JobPhase         jobphasepb.JobPhaseDomainServiceServer
	JobTask          jobtaskpb.JobTaskDomainServiceServer
	JobTemplate      jobtemplatepb.JobTemplateDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask  jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
}

// OperationUseCases contains all operation-related use cases
type OperationUseCases struct {
	Job              *jobUseCases.UseCases
	JobPhase         *jobPhaseUseCases.UseCases
	JobTask          *jobTaskUseCases.UseCases
	JobTemplate      *jobTemplateUseCases.UseCases
	JobTemplatePhase *jobTemplatePhaseUseCases.UseCases
	JobTemplateTask  *jobTemplateTaskUseCases.UseCases
}

// NewUseCases creates all operation use cases with proper constructor injection
func NewUseCases(
	repos OperationRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *OperationUseCases {
	jobUC := jobUseCases.NewUseCases(
		jobUseCases.JobRepositories{Job: repos.Job},
		jobUseCases.JobServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	jobPhaseUC := jobPhaseUseCases.NewUseCases(
		jobPhaseUseCases.JobPhaseRepositories{JobPhase: repos.JobPhase},
		jobPhaseUseCases.JobPhaseServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	jobTaskUC := jobTaskUseCases.NewUseCases(
		jobTaskUseCases.JobTaskRepositories{JobTask: repos.JobTask},
		jobTaskUseCases.JobTaskServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	jobTemplateUC := jobTemplateUseCases.NewUseCases(
		jobTemplateUseCases.JobTemplateRepositories{JobTemplate: repos.JobTemplate},
		jobTemplateUseCases.JobTemplateServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	jobTemplatePhaseUC := jobTemplatePhaseUseCases.NewUseCases(
		jobTemplatePhaseUseCases.JobTemplatePhaseRepositories{JobTemplatePhase: repos.JobTemplatePhase},
		jobTemplatePhaseUseCases.JobTemplatePhaseServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	jobTemplateTaskUC := jobTemplateTaskUseCases.NewUseCases(
		jobTemplateTaskUseCases.JobTemplateTaskRepositories{JobTemplateTask: repos.JobTemplateTask},
		jobTemplateTaskUseCases.JobTemplateTaskServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &OperationUseCases{
		Job:              jobUC,
		JobPhase:         jobPhaseUC,
		JobTask:          jobTaskUC,
		JobTemplate:      jobTemplateUC,
		JobTemplatePhase: jobTemplatePhaseUC,
		JobTemplateTask:  jobTemplateTaskUC,
	}
}
