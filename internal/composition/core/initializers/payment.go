package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/payment"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializePayment creates all payment use cases from provider repositories.
// Currently returns an empty PaymentUseCases since all legacy payment entities
// (Payment, PaymentAttribute, PaymentMethod, PaymentProfile) have been removed.
// Their functionality is superseded by Collection and Disbursement domains.
func InitializePayment(
	repos *domain.PaymentRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*payment.PaymentUseCases, error) {
	return payment.NewUseCases(
		payment.PaymentRepositories{},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
