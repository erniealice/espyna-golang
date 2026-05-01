//go:build fiber

package consumer

// Pulls in the Fiber v2 server adapter via the contrib/fiber sibling module.
// Only ONE server framework should be tagged in any given build (vanilla,
// fiber, fiber_v3, gin) — the user picks via .env CONFIG_SERVER_PROVIDER.
import _ "github.com/erniealice/espyna-golang/contrib/fiber"
