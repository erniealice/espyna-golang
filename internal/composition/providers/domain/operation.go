package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"
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
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"
)

// OperationRepositories contains all operation domain repositories.
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

	// Performance Evaluation (20260604 v1). Optional: when an adapter is not
	// registered (e.g. mock-only tests) the field stays nil and the use cases
	// degrade with a clear error rather than panicking.
	Evaluation             evaluationpb.EvaluationDomainServiceServer
	EvaluationResponse     evaluationresponsepb.EvaluationResponseDomainServiceServer
	EvaluationTemplate     evaluationtemplatepb.EvaluationTemplateDomainServiceServer
	EvaluationTemplateItem evaluationtemplateitempb.EvaluationTemplateItemDomainServiceServer
	EvaluationCycle        evaluationcyclepb.EvaluationCycleDomainServiceServer
	EvaluationCycleMember  evaluationcyclememberpb.EvaluationCycleMemberDomainServiceServer
	// SubscriptionSeat is consumed cross-domain by the evaluation anchor-ownership
	// IDOR check + the domain-layer evaluation_cycle.OpenUseCase denominator freeze.
	// Best-effort: nil when no adapter is registered.
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer

	// ClientWorkspaceUser + SubscriptionWorkspaceUser are the Q-SERVICING F-GATE
	// membership readers (CR-5), consumed cross-domain by GetPerformancePanelData
	// to gate rows to engagements the caller services (ACCOUNT scope via
	// client_workspace_user; PROJECT scope via subscription_workspace_user).
	// Best-effort: nil when no adapter is registered → the F-GATE fail-closes
	// (deny unless the caller holds evaluation:triage_all).
	ClientWorkspaceUser       clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
}

// NewOperationRepositories creates and returns a new set of OperationRepositories.
func NewOperationRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*OperationRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	jobRepo, err := repoCreator.CreateRepository(entityid.Job, conn, tableConfig.TableName(entityid.Job))
	if err != nil {
		return nil, fmt.Errorf("failed to create job repository: %w", err)
	}

	jobPhaseRepo, err := repoCreator.CreateRepository(entityid.JobPhase, conn, tableConfig.TableName(entityid.JobPhase))
	if err != nil {
		return nil, fmt.Errorf("failed to create job_phase repository: %w", err)
	}

	jobTaskRepo, err := repoCreator.CreateRepository(entityid.JobTask, conn, tableConfig.TableName(entityid.JobTask))
	if err != nil {
		return nil, fmt.Errorf("failed to create job_task repository: %w", err)
	}

	jobTemplateRepo, err := repoCreator.CreateRepository(entityid.JobTemplate, conn, tableConfig.TableName(entityid.JobTemplate))
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template repository: %w", err)
	}

	jobTemplatePhaseRepo, err := repoCreator.CreateRepository(entityid.JobTemplatePhase, conn, tableConfig.TableName(entityid.JobTemplatePhase))
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template_phase repository: %w", err)
	}

	jobTemplateTaskRepo, err := repoCreator.CreateRepository(entityid.JobTemplateTask, conn, tableConfig.TableName(entityid.JobTemplateTask))
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template_task repository: %w", err)
	}

	// JobTemplateRelation — auto-spawn-jobs-from-subscription Phase B.5 entity.
	// Best-effort: when no adapter is registered (e.g. mock-only tests),
	// MaterializeJobsForSubscription proceeds with the root template only.
	var jobTemplateRelationServer jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	if jobTemplateRelationRepo, jtrErr := repoCreator.CreateRepository(entityid.JobTemplateRelation, conn, tableConfig.TableName(entityid.JobTemplateRelation)); jtrErr == nil {
		jobTemplateRelationServer = jobTemplateRelationRepo.(jobtemplaterelationpb.JobTemplateRelationDomainServiceServer)
	}

	jobActivityRepo, err := repoCreator.CreateRepository(entityid.JobActivity, conn, tableConfig.TableName(entityid.JobActivity))
	if err != nil {
		return nil, fmt.Errorf("failed to create job_activity repository: %w", err)
	}

	outcomeCriteriaRepo, err := repoCreator.CreateRepository(entityid.OutcomeCriteria, conn, tableConfig.TableName(entityid.OutcomeCriteria))
	if err != nil {
		return nil, fmt.Errorf("failed to create outcome_criteria repository: %w", err)
	}

	criteriaThresholdRepo, err := repoCreator.CreateRepository(entityid.CriteriaThreshold, conn, tableConfig.TableName(entityid.CriteriaThreshold))
	if err != nil {
		return nil, fmt.Errorf("failed to create criteria_threshold repository: %w", err)
	}

	criteriaOptionRepo, err := repoCreator.CreateRepository(entityid.CriteriaOption, conn, tableConfig.TableName(entityid.CriteriaOption))
	if err != nil {
		return nil, fmt.Errorf("failed to create criteria_option repository: %w", err)
	}

	templateTaskCriteriaRepo, err := repoCreator.CreateRepository(entityid.TemplateTaskCriteria, conn, tableConfig.TableName(entityid.TemplateTaskCriteria))
	if err != nil {
		return nil, fmt.Errorf("failed to create template_task_criteria repository: %w", err)
	}

	taskOutcomeRepo, err := repoCreator.CreateRepository(entityid.TaskOutcome, conn, tableConfig.TableName(entityid.TaskOutcome))
	if err != nil {
		return nil, fmt.Errorf("failed to create task_outcome repository: %w", err)
	}

	taskOutcomeCheckRepo, err := repoCreator.CreateRepository(entityid.TaskOutcomeCheck, conn, tableConfig.TableName(entityid.TaskOutcomeCheck))
	if err != nil {
		return nil, fmt.Errorf("failed to create task_outcome_check repository: %w", err)
	}

	phaseOutcomeSummaryRepo, err := repoCreator.CreateRepository(entityid.PhaseOutcomeSummary, conn, tableConfig.TableName(entityid.PhaseOutcomeSummary))
	if err != nil {
		return nil, fmt.Errorf("failed to create phase_outcome_summary repository: %w", err)
	}

	jobOutcomeSummaryRepo, err := repoCreator.CreateRepository(entityid.JobOutcomeSummary, conn, tableConfig.TableName(entityid.JobOutcomeSummary))
	if err != nil {
		return nil, fmt.Errorf("failed to create job_outcome_summary repository: %w", err)
	}

	// Performance Evaluation (20260604 v1) — best-effort: nil when no adapter is
	// registered (mock/firestore builds), mirroring JobTemplateRelation.
	var evaluationServer evaluationpb.EvaluationDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.Evaluation, conn, tableConfig.TableName(entityid.Evaluation)); e == nil {
		evaluationServer = repo.(evaluationpb.EvaluationDomainServiceServer)
	}
	var evaluationResponseServer evaluationresponsepb.EvaluationResponseDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.EvaluationResponse, conn, tableConfig.TableName(entityid.EvaluationResponse)); e == nil {
		evaluationResponseServer = repo.(evaluationresponsepb.EvaluationResponseDomainServiceServer)
	}
	var evaluationTemplateServer evaluationtemplatepb.EvaluationTemplateDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.EvaluationTemplate, conn, tableConfig.TableName(entityid.EvaluationTemplate)); e == nil {
		evaluationTemplateServer = repo.(evaluationtemplatepb.EvaluationTemplateDomainServiceServer)
	}
	var evaluationTemplateItemServer evaluationtemplateitempb.EvaluationTemplateItemDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.EvaluationTemplateItem, conn, tableConfig.TableName(entityid.EvaluationTemplateItem)); e == nil {
		evaluationTemplateItemServer = repo.(evaluationtemplateitempb.EvaluationTemplateItemDomainServiceServer)
	}
	var evaluationCycleServer evaluationcyclepb.EvaluationCycleDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.EvaluationCycle, conn, tableConfig.TableName(entityid.EvaluationCycle)); e == nil {
		evaluationCycleServer = repo.(evaluationcyclepb.EvaluationCycleDomainServiceServer)
	}
	var evaluationCycleMemberServer evaluationcyclememberpb.EvaluationCycleMemberDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.EvaluationCycleMember, conn, tableConfig.TableName(entityid.EvaluationCycleMember)); e == nil {
		evaluationCycleMemberServer = repo.(evaluationcyclememberpb.EvaluationCycleMemberDomainServiceServer)
	}
	var subscriptionSeatServer subscriptionseatpb.SubscriptionSeatDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.SubscriptionSeat, conn, tableConfig.TableName(entityid.SubscriptionSeat)); e == nil {
		subscriptionSeatServer = repo.(subscriptionseatpb.SubscriptionSeatDomainServiceServer)
	}
	// Q-SERVICING F-GATE membership readers (CR-5) — best-effort, mirroring
	// SubscriptionSeat. The concrete postgres adapters additionally implement the
	// servicing reader ports (IsActiveAccountTeamMember / IsActiveServicer) as
	// extension methods; initServicePerformance reaches them via interface assertion.
	var clientWorkspaceUserServer clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.ClientWorkspaceUser, conn, tableConfig.TableName(entityid.ClientWorkspaceUser)); e == nil {
		clientWorkspaceUserServer = repo.(clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer)
	}
	var subscriptionWorkspaceUserServer subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
	if repo, e := repoCreator.CreateRepository(entityid.SubscriptionWorkspaceUser, conn, tableConfig.TableName(entityid.SubscriptionWorkspaceUser)); e == nil {
		subscriptionWorkspaceUserServer = repo.(subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer)
	}

	return &OperationRepositories{
		Job:                  jobRepo.(jobpb.JobDomainServiceServer),
		JobPhase:             jobPhaseRepo.(jobphasepb.JobPhaseDomainServiceServer),
		JobTask:              jobTaskRepo.(jobtaskpb.JobTaskDomainServiceServer),
		JobTemplate:          jobTemplateRepo.(jobtemplatepb.JobTemplateDomainServiceServer),
		JobTemplatePhase:     jobTemplatePhaseRepo.(jobtemplatephasepb.JobTemplatePhaseDomainServiceServer),
		JobTemplateTask:      jobTemplateTaskRepo.(jobtemplatetaskpb.JobTemplateTaskDomainServiceServer),
		JobTemplateRelation:  jobTemplateRelationServer,
		JobActivity:          jobActivityRepo.(jobactivitypb.JobActivityDomainServiceServer),
		OutcomeCriteria:      outcomeCriteriaRepo.(outcomecriteriapb.OutcomeCriteriaDomainServiceServer),
		CriteriaThreshold:    criteriaThresholdRepo.(criteriathresholdpb.CriteriaThresholdDomainServiceServer),
		CriteriaOption:       criteriaOptionRepo.(criteriaoptionpb.CriteriaOptionDomainServiceServer),
		TemplateTaskCriteria: templateTaskCriteriaRepo.(templatetaskcriteriapb.TemplateTaskCriteriaDomainServiceServer),
		TaskOutcome:          taskOutcomeRepo.(taskoutcomepb.TaskOutcomeDomainServiceServer),
		TaskOutcomeCheck:     taskOutcomeCheckRepo.(taskoutcomecheckpb.TaskOutcomeCheckDomainServiceServer),
		PhaseOutcomeSummary:  phaseOutcomeSummaryRepo.(phaseoutcomesummarypb.PhaseOutcomeSummaryDomainServiceServer),
		JobOutcomeSummary:    jobOutcomeSummaryRepo.(joboutcomesummarypb.JobOutcomeSummaryDomainServiceServer),

		Evaluation:             evaluationServer,
		EvaluationResponse:     evaluationResponseServer,
		EvaluationTemplate:     evaluationTemplateServer,
		EvaluationTemplateItem: evaluationTemplateItemServer,
		EvaluationCycle:        evaluationCycleServer,
		EvaluationCycleMember:  evaluationCycleMemberServer,
		SubscriptionSeat:       subscriptionSeatServer,

		ClientWorkspaceUser:       clientWorkspaceUserServer,
		SubscriptionWorkspaceUser: subscriptionWorkspaceUserServer,
	}, nil
}
