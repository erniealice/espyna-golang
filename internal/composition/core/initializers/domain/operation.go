package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeOperation creates all operation use cases from provider repositories.
//
// Optional cross-domain repositories (Subscription/PricePlan/ProductPricePlan/
// BillingEvent) are sourced from the SubscriptionRepositories provider and
// passed alongside the operation repos so MaterializeBillingEventsForJob and
// the OnJobPhaseCompleted hook can wire up. Pass nil for each when the
// caller does not have access — the use cases degrade with a clear error.
func InitializeOperation(
	repos *domain.OperationRepositories,
	subRepos *domain.SubscriptionRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*operation.OperationUseCases, error) {
	opRepos := operation.OperationRepositories{
		Job:                 repos.Job,
		JobPhase:            repos.JobPhase,
		JobTask:             repos.JobTask,
		JobTemplate:         repos.JobTemplate,
		JobTemplatePhase:    repos.JobTemplatePhase,
		JobTemplateTask:     repos.JobTemplateTask,
		JobTemplateRelation: repos.JobTemplateRelation,
		JobActivity:         repos.JobActivity,
		// JobCategory — per-workspace job taxonomy reference entity (20260714).
		JobCategory: repos.JobCategory,
		// JobOutcomeSummaryDocumentTemplate — report-card template binding (20260714).
		JobOutcomeSummaryDocumentTemplate: repos.JobOutcomeSummaryDocumentTemplate,
		// JobTemplateDocumentTemplate — sheet-family (grade-sheet) template binding (20260720).
		JobTemplateDocumentTemplate: repos.JobTemplateDocumentTemplate,
		OutcomeCriteria:                   repos.OutcomeCriteria,
		// TemplateTaskCriteria backs the A/B/C/D criterion ordering
		// (template_task_criteria.sequence_order) that the report-card document
		// builder prefers via ListTemplateTaskCriterias. Built by the provider
		// but — like TaskOutcome/PhaseOutcomeSummary below — it was not threaded
		// through here, leaving ListTemplateTaskCriteriaUseCase's repo nil (a
		// nil-deref the moment the builder lists criteria; ordering then degraded
		// to a stable criteria-id fallback). Threading it restores sequence_order.
		TemplateTaskCriteria: repos.TemplateTaskCriteria,
		// Education grading (20260616 v1).
		ScoringScheme:            repos.ScoringScheme,
		ScoringComponent:         repos.ScoringComponent,
		ScoringComponentCriteria: repos.ScoringComponentCriteria,
		ScoreScale:               repos.ScoreScale,
		ScoreScaleBand:           repos.ScoreScaleBand,
		// TaskOutcome + PhaseOutcomeSummary back the grade roll-up
		// (ComputePhaseOutcome reads recorded task_outcomes and upserts the
		// phase_outcome_summary). Both are built by the provider but were not
		// threaded through here, leaving GradeCompute's TaskOutcome /
		// PhaseOutcomeSummary repos nil (a nil-deref the moment the roll-up
		// lists outcomes). They also back the standalone TaskOutcome /
		// PhaseOutcomeSummary CRUD use-cases.
		TaskOutcome:         repos.TaskOutcome,
		PhaseOutcomeSummary: repos.PhaseOutcomeSummary,
		// JobOutcomeSummary + JobOutcomeLine are the year-final roll-up write
		// targets (ComputeJobOutcome). Without JobOutcomeSummary threaded here
		// the job roll-up nil-derefs on its GetByJob idempotency lookup.
		JobOutcomeSummary:   repos.JobOutcomeSummary,
		JobOutcomeLine:      repos.JobOutcomeLine,
		ReportingCheckpoint: repos.ReportingCheckpoint,
		// Performance Evaluation (20260604 v1).
		Evaluation:             repos.Evaluation,
		EvaluationResponse:     repos.EvaluationResponse,
		EvaluationTemplate:     repos.EvaluationTemplate,
		EvaluationTemplateItem: repos.EvaluationTemplateItem,
		EvaluationCycle:        repos.EvaluationCycle,
		EvaluationCycleMember:  repos.EvaluationCycleMember,
		// Work Requests (20260604-requests-workflow v1).
		WorkRequest:     repos.WorkRequest,
		WorkRequestType: repos.WorkRequestType,
		WorkspaceUser:   repos.WorkspaceUser,
	}
	if subRepos != nil {
		opRepos.BillingEvent = subRepos.BillingEvent
		opRepos.Subscription = subRepos.Subscription
		opRepos.PricePlan = subRepos.PricePlan
		opRepos.ProductPricePlan = subRepos.ProductPricePlan
		// SubscriptionSeat backs the evaluation anchor-ownership IDOR check.
		opRepos.SubscriptionSeat = subRepos.SubscriptionSeat
	}
	return operation.NewUseCases(
		opRepos,
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		actionGate,
	), nil
}
