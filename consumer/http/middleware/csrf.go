// Package middleware — csrf.go (AGNOSTIC surface).
//
// Framework-independent contract for the workspace-claim CSRF middleware. The
// net/http impl lives in contrib/http/internal/adapter/middleware/csrf.go
// (//go:build http) and is reached through the consumer/http build-tagged
// dispatcher (buildCSRF). This file owns only the config type; the impl-build
// is selected at compile time by the `http` server-provider tag.
package middleware

import "net/http"

// CSRFConfig configures the workspace-claim CSRF middleware.
type CSRFConfig struct {
	// Secret is the HMAC-SHA256 signing key for workspace-claim CSRF tokens.
	// Use SecretFromEnv to populate. When empty the impl issues/verifies
	// opaque random tokens (no workspace/session claim validation).
	Secret []byte

	// SessionToken extracts the current session token from the request.
	// Typically wired to consumer.GetSessionTokenFromContext. When nil the
	// impl skips the session claim.
	SessionToken func(r *http.Request) string

	// WorkspaceID extracts the session's current workspace_id from the request.
	// Typically wired to consumer.GetWorkspaceIDFromContext. When nil the impl
	// skips the workspace claim.
	WorkspaceID func(r *http.Request) string

	// PathPrefix scopes validation to mutation endpoints. Defaults to
	// "/action/". Cookie issuance still happens on every GET regardless.
	PathPrefix string
}

// CSRF returns a MiddlewareFunc that validates workspace-claim CSRF tokens on
// mutating requests and refreshes the double-submit cookie on GET.
//
// This agnostic entry point is a pass-through; consumer/http overrides it via
// the build-tagged buildCSRF when the `http` server provider is compiled in.
func CSRF(cfg CSRFConfig) MiddlewareFunc {
	return func(next http.Handler) http.Handler { return next }
}
