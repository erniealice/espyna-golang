package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription"
	subscriptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/subscription"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeSubscription creates all subscription use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeSubscription(
	repos *domain.SubscriptionRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	jobTemplateInstantiator subscriptionUseCases.JobTemplateInstantiator,
	refChecker ports.ReferenceChecker,
) (*subscription.SubscriptionUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return subscription.NewUseCases(
		subscription.SubscriptionRepositories{
			Balance:               repos.Balance,
			BalanceAttribute:      repos.BalanceAttribute,
			BillingEvent:          repos.BillingEvent,
			Client:                repos.Client,
			Invoice:               repos.Invoice,
			InvoiceAttribute:      repos.InvoiceAttribute,
			Plan:                  repos.Plan,
			PlanAttribute:         repos.PlanAttribute,
			PlanSettings:          repos.PlanSettings,
			PricePlan:             repos.PricePlan,
			PriceSchedule:         repos.PriceSchedule,
			ProductPlan:           repos.ProductPlan,
			ProductPricePlan:      repos.ProductPricePlan,
			Subscription:          repos.Subscription,
			SubscriptionAttribute: repos.SubscriptionAttribute,
			Attribute:             repos.Attribute,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		jobTemplateInstantiator,
		refChecker,
	), nil
}
