package domain

import (
	"fmt"

	ledgeruc "github.com/erniealice/espyna-golang/internal/application/usecases/ledger"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// ConfigureLedgerDomain configures routes for the Ledger domain.
// Note: Ledger reports (gross profit, balance sheet, etc.) are served via fycha view
// package and do not go through the standard API route configuration.
func ConfigureLedgerDomain(ledgerUseCases *ledgeruc.LedgerUseCases) contracts.DomainRouteConfiguration {
	if ledgerUseCases == nil {
		fmt.Printf("Ledger use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "ledger",
			Prefix:  "/ledger",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// Account routes
	if ledgerUseCases.Account != nil {
		if ledgerUseCases.Account.CreateAccount != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/account/create",
				Handler: contracts.NewGenericHandler(ledgerUseCases.Account.CreateAccount, &accountpb.CreateAccountRequest{}),
			})
		}
		if ledgerUseCases.Account.ReadAccount != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/account/read",
				Handler: contracts.NewGenericHandler(ledgerUseCases.Account.ReadAccount, &accountpb.ReadAccountRequest{}),
			})
		}
		if ledgerUseCases.Account.UpdateAccount != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/account/update",
				Handler: contracts.NewGenericHandler(ledgerUseCases.Account.UpdateAccount, &accountpb.UpdateAccountRequest{}),
			})
		}
		if ledgerUseCases.Account.DeleteAccount != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/account/delete",
				Handler: contracts.NewGenericHandler(ledgerUseCases.Account.DeleteAccount, &accountpb.DeleteAccountRequest{}),
			})
		}
		if ledgerUseCases.Account.ListAccounts != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/account/list",
				Handler: contracts.NewGenericHandler(ledgerUseCases.Account.ListAccounts, &accountpb.ListAccountsRequest{}),
			})
		}
		if ledgerUseCases.Account.GetAccountListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/account/get-list-page-data",
				Handler: contracts.NewGenericHandler(ledgerUseCases.Account.GetAccountListPageData, &accountpb.GetAccountListPageDataRequest{}),
			})
		}
	}

	// JournalEntry routes
	if ledgerUseCases.JournalEntry != nil {
		if ledgerUseCases.JournalEntry.CreateJournalEntry != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/create",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.CreateJournalEntry, &journalentrypb.CreateJournalEntryRequest{}),
			})
		}
		if ledgerUseCases.JournalEntry.ReadJournalEntry != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/read",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.ReadJournalEntry, &journalentrypb.ReadJournalEntryRequest{}),
			})
		}
		if ledgerUseCases.JournalEntry.UpdateJournalEntry != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/update",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.UpdateJournalEntry, &journalentrypb.UpdateJournalEntryRequest{}),
			})
		}
		if ledgerUseCases.JournalEntry.DeleteJournalEntry != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/delete",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.DeleteJournalEntry, &journalentrypb.DeleteJournalEntryRequest{}),
			})
		}
		if ledgerUseCases.JournalEntry.ListJournalEntries != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/list",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.ListJournalEntries, &journalentrypb.ListJournalEntriesRequest{}),
			})
		}
		if ledgerUseCases.JournalEntry.GetJournalEntryListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/get-list-page-data",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.GetJournalEntryListPageData, &journalentrypb.GetJournalEntryListPageDataRequest{}),
			})
		}
		if ledgerUseCases.JournalEntry.PostJournalEntry != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/post",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.PostJournalEntry, &journalentrypb.PostJournalEntryRequest{}),
			})
		}
		if ledgerUseCases.JournalEntry.ReverseJournalEntry != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/journal-entry/reverse",
				Handler: contracts.NewGenericHandler(ledgerUseCases.JournalEntry.ReverseJournalEntry, &journalentrypb.ReverseJournalEntryRequest{}),
			})
		}
	}

	// FiscalPeriod routes
	if ledgerUseCases.FiscalPeriod != nil {
		if ledgerUseCases.FiscalPeriod.CreateFiscalPeriod != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/fiscal-period/create",
				Handler: contracts.NewGenericHandler(ledgerUseCases.FiscalPeriod.CreateFiscalPeriod, &fiscalperiodpb.CreateFiscalPeriodRequest{}),
			})
		}
		if ledgerUseCases.FiscalPeriod.ReadFiscalPeriod != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/fiscal-period/read",
				Handler: contracts.NewGenericHandler(ledgerUseCases.FiscalPeriod.ReadFiscalPeriod, &fiscalperiodpb.ReadFiscalPeriodRequest{}),
			})
		}
		if ledgerUseCases.FiscalPeriod.ListFiscalPeriods != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/fiscal-period/list",
				Handler: contracts.NewGenericHandler(ledgerUseCases.FiscalPeriod.ListFiscalPeriods, &fiscalperiodpb.ListFiscalPeriodsRequest{}),
			})
		}
		if ledgerUseCases.FiscalPeriod.GetFiscalPeriodListPageData != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/fiscal-period/get-list-page-data",
				Handler: contracts.NewGenericHandler(ledgerUseCases.FiscalPeriod.GetFiscalPeriodListPageData, &fiscalperiodpb.GetFiscalPeriodListPageDataRequest{}),
			})
		}
		if ledgerUseCases.FiscalPeriod.CloseFiscalPeriod != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/ledger/fiscal-period/close",
				Handler: contracts.NewGenericHandler(ledgerUseCases.FiscalPeriod.CloseFiscalPeriod, &fiscalperiodpb.CloseFiscalPeriodRequest{}),
			})
		}
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "ledger",
		Prefix:  "/ledger",
		Enabled: len(routes) > 0,
		Routes:  routes,
	}
}
