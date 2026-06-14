package middleware

import (
	"context"

	impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"
)

// NonceFromContext retrieves the per-request CSP nonce. Returns "" if
// the security headers middleware did not run.
func NonceFromContext(ctx context.Context) string { return impl.NonceFromContext(ctx) }

// WithURLWorkspaceSlug stores the URL workspace slug on the context. Called by
// each server provider's WorkspacePath middleware so the slug is readable under
// one canonical key regardless of framework.
func WithURLWorkspaceSlug(ctx context.Context, slug string) context.Context {
	return impl.WithURLWorkspaceSlug(ctx, slug)
}

// GetURLWorkspaceSlugFromContext returns the URL workspace slug pinned by the
// WorkspacePath middleware (e.g. "leapfor" for /w/leapfor/...), or "" when the
// request is not workspace-scoped. Consumed by the app's workspace route
// rewriter to prefix sidebar + route-map URLs with /w/{slug}.
func GetURLWorkspaceSlugFromContext(ctx context.Context) string {
	return impl.GetURLWorkspaceSlugFromContext(ctx)
}

// WithActingAsClientID stores the /as/{client_id} acting-as target on the
// context. Called by each provider's WorkspacePath middleware.
func WithActingAsClientID(ctx context.Context, clientID string) context.Context {
	return impl.WithActingAsClientID(ctx, clientID)
}

// GetActingAsClientIDFromContext returns the /as/{client_id} acting-as target
// pinned by the WorkspacePath middleware, or "" when absent.
func GetActingAsClientIDFromContext(ctx context.Context) string {
	return impl.GetActingAsClientIDFromContext(ctx)
}
