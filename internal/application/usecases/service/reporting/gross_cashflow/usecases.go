// Package gross_cashflow hosts the service-driven gross-profit + cash-book
// reporting use cases (Wave B P1.E.3).
//
// Per docs/plan/20260520-service-domain-migration/decisions.md (Q-SDM-LEDGER-
// DECOMP + Q-SDM-LEDGER-INTERFACE) the 15-method `ledgerReportingInner`
// duck interface (formerly in `apps/service-admin/internal/composition/
// ledger_reporting.go`, FILE DELETED 20260521 per composition-reshape
// Q-SDM-LEDGER-RETIRE-PYEZA; logic inlined into `container.go`) decomposes
// into 5 report-group sub-candidates.
// This package is the third such sub-candidate (P1.E.3 — gross/cash flow)
// hosting:
//
//   - GetGrossProfitReport (formerly ledger_reporting.go:66)
//   - GetCashBookReport    (formerly ledger_reporting.go:79)
//
// The future `net_profit`/`cash_flow` reports named in Q-SDM-LEDGER-DECOMP
// don't exist yet; they will join this package when authored.
//
// Per Q-SDM-LEDGER-INTERFACE there is no replacement Go interface; the
// proto contract at `packages/esqyma/proto/v1/service/reporting/
// gross_cashflow/gross_cashflow.proto` is the canonical contract.
//
// Wiring: `internal/composition/core/initializers/service.go` constructs
// the per-group `UseCases` via `NewUseCases(deps)` and assigns the result
// into the `GrossCashFlow` field of `service/reporting.ReportingUseCases`.
// Apps consume `uc.Service.Reporting.GrossCashFlow.<Method>.Execute(ctx,
// *gross_cashflowpb.Get<X>Request)`.
//
// Codex pattern compliance (codex review of AR aging REVISE-MINOR P1+P2,
// 2026-05-21):
//   - `reporter` interface UNEXPORTED.
//   - `setReporter` UNEXPORTED.
//   - `SetReporterFromAny(any) bool` returns true on success.
//   - `Deps.Reporter any` (not typed) so apps can pass the raw adapter.
package gross_cashflow

import (
	"context"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
)

// reporter is the narrow port for gross-margin + cash-flow reports. The
// postgres `LedgerReportingAdapter` satisfies this interface structurally.
//
// **Unexported** (matching the AR aging pilot REVISE-MINOR P1, 2026-05-21).
// Defined locally rather than reusing `ports.LedgerReportingService` because
// the wider port carries 13 unrelated methods.
type reporter interface {
	GetGrossProfitReport(ctx context.Context, req *reportpb.GrossProfitReportRequest) (*reportpb.GrossProfitReportResponse, error)
	GetCashBookReport(ctx context.Context, req *reportpb.CashBookReportRequest) (*reportpb.CashBookReportResponse, error)
}

// Deps groups the construction-time dependencies. `Reporter` carries
// `any` from the umbrella; the assertion happens inside this package.
type Deps struct {
	Reporter   any
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases aggregates every gross/cashflow use case.
type UseCases struct {
	GetGrossProfitReport *GetGrossProfitReportUseCase
	GetCashBookReport    *GetCashBookReportUseCase
}

// setReporter rewires both use cases to a non-nil reporter after
// construction. The composition root calls [SetReporterFromAny] post-build
// because the adapter is assembled app-side (see container.go ledger
// reporting block).
//
// **Unexported** — public rewire path is [SetReporterFromAny].
func (u *UseCases) setReporter(r reporter) {
	if u == nil {
		return
	}
	if u.GetGrossProfitReport != nil {
		u.GetGrossProfitReport.reporter = r
	}
	if u.GetCashBookReport != nil {
		u.GetCashBookReport.reporter = r
	}
}

// SetReporterFromAny is the canonical wiring entry point. Returns `true` on
// success, `false` when u is nil, v is nil, or v doesn't satisfy the
// unexported `reporter` interface. Composition root logs loudly on
// (`v != nil && !ok`) to surface adapter-signature drift at boot.
func (u *UseCases) SetReporterFromAny(v any) bool {
	if u == nil || v == nil {
		return false
	}
	r, ok := v.(reporter)
	if !ok {
		return false
	}
	u.setReporter(r)
	return true
}

// NewUseCases wires the gross/cashflow sub-aggregate. `deps` may be nil; on
// nil-port construction Execute degrades to an empty response.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	var r reporter
	if deps.Reporter != nil {
		if v, ok := deps.Reporter.(reporter); ok {
			r = v
		}
	}
	return &UseCases{
		GetGrossProfitReport: NewGetGrossProfitReportUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
			deps.ActionGatekeeper,
		),
		GetCashBookReport: NewGetCashBookReportUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
			deps.ActionGatekeeper,
		),
	}
}
