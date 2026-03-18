package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Document domain (existing)
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"

	// Protobuf domain services - Ledger / Chart of Accounts domain
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

// LedgerRepositories contains all ledger domain repositories
type LedgerRepositories struct {
	// Existing document repositories
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
	Attachment       attachmentpb.AttachmentDomainServiceServer

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

// NewLedgerRepositories creates and returns a new set of LedgerRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewLedgerRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*LedgerRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &LedgerRepositories{}
	var skipped []string

	// Helper: try to create a repository, log and skip on failure
	tryCreate := func(entity string) interface{} {
		repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
		if err != nil {
			skipped = append(skipped, entity)
			return nil
		}
		return repo
	}

	// Existing document repositories
	if r := tryCreate(entityid.DocumentTemplate); r != nil {
		repos.DocumentTemplate = r.(documenttemplatepb.DocumentTemplateDomainServiceServer)
	}
	if r := tryCreate(entityid.Attachment); r != nil {
		repos.Attachment = r.(attachmentpb.AttachmentDomainServiceServer)
	}

	// Chart of Accounts repositories
	if r := tryCreate(entityid.Account); r != nil {
		repos.Account = r.(accountpb.AccountDomainServiceServer)
	}
	if r := tryCreate(entityid.AccountGroup); r != nil {
		repos.AccountGroup = r.(accountgrouppb.AccountGroupDomainServiceServer)
	}
	if r := tryCreate(entityid.AccountTemplate); r != nil {
		repos.AccountTemplate = r.(accounttemplatepb.AccountTemplateDomainServiceServer)
	}
	if r := tryCreate(entityid.JournalEntry); r != nil {
		repos.JournalEntry = r.(journalentrypb.JournalEntryDomainServiceServer)
	}
	if r := tryCreate(entityid.JournalLine); r != nil {
		repos.JournalLine = r.(journallinepb.JournalLineDomainServiceServer)
	}
	if r := tryCreate(entityid.FiscalPeriod); r != nil {
		repos.FiscalPeriod = r.(fiscalperiodpb.FiscalPeriodDomainServiceServer)
	}
	if r := tryCreate(entityid.RecurringJournalTemplate); r != nil {
		repos.RecurringJournalTemplate = r.(recurringjournaltemplatepb.RecurringJournalTemplateDomainServiceServer)
	}
	if r := tryCreate(entityid.EquityAccount); r != nil {
		repos.EquityAccount = r.(equityaccountpb.EquityAccountDomainServiceServer)
	}
	if r := tryCreate(entityid.EquityTransaction); r != nil {
		repos.EquityTransaction = r.(equitytransactionpb.EquityTransactionDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Ledger repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
