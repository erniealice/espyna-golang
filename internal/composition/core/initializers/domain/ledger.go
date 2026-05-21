package domain

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
			// Existing document repositories
			DocumentTemplate: repos.DocumentTemplate,
			Attachment:       repos.Attachment,

			// Chart of Accounts repositories
			Account:                  repos.Account,
			AccountGroup:             repos.AccountGroup,
			AccountTemplate:          repos.AccountTemplate,
			JournalEntry:             repos.JournalEntry,
			JournalLine:              repos.JournalLine,
			FiscalPeriod:             repos.FiscalPeriod,
			RecurringJournalTemplate: repos.RecurringJournalTemplate,
			EquityAccount:            repos.EquityAccount,
			EquityTransaction:        repos.EquityTransaction,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
