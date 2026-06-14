package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

const (
	// EnvKeyWorkspaceFormHMAC is the canonical env var for the action
	// guard + CSRF HMAC signing key.
	EnvKeyWorkspaceFormHMAC = impl.EnvKeyWorkspaceFormHMAC

	// EnvKeyFallbackHMAC is the fallback env var for dev/test.
	EnvKeyFallbackHMAC = impl.EnvKeyFallbackHMAC
)

// SecretFromEnv reads the HMAC signing key from the environment,
// preferring EnvKeyWorkspaceFormHMAC and falling back to
// EnvKeyFallbackHMAC. Returns "" when neither is set.
func SecretFromEnv(getenv func(string) string) string { return impl.SecretFromEnv(getenv) }

// ActionGuardConfig configures the ActionGuard middleware wrapper.
type ActionGuardConfig = impl.ActionGuardConfig

// ActionGuard returns a MiddlewareFunc that enforces the signed
// _workspace_id hidden-field invariant on /action/* mutating requests.
func ActionGuard(cfg ActionGuardConfig) MiddlewareFunc { return impl.ActionGuard(cfg) }
