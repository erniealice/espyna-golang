//go:build gin

package consumer

// Pulls in the Gin server adapter via the contrib/gin sibling module.
// Only ONE server framework should be tagged in any given build.
import _ "github.com/erniealice/espyna-golang/contrib/gin"
