package ledger

import (
	// Document template use cases
	documentTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/document/template"

	// Attachment use cases
	attachmentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/document/attachment"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Reporting use cases
	cashbookreporting "github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/reporting/cash_book"
	grossprofit "github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/reporting/gross_profit"
	simplepayablesaging "github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/reporting/simple_payables_aging"

	// Chart of Accounts use cases
	accountUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/account"
	fiscalPeriodUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/fiscal_period"
	journalEntryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/journal_entry"

	// Protobuf domain services for ledger repositories
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	accountgrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account_group"
	accounttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account_template"
	equityaccountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_account"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
	journallinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_line"
	recurringjournaltemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/recurring_journal_template"
)

// LedgerRepositories groups all repository dependencies for ledger use cases.
type LedgerRepositories struct {
	// Existing document repositories
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
	Attachment       attachmentpb.AttachmentDomainServiceServer
	ReportingService ports.LedgerReportingService

	// Chart of Accounts repositories
	Account                  accountpb.AccountDomainServiceServer
	AccountGroup             accountgrouppb.AccountGroupDomainServiceServer
	AccountTemplate          accounttemplatepb.AccountTemplateDomainServiceServer
	JournalEntry             journalentrypb.JournalEntryDomainServiceServer
	JournalLine              journallinepb.JournalLineDomainServiceServer
	FiscalPeriod             fiscalperiodpb.FiscalPeriodDomainServiceServer
	RecurringJournalTemplate recurringjournaltemplatepb.RecurringJournalTemplateDomainServiceServer
	EquityAccount            equityaccountpb.EquityAccountDomainServiceServer
	EquityTransaction        equitytransactionpb.EquityTransactionDomainServiceServer
}

// LedgerUseCases contains all ledger-related use cases.
type LedgerUseCases struct {
	DocumentTemplate             *documentTemplateUseCases.UseCases
	Attachment                   *attachmentUseCases.UseCases
	GetGrossProfitReport         *grossprofit.GetGrossProfitReportUseCase
	GetCashBookReport            *cashbookreporting.GetCashBookReportUseCase
	GetSimplePayablesAgingReport *simplepayablesaging.GetSimplePayablesAgingReportUseCase

	// Chart of Accounts use cases (Phase 2 priority)
	Account      *accountUseCases.UseCases
	JournalEntry *journalEntryUseCases.UseCases
	FiscalPeriod *fiscalPeriodUseCases.UseCases

	// Dashboard fields retired 2026-05-21 (Wave B P1.C.3 Ledger + P1.C.4
	// Equity) — both ledger + equity dashboards now live under
	// `service.Dashboard.Ledger` and `service.Dashboard.Equity`. The
	// `usecases/ledger/dashboard/` and `usecases/ledger/equity_dashboard/`
	// packages are retired in the same commit per Q-SDM-DASHBOARD-DOWNSTREAM.
}

// NewUseCases creates all ledger use cases with proper constructor injection.
func NewUseCases(
	repos LedgerRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
) *LedgerUseCases {
	var documentTemplateUC *documentTemplateUseCases.UseCases
	if repos.DocumentTemplate != nil {
		documentTemplateUC = documentTemplateUseCases.NewUseCases(
			documentTemplateUseCases.DocumentTemplateRepositories{
				DocumentTemplate: repos.DocumentTemplate,
			},
			documentTemplateUseCases.DocumentTemplateServices{
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
		)
	}

	var attachmentUC *attachmentUseCases.UseCases
	if repos.Attachment != nil {
		attachmentUC = attachmentUseCases.NewUseCases(
			attachmentUseCases.AttachmentRepositories{
				Attachment: repos.Attachment,
			},
			attachmentUseCases.AttachmentServices{
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
		)
	}

	var accountUC *accountUseCases.UseCases
	if repos.Account != nil {
		accountUC = accountUseCases.NewUseCases(
			accountUseCases.AccountRepositories{
				Account: repos.Account,
			},
			accountUseCases.AccountServices{
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
		)
	}

	var journalEntryUC *journalEntryUseCases.UseCases
	if repos.JournalEntry != nil {
		journalEntryUC = journalEntryUseCases.NewUseCases(
			journalEntryUseCases.JournalEntryRepositories{
				JournalEntry: repos.JournalEntry,
			},
			journalEntryUseCases.JournalEntryServices{
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
		)
	}

	var fiscalPeriodUC *fiscalPeriodUseCases.UseCases
	if repos.FiscalPeriod != nil {
		fiscalPeriodUC = fiscalPeriodUseCases.NewUseCases(
			fiscalPeriodUseCases.FiscalPeriodRepositories{
				FiscalPeriod: repos.FiscalPeriod,
			},
			fiscalPeriodUseCases.FiscalPeriodServices{
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idService,
			},
		)
	}

	// Ledger + Equity dashboard wiring retired 2026-05-21 (Wave B P1.C.3 +
	// P1.C.4) — type-assertion + factory wiring now lives in the service-
	// layer initializer at `internal/composition/core/initializers/service.go`
	// (search "Wave B P1.C.3 Ledger" and "Wave B P1.C.4 Equity").
	return &LedgerUseCases{
		DocumentTemplate:             documentTemplateUC,
		Attachment:                   attachmentUC,
		GetGrossProfitReport:         grossprofit.NewGetGrossProfitReportUseCase(repos.ReportingService),
		GetCashBookReport:            cashbookreporting.NewGetCashBookReportUseCase(repos.ReportingService),
		GetSimplePayablesAgingReport: simplepayablesaging.NewGetSimplePayablesAgingReportUseCase(repos.ReportingService),
		Account:                      accountUC,
		JournalEntry:                 journalEntryUC,
		FiscalPeriod:                 fiscalPeriodUC,
	}
}
