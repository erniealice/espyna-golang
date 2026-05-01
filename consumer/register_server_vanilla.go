//go:build vanilla

package consumer

// Pulls in the vanilla net/http server adapter via the contrib/http sibling
// module. Only ONE server framework should be tagged in any given build —
// each framework's register file uses a single positive tag (vanilla, fiber,
// gin) so the user picks via .env CONFIG_SERVER_PROVIDER and Go enforces
// exclusivity at link time.
import _ "github.com/erniealice/espyna-golang/contrib/http"
