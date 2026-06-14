//go:build http

package consumer

// Pulls in the stdlib net/http server adapter via the contrib/http/provider
// sibling package. The provider subpackage (not the contrib/http root) carries
// the //go:build http VanillaAdapter registration + the middleware bridge, so
// the dep-free HTTP utils in the contrib/http root stay importable by the
// postgres adapters WITHOUT dragging in the net/http adapter (which would form
// an import cycle). Only ONE server framework should be tagged in any given
// build — each framework's register file uses a single positive tag (http,
// fiber, gin, fiber_v3, grpc) so the user picks via .env CONFIG_SERVER_PROVIDER
// and Go enforces exclusivity at link time.
import _ "github.com/erniealice/espyna-golang/contrib/http/provider"
