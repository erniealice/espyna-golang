package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

const (
	// LoginRateLimitPath is the path this middleware gates.
	LoginRateLimitPath = impl.LoginRateLimitPath

	// EnvKeyLoginRateLimit is the env var controlling max attempts per IP
	// per minute.
	EnvKeyLoginRateLimit = impl.EnvKeyLoginRateLimit
)

// LoginRateLimit returns a MiddlewareFunc that rate-limits POST /auth/login
// by client IP.
func LoginRateLimit() MiddlewareFunc { return impl.LoginRateLimit() }

// LoginRateLimitWithStop is like LoginRateLimit but also returns a stop
// function that halts the background cleanup goroutine (useful in tests).
func LoginRateLimitWithStop() (MiddlewareFunc, func()) { return impl.LoginRateLimitWithStop() }
