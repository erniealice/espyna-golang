// Package fiber registers the Fiber HTTP framework adapters (v2 and v3) with espyna's registry.
// Import this package with a blank identifier to enable Fiber support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/fiber"
package fiber

import (
	// Import triggers adapter's init() which self-registers with the registry.
	_ "github.com/erniealice/espyna-golang/contrib/fiber/internal/adapter"
	_ "github.com/erniealice/espyna-golang/contrib/fiber/internal/adapterv3"
)
