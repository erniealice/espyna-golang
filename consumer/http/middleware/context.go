package middleware

import (
	"context"

	impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"
)

// NonceFromContext retrieves the per-request CSP nonce. Returns "" if
// the security headers middleware did not run.
func NonceFromContext(ctx context.Context) string { return impl.NonceFromContext(ctx) }
