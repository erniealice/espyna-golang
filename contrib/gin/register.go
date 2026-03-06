// Package gin registers the Gin HTTP framework adapter with espyna's registry.
// Import this package with a blank identifier to enable Gin support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/gin"
package gin

import (
	// Import triggers adapter's init() which self-registers with the registry.
	_ "github.com/erniealice/espyna-golang/contrib/gin/internal/adapter"
)
