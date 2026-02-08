package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/subscription"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeSubscription creates all subscription use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeSubscription(
	repos *domain.SubscriptionRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*subscription.SubscriptionUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return subscription.NewUseCases(
		subscription.SubscriptionRepositories{
			Balance:               repos.Balance,
			BalanceAttribute:      repos.BalanceAttribute,
			Client:                repos.Client,
			Invoice:               repos.Invoice,
			InvoiceAttribute:      repos.InvoiceAttribute,
			Plan:                  repos.Plan,
			PlanAttribute:         repos.PlanAttribute,
			PlanSettings:          repos.PlanSettings,
			PricePlan:             repos.PricePlan,
			Subscription:          repos.Subscription,
			SubscriptionAttribute: repos.SubscriptionAttribute,
			Attribute:             repos.Attribute,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
