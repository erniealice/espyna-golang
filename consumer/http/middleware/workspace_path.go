package middleware

import "net/http"

// WorkspacePath returns a MiddlewareFunc that parses /w/{slug}/* URL paths,
// resolves workspace slugs to workspace IDs, validates user bindings, and
// optionally rotates sessions on cross-workspace navigation.
//
// Stub: the full implementation will be wired when the Server API lands.
// The middleware will accept a WorkspacePathConfig carrying slug lookup,
// session lookup, binding resolver, and execute-switch callbacks. The
// construction logic currently in container.go (~150 lines) will move here.
func WorkspacePath() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		// Pass-through until the Server API provides workspace config.
		return next
	}
}
