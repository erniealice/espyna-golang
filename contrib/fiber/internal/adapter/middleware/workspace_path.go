//go:build fiber

package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// WorkspacePathConfig configures the WorkspacePath middleware wrapper for Fiber.
// All closure fields are wired by the caller from the app's internal use cases
// and adapters.
//
// Mirrors the vanilla WorkspacePathConfig
// (consumer/http/middleware/workspace_path.go) adapted for Fiber's handler
// signature.
type WorkspacePathConfig struct {
	// SlugLookup resolves a workspace slug to a workspace_id. Returns ("", nil)
	// on miss; ("", err) on infrastructure failure. When nil every lookup
	// returns miss (safe for boot-time stub configurations).
	SlugLookup func(ctx context.Context, slug string) (string, error)

	// SessionLookup reads the current session identity from the Fiber context.
	// Returns (userID, workspaceID, token, ok). ok=false means no session
	// context was found (unauthenticated request). Required.
	SessionLookup func(c *fiber.Ctx) (userID, workspaceID, token string, ok bool)

	// BindingResolver validates the user's binding in the URL workspace.
	// The kind and principalID parameters carry the session's current
	// principal hint so the resolver can "stay in the current lane" for
	// multi-binding users. Required.
	BindingResolver func(ctx context.Context, userID, workspaceID string, kind int32, principalID string) (binding interface{}, err error)

	// PrincipalLookup reads the session's current principal kind + id
	// from the Fiber context. Optional; when nil the middleware treats every
	// request as "no hint" and the resolver falls back to single-binding
	// or picker behaviour.
	PrincipalLookup func(c *fiber.Ctx) (kind int32, principalID string)

	// ExecuteSwitch performs the atomic session update for a URL-driven
	// workspace navigation. Required.
	ExecuteSwitch func(ctx context.Context, userID, token string, binding interface{}, urlActingAs string, requestURL, referer, secFetchSite, userAgent string) (*FiberWorkspaceSwitchResult, error)

	// SetCSRFCookie issues a fresh workspace-claim CSRF cookie alongside
	// the rotated session cookie. Called with (c, newSessionToken,
	// newWorkspaceID) only when rotation occurred. When nil no CSRF cookie
	// is issued on URL-driven rotation.
	SetCSRFCookie func(c *fiber.Ctx, newSessionToken, newWorkspaceID string)

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

	// Handler is a pre-built Fiber middleware function. When set, all other
	// config fields are ignored and this handler is used directly. This
	// allows the caller to construct the full middleware implementation
	// and pass it through. When nil, the config fields above must be
	// populated.
	Handler fiber.Handler
}

// FiberWorkspaceSwitchResult is the outcome of a URL-driven principal switch.
// Mirrors the vanilla WorkspaceSwitchResult.
type FiberWorkspaceSwitchResult struct {
	// NewToken is non-empty when the session was rotated.
	NewToken string
	// RedirectURL is the target URL after rotation (may be empty).
	RedirectURL string
}

// WorkspacePath returns a Fiber middleware that parses /w/{slug}/* URL paths,
// resolves workspace slugs to workspace IDs, validates user bindings, and
// optionally rotates sessions on cross-workspace navigation.
//
// When cfg.Handler is set, it is used directly (the caller has already
// constructed the full middleware). Otherwise, the middleware is a
// pass-through.
//
// Mirrors the vanilla net/http reference implementation
// (consumer/http/middleware/workspace_path.go).
//
// TODO: Implement full slug resolution and session rotation logic when the
// workspace-path adapter is ported to the fiber contrib layer.
func WorkspacePath(cfg WorkspacePathConfig) fiber.Handler {
	// Fast path: pre-built handler provided.
	if cfg.Handler != nil {
		return cfg.Handler
	}

	// No handler and no session lookup: pass-through (boot-time stub).
	if cfg.SessionLookup == nil {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// TODO: When the full implementation moves to espyna's fiber contrib,
	// it will be constructed here from the config closures. For now,
	// callers must either set cfg.Handler (delegating to service-admin's
	// middleware) or accept the pass-through.
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
