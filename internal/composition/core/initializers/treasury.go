package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/treasury"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeTreasury creates all treasury use cases from provider repositories.
// Currently returns an empty TreasuryUseCases since all legacy payment entities
// (Payment, PaymentAttribute, PaymentMethod, PaymentProfile) have been removed.
// Their functionality is superseded by Collection and Disbursement domains.
func InitializeTreasury(
	repos *domain.TreasuryRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*treasury.TreasuryUseCases, error) {
	return treasury.NewUseCases(
		treasury.TreasuryRepositories{},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
