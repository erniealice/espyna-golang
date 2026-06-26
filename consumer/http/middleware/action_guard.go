// Package middleware — action_guard.go (AGNOSTIC surface).
//
// Framework-independent contract for the /action/* workspace form-guard
// middleware plus the HMAC-secret env helpers (pure stdlib, framework-neutral,
// so they live here rather than in the impl). The net/http impl lives in
// contrib/http/internal/adapter/middleware/action_guard.go (//go:build http)
// and is reached through the consumer/http build-tagged dispatcher
// (buildActionGuard).
package middleware

import (
	"context"
	"net/http"
)

const (
	// EnvKeyWorkspaceFormHMAC is the canonical env var for the action
	// guard + CSRF HMAC signing key.
	EnvKeyWorkspaceFormHMAC = "SECURITY_WORKSPACEFORM_HMAC_KEY"

	// EnvKeyFallbackHMAC is the fallback env var for dev/test. Re-uses the
	// password-auth reset-token secret so a single secret serves both
	// purposes in small deployments.
	EnvKeyFallbackHMAC = "AUTH_PASSWORD_RESET_TOKEN_SECRET"
)

// SecretFromEnv reads the HMAC signing key from the environment, preferring
// EnvKeyWorkspaceFormHMAC and falling back to EnvKeyFallbackHMAC. Returns ""
// when neither is set. Pure stdlib — no impl dependency.
func SecretFromEnv(getenv func(string) string) string {
	if v := getenv(EnvKeyWorkspaceFormHMAC); v != "" {
		return v
	}
	if v := getenv(EnvKeyFallbackHMAC); v != "" {
		return v
	}
	return ""
}

// ActionGuardConfig configures the /action/* workspace form-guard middleware.
type ActionGuardConfig struct {
	// Secret is the HMAC-SHA256 key used to verify the _workspace_id_sig form
	// field. When empty the impl is a pass-through (action guard disabled —
	// the consumer/http boot guard fatals before this for a real auth provider).
	Secret []byte

	// SessionWorkspaceID reads the session's current workspace_id from the
	// request context. Typically wired to consumer.GetWorkspaceIDFromContext.
	// When nil the impl defaults to "" (pre-workspace actions pass through).
	SessionWorkspaceID func(ctx context.Context) string

	// PathPrefix scopes the guard. Defaults to "/action/".
	PathPrefix string
}

// ActionGuard returns a MiddlewareFunc that enforces the signed _workspace_id
// hidden-field invariant on /action/* mutating requests (cross-workspace form
// replay defence).
//
// This agnostic entry point is a pass-through; consumer/http overrides it via
// the build-tagged buildActionGuard when the `http` server provider is
// compiled in.
func ActionGuard(cfg ActionGuardConfig) MiddlewareFunc {
	return func(next http.Handler) http.Handler { return next }
}
