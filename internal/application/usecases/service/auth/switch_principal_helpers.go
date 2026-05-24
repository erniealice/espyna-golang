package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// homeRouteForProtoPrincipal mirrors adapthttp.Principal.HomeRoute() from
// apps/service-admin/internal/infrastructure/input/http/principal_loader.go
// (Principal.HomeRoute at line 132 and PrincipalType.HomeRoute at line 84) for
// proto-typed Principal messages. The composition layer's adapthttp.Principal.
// HomeRoute() stays for non-rotation callers (login picker, login redirect
// resolution); this helper is for the SwitchPrincipal use case's response
// shaping — the Phase-2 adapter intentionally leaves RedirectUrl empty because
// the proto Principal message has no method, and computing HTTP-routing
// vocabulary from inside an adapter would invert the dependency direction
// (hexagonal-rules.md §1 principle 3 / §6 anti-patterns: adapters never reach
// upward).
//
// All concrete operator/client/supplier/delegate principal types currently
// land on "/me/inbox" per the P12 cutover (2026-05-22, Q-WS-7 → B); delegates
// auto-enter when N=1, otherwise route to the picker. PrincipalTypeUnspecified
// degrades to "/auth/no-access" (parity with PrincipalType.HomeRoute's default).
//
// Keep this in lockstep with the composition-layer HomeRoute() — when one
// moves, the other moves with it. A regression test would diff the two for
// every PrincipalType × ActingAsTargets length, but the routing surface is
// stable enough that mirroring-by-comment is acceptable today.
func homeRouteForProtoPrincipal(p *authpb.Principal) string {
	if p == nil {
		return "/auth/no-access"
	}
	base := protoPrincipalTypeHomeRoute(p.GetType())
	switch p.GetType() {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE:
		targets := p.GetActingAsTargets()
		if len(targets) == 1 {
			return fmt.Sprintf("%s?acting_as_client_id=%s", base, targets[0].GetId())
		}
		return base + "select"
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE:
		targets := p.GetActingAsTargets()
		if len(targets) == 1 {
			return fmt.Sprintf("%s?acting_as_supplier_id=%s", base, targets[0].GetId())
		}
		return base + "select"
	}
	return base
}

// protoPrincipalTypeHomeRoute mirrors adapthttp.PrincipalType.HomeRoute() at
// principal_loader.go:84. All six concrete principal types land on /me/inbox;
// unspecified routes to /auth/no-access.
func protoPrincipalTypeHomeRoute(t principaltypepb.PrincipalType) string {
	switch t {
	case principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE:
		return "/me/inbox"
	}
	return "/auth/no-access"
}

// translateSwitchPrincipalError maps the small set of well-known adapter
// error signatures emitted by
//
//	packages/espyna-golang/contrib/postgres/internal/adapter/entity/session_switch_principal.go
//
// to translator keys. The adapter returns raw errors wrapped with
// `fmt.Errorf("session adapter: SwitchPrincipal: <where>: %w", err)` (or a
// formatted error string when the failure is a semantic precondition like a
// revoked binding); we substring-match on the stable middle segment so that
// future error-text drift in the outer wrapping doesn't break translation.
//
// Unrecognised errors fall through to a generic auth.switch_principal.failed
// key — the raw error text is appended to the default fallback so dev
// builds still surface the underlying message when no translation file is
// loaded.
//
// Translator keys (flag for lyngua catalog follow-up):
//   - auth.switch_principal.binding_revoked
//   - auth.switch_principal.session_invalid
//   - auth.switch_principal.acting_as_mismatch
//   - auth.switch_principal.unsupported_principal_type
//   - auth.switch_principal.failed
func translateSwitchPrincipalError(ctx context.Context, t ports.Translator, raw error) error {
	if raw == nil {
		return nil
	}
	msg := raw.Error()
	switch {
	case strings.Contains(msg, "binding revoked or not in workspace"):
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, t,
			"auth.switch_principal.binding_revoked",
			"Selected binding has been revoked or is not in this workspace [DEFAULT]"))
	case strings.Contains(msg, "read current session"),
		strings.Contains(msg, "invalidate old session"),
		strings.Contains(msg, "in-place update"),
		strings.Contains(msg, "insert new session"):
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, t,
			"auth.switch_principal.session_invalid",
			"Current session is invalid or could not be rotated [DEFAULT]"))
	case strings.Contains(msg, "is not in the resolved binding's targets"):
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, t,
			"auth.switch_principal.acting_as_mismatch",
			"Requested acting-as target is not in the resolved binding [DEFAULT]"))
	case strings.Contains(msg, "unsupported principal type"):
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, t,
			"auth.switch_principal.unsupported_principal_type",
			"Unsupported principal type for binding lock [DEFAULT]"))
	default:
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, t,
			"auth.switch_principal.failed",
			fmt.Sprintf("Principal switch failed: %s [DEFAULT]", msg)))
	}
}
