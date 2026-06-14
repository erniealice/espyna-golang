//go:build http

// middleware_dispatch_http.go
//
// Build-tagged dispatcher that binds the AGNOSTIC consumer/http/middleware
// configs to the FRAMEWORK-NATIVE net/http impls in contrib/http. Compiled
// when CONFIG_SERVER_PROVIDER=http selects the net/http server provider
// (scripts/build.sh adds the value as a build tag).
//
// This is the ONLY place consumer/http imports contrib/http/provider. It is
// safe: contrib/http/provider imports the agnostic consumer/http/middleware
// (a dep-free leaf) but NOT consumer or consumer/http, so no import cycle is
// formed. The contrib builders own the actual middleware construction (they
// alone can import contrib/http/internal/... per Go visibility); this file just
// forwards the config.
//
// Without the `http` tag, middleware_dispatch_stub.go provides pass-throughs so
// `go build ./...` (no tags) still compiles.
package http

import (
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	provider "github.com/erniealice/espyna-golang/contrib/http/provider"
)

// buildWorkspacePath returns the real net/http WorkspacePath middleware wired
// from cfg. See contrib/http/middleware_http.go BuildWorkspacePath.
func buildWorkspacePath(cfg consumermw.WorkspacePathConfig) func(http.Handler) http.Handler {
	return provider.BuildWorkspacePath(cfg)
}

// buildCSRF returns the real net/http workspace-claim CSRF middleware wired
// from cfg.
func buildCSRF(cfg consumermw.CSRFConfig) func(http.Handler) http.Handler {
	return provider.BuildCSRF(cfg)
}

// buildActionGuard returns the real net/http /action/* form-guard middleware
// wired from cfg.
func buildActionGuard(cfg consumermw.ActionGuardConfig) func(http.Handler) http.Handler {
	return provider.BuildActionGuard(cfg)
}

// issueWorkspaceCSRFCookie issues a fresh workspace-claim CSRF cookie. Wired
// into WorkspacePathConfig.SetCSRFCookie so the rotation path emits a matching
// CSRF cookie alongside the rotated session cookie.
func issueWorkspaceCSRFCookie(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string {
	return provider.IssueWorkspaceCSRFCookie(w, secret, sessionToken, workspaceID)
}
