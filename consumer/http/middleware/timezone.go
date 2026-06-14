package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// TimezoneConfig configures the Timezone middleware.
type TimezoneConfig = impl.TimezoneConfig

// Timezone returns a MiddlewareFunc that resolves the authenticated user's
// timezone preference and stores the resulting *time.Location on the request
// context.
func Timezone(cfg TimezoneConfig) MiddlewareFunc { return impl.Timezone(cfg) }
