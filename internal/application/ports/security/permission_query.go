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
// Delegate target scoping (added 2026-05-24 per A2-followup / codex
// A2-P0-1 fix): actingAsClientID / actingAsSupplierID identify the
// per-target grant row a delegate is currently acting through. For
// CLIENT_DELEGATE the adapter scopes resolution to the delegate_client
// row (delegate_id = bindingID, client_id = actingAsClientID); for
// SUPPLIER_DELEGATE it scopes to delegate_supplier (delegate_id =
// bindingID, supplier_id = actingAsSupplierID). These are IGNORED for
// non-delegate bindingKinds. When the relevant acting-as value is
// empty for a delegate binding kind, the adapter FAILS CLOSED (empty
// result) rather than unioning across all per-target rows — closing
// the multi-target leakage hole (codex A2-P0-1).
//
// Fail-closed posture (codex A2-P1-1 fix): the legacy union-fallback
// path is reserved for the EXACT zero pair `(bindingKind=0,
// bindingID="")`. Any other combination — partial hints
// (CLIENT, ""), out-of-range bindingKinds, kind set with empty id,
// id set with UNSPECIFIED kind, or a delegate kind with no acting-as
// id — returns an empty permission set. Production callers MUST
// always supply a complete binding hint; only legacy bootstrap and
// test paths exercise the union path.
type PermissionQuery interface {
	// GetUserPermissionCodes returns all effective ALLOW permission codes
	// for the given user within the given workspace, restricted to the
	// grant chain identified by (bindingKind, bindingID) when both are
	// supplied. For delegate kinds, actingAsClientID (CLIENT_DELEGATE)
	// or actingAsSupplierID (SUPPLIER_DELEGATE) additionally scope the
	// lookup to the per-target row. Returns an empty slice (not nil)
	// when the user has no permissions. See package-level doc for the
	// binding-hint + fail-closed semantics.
	GetUserPermissionCodes(
		ctx context.Context,
		userID, workspaceID string,
		bindingKind int32,
		bindingID string,
		actingAsClientID, actingAsSupplierID string,
	) ([]string, error)
}
