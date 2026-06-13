package service

import (
	"database/sql"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	reportingusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting"
)

// ledgerReportingTableConfig configures table names for the registry
// factory. Field names + types must match the reflection-based reader in
// `contrib/postgres/ledger.go` — this struct is passed to the factory as
// `any`, so the field names are load-bearing.
//
// Moved from apps/service-admin/internal/composition/container.go
// (20260614-composition-slim): the table config is an espyna-internal
// concern; apps should never see raw adapter configuration.
type ledgerReportingTableConfig struct {
	Revenue              string
	RevenueLineItem      string
	InventoryTransaction string
	InventoryItem        string
	Product              string
	Location             string
	RevenueCategory      string
	Expenditure          string
	ExpenditureLineItem  string
	ExpenditureCategory  string
	Supplier             string
	ProductCollection    string
	Collection           string
	Line                 string
	LocationArea         string
	TreasuryDisbursement string
	DisbursementMethod   string
	SupplierCategory     string
	Client               string
	ClientCategory       string
	Category             string
	TreasuryCollection   string
	CollectionMethod     string
	PaymentTerm          string
}

// buildLedgerReportingAdapter creates the raw ledger reporting adapter from
// the registry factory using the canonical table config. Returns nil when
// no SQL provider or no factory is available (mock / non-postgres builds).
//
// Moved from apps/service-admin/internal/composition/reporting.go
// (20260614-composition-slim): the adapter construction is an espyna-
// internal concern; apps should never import the registry or configure
// table names directly.
func buildLedgerReportingAdapter(db *sql.DB) any {
	if db == nil {
		return nil
	}
	factory, ok := internalregistry.GetLedgerReportingFactory()
	if !ok || factory == nil {
		return nil
	}
	tableConfig := ledgerReportingTableConfig{
		Revenue:              "revenue",
		RevenueLineItem:      "revenue_line_item",
		InventoryTransaction: "inventory_transaction",
		InventoryItem:        "inventory_item",
		Product:              "product",
		Location:             "location",
		RevenueCategory:      "revenue_category",
		Expenditure:          "expenditure",
		ExpenditureLineItem:  "expenditure_line_item",
		ExpenditureCategory:  "expenditure_category",
		Supplier:             "supplier",
		ProductCollection:    "product_collection",
		Collection:           "collection",
		Line:                 "line",
		LocationArea:         "location_area",
		TreasuryDisbursement: "treasury_disbursement",
		DisbursementMethod:   "disbursement_method",
		SupplierCategory:     "supplier_category",
		Client:               "client",
		ClientCategory:       "client_category",
		Category:             "category",
		TreasuryCollection:   "treasury_collection",
		CollectionMethod:     "collection_method",
		PaymentTerm:          "payment_term",
	}
	return factory(db, tableConfig)
}

// initServiceReporting wires the service-layer Reporting umbrella sub-aggregate.
//
// 20260614 — the adapter is now built internally via
// buildLedgerReportingAdapter + GetLedgerReportingFactory. Apps no longer
// need to construct the raw adapter or call SetReporterFromAny post-hoc;
// the table config and wiring are fully encapsulated in espyna.
//
// One shared `rawAdapter` value satisfies every group's reporter port
// because the postgres `LedgerReportingAdapter` exposes the union of all
// methods structurally — splitting the assertion across 5 leaves (rather
// than asserting a fat union here) keeps each leaf's port narrow.
//
// Loud-fail: after construction, each sub-group's reporter may be nil if
// the raw adapter does not satisfy its narrow interface. We call
// SetReporterFromAny to detect and log assertion failures, matching the
// previous app-side behavior.
func initServiceReporting(
	db *sql.DB,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
	actionGate *actiongate.ActionGatekeeper,
) *reportingusecases.ReportingUseCases {
	rawAdapter := buildLedgerReportingAdapter(db)

	reportingDeps := &reportingusecases.Deps{
		DB:                     db,
		Authorizer:             authSvc,
		Translator:             i18nSvc,
		ActionGatekeeper:       actionGate,
		ARAgingReporter:        rawAdapter,
		APAgingReporter:        rawAdapter,
		GrossCashFlowReporter:  rawAdapter,
		StatementsReporter:     rawAdapter,
		DomainSpecificReporter: rawAdapter,
	}
	rpt := reportingusecases.NewReportingUseCases(reportingDeps)

	// Loud-fail logging: SetReporterFromAny returns false when the assertion
	// fails despite a non-nil adapter. Log per group so a drift in the
	// postgres adapter's method signatures surfaces at boot, not via empty
	// reports in production.
	if rawAdapter != nil {
		if rpt.ARAging != nil {
			if ok := rpt.ARAging.SetReporterFromAny(rawAdapter); !ok {
				log.Printf("WARN: AR aging reporter assertion failed; raw adapter %T does not satisfy ar_aging.reporter — AR aging reports will render empty. Check postgres LedgerReportingAdapter method signatures.", rawAdapter)
			}
		}
		if rpt.APAging != nil {
			if ok := rpt.APAging.SetReporterFromAny(rawAdapter); !ok {
				log.Printf("WARN: AP aging reporter assertion failed; raw adapter %T does not satisfy ap_aging.reporter — AP aging reports will render empty. Check postgres LedgerReportingAdapter method signatures.", rawAdapter)
			}
		}
		if rpt.GrossCashFlow != nil {
			if ok := rpt.GrossCashFlow.SetReporterFromAny(rawAdapter); !ok {
				log.Printf("WARN: Gross/CashFlow reporter assertion failed; raw adapter %T does not satisfy gross_cashflow.reporter — Gross profit + cash book reports will render empty. Check postgres LedgerReportingAdapter method signatures.", rawAdapter)
			}
		}
		if rpt.Statements != nil {
			if ok := rpt.Statements.SetReporterFromAny(rawAdapter); !ok {
				log.Printf("WARN: Statements reporter assertion failed; raw adapter %T does not satisfy statements.reporter — Client/Supplier statements + balances will render empty. Check postgres LedgerReportingAdapter method signatures.", rawAdapter)
			}
		}
		if rpt.DomainSpecific != nil {
			if ok := rpt.DomainSpecific.SetReporterFromAny(rawAdapter); !ok {
				log.Printf("WARN: Domain-specific reporter assertion failed; raw adapter %T does not satisfy domain_specific.reporter — Revenue/Expenditure/Disbursement reports + ListRevenue/ListExpenses feeders will render empty. Check postgres LedgerReportingAdapter method signatures.", rawAdapter)
			}
		}
	}

	return rpt
}
