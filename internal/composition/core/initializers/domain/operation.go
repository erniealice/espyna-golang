package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/operation"
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
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
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
	}
	if subRepos != nil {
		opRepos.BillingEvent = subRepos.BillingEvent
		opRepos.Subscription = subRepos.Subscription
		opRepos.PricePlan = subRepos.PricePlan
		opRepos.ProductPricePlan = subRepos.ProductPricePlan
	}
	return operation.NewUseCases(
		opRepos,
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
