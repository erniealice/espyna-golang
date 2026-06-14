package middleware

import (
	"context"

	"github.com/erniealice/espyna-golang/consumer/http/httpctx"
	"github.com/erniealice/pyeza-golang/render"
)

// NonceFromContext retrieves the per-request CSP nonce. Returns "" if the
// security headers middleware did not run.
//
// The nonce ctx-key is canonical in pyeza render (the render pipeline is the
// reader; the SecurityHeaders middleware is the writer). This shim re-exports
// render's accessor so the surface stays stable for any consumer reading the
// nonce off the agnostic middleware package.
func NonceFromContext(ctx context.Context) string { return render.NonceFromContext(ctx) }

// WithURLWorkspaceSlug stores the URL workspace slug on the context. Called by
// each server provider's WorkspacePath middleware so the slug is readable under
// one canonical key regardless of framework. Re-exported from the httpctx leaf.
func WithURLWorkspaceSlug(ctx context.Context, slug string) context.Context {
	return httpctx.WithURLWorkspaceSlug(ctx, slug)
}

// GetURLWorkspaceSlugFromContext returns the URL workspace slug pinned by the
// WorkspacePath middleware (e.g. "leapfor" for /w/leapfor/...), or "" when the
// request is not workspace-scoped. Consumed by the app's workspace route
// rewriter to prefix sidebar + route-map URLs with /w/{slug}. Re-exported from
// the httpctx leaf.
func GetURLWorkspaceSlugFromContext(ctx context.Context) string {
	return httpctx.GetURLWorkspaceSlug(ctx)
}

// WithActingAsClientID stores the /as/{client_id} acting-as target on the
// context. Called by each provider's WorkspacePath middleware. Re-exported from
// the httpctx leaf.
func WithActingAsClientID(ctx context.Context, clientID string) context.Context {
	return httpctx.WithActingAsClientID(ctx, clientID)
}

// GetActingAsClientIDFromContext returns the /as/{client_id} acting-as target
// pinned by the WorkspacePath middleware, or "" when absent. Re-exported from
// the httpctx leaf.
func GetActingAsClientIDFromContext(ctx context.Context) string {
	return httpctx.GetActingAsClientID(ctx)
}
