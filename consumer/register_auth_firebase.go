//go:build firebase

package consumer

// NOTE: auth/firebase adapter does not yet have an init() with
// registry.RegisterAuthProvider. This import is ready for when
// self-registration is added to that package.
import _ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/firebase"
