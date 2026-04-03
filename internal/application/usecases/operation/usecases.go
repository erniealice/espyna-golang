package operation

import (
	// Operation use cases
	criteriaOptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/criteria_option"
	criteriaThresholdUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/criteria_threshold"
	jobUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job"
	jobOutcomeSummaryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_outcome_summary"
	jobPhaseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_phase"
	jobTaskUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_task"
	jobTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_template"
	jobTemplatePhaseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_template_phase"
	jobTemplateTaskUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job_template_task"
	outcomeCriteriaUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/outcome_criteria"
	phaseOutcomeSummaryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/phase_outcome_summary"
	taskOutcomeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/task_outcome"
	taskOutcomeCheckUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/task_outcome_check"
	templateTaskCriteriaUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/operation/template_task_criteria"
	jobActivityUseCases "github.com/erniealice/espyna-golang/operation/job_activity"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for operation repositories
	criteriaoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
	criteriathresholdpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
	joboutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	taskoutcomecheckpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
	templatetaskcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

// OperationRepositories contains all operation domain repositories
type OperationRepositories struct {
	Job                  jobpb.JobDomainServiceServer
	JobPhase             jobphasepb.JobPhaseDomainServiceServer
	JobTask              jobtaskpb.JobTaskDomainServiceServer
	JobTemplate          jobtemplatepb.JobTemplateDomainServiceServer
	JobTemplatePhase     jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask      jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
	JobActivity          jobactivitypb.JobActivityDomainServiceServer
	OutcomeCriteria      outcomecriteriapb.OutcomeCriteriaDomainServiceServer
	CriteriaThreshold    criteriathresholdpb.CriteriaThresholdDomainServiceServer
	CriteriaOption       criteriaoptionpb.CriteriaOptionDomainServiceServer
	TemplateTaskCriteria templatetaskcriteriapb.TemplateTaskCriteriaDomainServiceServer
	TaskOutcome          taskoutcomepb.TaskOutcomeDomainServiceServer
	TaskOutcomeCheck     taskoutcomecheckpb.TaskOutcomeCheckDomainServiceServer
	PhaseOutcomeSummary  phaseoutcomesummarypb.PhaseOutcomeSummaryDomainServiceServer
	JobOutcomeSummary    joboutcomesummarypb.JobOutcomeSummaryDomainServiceServer
}

// OperationUseCases contains all operation-related use cases
type OperationUseCases struct {
	Job                  *jobUseCases.UseCases
	JobPhase             *jobPhaseUseCases.UseCases
	JobTask              *jobTaskUseCases.UseCases
	JobTemplate          *jobTemplateUseCases.UseCases
	JobTemplatePhase     *jobTemplatePhaseUseCases.UseCases
	JobTemplateTask      *jobTemplateTaskUseCases.UseCases
	JobActivity          *jobActivityUseCases.UseCases
	OutcomeCriteria      *outcomeCriteriaUseCases.UseCases
	CriteriaThreshold    *criteriaThresholdUseCases.UseCases
	CriteriaOption       *criteriaOptionUseCases.UseCases
	TemplateTaskCriteria *templateTaskCriteriaUseCases.UseCases
	TaskOutcome          *taskOutcomeUseCases.UseCases
	TaskOutcomeCheck     *taskOutcomeCheckUseCases.UseCases
	PhaseOutcomeSummary  *phaseOutcomeSummaryUseCases.UseCases
	JobOutcomeSummary    *jobOutcomeSummaryUseCases.UseCases
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

	jobActivityUC := jobActivityUseCases.NewUseCases(
		jobActivityUseCases.JobActivityRepositories{JobActivity: repos.JobActivity},
		jobActivityUseCases.JobActivityServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	outcomeCriteriaUC := outcomeCriteriaUseCases.NewUseCases(
		outcomeCriteriaUseCases.OutcomeCriteriaRepositories{OutcomeCriteria: repos.OutcomeCriteria},
		outcomeCriteriaUseCases.OutcomeCriteriaServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	criteriaThresholdUC := criteriaThresholdUseCases.NewUseCases(
		criteriaThresholdUseCases.CriteriaThresholdRepositories{CriteriaThreshold: repos.CriteriaThreshold},
		criteriaThresholdUseCases.CriteriaThresholdServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	criteriaOptionUC := criteriaOptionUseCases.NewUseCases(
		criteriaOptionUseCases.CriteriaOptionRepositories{CriteriaOption: repos.CriteriaOption},
		criteriaOptionUseCases.CriteriaOptionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	templateTaskCriteriaUC := templateTaskCriteriaUseCases.NewUseCases(
		templateTaskCriteriaUseCases.TemplateTaskCriteriaRepositories{TemplateTaskCriteria: repos.TemplateTaskCriteria},
		templateTaskCriteriaUseCases.TemplateTaskCriteriaServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	taskOutcomeUC := taskOutcomeUseCases.NewUseCases(
		taskOutcomeUseCases.TaskOutcomeRepositories{TaskOutcome: repos.TaskOutcome},
		taskOutcomeUseCases.TaskOutcomeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	taskOutcomeCheckUC := taskOutcomeCheckUseCases.NewUseCases(
		taskOutcomeCheckUseCases.TaskOutcomeCheckRepositories{TaskOutcomeCheck: repos.TaskOutcomeCheck},
		taskOutcomeCheckUseCases.TaskOutcomeCheckServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	phaseOutcomeSummaryUC := phaseOutcomeSummaryUseCases.NewUseCases(
		phaseOutcomeSummaryUseCases.PhaseOutcomeSummaryRepositories{PhaseOutcomeSummary: repos.PhaseOutcomeSummary},
		phaseOutcomeSummaryUseCases.PhaseOutcomeSummaryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	jobOutcomeSummaryUC := jobOutcomeSummaryUseCases.NewUseCases(
		jobOutcomeSummaryUseCases.JobOutcomeSummaryRepositories{JobOutcomeSummary: repos.JobOutcomeSummary},
		jobOutcomeSummaryUseCases.JobOutcomeSummaryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &OperationUseCases{
		Job:                  jobUC,
		JobPhase:             jobPhaseUC,
		JobTask:              jobTaskUC,
		JobTemplate:          jobTemplateUC,
		JobTemplatePhase:     jobTemplatePhaseUC,
		JobTemplateTask:      jobTemplateTaskUC,
		JobActivity:          jobActivityUC,
		OutcomeCriteria:      outcomeCriteriaUC,
		CriteriaThreshold:    criteriaThresholdUC,
		CriteriaOption:       criteriaOptionUC,
		TemplateTaskCriteria: templateTaskCriteriaUC,
		TaskOutcome:          taskOutcomeUC,
		TaskOutcomeCheck:     taskOutcomeCheckUC,
		PhaseOutcomeSummary:  phaseOutcomeSummaryUC,
		JobOutcomeSummary:    jobOutcomeSummaryUC,
	}
}
