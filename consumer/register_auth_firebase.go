//go:build firebase

package consumer

// Pulls in the firebase auth adapter via the contrib/google sibling module,
// which only registers firebase when the firebase build tag is active.
import _ "github.com/erniealice/espyna-golang/contrib/google"
