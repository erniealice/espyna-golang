//go:build grpc

package consumer

// Pulls in the gRPC server adapter via the contrib/grpc sibling module.
// Only ONE server framework should be tagged in any given build —
// each framework's register file uses a single positive tag (http, fiber,
// gin, fiber_v3, grpc) so the user picks via .env CONFIG_SERVER_PROVIDER
// and Go enforces exclusivity at link time.
import _ "github.com/erniealice/espyna-golang/contrib/grpc"
