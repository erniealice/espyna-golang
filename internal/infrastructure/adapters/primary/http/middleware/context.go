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
