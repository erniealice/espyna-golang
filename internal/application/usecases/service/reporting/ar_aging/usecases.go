// Package ar_aging hosts the service-driven AR aging reporting use cases.
//
// Per docs/plan/20260520-service-domain-migration/decisions.md (Q-SDM-LEDGER-
// DECOMP + Q-SDM-LEDGER-INTERFACE) the 15-method `ledgerReportingInner`
// duck interface at `apps/service-admin/internal/composition/
// ledger_reporting.go:65-81` decomposes into 5 report-group sub-candidates;
// this package is the first such sub-candidate (P1.E.1 — AR aging) hosting:
//
//   - GetReceivablesAgingReport (formerly ledger_reporting.go:70)
//   - GetCollectionSummaryReport (formerly ledger_reporting.go:72)
//
// Per Q-SDM-LEDGER-INTERFACE there is no replacement Go interface; the
// proto contract at `packages/esqyma/proto/v1/service/reporting/ar_aging/
// ar_aging.proto` is the canonical contract. Each use case here Execute()s
// a proto-shaped `*ar_agingpb.Get<X>Request` and returns a proto-shaped
// `*ar_agingpb.Get<X>Response`, internally translating to/from the existing
// entity-domain proto shapes that the postgres `LedgerReportingAdapter`
// speaks.
//
// The Reporter port narrows the existing fat `ports.LedgerReportingService`
// interface (the postgres adapter satisfies the wider shape; we only need 2
// methods here). This keeps the AR aging package independent of every
// other report-group sub-candidate and avoids dragging the entire fat
// interface into this leaf.
//
// Wiring: `internal/composition/core/initializers/service.go` constructs
// the per-group `UseCases` via `NewUseCases(deps)` and assigns the result
// into the `ARAging` field of `service/reporting.ReportingUseCases` (the
// Wave B umbrella). Apps consume `uc.Service.Reporting.ARAging.<Method>.
// Execute(ctx, *ar_agingpb.Get<X>Request)`.
package ar_aging

import (
	"context"

	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	collsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"

	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// reporter is the narrow port the AR aging use cases need from the
// underlying ledger reporting adapter. The postgres `LedgerReportingAdapter`
// satisfies this interface structurally.
//
// **Unexported** (codex review REVISE-MINOR P1, 2026-05-21): apps cannot
// name `ar_aging.reporter` as a type assertion target (Go's `internal/`
// visibility rule + lower-case unexported name), which is exactly the
// behavior we want — the port stays private to this package and the
// composition root threads adapters in via [SetReporterFromAny]. The
// umbrella `Reporting.Deps.ARAgingReporter` field carries `any` for the
// same reason (see service/reporting/usecases.go).
//
// Defined locally (rather than reusing `ports.LedgerReportingService`)
// because the wider port carries 12 unrelated methods. Each P1.E.* sub-
// candidate defines its own narrow port over the same adapter.
type reporter interface {
	GetReceivablesAgingReport(ctx context.Context, req *agingpb.ReceivablesAgingRequest) (*agingpb.ReceivablesAgingResponse, error)
	GetCollectionSummaryReport(ctx context.Context, req *collsumpb.CollectionSummaryRequest) (*collsumpb.CollectionSummaryResponse, error)
}

// Deps groups the construction-time dependencies. `ARAgingReporter` carries
// `any` from the umbrella; the assertion happens inside this package.
// The other fields mirror the parent service-layer `Services` shape
// (TranslationService for error messages, AuthorizationService for the
// standard "reports" + ActionList gate).
type Deps struct {
	// Reporter carries the raw postgres LedgerReportingAdapter value. The
	// concrete type satisfies the unexported `reporter` interface; the
	// assertion happens inside [NewUseCases] / [SetReporterFromAny].
	Reporter             any
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UseCases aggregates every AR aging use case. Apps reach it via
// `uc.Service.Reporting.ARAging.<UseCaseField>.Execute(ctx, req)`.
type UseCases struct {
	GetReceivablesAgingReport  *GetReceivablesAgingReportUseCase
	GetCollectionSummaryReport *GetCollectionSummaryReportUseCase
}

// setReporter rewires both AR aging use cases to a non-nil reporter
// after construction. Needed because the espyna container builds the
// service use case aggregate BEFORE the app's composition root assembles
// the concrete ledger reporting service (the table config lives in the
// app; see apps/service-admin/internal/composition/ledger_reporting.go).
//
// Calling setReporter with nil leaves the use cases in their constructed
// state (nil reporter → empty Response on Execute). Subsequent calls
// overwrite previous wiring. Concurrency: the rewire happens during app
// bootstrap before any Execute is dispatched; no mutex is required.
//
// This is the same post-construction-setter pattern that
// service/tax/init.go uses for SetEntityCompute; see the Q-PERMQ-
// COMPOSITION-PATTERN lock and decisions.md `SetEntityCompute` caveat
// note for the rationale + the alternatives an author should weigh
// before adopting this shape on a new candidate.
//
// **Unexported** (codex review REVISE-MINOR P1, 2026-05-21): the
// rewire path apps care about is [SetReporterFromAny]; the typed
// variant is package-private.
func (u *UseCases) setReporter(r reporter) {
	if u == nil {
		return
	}
	if u.GetReceivablesAgingReport != nil {
		u.GetReceivablesAgingReport.reporter = r
	}
	if u.GetCollectionSummaryReport != nil {
		u.GetCollectionSummaryReport.reporter = r
	}
}

// SetReporterFromAny is the canonical wiring entry point. Apps
// (e.g. apps/service-admin) cannot name the unexported `ar_aging.reporter`
// interface as a type assertion target on their side; this helper accepts
// `any`, runs the assertion inside the `ar_aging` package, and returns
// whether the assertion succeeded.
//
// **Return contract** (codex review REVISE-MINOR P2, 2026-05-21): returns
// `true` when the rewire took effect, `false` when either u is nil, v is
// nil, or v does NOT satisfy the unexported `reporter` interface. The
// composition root is expected to log on `false` when v was non-nil — a
// non-nil adapter that fails the assertion is a silent-wiring trap
// (Wave B P1.C.1 admin role bug — codex review P0, 2026-05-20). Loud
// failure here keeps that trap from re-opening.
//
// The composition root passes the raw postgres LedgerReportingAdapter
// (returned by the registry factory) — that adapter exposes
// GetReceivablesAgingReport + GetCollectionSummaryReport with the
// expected proto-shape, so the assertion succeeds. On mock builds
// where the factory is unregistered, the caller passes nil and this
// is a no-op (returns false; caller skips logging on nil input).
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

// NewUseCases wires the AR aging sub-aggregate. `deps` may be nil (then
// every use case is constructed with a nil port and degrades to an empty
// response on Execute — matches the Audit/Security nil-safety contract).
//
// `deps.Reporter` carries `any` (so the umbrella `Reporting.Deps.ARAgingReporter`
// can also be `any` and apps can pass the raw postgres adapter without
// naming the unexported `reporter` interface). If `deps.Reporter` is non-nil
// but does NOT satisfy `reporter`, construction returns the use cases with
// a nil port (Execute degrades to empty Response). The composition root
// is expected to call [SetReporterFromAny] after construction to inspect
// the bool return and log on silent-assertion failure.
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
		GetReceivablesAgingReport: NewGetReceivablesAgingReportUseCase(
			r,
			deps.AuthorizationService,
			deps.TranslationService,
		),
		GetCollectionSummaryReport: NewGetCollectionSummaryReportUseCase(
			r,
			deps.AuthorizationService,
			deps.TranslationService,
		),
	}
}
