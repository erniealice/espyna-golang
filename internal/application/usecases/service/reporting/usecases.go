// Package reporting hosts the Reporting umbrella sub-aggregate for the
// service-driven domain.
//
// Per docs/plan/20260520-service-domain-migration/wave-b-surface-map.md
// and the Q-SDM-LEDGER-INTERFACE lock (no Go interface; proto-RPC
// contracts directly), the previous 15-method `ledgerReportingInner`
// duck interface (formerly in `apps/service-admin/internal/composition/
// ledger_reporting.go`, now DELETED per 20260521-composition-reshape;
// logic inlined into `container.go`) decomposes into 5 report-group
// sub-candidates per Q-SDM-LEDGER-DECOMP. Each sub-candidate is
// **app-visible** (consumed by service-admin through fycha-golang,
// entydad-golang, and centymo-golang report views), so per
// Q-ORCH-2-REFINEMENT they MUST use typed fields on `ServiceUseCases`
// (the dynamic registry path is blocked by Go's `internal/` visibility
// rule at app callsites).
//
// Rather than add 5 separate typed fields to `service.ServiceUseCases`
// (high merge contention during the parallel Wave B sprint), this
// package introduces a single `Reporting *ReportingUseCases` umbrella
// on `ServiceUseCases`. Each Wave B candidate ADDS its typed pointer
// to this struct (low contention — small file, isolated changes) instead
// of editing the main aggregate.
//
// **Wave B candidate assignment points:** every field on
// [ReportingUseCases] is initially `any` carrying nil. Each candidate
// Wave B agent will:
//
//  1. Land the per-candidate package `service/reporting/<group>/`.
//  2. Promote its field on [ReportingUseCases] from `any` to
//     `*<group>.UseCases` (typed pointer).
//  3. Replace the nil literal in [NewReportingUseCases] with the
//     real factory call.
//  4. Thread any new typed deps through [Deps] (or piggyback on
//     existing deps when sufficient).
//
// Until a candidate lands, the field stays `any` carrying nil so apps
// can already navigate the chain `uc.Service.Reporting.ARAging`
// (returning nil) without compile-time visibility issues. Once
// promoted to a typed pointer, callsites can dereference `.Execute(...)`.
//
// The 15-method → 5-group allocation is locked in decisions.md
// §Q-SDM-LEDGER-INTERFACE:
//
//	ARAging          — GetReceivablesAgingReport, GetCollectionSummaryReport
//	APAging          — GetPayablesAgingReport, GetSimplePayablesAgingReport
//	GrossCashFlow    — GetGrossProfitReport, GetCashBookReport
//	Statements       — GetClientStatement, GetSupplierStatement,
//	                   GetClientBalances, GetSupplierBalances
//	DomainSpecific   — GetRevenueReport, GetExpenditureReport,
//	                   GetDisbursementReport, ListRevenue, ListExpenses
package reporting

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting/ap_aging"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting/ar_aging"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting/domain_specific"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting/gross_cashflow"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting/statements"
)

// Deps is the dependency surface for the Reporting umbrella factory.
//
// **Wave B extension point:** as each candidate lands and discovers
// what underlying adapter/repository it needs (the postgres
// `LedgerReportingAdapter` is the current source of truth for every
// method), the candidate's Wave B agent extends this struct rather
// than the top-level `service.Deps` — keeps reporting-only deps out
// of the service-wide deps junk drawer.
//
// The initial shape mirrors the parts of `service.Deps` every report
// group plausibly needs (DB handle, authorization, translation).
// Defined independently here to avoid a `reporting → service →
// reporting` import cycle.
type Deps struct {
	DB         *sql.DB
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper

	// ARAgingReporter carries the raw postgres LedgerReportingAdapter as
	// `any`. The unexported `ar_aging.reporter` interface is the structural
	// contract; the assertion happens inside the ar_aging package so apps
	// (and the umbrella here) never need to name the unexported type.
	//
	// May be nil — `ar_aging.NewUseCases` degrades to an empty response.
	//
	// **Type changed `ar_aging.Reporter` → `any`** (codex review
	// REVISE-MINOR P1, 2026-05-21): the prior exported `Reporter` name
	// invited apps to assert against it directly, which Go's `internal/`
	// visibility rule blocks anyway. Carrying `any` here matches the
	// `[SetReporterFromAny]` entry point on `*ar_aging.UseCases` and keeps
	// the narrow port private to the leaf package.
	//
	// **Per-candidate extension point:** each P1.E.* sub-candidate that
	// lands adds its own `any`-typed reporter field (per this lock).
	ARAgingReporter any

	// APAgingReporter carries the raw postgres LedgerReportingAdapter as
	// `any` for the P1.E.2 AP aging sub-candidate. Same shape + rationale
	// as ARAgingReporter — the unexported `ap_aging.reporter` interface
	// stays private to the leaf package; the assertion happens inside
	// `ap_aging.NewUseCases` / [SetReporterFromAny].
	//
	// May be nil — `ap_aging.NewUseCases` degrades to an empty response.
	APAgingReporter any

	// GrossCashFlowReporter carries the raw postgres LedgerReportingAdapter
	// as `any` for the P1.E.3 gross/cashflow sub-candidate. Same shape +
	// rationale as ARAgingReporter — the unexported `gross_cashflow.
	// reporter` interface stays private to the leaf package.
	//
	// May be nil — `gross_cashflow.NewUseCases` degrades to an empty
	// response.
	GrossCashFlowReporter any

	// StatementsReporter carries the raw postgres LedgerReportingAdapter
	// as `any` for the P1.E.4 statements sub-candidate. Same shape +
	// rationale as ARAgingReporter — the unexported `statements.reporter`
	// interface stays private to the leaf package.
	//
	// May be nil — `statements.NewUseCases` degrades to an empty response.
	StatementsReporter any

	// DomainSpecificReporter carries the raw postgres LedgerReportingAdapter
	// as `any` for the P1.E.5 domain-specific sub-candidate. Same shape +
	// rationale as ARAgingReporter — the unexported `domain_specific.
	// reporter` interface stays private to the leaf package.
	//
	// May be nil — `domain_specific.NewUseCases` degrades to an empty
	// response. P1.E.5 is the LAST sub-candidate; its commit retired
	// `apps/service-admin/internal/composition/ledger_reporting.go`
	// entirely (FILE DELETED, 20260521-composition-reshape; logic inlined
	// into `container.go`) and removed the `pyeza.AppContext.LedgerReportingSvc`
	// field per Q-SDM-LEDGER-RETIRE-PYEZA. Both retirements landed 2026-05-21
	// alongside the downstream rewires (fycha/centymo/entydad view
	// consumers now thread typed closures off
	// `useCases.Service.Reporting.<Group>` instead of the wrapper).
	DomainSpecificReporter any
}

// ReportingUseCases aggregates every service-driven ledger reporting
// sub-candidate. Each field corresponds to one P1.E.* sub-phase from
// wave-b-surface-map.md / decisions.md §Q-SDM-LEDGER-INTERFACE.
//
// **Field typing convention:** every field is initially `any` carrying
// nil. Wave B candidate agents promote their field to the typed pointer
// `*<package>.UseCases` (matching the Audit/Security/Auth canonical
// shape on `ServiceUseCases`) once the per-candidate package lands.
type ReportingUseCases struct {
	// ARAging — LANDED 2026-05-20 (P1.E.1).
	// Hosts GetReceivablesAgingReport + GetCollectionSummaryReport,
	// migrated from ledger_reporting.go:70, :72.
	ARAging *ar_aging.UseCases

	// APAging — LANDED 2026-05-21 (P1.E.2).
	// Hosts GetPayablesAgingReport + GetSimplePayablesAgingReport,
	// migrated from ledger_reporting.go:71 (parameterized) + :80 (simple).
	APAging *ap_aging.UseCases

	// GrossCashFlow — LANDED 2026-05-21 (P1.E.3).
	// Hosts GetGrossProfitReport + GetCashBookReport, migrated from
	// ledger_reporting.go:66, :79. The future `net_profit`/`cash_flow`
	// reports named in Q-SDM-LEDGER-DECOMP will join this package when
	// authored — the directory name is forward-compatible.
	GrossCashFlow *gross_cashflow.UseCases

	// Statements — LANDED 2026-05-21 (P1.E.4).
	// Hosts GetClientStatement + GetSupplierStatement + ListClientBalances
	// + ListSupplierBalances, migrated from ledger_reporting.go:73, :74,
	// :75, :76. Per Q-SDM-MAP-SHAPES the Balances responses use typed
	// `repeated BalanceRow` proto (no `google.protobuf.Struct`).
	Statements *statements.UseCases

	// DomainSpecific — LANDED 2026-05-21 (P1.E.5, FINAL sub-candidate).
	// Hosts GetRevenueReport + GetExpenditureReport + GetDisbursementReport
	// (proto-shaped) + ListRevenue + ListExpenses (Go-only per
	// Q-SDM-MAP-SHAPES), migrated from the former `ledger_reporting.go`
	// (methods :67, :68, :69, :77, :78). This sub-phase's commit also
	// DELETED `apps/service-admin/internal/composition/ledger_reporting.go`
	// entirely (logic inlined into `container.go`) and removed the
	// `pyeza.AppContext.LedgerReportingSvc` field per Q-SDM-LEDGER-RETIRE-
	// PYEZA. Both retirements landed 2026-05-21 alongside the downstream
	// rewires.
	DomainSpecific *domain_specific.UseCases
}

// NewReportingUseCases constructs the umbrella aggregate. Initial body
// returns a struct with every field set to nil — Wave B candidate
// agents replace each nil with their candidate's factory call as they
// land their package.
//
// **Wave B per-candidate edit pattern:**
//
//  1. Replace the typed `any` on the relevant field above with the
//     candidate's typed pointer (e.g. `ARAging *ar_aging.UseCases`).
//  2. Replace the nil literal below with the factory call (e.g.
//     `ARAging: ar_aging.NewUseCases(ar_aging.Repositories{...}, ...)`).
//  3. Add any new typed deps needed to [Deps] above — keeps
//     reporting-specific deps out of the parent service.Deps.
//
// `deps` may be nil during unit tests that don't exercise any sub-
// candidate's wiring. Each sub-candidate's factory must tolerate nil
// dependencies (same nil-safety contract as Audit/Security on the
// parent aggregate).
func NewReportingUseCases(deps *Deps) *ReportingUseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &ReportingUseCases{
		ARAging: ar_aging.NewUseCases(&ar_aging.Deps{
			Reporter:         deps.ARAgingReporter,
			Authorizer:       deps.Authorizer,
			Translator:       deps.Translator,
			ActionGatekeeper: deps.ActionGatekeeper,
		}),
		APAging: ap_aging.NewUseCases(&ap_aging.Deps{
			Reporter:         deps.APAgingReporter,
			Authorizer:       deps.Authorizer,
			Translator:       deps.Translator,
			ActionGatekeeper: deps.ActionGatekeeper,
		}),
		GrossCashFlow: gross_cashflow.NewUseCases(&gross_cashflow.Deps{
			Reporter:         deps.GrossCashFlowReporter,
			Authorizer:       deps.Authorizer,
			Translator:       deps.Translator,
			ActionGatekeeper: deps.ActionGatekeeper,
		}),
		Statements: statements.NewUseCases(&statements.Deps{
			Reporter:         deps.StatementsReporter,
			Authorizer:       deps.Authorizer,
			Translator:       deps.Translator,
			ActionGatekeeper: deps.ActionGatekeeper,
		}),
		DomainSpecific: domain_specific.NewUseCases(&domain_specific.Deps{
			Reporter:         deps.DomainSpecificReporter,
			Authorizer:       deps.Authorizer,
			Translator:       deps.Translator,
			ActionGatekeeper: deps.ActionGatekeeper,
		}),
	}
}
