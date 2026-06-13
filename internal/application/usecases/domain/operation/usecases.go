package operation

import (
	// Operation use cases
	criteriaOptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/criteria_option"
	criteriaThresholdUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/criteria_threshold"
	evaluationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/evaluation"
	evaluationCycleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/evaluation_cycle"
	evaluationCycleMemberUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/evaluation_cycle_member"
	evaluationResponseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/evaluation_response"
	evaluationTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/evaluation_template"
	evaluationTemplateItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/evaluation_template_item"
	jobUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job"
	jobActivityUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_activity"
	jobOutcomeSummaryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_outcome_summary"
	jobPhaseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_phase"
	jobTaskUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_task"
	jobTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_template"
	jobTemplatePhaseUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_template_phase"
	jobTemplateRelationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_template_relation"
	jobTemplateTaskUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job_template_task"
	outcomeCriteriaUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/outcome_criteria"
	phaseOutcomeSummaryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/phase_outcome_summary"
	taskOutcomeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/task_outcome"
	taskOutcomeCheckUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/task_outcome_check"
	templateTaskCriteriaUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/template_task_criteria"
	workRequestUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/work_request"
	workRequestTypeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/work_request_type"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

	// Protobuf domain services for operation repositories
	criteriaoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
	criteriathresholdpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	evaluationcyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle"
	evaluationcyclememberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle_member"
	evaluationresponsepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_response"
	evaluationtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template"
	evaluationtemplateitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template_item"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
	joboutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	taskoutcomecheckpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
	templatetaskcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
	workrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
	workrequesttypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"

	// Cross-domain dependencies: entity domain
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"

	// Cross-domain dependency for the OnJobPhaseCompleted hook + the
	// MaterializeBillingEventsForJob use case (milestone-billing plan §3).
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// OperationRepositories contains all operation domain repositories.
//
// BillingEvent + Subscription + PricePlan + ProductPricePlan are cross-domain
// reads required by:
//
//   - OnJobPhaseCompleted hook in UpdateJobPhaseUseCase (milestone-billing
//     plan §3 / flow.md §11) — reads BillingEvent.
//   - MaterializeBillingEventsForJob use case — reads Subscription / PricePlan /
//     ProductPricePlan to resolve per-phase billable amounts.
//
// All four are optional. When nil, the milestone-billing branches no-op or
// return a clear validation error.
type OperationRepositories struct {
	Job                  jobpb.JobDomainServiceServer
	JobPhase             jobphasepb.JobPhaseDomainServiceServer
	JobTask              jobtaskpb.JobTaskDomainServiceServer
	JobTemplate          jobtemplatepb.JobTemplateDomainServiceServer
	JobTemplatePhase     jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask      jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
	JobTemplateRelation  jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	JobActivity          jobactivitypb.JobActivityDomainServiceServer
	OutcomeCriteria      outcomecriteriapb.OutcomeCriteriaDomainServiceServer
	CriteriaThreshold    criteriathresholdpb.CriteriaThresholdDomainServiceServer
	CriteriaOption       criteriaoptionpb.CriteriaOptionDomainServiceServer
	TemplateTaskCriteria templatetaskcriteriapb.TemplateTaskCriteriaDomainServiceServer
	TaskOutcome          taskoutcomepb.TaskOutcomeDomainServiceServer
	TaskOutcomeCheck     taskoutcomecheckpb.TaskOutcomeCheckDomainServiceServer
	PhaseOutcomeSummary  phaseoutcomesummarypb.PhaseOutcomeSummaryDomainServiceServer
	JobOutcomeSummary    joboutcomesummarypb.JobOutcomeSummaryDomainServiceServer

	// Performance Evaluation (20260604 v1).
	Evaluation             evaluationpb.EvaluationDomainServiceServer
	EvaluationResponse     evaluationresponsepb.EvaluationResponseDomainServiceServer
	EvaluationTemplate     evaluationtemplatepb.EvaluationTemplateDomainServiceServer
	EvaluationTemplateItem evaluationtemplateitempb.EvaluationTemplateItemDomainServiceServer
	EvaluationCycle        evaluationcyclepb.EvaluationCycleDomainServiceServer
	EvaluationCycleMember  evaluationcyclememberpb.EvaluationCycleMemberDomainServiceServer
	// SubscriptionSeat backs the evaluation anchor-ownership IDOR validation.
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer

	// Milestone-billing cross-domain reads (Phase C).
	BillingEvent     billingeventpb.BillingEventDomainServiceServer
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer

	// Work Requests (20260604-requests-workflow v1).
	WorkRequest     workrequestpb.WorkRequestDomainServiceServer
	WorkRequestType workrequesttypepb.WorkRequestTypeDomainServiceServer
	// WorkspaceUser — cross-domain FK validation for work_request.AssignWorkRequest.
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer
}

// OperationUseCases contains all operation-related use cases.
//
// 20260518-hexagonal-strict-adherence Phase 3 F7 closure — JobTemplateRelation
// (raw DomainServiceServer leak) is wrapped in a Layer-7 use case sub-aggregate
// and now lives at .JobTemplateRelation as a *jobtemplaterelation.UseCases.
type OperationUseCases struct {
	Job                  *jobUseCases.UseCases
	JobPhase             *jobPhaseUseCases.UseCases
	JobTask              *jobTaskUseCases.UseCases
	JobTemplate          *jobTemplateUseCases.UseCases
	JobTemplatePhase     *jobTemplatePhaseUseCases.UseCases
	JobTemplateRelation  *jobTemplateRelationUseCases.UseCases
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

	// Performance Evaluation (20260604 v1).
	Evaluation             *evaluationUseCases.UseCases
	EvaluationResponse     *evaluationResponseUseCases.UseCases
	EvaluationTemplate     *evaluationTemplateUseCases.UseCases
	EvaluationTemplateItem *evaluationTemplateItemUseCases.UseCases
	EvaluationCycle        *evaluationCycleUseCases.UseCases
	EvaluationCycleMember  *evaluationCycleMemberUseCases.UseCases

	// Work Requests (20260604-requests-workflow v1).
	WorkRequest     *workRequestUseCases.UseCases
	WorkRequestType *workRequestTypeUseCases.UseCases

	// Dashboard field retired 2026-05-21 (Wave C P1.C.9 Job) — the dashboard
	// now lives under `service.Dashboard.Job` (note the candidate-name vs.
	// source-aggregate divergence: source is `operation`, surface is `Job`).
	// The `usecases/operation/dashboard/` package is retired in the same
	// commit; the repository composition relocated to
	// `usecases/service/dashboard/job/`.
}

// NewUseCases creates all operation use cases with proper constructor injection
func NewUseCases(
	repos OperationRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) *OperationUseCases {
	jobUC := jobUseCases.NewUseCases(
		jobUseCases.JobRepositories{
			Job:              repos.Job,
			JobTemplate:      repos.JobTemplate,
			JobTemplatePhase: repos.JobTemplatePhase,
			JobPhase:         repos.JobPhase,
			BillingEvent:     repos.BillingEvent,
			Subscription:     repos.Subscription,
			PricePlan:        repos.PricePlan,
			ProductPricePlan: repos.ProductPricePlan,
		},
		jobUseCases.JobServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	jobPhaseUC := jobPhaseUseCases.NewUseCases(
		jobPhaseUseCases.JobPhaseRepositories{
			JobPhase:     repos.JobPhase,
			BillingEvent: repos.BillingEvent,
		},
		jobPhaseUseCases.JobPhaseServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	jobTaskUC := jobTaskUseCases.NewUseCases(
		jobTaskUseCases.JobTaskRepositories{JobTask: repos.JobTask},
		jobTaskUseCases.JobTaskServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	jobTemplateUC := jobTemplateUseCases.NewUseCases(
		jobTemplateUseCases.JobTemplateRepositories{JobTemplate: repos.JobTemplate},
		jobTemplateUseCases.JobTemplateServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	jobTemplatePhaseUC := jobTemplatePhaseUseCases.NewUseCases(
		jobTemplatePhaseUseCases.JobTemplatePhaseRepositories{JobTemplatePhase: repos.JobTemplatePhase},
		jobTemplatePhaseUseCases.JobTemplatePhaseServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	jobTemplateTaskUC := jobTemplateTaskUseCases.NewUseCases(
		jobTemplateTaskUseCases.JobTemplateTaskRepositories{JobTemplateTask: repos.JobTemplateTask},
		jobTemplateTaskUseCases.JobTemplateTaskServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	jobActivityUC := jobActivityUseCases.NewUseCases(
		jobActivityUseCases.JobActivityRepositories{JobActivity: repos.JobActivity},
		jobActivityUseCases.JobActivityServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	outcomeCriteriaUC := outcomeCriteriaUseCases.NewUseCases(
		outcomeCriteriaUseCases.OutcomeCriteriaRepositories{OutcomeCriteria: repos.OutcomeCriteria},
		outcomeCriteriaUseCases.OutcomeCriteriaServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	criteriaThresholdUC := criteriaThresholdUseCases.NewUseCases(
		criteriaThresholdUseCases.CriteriaThresholdRepositories{CriteriaThreshold: repos.CriteriaThreshold},
		criteriaThresholdUseCases.CriteriaThresholdServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	criteriaOptionUC := criteriaOptionUseCases.NewUseCases(
		criteriaOptionUseCases.CriteriaOptionRepositories{CriteriaOption: repos.CriteriaOption},
		criteriaOptionUseCases.CriteriaOptionServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	templateTaskCriteriaUC := templateTaskCriteriaUseCases.NewUseCases(
		templateTaskCriteriaUseCases.TemplateTaskCriteriaRepositories{TemplateTaskCriteria: repos.TemplateTaskCriteria},
		templateTaskCriteriaUseCases.TemplateTaskCriteriaServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	taskOutcomeUC := taskOutcomeUseCases.NewUseCases(
		taskOutcomeUseCases.TaskOutcomeRepositories{TaskOutcome: repos.TaskOutcome},
		taskOutcomeUseCases.TaskOutcomeServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	taskOutcomeCheckUC := taskOutcomeCheckUseCases.NewUseCases(
		taskOutcomeCheckUseCases.TaskOutcomeCheckRepositories{TaskOutcomeCheck: repos.TaskOutcomeCheck},
		taskOutcomeCheckUseCases.TaskOutcomeCheckServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	phaseOutcomeSummaryUC := phaseOutcomeSummaryUseCases.NewUseCases(
		phaseOutcomeSummaryUseCases.PhaseOutcomeSummaryRepositories{PhaseOutcomeSummary: repos.PhaseOutcomeSummary},
		phaseOutcomeSummaryUseCases.PhaseOutcomeSummaryServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	jobOutcomeSummaryUC := jobOutcomeSummaryUseCases.NewUseCases(
		jobOutcomeSummaryUseCases.JobOutcomeSummaryRepositories{JobOutcomeSummary: repos.JobOutcomeSummary},
		jobOutcomeSummaryUseCases.JobOutcomeSummaryServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	// Performance Evaluation (20260604 v1).
	evaluationUC := evaluationUseCases.NewUseCases(
		evaluationUseCases.EvaluationRepositories{
			Evaluation:         repos.Evaluation,
			EvaluationResponse: repos.EvaluationResponse,
			OutcomeCriteria:    repos.OutcomeCriteria,
			SubscriptionSeat:   repos.SubscriptionSeat,
		},
		evaluationUseCases.EvaluationServices{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idService, ActionGatekeeper: actionGate},
	)
	evaluationResponseUC := evaluationResponseUseCases.NewUseCases(
		evaluationResponseUseCases.Repositories{EvaluationResponse: repos.EvaluationResponse, Evaluation: repos.Evaluation},
		evaluationResponseUseCases.Services{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idService, ActionGatekeeper: actionGate},
	)
	evaluationTemplateUC := evaluationTemplateUseCases.NewUseCases(
		evaluationTemplateUseCases.Repositories{EvaluationTemplate: repos.EvaluationTemplate, EvaluationTemplateItem: repos.EvaluationTemplateItem, OutcomeCriteria: repos.OutcomeCriteria},
		evaluationTemplateUseCases.Services{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idService, ActionGatekeeper: actionGate},
	)
	evaluationTemplateItemUC := evaluationTemplateItemUseCases.NewUseCases(
		evaluationTemplateItemUseCases.Repositories{EvaluationTemplateItem: repos.EvaluationTemplateItem, EvaluationTemplate: repos.EvaluationTemplate},
		evaluationTemplateItemUseCases.Services{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idService, ActionGatekeeper: actionGate},
	)
	evaluationCycleUC := evaluationCycleUseCases.NewUseCases(
		evaluationCycleUseCases.Repositories{
			EvaluationCycle:       repos.EvaluationCycle,
			EvaluationCycleMember: repos.EvaluationCycleMember,
			SubscriptionSeat:      repos.SubscriptionSeat,
		},
		evaluationCycleUseCases.Services{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idService, ActionGatekeeper: actionGate},
	)
	evaluationCycleMemberUC := evaluationCycleMemberUseCases.NewUseCases(
		evaluationCycleMemberUseCases.Repositories{EvaluationCycleMember: repos.EvaluationCycleMember},
		evaluationCycleMemberUseCases.Services{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idService, ActionGatekeeper: actionGate},
	)

	// Work Requests (20260604-requests-workflow v1).
	var workRequestUC *workRequestUseCases.UseCases
	if repos.WorkRequest != nil {
		workRequestUC = workRequestUseCases.NewUseCases(
			workRequestUseCases.WorkRequestRepositories{
				WorkRequest:     repos.WorkRequest,
				WorkRequestType: repos.WorkRequestType,
				WorkspaceUser:   repos.WorkspaceUser,
			},
			workRequestUseCases.WorkRequestServices{
				ActionGatekeeper: actionGate,
				Transactor:       txSvc,
				Translator:       i18nSvc,
				IDGenerator:      idService,
			},
		)
	}
	var workRequestTypeUC *workRequestTypeUseCases.UseCases
	if repos.WorkRequestType != nil {
		workRequestTypeUC = workRequestTypeUseCases.NewUseCases(
			workRequestTypeUseCases.WorkRequestTypeRepositories{
				WorkRequestType: repos.WorkRequestType,
			},
			workRequestTypeUseCases.WorkRequestTypeServices{
				Transactor:       txSvc,
				Translator:       i18nSvc,
				ActionGatekeeper: actionGate,
				IDGenerator:      idService,
			},
		)
	}

	// Job dashboard wiring retired 2026-05-21 (Wave C P1.C.9 Job) —
	// type-assertion + factory wiring now lives in the service-layer
	// initializer at `internal/composition/core/initializers/service.go`
	// (search "Wave C P1.C.9 Job").

	// Phase 3 F7 closure — wrap the raw JobTemplateRelation
	// DomainServiceServer in a Layer-7 use case sub-aggregate. nil-safe when
	// the adapter isn't registered.
	jobTemplateRelationUC := jobTemplateRelationUseCases.NewUseCases(
		jobTemplateRelationUseCases.JobTemplateRelationRepositories{
			JobTemplateRelation: repos.JobTemplateRelation,
		},
		jobTemplateRelationUseCases.JobTemplateRelationServices{
			Authorizer:       authSvc,
			Translator:       i18nSvc,
			ActionGatekeeper: actionGate,
		},
	)

	return &OperationUseCases{
		Job:                  jobUC,
		JobPhase:             jobPhaseUC,
		JobTask:              jobTaskUC,
		JobTemplate:          jobTemplateUC,
		JobTemplatePhase:     jobTemplatePhaseUC,
		JobTemplateRelation:  jobTemplateRelationUC,
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

		Evaluation:             evaluationUC,
		EvaluationResponse:     evaluationResponseUC,
		EvaluationTemplate:     evaluationTemplateUC,
		EvaluationTemplateItem: evaluationTemplateItemUC,
		EvaluationCycle:        evaluationCycleUC,
		EvaluationCycleMember:  evaluationCycleMemberUC,

		WorkRequest:     workRequestUC,
		WorkRequestType: workRequestTypeUC,
	}
}
