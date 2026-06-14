//go:build !http

// middleware_dispatch_stub.go
//
// Pass-through fallbacks for the build-tagged middleware seam when the `http`
// server provider is NOT compiled in (e.g. plain `go build ./...`, or a gin /
// fiber / grpc provider build). Mirrors the API of middleware_dispatch_http.go
// so consumer/http/server.go compiles under any provider tag.
//
// The framework-native net/http chain assembler lives in contrib/http/provider
// (chain_http.go) and is bound by middleware_dispatch_http.go under
// //go:build http. Other server providers (gin, fiber) supply their own chain
// assemblers through their own dispatcher files (follow-on track).
package http

import (
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

// buildChain (no-http stub): returns the inner handler unwrapped. See
// contrib/http/provider/chain_stub.go for the fail-safety rationale.
func buildChain(_ consumermw.Preset, inner http.Handler) http.Handler { return inner }

// NOTE (A.3.0): the build-tagged issueWorkspaceCSRFCookie twin (and its `!http`
// `return ""` fail-open stub) are DELETED. The workspace-CSRF cookie writer is
// now the AGNOSTIC no-tag consumermw.IssueWorkspaceCSRFCookie (it computes the
// HMAC inline + http.SetCookie), so the cookie is written under EVERY build —
// there is no tagged stub left to fail open.

// buildWorkspacePath (no-http stub): pass-through twin of the http forwarder.
// Retained for build-tag-twin symmetry with middleware_dispatch_http.go.
func buildWorkspacePath(cfg consumermw.WorkspacePathConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}
