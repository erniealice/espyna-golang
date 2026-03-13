package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/ledger"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeLedger creates all ledger use cases from provider repositories
func InitializeLedger(
	repos *domain.LedgerRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*ledger.LedgerUseCases, error) {
	return ledger.NewUseCases(
		ledger.LedgerRepositories{
			DocumentTemplate: repos.DocumentTemplate,
			Attachment:       repos.Attachment,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
