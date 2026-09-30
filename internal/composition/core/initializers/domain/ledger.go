package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeLedger creates all ledger use cases from provider repositories
func InitializeLedger(
	repos *domain.LedgerRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*ledger.LedgerUseCases, error) {
	return ledger.NewUseCases(
		ledger.LedgerRepositories{
			// Slice B known-cost recovery (S1)
			ChargeEffect: repos.ChargeEffect,
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

			ChargePolicy:              repos.ChargePolicy,
			ChargePolicyVersion:       repos.ChargePolicyVersion,
			ChargePolicyComponent:     repos.ChargePolicyComponent,
			ChargePolicyPosting:       repos.ChargePolicyPosting,
			ChargePolicyVersionEditor: repos.ChargePolicyVersionEditor,
			TaxTreatment:              repos.TaxTreatment,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		actionGate,
	), nil
}
