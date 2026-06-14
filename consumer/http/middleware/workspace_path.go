// Package middleware — workspace_path.go (AGNOSTIC surface).
//
// This file defines the framework-INDEPENDENT contract for the workspace-path
// middleware: the config struct, the neutral binding type, and the sentinel
// errors. It has NO build tag and imports NO impl package — it is pure
// stdlib + this package's own MiddlewareFunc.
//
// The framework-native net/http IMPLEMENTATION lives in
// contrib/http/internal/adapter/middleware/workspace_path.go (//go:build http)
// and is selected at runtime via CONFIG_SERVER_PROVIDER=http (build tag `http`).
// The bridge that adapts this agnostic config to that impl is
// contrib/http/middleware_http.go (BuildWorkspacePath), invoked from
// consumer/http through a build-tagged dispatcher (workspace_path_http.go /
// workspace_path_stub.go). See docs/plan/20260614-composition-model-a/contrib-pattern.md.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Principal-binding kind constants. PROTO-ALIGNED: identical to esqyma's
// domain.entity.principal_type.PrincipalType enum AND the contrib impl's
// PrincipalType, so the int32 value is carried verbatim across the boundary
// with no translation table. The agnostic surface uses int32 so it does not
// import the contrib impl's named type or the proto enum.
const (
	BindingKindUnspecified      int32 = 0
	BindingKindOperatorOwner    int32 = 1
	BindingKindOperatorStaff    int32 = 2
	BindingKindClient           int32 = 3
	BindingKindClientDelegate   int32 = 4
	BindingKindSupplier         int32 = 5
	BindingKindSupplierDelegate int32 = 6
)

// Sentinel errors a BindingResolver may return. The impl recognizes these by
// value (errors.Is) to choose the picker (ambiguous) vs unified-not-found
// (no binding) response, so resolvers MUST return these exact values rather
// than wrapped copies from another package.
var (
	// ErrNoBinding signals the user has no active binding in the workspace.
	// Maps to the unified 303 -> /auth/select-workspace-role response.
	ErrNoBinding = errors.New("workspace_path: user has no active binding in workspace")

	// ErrAmbiguousBinding signals multiple bindings with no disambiguating
	// session hint. Maps to the picker (303 -> /auth/select-workspace-role).
	// NO auto-elect by privilege (security invariant A3).
	ErrAmbiguousBinding = errors.New("workspace_path: ambiguous binding")
)

// WorkspaceBinding is the neutral, framework-agnostic representation of a
// resolved principal binding in a workspace. BindingResolver produces one and
// the impl bridge converts it to the contrib Principal; ExecuteSwitch receives
// the same value back so the switch primitive can act on the exact binding the
// resolver returned. Stdlib/scalar fields only — no proto, no impl types.
type WorkspaceBinding struct {
	// Kind is the principal kind (BindingKind* constants / esqyma int32 value).
	Kind int32
	// PrincipalID is the binding's principal_id.
	PrincipalID string
	// WorkspaceID is the workspace the binding belongs to.
	WorkspaceID string
	// DisplayName is the principal's display name (for the rotation audit/banner).
	DisplayName string
	// ActingAsTargets enumerates delegate acting-as targets (client/supplier).
	ActingAsTargets []WorkspaceActingAsTarget
}

// WorkspaceActingAsTarget is one acting-as target a delegate binding may serve.
type WorkspaceActingAsTarget struct {
	ID          string
	WorkspaceID string
	DisplayName string
}

// WorkspaceSwitchResult is the outcome of a URL-driven principal switch.
type WorkspaceSwitchResult struct {
	// NewToken is non-empty when the session was rotated.
	NewToken string
	// RedirectURL is the target URL after rotation (may be empty).
	RedirectURL string
}

// WorkspacePathConfig configures the WorkspacePath middleware. All closure
// fields are framework-agnostic (stdlib types + WorkspaceBinding). The
// consumer/http Server populates them from espyna's OWN use cases
// (ResolveWorkspaceBySlug, ResolveBinding, SwitchPrincipal, LookupSessionPrincipal)
// — the consumer app supplies none of them.
type WorkspacePathConfig struct {
	// SlugLookup resolves a workspace slug to a workspace_id. Returns ("", nil)
	// on miss; ("", err) on infrastructure failure. When nil every lookup
	// returns miss.
	SlugLookup func(ctx context.Context, slug string) (string, error)

	// SessionLookup reads the current session identity from the request.
	// Returns (userID, workspaceID, token, ok). ok=false means no session
	// context was found. Required (the dispatcher treats a nil SessionLookup
	// as a disabled / pass-through middleware).
	SessionLookup func(r *http.Request) (userID, workspaceID, token string, ok bool)

	// BindingResolver validates the user's binding in the URL workspace. The
	// kind + principalID carry the session's current principal hint so the
	// resolver stays in the session's lane (A3 — no auto-elect by privilege).
	// Returns (nil, ErrNoBinding) / (nil, ErrAmbiguousBinding) for the two
	// fail-closed branches. Required.
	BindingResolver func(ctx context.Context, userID, workspaceID string, kind int32, principalID string) (*WorkspaceBinding, error)

	// PrincipalLookup reads the session's current principal kind + id from the
	// request. Optional; nil = "no hint".
	PrincipalLookup func(r *http.Request) (kind int32, principalID string)

	// ExecuteSwitch performs the atomic session update for a URL-driven
	// workspace navigation. binding is the exact value BindingResolver
	// returned (with URL-derived acting-as applied). Required.
	ExecuteSwitch func(ctx context.Context, userID, token string, binding *WorkspaceBinding, urlActingAs string, requestURL, referer, secFetchSite, userAgent string) (*WorkspaceSwitchResult, error)

	// SetCSRFCookie issues a fresh workspace-claim CSRF cookie alongside the
	// rotated session cookie. Called with (w, newSessionToken, newWorkspaceID)
	// only when rotation occurred. When nil no CSRF cookie is issued on rotation.
	SetCSRFCookie func(w http.ResponseWriter, newSessionToken, newWorkspaceID string)

	// SetSessionCookie writes the rotated session cookie. When nil the impl
	// writes a SameSite=Strict cookie via its own writer (security invariant).
	SetSessionCookie func(w http.ResponseWriter, token string)

	// WithWorkspaceID pins the URL-canonical workspace_id into the request
	// context BEFORE downstream guards read it (Q-WS-13: URL is canonical).
	// Typically wired to consumer.WithWorkspaceID. When nil the impl still
	// sets its own CtxKeyURLWorkspaceID, but the espyna-owned workspace_id key
	// is NOT overridden — callers SHOULD wire this for correct guard behaviour.
	WithWorkspaceID func(ctx context.Context, workspaceID string) context.Context

	// IsReservedSlug reports whether a slug is reserved (e.g. "auth", "me",
	// "portal"). When nil no slugs are treated as reserved.
	IsReservedSlug func(slug string) bool

	// AppOrigin is the canonical origin (scheme://host[:port]) for the CSRF
	// preflight Referer-fallback. Empty = strict deny on missing Sec-Fetch-*
	// headers (production-safe default).
	AppOrigin string

	// SlugCacheTTL is how long a slug->workspace_id mapping is cached. Default 5m.
	SlugCacheTTL time.Duration

	// RotationRateLimitPerMin is the maximum URL-driven rotations per user per
	// minute. Default 10.
	RotationRateLimitPerMin int
}

// WorkspacePath returns a MiddlewareFunc that parses /w/{slug}/* URL paths,
// resolves workspace slugs to workspace IDs, validates user bindings, pins the
// URL-canonical workspace_id into context, rotates the session (+ Strict cookie
// + fresh CSRF cookie) on cross-workspace navigation, strips the /w/{slug}
// prefix, and dispatches the bare route to the downstream handler.
//
// The implementation is selected at build time:
//   - //go:build http  -> contrib/http net/http impl (workspace_path_http.go)
//   - otherwise         -> pass-through stub (workspace_path_stub.go)
//
// The dispatcher lives in package http (consumer/http) because the contrib
// impl is under contrib/http/internal/ (Go visibility) and because building
// the impl needs consumer context helpers; this agnostic package only owns the
// contract. consumer/http/server.go calls buildWorkspacePath(cfg) directly.
func WorkspacePath(cfg WorkspacePathConfig) MiddlewareFunc {
	// Agnostic fallback: with no SessionLookup there is nothing to drive the
	// middleware, so it is a pass-through regardless of build tag. consumer/http
	// overrides this via the build-tagged buildWorkspacePath when the `http`
	// provider is compiled in.
	return func(next http.Handler) http.Handler { return next }
}
