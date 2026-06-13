// Package ap_aging hosts the service-driven AP aging reporting use cases
// (Wave B P1.E.2).
//
// Per docs/plan/20260520-service-domain-migration/decisions.md (Q-SDM-LEDGER-
// DECOMP + Q-SDM-LEDGER-INTERFACE) the 15-method `ledgerReportingInner`
// duck interface (formerly in `apps/service-admin/internal/composition/
// ledger_reporting.go`, FILE DELETED 20260521 per composition-reshape
// Q-SDM-LEDGER-RETIRE-PYEZA; logic inlined into `container.go`) decomposes
// into 5 report-group sub-candidates.
// This package is the second such sub-candidate (P1.E.2 — AP aging) hosting:
//
//   - GetPayablesAgingReport       (formerly ledger_reporting.go:71)
//   - GetSimplePayablesAgingReport (formerly ledger_reporting.go:80)
//
// Per Q-SDM-LEDGER-INTERFACE there is no replacement Go interface; the
// proto contract at `packages/esqyma/proto/v1/service/reporting/ap_aging/
// ap_aging.proto` is the canonical contract. Each use case here Execute()s
// a proto-shaped `*ap_agingpb.Get<X>Request` and returns a proto-shaped
// `*ap_agingpb.Get<X>Response`, internally translating to/from the existing
// entity-domain proto shapes that the postgres `LedgerReportingAdapter`
// speaks.
//
// Wiring: `internal/composition/core/initializers/service.go` constructs
// the per-group `UseCases` via `NewUseCases(deps)` and assigns the result
// into the `APAging` field of `service/reporting.ReportingUseCases` (the
// Wave B umbrella). Apps consume `uc.Service.Reporting.APAging.<Method>.
// Execute(ctx, *ap_agingpb.Get<X>Request)`.
//
// Codex pattern compliance (codex review of AR aging REVISE-MINOR P1+P2,
// 2026-05-21):
//   - `reporter` interface UNEXPORTED (apps can't name it; assertion happens
//     inside this package via SetReporterFromAny).
//   - `setReporter` UNEXPORTED — public rewire path is SetReporterFromAny.
//   - `SetReporterFromAny(any) bool` returns true on success; composition
//     root logs loudly on (`v != nil && !ok`) to dodge silent-wiring trap.
//   - `Deps.Reporter any` (not typed) — typing as `*reporter` would force
//     callers to name an unexported type.
package ap_aging

import (
	"context"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	payagingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/payables_aging"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
)

// reporter is the narrow port the AP aging use cases need from the
// underlying ledger reporting adapter. The postgres `LedgerReportingAdapter`
// satisfies this interface structurally.
//
// **Unexported** (matching the AR aging pilot REVISE-MINOR P1, 2026-05-21):
// apps cannot name `ap_aging.reporter` as a type assertion target — that's
// the whole point. The umbrella `Reporting.Deps.APAgingReporter` field
// carries `any` for the same reason; the assertion happens inside this
// package via [SetReporterFromAny].
//
// Defined locally rather than reusing `ports.LedgerReportingService`
// because the wider port carries 13 unrelated methods. Each P1.E.* sub-
// candidate defines its own narrow port over the same adapter.
type reporter interface {
	GetPayablesAgingReport(ctx context.Context, req *payagingpb.PayablesAgingRequest) (*payagingpb.PayablesAgingResponse, error)
	GetSimplePayablesAgingReport(ctx context.Context, req *reportpb.PayablesAgingReportRequest) (*reportpb.PayablesAgingReportResponse, error)
}

// Deps groups the construction-time dependencies. `Reporter` carries
// `any` from the umbrella; the assertion happens inside this package.
// The other fields mirror the parent service-layer `Services` shape
// (Translator for error messages, Authorizer for the
// standard "reports" + ActionList gate).
type Deps struct {
	// Reporter carries the raw postgres LedgerReportingAdapter value. The
	// concrete type satisfies the unexported `reporter` interface; the
	// assertion happens inside [NewUseCases] / [SetReporterFromAny].
	Reporter   any
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases aggregates every AP aging use case. Apps reach it via
// `uc.Service.Reporting.APAging.<UseCaseField>.Execute(ctx, req)`.
type UseCases struct {
	GetPayablesAgingReport       *GetPayablesAgingReportUseCase
	GetSimplePayablesAgingReport *GetSimplePayablesAgingReportUseCase
}

// setReporter rewires both AP aging use cases to a non-nil reporter after
// construction. The espyna container builds the service use case aggregate
// BEFORE the app's composition root assembles the concrete ledger reporting
// adapter (the table config struct is inlined in
// apps/service-admin/internal/composition/container.go — the former
// ledger_reporting.go is DELETED per 20260521-composition-reshape). The
// composition root therefore calls [SetReporterFromAny] post-construction.
//
// Calling setReporter with nil leaves the use cases in their constructed
// state (nil reporter → empty Response on Execute). Subsequent calls
// overwrite previous wiring. Concurrency: the rewire happens during app
// bootstrap before any Execute is dispatched; no mutex is required.
//
// **Unexported** — the rewire path apps care about is [SetReporterFromAny].
func (u *UseCases) setReporter(r reporter) {
	if u == nil {
		return
	}
	if u.GetPayablesAgingReport != nil {
		u.GetPayablesAgingReport.reporter = r
	}
	if u.GetSimplePayablesAgingReport != nil {
		u.GetSimplePayablesAgingReport.reporter = r
	}
}

// SetReporterFromAny is the canonical wiring entry point. Apps cannot name
// the unexported `ap_aging.reporter` interface as a type assertion target
// on their side; this helper accepts `any`, runs the assertion inside the
// `ap_aging` package, and returns whether the assertion succeeded.
//
// **Return contract:** returns `true` when the rewire took effect, `false`
// when either u is nil, v is nil, or v does NOT satisfy the unexported
// `reporter` interface. The composition root logs loudly on (`v != nil && !ok`)
// to surface a drift in the postgres adapter's method signatures at boot,
// not via empty reports in production. This matches the AR aging
// REVISE-MINOR P2 contract.
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

// NewUseCases wires the AP aging sub-aggregate. `deps` may be nil (then
// every use case is constructed with a nil port and degrades to an empty
// response on Execute — matches the Audit/Security nil-safety contract).
//
// `deps.Reporter` carries `any`. If non-nil but failing the `reporter`
// assertion, construction returns the use cases with a nil port; the
// composition root is expected to call [SetReporterFromAny] after
// construction to inspect the bool return and log on assertion failure.
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
		GetPayablesAgingReport: NewGetPayablesAgingReportUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
			deps.ActionGatekeeper,
		),
		GetSimplePayablesAgingReport: NewGetSimplePayablesAgingReportUseCase(
			r,
			deps.Authorizer,
			deps.Translator,
			deps.ActionGatekeeper,
		),
	}
}
