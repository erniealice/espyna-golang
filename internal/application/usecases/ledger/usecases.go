package ledger

import (
	// Document template use cases
	documentTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/document/template"

	// Attachment use cases
	attachmentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/document/attachment"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Reporting use cases
	cashbookreporting "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/reporting/cash_book"
	grossprofit "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/reporting/gross_profit"
	simplepayablesaging "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/reporting/simple_payables_aging"

	// Chart of Accounts use cases
	accountUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/account"
	fiscalPeriodUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/fiscal_period"
	journalEntryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/journal_entry"

	// Dashboard use cases
	ledgerdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/dashboard"
	equitydashboard "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/equity_dashboard"

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

	// Dashboard use cases (nil when postgres build tag is inactive).
	Dashboard       *ledgerdashboard.GetLedgerDashboardPageDataUseCase
	EquityDashboard *equitydashboard.GetEquityDashboardPageDataUseCase
}

// NewUseCases creates all ledger use cases with proper constructor injection.
func NewUseCases(
	repos LedgerRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *LedgerUseCases {
	var documentTemplateUC *documentTemplateUseCases.UseCases
	if repos.DocumentTemplate != nil {
		documentTemplateUC = documentTemplateUseCases.NewUseCases(
			documentTemplateUseCases.DocumentTemplateRepositories{
				DocumentTemplate: repos.DocumentTemplate,
			},
			documentTemplateUseCases.DocumentTemplateServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
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
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
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
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
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
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
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
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
		)
	}

	// Wire ledger dashboard use case via type assertions — safe when postgres
	// build tag is inactive (assertion fails and Dashboard stays nil).
	var ledgerDash *ledgerdashboard.GetLedgerDashboardPageDataUseCase
	if repos.Account != nil && repos.JournalEntry != nil {
		accountQ, aOK := repos.Account.(ledgerdashboard.AccountDashboardQueries)
		journalQ, jOK := repos.JournalEntry.(ledgerdashboard.JournalEntryDashboardQueries)
		if aOK && jOK {
			ledgerDash = ledgerdashboard.NewGetLedgerDashboardPageDataUseCase(accountQ, journalQ)
		}
	}

	// Wire equity dashboard use case via type assertions.
	var equityDash *equitydashboard.GetEquityDashboardPageDataUseCase
	if repos.EquityAccount != nil && repos.EquityTransaction != nil {
		eaQ, eaOK := repos.EquityAccount.(equitydashboard.EquityAccountDashboardQueries)
		etQ, etOK := repos.EquityTransaction.(equitydashboard.EquityTransactionDashboardQueries)
		if eaOK && etOK {
			equityDash = equitydashboard.NewGetEquityDashboardPageDataUseCase(eaQ, etQ)
		}
	}

	return &LedgerUseCases{
		DocumentTemplate:             documentTemplateUC,
		Attachment:                   attachmentUC,
		GetGrossProfitReport:         grossprofit.NewGetGrossProfitReportUseCase(repos.ReportingService),
		GetCashBookReport:            cashbookreporting.NewGetCashBookReportUseCase(repos.ReportingService),
		GetSimplePayablesAgingReport: simplepayablesaging.NewGetSimplePayablesAgingReportUseCase(repos.ReportingService),
		Account:                      accountUC,
		JournalEntry:                 journalEntryUC,
		FiscalPeriod:                 fiscalPeriodUC,
		Dashboard:                    ledgerDash,
		EquityDashboard:              equityDash,
	}
}
