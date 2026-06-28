// Package identity is the canonical home for request-scoped user/workspace
// identity in the espyna framework. It replaces the scattered
// consumer.Get*/shared/context/appcontext helpers with a single, typed,
// fail-CLOSED struct stored atomically on the context.
//
// The RequestIdentity struct carries every field that authentication and
// principal-switching middleware stamp onto a request: user ID, workspace ID,
// workspace-user ID, email, session token, and acting-as scopes. Callers
// access fields directly (identity.Must(ctx).WorkspaceID) instead of through
// individual Get* helpers — this is deliberate: a missing field is a zero
// string, and a missing struct panics (Must) or returns an error (Require),
// making the fail-OPEN "return empty string on missing" pattern impossible.
//
// Layer: Shared Adapter Toolkit (L4). Imported by contrib/ adapters,
// consumer/ (re-export), and sibling view packages. Zero dependencies beyond
// the Go standard library.
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...)
//   - DB drivers or adapter packages
//   - anything under internal/
//   - ports, registry, composition, or consumer
//
// Depends only on the Go standard library.
package identity

import (
	"context"
	"errors"
)

// DefaultSessionCookieName is the default cookie name for session tokens.
// Canonical constant — consumer/ and appcontext/ re-exported copies of this.
const DefaultSessionCookieName = "ichizen_session"

// ErrIdentityNotInContext is returned by Require when no RequestIdentity has
// been stored on the context. This replaces the old ErrUserNotFoundInContext
// with a broader, struct-level error.
var ErrIdentityNotInContext = errors.New("identity: no RequestIdentity in context")

// contextKey is unexported — forces usage through the typed API.
type contextKey struct{}

// RequestIdentity carries the full authenticated identity for a single HTTP
// request. Middleware constructs it once and stores it atomically via
// WithRequestIdentity; downstream code reads it via Must or Require.
//
// All fields are exported for direct field access (the design contract —
// callers use identity.Must(ctx).WorkspaceID, not a getter method).
//
// Zero-value fields are meaningful: an empty WorkspaceID means the user has
// not yet selected a workspace (pre-selection state); an empty
// ActingAsClientID means staff/operator scope (no client row-scoping).
type RequestIdentity struct {
	// UserID is the authenticated user's ID. Always set for authenticated
	// requests; the session middleware panics or redirects before reaching
	// use cases if this is empty.
	UserID string

	// WorkspaceID is the workspace the user is currently operating in.
	// Empty for pre-workspace-selection state (valid session, no workspace
	// chosen yet).
	WorkspaceID string

	// WorkspaceUserID is the workspace-user binding ID. Empty when
	// WorkspaceID is empty (no binding without a workspace).
	WorkspaceUserID string

	// Email is the user's email address. May be empty for sessions that
	// don't carry an email (e.g. service-to-service calls).
	Email string

	// SessionToken is the raw session cookie value. Carried so that
	// downstream code (e.g. workspace-path middleware) can read the token
	// without re-parsing the cookie.
	SessionToken string

	// PrincipalType is the kind of the session's ACTIVE binding, as the
	// domain.entity.v1.PrincipalType enum's integer value (e.g. 7 =
	// PRINCIPAL_TYPE_STAFF). Stamped by the session middleware from the
	// session row via LookupSessionPrincipal. A zero value (UNSPECIFIED)
	// means "no resolved binding" — a pre-selection session or a
	// service-to-service / no-session context; downstream authorization
	// treats zero as the legacy union-across-all-bindings sentinel (see
	// internal/.../rbac/authorizer.go loadCodes). Pairs with PrincipalID:
	// a REAL binding is (PrincipalType != 0 && PrincipalID != "").
	PrincipalType int32

	// PrincipalID is the grant-row ID the session's ACTIVE binding
	// identifies (the staff.id / workspace_user.id / client.id … row the
	// PrincipalType selects). Stamped by the session middleware from the
	// session row via LookupSessionPrincipal. Empty when no binding is
	// resolved. Row-scoping adapters read identity.Must(ctx).PrincipalID
	// alongside PrincipalType (e.g. PrincipalType == 7 → PrincipalID is the
	// acting staff.id).
	PrincipalID string

	// ActingAsClientID is the client ID the user is currently acting as
	// (client-portal principal or delegate). Empty for staff/operator
	// principals (workspace-wide scope). Use cases that row-scope by client
	// read this field; an empty value means "no client filter" (fail-open
	// for staff is intentional — RBAC gates the permission, not the row
	// scope).
	ActingAsClientID string

	// ActingAsSupplierID is the supplier ID the user is currently acting as
	// (supplier-portal principal or delegate). Empty for staff/operator
	// principals. Mirrors ActingAsClientID for the supplier portal.
	ActingAsSupplierID string
}

// WithRequestIdentity stores the identity on the context atomically. This is
// the ONLY writer — middleware calls this once per request. The struct is
// stored by pointer so subsequent reads via Must/Require share the same
// allocation (no copy per context.Value call).
func WithRequestIdentity(ctx context.Context, id *RequestIdentity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// WithSessionBinding stamps the session's ACTIVE principal binding
// (PrincipalType, PrincipalID, ActingAsClientID, ActingAsSupplierID) onto the
// RequestIdentity already stored on the context, returning the context.
//
// Additive companion to WithRequestIdentity: the session middleware first
// stores the user/workspace identity (WithSessionIdentity → WithRequestIdentity),
// then resolves the session row's binding and calls this to stamp it. Because
// the identity is stored by pointer, this mutates the SAME allocation in place
// (matching how the middleware stamps SessionToken), so every downstream
// Must/Require read observes the binding without a context re-derivation.
//
// Fail-closed: if no RequestIdentity is present (pre-auth / service-to-service
// context), this stores a fresh struct carrying ONLY the binding — user and
// workspace remain zero, so nothing is silently elevated.
func WithSessionBinding(ctx context.Context, principalType int32, principalID, actingAsClientID, actingAsSupplierID string) context.Context {
	if id, ok := FromContext(ctx); ok {
		id.PrincipalType = principalType
		id.PrincipalID = principalID
		id.ActingAsClientID = actingAsClientID
		id.ActingAsSupplierID = actingAsSupplierID
		return ctx
	}
	return WithRequestIdentity(ctx, &RequestIdentity{
		PrincipalType:      principalType,
		PrincipalID:        principalID,
		ActingAsClientID:   actingAsClientID,
		ActingAsSupplierID: actingAsSupplierID,
	})
}

// Must returns the RequestIdentity from the context, panicking if absent.
// Use on middleware-protected routes where the identity is guaranteed to exist;
// a panic here indicates a middleware misconfiguration (the route should have
// been excluded from session validation, not silently missing identity).
func Must(ctx context.Context) *RequestIdentity {
	id, ok := ctx.Value(contextKey{}).(*RequestIdentity)
	if !ok || id == nil {
		panic("identity.Must: no RequestIdentity in context — is the session middleware applied?")
	}
	return id
}

// Require returns the RequestIdentity from the context or an error if absent.
// Use on pre-auth routes (login, health check, public endpoints) where the
// identity may legitimately not exist.
func Require(ctx context.Context) (*RequestIdentity, error) {
	id, ok := ctx.Value(contextKey{}).(*RequestIdentity)
	if !ok || id == nil {
		return nil, ErrIdentityNotInContext
	}
	return id, nil
}

// FromContext is a convenience alias for Require — returns the identity and a
// boolean indicating presence. Useful in contexts where an error value is
// unnecessary (e.g. optional identity reads in shared utilities).
func FromContext(ctx context.Context) (*RequestIdentity, bool) {
	id, ok := ctx.Value(contextKey{}).(*RequestIdentity)
	return id, ok && id != nil
}
