//go:build !http

// middleware_dispatch_stub.go
//
// Pass-through fallbacks for the workspace-security middleware when the `http`
// server provider is NOT compiled in (e.g. plain `go build ./...`, or a gin /
// fiber / grpc provider build). Mirrors the API of middleware_dispatch_http.go
// so consumer/http/server.go compiles under any provider tag.
//
// The framework-native net/http impls live in contrib/http and are bound by
// middleware_dispatch_http.go under //go:build http. Other server providers
// (gin, fiber) supply their own equivalents through their own dispatcher files
// (follow-on track).
package http

import (
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

func buildWorkspacePath(cfg consumermw.WorkspacePathConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func buildCSRF(cfg consumermw.CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func buildActionGuard(cfg consumermw.ActionGuardConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func issueWorkspaceCSRFCookie(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string {
	return ""
}
