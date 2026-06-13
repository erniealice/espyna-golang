package auth

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// SessionSwitchAdapter is the narrow extension interface the SwitchPrincipal
// use case requires from the session repository.
//
// Why a separate interface (not the auto-generated SessionDomainServiceServer
// shape): SwitchPrincipal is NOT an entity-CRUD operation — it owns a
// security-critical multi-table transaction (session row + binding lock +
// audit insert) that doesn't fit the CRUD-shaped service surface. The
// Phase-2 adapter exposes the method directly on the concrete
// *PostgresSessionRepository struct rather than threading it through the
// generated SessionDomainServiceServer interface. The use case here consumes
// it via this narrow Go-only extension interface; other backends (mock,
// firestore) can leave it unimplemented — Execute nil-guards the field and
// fails closed with service_unavailable when the asserted adapter is nil.
//
// Phase-2 implementation:
//
//	packages/espyna-golang/contrib/postgres/internal/adapter/entity/session_switch_principal.go
//
// Phase-4 wiring (pending): initServiceAuth at
// internal/composition/core/initializers/service/auth.go type-asserts the
// session repo to this interface; on success threads it into
// Repositories.SessionSwitch, on failure leaves the field nil.
type SessionSwitchAdapter interface {
	SwitchPrincipal(ctx context.Context, req *authpb.SwitchPrincipalRequest) (*authpb.SwitchPrincipalResponse, error)
}

// SwitchPrincipalRepositories groups the adapters this use case consumes.
// SessionSwitch may be nil — Execute nil-guards at body entry and fails
// closed with auth.errors.service_unavailable (Q2 lock from the predecessor
// auth-collapse plan).
type SwitchPrincipalRepositories struct {
	SessionSwitch SessionSwitchAdapter
}

// SwitchPrincipalServices groups infrastructure services. No Authorizer —
// per the package invariant in usecases.go, principal-switch is an
// identity-establishment-adjacent operation that runs AFTER initial
// authentication but BEFORE per-action authorization re-grants on the new
// session.
type SwitchPrincipalServices struct {
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SwitchPrincipalUseCase orchestrates the session-rotation primitive.
//
// Its sole responsibilities are:
//  1. Nil-guard the adapter (fail closed with service_unavailable).
//  2. Validate the proto request shape (required fields).
//  3. Delegate the security-critical transactional work to the adapter.
//  4. Translate adapter errors via the Translator port (adapter returns raw
//     errors per hexagonal-rules §1 principle 3).
//  5. Compute the response RedirectUrl from the target Principal — the
//     adapter intentionally leaves this empty because HTTP routing
//     vocabulary lives in the composition layer; the proto Principal
//     message has no HomeRoute method (see switch_principal_helpers.go's
//     homeRouteForProtoPrincipal for the mirror logic).
//
// Per Q2-A atomicity lock: the adapter writes the audit row inside the same
// transaction as the session UPDATE/INSERT (see
// session_switch_principal.go::writeSwitchAuditRow), so RequireAudit=true
// rolls the rotation back when the audit insert fails — red-team A-4
// stealth-rotation defense is preserved across the typed-stack migration.
type SwitchPrincipalUseCase struct {
	repositories SwitchPrincipalRepositories
	services     SwitchPrincipalServices
}

// NewSwitchPrincipalUseCase wires the use case from grouped dependencies.
func NewSwitchPrincipalUseCase(
	repositories SwitchPrincipalRepositories,
	services SwitchPrincipalServices,
) *SwitchPrincipalUseCase {
	return &SwitchPrincipalUseCase{repositories: repositories, services: services}
}

// Execute runs the principal switch.
//
// Validation order (matches the predecessor auth-collapse split between
// service_unavailable, request_required, and field-level keys):
//
//  1. Adapter nil-guard      → auth.errors.service_unavailable
//  2. Request nil-guard       → auth.validation.request_required
//  3. user_id required        → auth.validation.user_id_required
//  4. target_principal req    → auth.validation.target_principal_required
//  5. Adapter call            → adapter errors translated via
//     translateSwitchPrincipalError
//  6. Nil response defense    → auth.switch_principal.failed
//  7. RedirectUrl computation → adapter leaves empty; we fill from target
//
// Read-only callers (no token mutation) get the in-place mutation path
// inside the adapter; rotating callers get the new token in
// SwitchPrincipalResponse.NewToken. Either way RedirectUrl is set here so
// every caller can render the post-switch destination uniformly.
func (uc *SwitchPrincipalUseCase) Execute(
	ctx context.Context,
	req *authpb.SwitchPrincipalRequest,
) (*authpb.SwitchPrincipalResponse, error) {
	// 1. Fail-closed at body entry (Q2 lock from predecessor auth-collapse).
	if uc.repositories.SessionSwitch == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable",
			"Auth service is not available [DEFAULT]"))
	}
	// 2. Split nil-request from empty-fields keys (parity with
	//    authenticate_session: nil request → request_required; well-formed
	//    request with empty fields → field-level keys below).
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.request_required",
			"Principal switch request is required [DEFAULT]"))
	}
	// 3. user_id is required for every switch — the adapter also rejects
	//    this but checking here keeps the error surface translated +
	//    avoids one round trip into the database layer.
	if req.GetUserId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.user_id_required",
			"User ID is required for principal switch [DEFAULT]"))
	}
	// 4. target_principal is required (the destination of the switch).
	if req.GetTargetPrincipal() == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.target_principal_required",
			"Target principal is required for principal switch [DEFAULT]"))
	}

	// 5. Delegate the transactional work. Adapter returns raw errors per
	//    hexagonal-rules §1 principle 3; translate before returning.
	resp, err := uc.repositories.SessionSwitch.SwitchPrincipal(ctx, req)
	if err != nil {
		return nil, translateSwitchPrincipalError(ctx, uc.services.Translator, err)
	}
	// 6. Defensive nil-response check (an adapter contract violation —
	//    a successful return should always come with a non-nil response).
	if resp == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.switch_principal.failed",
			"Principal switch did not return a result [DEFAULT]"))
	}

	// 7. Phase-2 note: the adapter leaves RedirectUrl empty because
	//    computing it requires HomeRoute(), which is composition-layer
	//    Go vocabulary on adapthttp.Principal — the proto Principal
	//    message has no method. Fill it in here from the target so every
	//    caller (URL middleware, explicit form handler) receives the
	//    post-switch destination uniformly.
	if resp.GetRedirectUrl() == "" {
		resp.RedirectUrl = homeRouteForProtoPrincipal(req.GetTargetPrincipal())
	}

	return resp, nil
}
