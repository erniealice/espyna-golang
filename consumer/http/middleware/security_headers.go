package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

const (
	// EnvKeyHSTSEnabled gates the Strict-Transport-Security header.
	EnvKeyHSTSEnabled = impl.EnvKeyHSTSEnabled

	// EnvKeyCSPEnforce gates the CSP enforce/report-only choice.
	EnvKeyCSPEnforce = impl.EnvKeyCSPEnforce
)

// SecurityHeadersConfig configures the security-header middleware.
type SecurityHeadersConfig = impl.SecurityHeadersConfig

// SecurityHeadersConfigFromEnv builds the config from environment variables.
func SecurityHeadersConfigFromEnv(getenv func(string) string) SecurityHeadersConfig {
	return impl.SecurityHeadersConfigFromEnv(getenv)
}

// SecurityHeaders returns a MiddlewareFunc that sets defense-in-depth security
// response headers on every response.
func SecurityHeaders(cfg ...SecurityHeadersConfig) MiddlewareFunc { return impl.SecurityHeaders(cfg...) }
