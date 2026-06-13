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
		OutcomeCriteria:     repos.OutcomeCriteria,
		// Performance Evaluation (20260604 v1).
		Evaluation:             repos.Evaluation,
		EvaluationResponse:     repos.EvaluationResponse,
		EvaluationTemplate:     repos.EvaluationTemplate,
		EvaluationTemplateItem: repos.EvaluationTemplateItem,
		EvaluationCycle:        repos.EvaluationCycle,
		EvaluationCycleMember:  repos.EvaluationCycleMember,
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
