// Package domain_specific hosts the service-driven per-domain pivot
// reporting use cases (Wave B P1.E.5 — the LAST decomposition slice).
//
// Per docs/plan/20260520-service-domain-migration/decisions.md (Q-SDM-LEDGER-
// DECOMP + Q-SDM-LEDGER-INTERFACE + Q-SDM-MAP-SHAPES + Q-SDM-LEDGER-RETIRE-
// PYEZA) the 15-method `ledgerReportingInner` duck interface (formerly in
// `apps/service-admin/internal/composition/ledger_reporting.go`, FILE DELETED
// 20260521 per composition-reshape Q-SDM-LEDGER-RETIRE-PYEZA; logic inlined
// into `container.go`) decomposes into 5 report-group sub-candidates. This package is the
// fifth — and final — such sub-candidate (P1.E.5 — domain-specific pivots)
// hosting:
//
//   - GetRevenueReport      (formerly ledger_reporting.go:67) — proto-shaped
//   - GetExpenditureReport  (formerly ledger_reporting.go:68) — proto-shaped
//   - GetDisbursementReport (formerly ledger_reporting.go:69) — proto-shaped
//   - ListRevenue           (formerly ledger_reporting.go:77) — Go-only
//   - ListExpenses          (formerly ledger_reporting.go:78) — Go-only
//
// **Q-SDM-MAP-SHAPES enforcement:** `ListRevenue`/`ListExpenses` return
// `[]map[string]any` because they walk operational entities directly and
// have no stable column schema. Q-SDM-MAP-SHAPES rejects
// `google.protobuf.Struct` and locks "stay Go-only until a real column
// schema is chosen." Each use case here exposes a proto-shaped Execute()
// for the typed pivots AND a separate Go-only method (`Execute()` on the
// `ListRevenueUseCase` etc.) that returns the raw rows. Downstream views
// that need CSV/PDF feeders call the Go-only method; future typed
// consumers can replace it without breaking the proto-shaped pivots.
//
// **Q-SDM-LEDGER-RETIRE-PYEZA cleanup (LANDED 2026-05-21):** the LAST
// sub-candidate's commit retired:
//   - The `*ledgerReporting` wrapper + `ledgerReportingInner` duck
//     interface in `apps/service-admin/internal/composition/
//     ledger_reporting.go`. The FILE itself is DELETED (20260521-
//     composition-reshape); its logic is inlined into
//     `apps/service-admin/internal/composition/container.go`.
//   - The populated `pyeza.AppContext.LedgerReportingSvc any` field at
//     `packages/pyeza-golang/app_context.go`. The field is removed from
//     the AppContext struct; service-admin no longer populates it.
//
// Per Q-SDM-LEDGER-INTERFACE there is no replacement Go interface; the
// proto contract at `packages/esqyma/proto/v1/service/reporting/
// domain_specific/domain_specific.proto` is the canonical contract for
// the typed pivots.
//
// Wiring: `internal/composition/core/initializers/service/reporting.go`
// constructs the per-group `UseCases` via `NewUseCases(deps)` and assigns
// the result into the `DomainSpecific` field of
// `service/reporting.ReportingUseCases`.
// Apps consume `uc.Service.Reporting.DomainSpecific.<Method>.Execute(...)`.
//
// Codex pattern compliance (codex review of AR aging REVISE-MINOR P1+P2,
// 2026-05-21):
//   - `reporter` interface UNEXPORTED.
//   - `setReporter` UNEXPORTED.
//   - `SetReporterFromAny(any) bool` returns true on success.
//   - `Deps.Reporter any` (not typed) so apps can pass the raw adapter.
package domain_specific

import (
	"context"
	"time"

	expreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/expenditure_report"
	revreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/revenue_report"
	disbreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/disbursement_report"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
)

// reporter is the narrow port for the 3 typed pivot reports + the 2 Go-
// only list helpers. The postgres `LedgerReportingAdapter` satisfies this
// interface structurally.
//
// **Unexported** (matching the AR aging pilot REVISE-MINOR P1, 2026-05-21).
type reporter interface {
	GetRevenueReport(ctx context.Context, req *revreportpb.RevenueReportRequest) (*revreportpb.RevenueReportResponse, error)
	GetExpenditureReport(ctx context.Context, req *expreportpb.ExpenditureReportRequest) (*expreportpb.ExpenditureReportResponse, error)
	GetDisbursementReport(ctx context.Context, req *disbreportpb.DisbursementReportRequest) (*disbreportpb.DisbursementReportResponse, error)
	ListRevenue(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
	ListExpenses(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
}

// Deps groups the construction-time dependencies. `Reporter` carries
// `any` from the umbrella; the assertion happens inside this package.
type Deps struct {
	Reporter   any
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases aggregates every domain-specific use case.
type UseCases struct {
	GetRevenueReport      *GetRevenueReportUseCase
	GetExpenditureReport  *GetExpenditureReportUseCase
	GetDisbursementReport *GetDisbursementReportUseCase
	ListRevenue           *ListRevenueUseCase
	ListExpenses          *ListExpensesUseCase
}

// setReporter rewires every use case to a non-nil reporter after
// construction. The composition root calls [SetReporterFromAny] post-build
// because the adapter is assembled app-side.
//
// **Unexported** — public rewire path is [SetReporterFromAny].
func (u *UseCases) setReporter(r reporter) {
	if u == nil {
		return
	}
	if u.GetRevenueReport != nil {
		u.GetRevenueReport.reporter = r
	}
	if u.GetExpenditureReport != nil {
		u.GetExpenditureReport.reporter = r
	}
	if u.GetDisbursementReport != nil {
		u.GetDisbursementReport.reporter = r
	}
	if u.ListRevenue != nil {
		u.ListRevenue.reporter = r
	}
	if u.ListExpenses != nil {
		u.ListExpenses.reporter = r
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

// NewUseCases wires the domain-specific sub-aggregate. `deps` may be nil;
// on nil-port construction Execute degrades to an empty response.
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
		GetRevenueReport: NewGetRevenueReportUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
		),
		GetExpenditureReport: NewGetExpenditureReportUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
		),
		GetDisbursementReport: NewGetDisbursementReportUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
		),
		ListRevenue: NewListRevenueUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
		),
		ListExpenses: NewListExpensesUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
		),
	}
}
