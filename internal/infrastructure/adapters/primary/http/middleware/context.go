package middleware

import "context"

type nonceCtxKey struct{}

// contextWithNonce stores the CSP nonce on the context for downstream
// template rendering.
func contextWithNonce(ctx context.Context, nonce string) context.Context {
	return context.WithValue(ctx, nonceCtxKey{}, nonce)
}

// NonceFromContext retrieves the per-request CSP nonce. Returns "" if
// the security headers middleware did not run.
func NonceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(nonceCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// --- Workspace-path scoping (framework-agnostic) ---
//
// The WorkspacePath middleware (one impl per server provider) pins the URL
// workspace slug and the optional /as/{client_id} acting-as target onto the
// request context. The KEYS + accessors live here — a single canonical home,
// keyed by unexported value types so no other package can collide — and are
// re-exported by consumer/http/middleware (the agnostic seam). Every provider
// writes through WithURLWorkspaceSlug / WithActingAsClientID so a downstream
// consumer (e.g. the app's workspace route rewriter) reads ONE key regardless
// of which server framework served the request.

type urlWorkspaceSlugCtxKey struct{}
type actingAsClientIDCtxKey struct{}

// WithURLWorkspaceSlug stores the URL workspace slug on the context.
func WithURLWorkspaceSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, urlWorkspaceSlugCtxKey{}, slug)
}

// GetURLWorkspaceSlugFromContext returns the URL workspace slug pinned by the
// WorkspacePath middleware (e.g. "leapfor" for /w/leapfor/...), or "" when the
// request is not workspace-scoped or the middleware did not run.
func GetURLWorkspaceSlugFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(urlWorkspaceSlugCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// WithActingAsClientID stores the /as/{client_id} acting-as target on the
// context.
func WithActingAsClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, actingAsClientIDCtxKey{}, clientID)
}

// GetActingAsClientIDFromContext returns the /as/{client_id} acting-as target
// pinned by the WorkspacePath middleware, or "" when absent.
func GetActingAsClientIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(actingAsClientIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}
