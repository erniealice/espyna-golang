// Package statements hosts the service-driven counterparty-statement +
// balance reporting use cases (Wave B P1.E.4).
//
// Per docs/plan/20260520-service-domain-migration/decisions.md (Q-SDM-LEDGER-
// DECOMP + Q-SDM-LEDGER-INTERFACE + Q-SDM-MAP-SHAPES) the 15-method
// `ledgerReportingInner` duck interface at
// `apps/service-admin/internal/composition/ledger_reporting.go:65-81`
// decomposes into 5 report-group sub-candidates. This package is the
// fourth such sub-candidate (P1.E.4 — counterparty statements + balances)
// hosting:
//
//   - GetClientStatement    (formerly ledger_reporting.go:73)
//   - GetSupplierStatement  (formerly ledger_reporting.go:74)
//   - ListClientBalances    (formerly ledger_reporting.go:76 — GetClientBalances)
//   - ListSupplierBalances  (formerly ledger_reporting.go:75 — GetSupplierBalances)
//
// **Q-SDM-MAP-SHAPES enforcement:** the legacy
// `GetClientBalances`/`GetSupplierBalances` returned `map[string]int64`
// (counterparty_id → centavo balance). Q-SDM-MAP-SHAPES rejects
// `google.protobuf.Struct` as a primary contract and locks "typed proto
// where possible". The new responses use `repeated BalanceRow` (proto-
// shaped rows with `counterparty_id` + `amount_centavos`) — Apps can
// consume the rows directly or convert to a `map[string]int64` at the
// call boundary for backward compatibility.
//
// The future `income_statement`/`balance_sheet`/`equity_changes` reports
// named in Q-SDM-LEDGER-DECOMP don't exist yet; they will join this
// package when authored.
//
// Per Q-SDM-LEDGER-INTERFACE there is no replacement Go interface; the
// proto contract at `packages/esqyma/proto/v1/service/reporting/
// statements/statements.proto` is the canonical contract.
//
// Wiring: `internal/composition/core/initializers/service.go` constructs
// the per-group `UseCases` via `NewUseCases(deps)` and assigns the result
// into the `Statements` field of `service/reporting.ReportingUseCases`.
// Apps consume `uc.Service.Reporting.Statements.<Method>.Execute(ctx,
// *statementspb.<X>Request)`.
//
// Codex pattern compliance (codex review of AR aging REVISE-MINOR P1+P2,
// 2026-05-21):
//   - `reporter` interface UNEXPORTED.
//   - `setReporter` UNEXPORTED.
//   - `SetReporterFromAny(any) bool` returns true on success.
//   - `Deps.Reporter any` (not typed) so apps can pass the raw adapter.
package statements

import (
	"context"

	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	suppstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/supplier_statement"

	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// reporter is the narrow port for counterparty statement + balance reads.
// The postgres `LedgerReportingAdapter` satisfies this interface
// structurally.
//
// **Unexported** (matching the AR aging pilot REVISE-MINOR P1, 2026-05-21).
//
// Note: the underlying `GetClientBalances`/`GetSupplierBalances` adapter
// methods return `map[string]int64`; the use cases convert these into the
// typed `repeated BalanceRow` proto response per Q-SDM-MAP-SHAPES.
type reporter interface {
	GetClientStatement(ctx context.Context, req *clientstmtpb.ClientStatementRequest) (*clientstmtpb.ClientStatementResponse, error)
	GetSupplierStatement(ctx context.Context, req *suppstmtpb.SupplierStatementRequest) (*suppstmtpb.SupplierStatementResponse, error)
	GetClientBalances(ctx context.Context) (map[string]int64, error)
	GetSupplierBalances(ctx context.Context) (map[string]int64, error)
}

// Deps groups the construction-time dependencies. `Reporter` carries
// `any` from the umbrella; the assertion happens inside this package.
type Deps struct {
	Reporter             any
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UseCases aggregates every statement + balance use case.
type UseCases struct {
	GetClientStatement    *GetClientStatementUseCase
	GetSupplierStatement  *GetSupplierStatementUseCase
	ListClientBalances    *ListClientBalancesUseCase
	ListSupplierBalances  *ListSupplierBalancesUseCase
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
	if u.GetClientStatement != nil {
		u.GetClientStatement.reporter = r
	}
	if u.GetSupplierStatement != nil {
		u.GetSupplierStatement.reporter = r
	}
	if u.ListClientBalances != nil {
		u.ListClientBalances.reporter = r
	}
	if u.ListSupplierBalances != nil {
		u.ListSupplierBalances.reporter = r
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

// NewUseCases wires the statements sub-aggregate. `deps` may be nil; on
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
		GetClientStatement: NewGetClientStatementUseCase(
			r,
			deps.AuthorizationService,
			deps.TranslationService,
		),
		GetSupplierStatement: NewGetSupplierStatementUseCase(
			r,
			deps.AuthorizationService,
			deps.TranslationService,
		),
		ListClientBalances: NewListClientBalancesUseCase(
			r,
			deps.AuthorizationService,
			deps.TranslationService,
		),
		ListSupplierBalances: NewListSupplierBalancesUseCase(
			r,
			deps.AuthorizationService,
			deps.TranslationService,
		),
	}
}
