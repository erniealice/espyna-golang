//go:build http

// middleware_dispatch_http.go
//
// Build-tagged dispatcher that binds the AGNOSTIC consumer/http/middleware
// surface to the FRAMEWORK-NATIVE net/http impls in contrib/http. Compiled when
// CONFIG_SERVER_PROVIDER=http selects the net/http server provider
// (scripts/build.sh adds the value as a build tag).
//
// This is the ONLY place consumer/http imports contrib/http/provider. It is
// safe: contrib/http/provider imports the agnostic consumer/http/middleware (a
// dep-free leaf) but NOT consumer or consumer/http, so no import cycle is
// formed. The contrib builders own the actual middleware construction (they
// alone can import contrib/http/internal/... per Go visibility); this file just
// forwards.
//
// W2 (docs/plan/20260614-composition-model-a/w2-plan.md §5): the W1 per-function
// trio (buildWorkspacePath/buildCSRF/buildActionGuard) is COLLAPSED into one
// buildChain seam that forwards the agnostic Preset to provider.BuildChain (the
// fixed-order assembler). issueWorkspaceCSRFCookie is retained because
// server.go's finalizePreset wires it into WorkspacePathConfig.SetCSRFCookie
// (the rotation-time fresh-CSRF-cookie writer) — it crosses the build-tag
// boundary so it needs a tag-twin here.
//
// Without the `http` tag, middleware_dispatch_stub.go provides pass-throughs so
// `go build ./...` (no tags) still compiles.
package http

import (
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	provider "github.com/erniealice/espyna-golang/contrib/http/provider"
	pyeza "github.com/erniealice/pyeza-golang"
)

// buildChain assembles the full fixed-order net/http middleware chain from the
// agnostic Preset and wraps the inner handler. The single seam that subsumes the
// W1 buildWorkspacePath/buildCSRF/buildActionGuard dispatch + the inline
// 11-middleware nest. See contrib/http/provider/chain_http.go BuildChain.
func buildChain(p consumermw.Preset, inner http.Handler) http.Handler {
	return provider.BuildChain(p, inner)
}

// NOTE (A.3.0): the issueWorkspaceCSRFCookie dispatch twin is DELETED. The
// workspace-CSRF cookie writer moved to the AGNOSTIC no-tag
// consumermw.IssueWorkspaceCSRFCookie (HMAC computed inline + http.SetCookie),
// so server.go's finalizePreset wires WorkspacePathConfig.SetCSRFCookie to it
// directly — no build-tag boundary crossing, no provider re-export.

// buildWorkspacePath returns the real net/http WorkspacePath middleware wired
// from cfg. The production chain reaches it through provider.BuildChain (which
// calls provider.BuildWorkspacePath in the correct fixed-order slot); this thin
// forwarder is RETAINED so the workspace-path security slice-proof
// (workspace_path_slice_test.go) can exercise the WorkspacePath middleware in
// isolation without depending on the full chain assembly.
func buildWorkspacePath(cfg consumermw.WorkspacePathConfig) func(http.Handler) http.Handler {
	return provider.BuildWorkspacePath(cfg)
}

// buildWorkspaceFormSigner builds the render-side {{actionForm}} signer from the
// SAME guard secret the ActionGuard verifier uses, returning it as the agnostic
// pyeza.WorkspaceFormSigner interface so finalizeHTTPAdapter can install it on
// the app-supplied renderer. This is the symmetric half of provider.BuildActionGuard
// (which builds the verifier from the same secret), so signer and verifier can
// NEVER drift. Returns nil on an empty secret (guard disabled → renderer needs
// no signer), mirroring the guard's pass-through-on-empty. Crosses the build-tag
// boundary into contrib/http/provider, the ONE place consumer/http may reach it.
func buildWorkspaceFormSigner(secret string) pyeza.WorkspaceFormSigner {
	s := provider.NewWorkspaceFormSigner(secret)
	if s == nil {
		// Return an explicit nil interface, not a typed-nil pointer, so callers
		// can compare == nil reliably.
		return nil
	}
	return s
}
