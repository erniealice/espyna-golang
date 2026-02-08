package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/payment"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializePayment creates all payment use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializePayment(
	repos *domain.PaymentRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*payment.PaymentUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return payment.NewUseCases(
		payment.PaymentRepositories{
			Payment:          repos.Payment,
			PaymentAttribute: repos.PaymentAttribute,
			PaymentMethod:    repos.PaymentMethod,
			PaymentProfile:   repos.PaymentProfile,
			Attribute:        repos.Attribute,
			Client:           repos.Client,
			Subscription:     repos.Subscription,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
