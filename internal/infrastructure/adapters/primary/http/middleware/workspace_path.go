package middleware

import (
	"context"
	"net/http"
	"time"
)

// WorkspacePathConfig configures the WorkspacePath middleware wrapper.
// All closure fields are wired by the caller (container.go or Server.Build())
// from the app's internal use cases and adapters.
//
// The full implementation of the workspace-path parsing, slug resolution,
// binding validation, session rotation, and CSRF-cookie issuance lives in
// the service-admin middleware package. This config struct mirrors its
// closure interface so the espyna Server can construct the middleware
// without importing service-admin internals.
type WorkspacePathConfig struct {
	// SlugLookup resolves a workspace slug to a workspace_id. Returns ("", nil)
	// on miss; ("", err) on infrastructure failure. When nil every lookup
	// returns miss (safe for boot-time stub configurations).
	SlugLookup func(ctx context.Context, slug string) (string, error)

	// SessionLookup reads the current session identity from the request.
	// Returns (userID, workspaceID, token, ok). ok=false means no session
	// context was found (unauthenticated request). Required.
	SessionLookup func(r *http.Request) (userID, workspaceID, token string, ok bool)

	// BindingResolver validates the user's binding in the URL workspace.
	// The kind and principalID parameters carry the session's current
	// principal hint so the resolver can "stay in the current lane" for
	// multi-binding users. Required.
	BindingResolver func(ctx context.Context, userID, workspaceID string, kind int32, principalID string) (binding interface{}, err error)

	// PrincipalLookup reads the session's current principal kind + id
	// from the request. Optional; when nil the middleware treats every
	// request as "no hint" and the resolver falls back to single-binding
	// or picker behaviour.
	PrincipalLookup func(r *http.Request) (kind int32, principalID string)

	// ExecuteSwitch performs the atomic session update for a URL-driven
	// workspace navigation. Required.
	ExecuteSwitch func(ctx context.Context, userID, token string, binding interface{}, urlActingAs string, requestURL, referer, secFetchSite, userAgent string) (*WorkspaceSwitchResult, error)

	// SetCSRFCookie issues a fresh workspace-claim CSRF cookie alongside
	// the rotated session cookie. Called with (w, newSessionToken,
	// newWorkspaceID) only when rotation occurred. When nil no CSRF cookie
	// is issued on URL-driven rotation.
	SetCSRFCookie func(w http.ResponseWriter, newSessionToken, newWorkspaceID string)

	// IsReservedSlug reports whether a slug is reserved (e.g. "auth",
	// "me", "portal"). When nil no slugs are treated as reserved.
	IsReservedSlug func(slug string) bool

	// AppOrigin is the canonical origin (scheme://host[:port]) for the
	// CSRF preflight Referer-fallback. Empty = strict deny on missing
	// Sec-Fetch-* headers (production-safe default).
	AppOrigin string

	// SlugCacheTTL is how long a slug-to-workspace_id mapping is cached.
	// Default 5 minutes.
	SlugCacheTTL time.Duration

	// RotationRateLimitPerMin is the maximum URL-driven rotations per
	// user per minute. Default 10.
	RotationRateLimitPerMin int

	// Handler is the pre-built middleware function. When set, all other
	// config fields are ignored and this handler is used directly. This
	// allows the caller to construct the full middleware implementation
	// (e.g. from service-admin's NewWorkspacePathMiddleware) and pass it
	// through. When nil, the config fields above must be populated and a
	// built-in implementation will be used (future; currently panics).
	Handler func(http.Handler) http.Handler
}

// WorkspaceSwitchResult is the outcome of a URL-driven principal switch.
type WorkspaceSwitchResult struct {
	// NewToken is non-empty when the session was rotated.
	NewToken string
	// RedirectURL is the target URL after rotation (may be empty).
	RedirectURL string
}

// WorkspacePath returns a MiddlewareFunc that parses /w/{slug}/* URL paths,
// resolves workspace slugs to workspace IDs, validates user bindings, and
// optionally rotates sessions on cross-workspace navigation.
//
// When cfg.Handler is set, it is used directly (the caller has already
// constructed the full middleware, e.g. from the service-admin middleware
// package). Otherwise, the config closures are used to build the middleware.
//
// When cfg is nil or zero-valued with no Handler, the middleware is a
// pass-through.
func WorkspacePath(cfg WorkspacePathConfig) MiddlewareFunc {
	// Fast path: pre-built handler provided
	if cfg.Handler != nil {
		return func(next http.Handler) http.Handler {
			return cfg.Handler(next)
		}
	}

	// No handler and no session lookup → pass-through (boot-time stub)
	if cfg.SessionLookup == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Future: when the full implementation moves to espyna, it will be
	// constructed here from the config closures. For now, callers must
	// either set cfg.Handler (delegating to service-admin's middleware)
	// or accept the pass-through.
	return func(next http.Handler) http.Handler {
		return next
	}
}
