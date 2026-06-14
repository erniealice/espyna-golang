// Package httpctx is the dependency-free, stdlib-only leaf that owns the
// canonical request-scoped context keys crossing the middleware ↔ view ↔
// template boundary for the URL-driven workspace routing slice.
//
// ── Why this package exists (the ctx-key single-source rule) ──────────────────
//
// context.WithValue key identity is by Go TYPE identity: an identically-spelled
// `type fooKey struct{}` declared in two packages produces TWO distinct keys.
// Before this leaf, the same logical keys were re-declared in four packages
// (the app's input/http, the espyna-internal primary/http/middleware stub, the
// contrib impl, and pyeza render), so a writer in one package and a reader in
// another silently used different keys — no compile error, just an empty read.
//
// This leaf collapses the slug + acting-as keys to ONE definition each. Every
// side (contrib WorkspacePath writer, the consumer/http/middleware surface
// shim, the app's route rewriter) funnels through these accessors so the key
// agrees byte-for-byte.
//
// ── The leaf split (espyna ↔ pyeza arrow is espyna → pyeza) ───────────────────
//
// The import arrow runs espyna → pyeza (espyna consumer/ imports pyeza render;
// pyeza render imports nothing from espyna). pyeza render is therefore the
// canonical READER of the per-request CSP nonce + the post-rotation banner
// (render.NonceFromContext / render.PostRotationBannerFromContext, invoked from
// the render pipeline). Those two keys CANNOT live here without inverting the
// arrow (pyeza would have to import espyna → cycle). So:
//
//   - nonce + PostRotationBanner keys  → canonical in pyeza render; the espyna
//     contrib writers (SecurityHeaders / WorkspacePath) WRITE pyeza's keys via
//     render.WithNonce / render.WithPostRotationBanner. (espyna writes, pyeza
//     reads — the arrow is preserved.)
//   - URL workspace slug + acting-as client id → canonical HERE (pyeza does not
//     read them; only espyna middleware writes them and the app route-rewriter
//     reads them).
//
// This package carries NO build tag, NO contrib import, NO `consumer` import,
// so it can never close an import cycle and is importable by every side.
package httpctx

import "context"

// urlWorkspaceSlugKey is the canonical, unexported context key for the URL
// workspace slug pinned by the WorkspacePath middleware (e.g. "leapfor" for
// /w/leapfor/...). Empty-struct typed keys are collision-free.
type urlWorkspaceSlugKey struct{}

// actingAsClientIDKey is the canonical, unexported context key for the
// /as/{client_id} acting-as target pinned by the WorkspacePath middleware.
type actingAsClientIDKey struct{}

// WithURLWorkspaceSlug stores the URL workspace slug on the context. Called by
// each server provider's WorkspacePath middleware so the slug is readable under
// ONE canonical key regardless of which framework served the request.
func WithURLWorkspaceSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, urlWorkspaceSlugKey{}, slug)
}

// GetURLWorkspaceSlug returns the URL workspace slug pinned by the WorkspacePath
// middleware, or "" when the request is not workspace-scoped or the middleware
// did not run. Consumed by the app's workspace route rewriter to prefix sidebar
// + route-map URLs with /w/{slug}.
func GetURLWorkspaceSlug(ctx context.Context) string {
	if v, ok := ctx.Value(urlWorkspaceSlugKey{}).(string); ok {
		return v
	}
	return ""
}

// WithActingAsClientID stores the /as/{client_id} acting-as target on the
// context. Called by each provider's WorkspacePath middleware.
func WithActingAsClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, actingAsClientIDKey{}, clientID)
}

// GetActingAsClientID returns the /as/{client_id} acting-as target pinned by the
// WorkspacePath middleware, or "" when absent.
func GetActingAsClientID(ctx context.Context) string {
	if v, ok := ctx.Value(actingAsClientIDKey{}).(string); ok {
		return v
	}
	return ""
}
