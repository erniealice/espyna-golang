// Package middleware provides re-usable HTTP middleware wrappers for espyna
// consumer apps. Each wrapper returns a MiddlewareFunc that can be passed to
// server.WithMiddleware. The simple wrappers (Logger, Recovery, Gzip,
// SecurityHeaders, CookieSecure, LoginRateLimit, Timezone) are self-contained;
// the complex wrappers (Session, WorkspacePath, CSRF, ActionGuard) accept
// configuration and will be filled when the Server API lands.
package middleware

import "net/http"

// MiddlewareFunc is a function that wraps an http.Handler with middleware
// behaviour. This is the standard Go middleware signature.
type MiddlewareFunc func(http.Handler) http.Handler
