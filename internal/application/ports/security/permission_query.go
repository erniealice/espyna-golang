package security

import "context"

// PermissionQuery is the narrow port for RBAC permission-code lookups.
//
// Separate from Authorizer: Authorizer answers
// "does user X have permission Y?" (boolean), while PermissionQuery returns
// the full set of ALLOW codes net of DENYs so callers can cache or filter
// in bulk (e.g. for sidebar building, batch authorization, permission
// hydration on session cookie).
//
// DENY-wins semantics: if the user holds both an ALLOW and a DENY grant
// for a given permission, it is treated as denied and omitted from the
// returned set.
//
// Binding-scoped lookup (added 2026-05-24 per A2 / WKR-P0-2 in
// docs/plan/20260522-codex-redteam-sweep): bindingKind + bindingID
// constrain the lookup to the SINGLE selected binding row from the
// session, so a user holding multiple bindings in one workspace
// (e.g. CLIENT + OPERATOR_STAFF) does NOT receive the UNION of
// permissions across all bindings. Field-name vocabulary is `binding_*`
// per Q-BR-1 lock 2026-05-24; the proto-level enum rename to BindingKind
// is owned by docs/plan/20260522-principal-to-binding-rename/'s P1
// sweep — until then the values flow through the existing
// domain.entity.v1.PrincipalType enum.
//
// Backwards compatibility: bindingKind == 0 (UNSPECIFIED) and
// bindingID == "" together mean "no binding hint — fall back to the
// legacy union-across-all-bindings behaviour." Production callers MUST
// supply both values; the silent-elevation fix only applies when both
// are non-zero.
type PermissionQuery interface {
	// GetUserPermissionCodes returns all effective ALLOW permission codes
	// for the given user within the given workspace, restricted to the
	// grant chain identified by (bindingKind, bindingID) when both are
	// supplied. Returns an empty slice (not nil) when the user has no
	// permissions. See package-level doc for the binding-hint semantics.
	GetUserPermissionCodes(
		ctx context.Context,
		userID, workspaceID string,
		bindingKind int32,
		bindingID string,
	) ([]string, error)
}
