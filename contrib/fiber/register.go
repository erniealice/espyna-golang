// Package fiber registers the Fiber HTTP framework adapter (v2 and v3) with
// espyna's registry. Import this package with a blank identifier to enable
// Fiber support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/fiber"
//	// build with: -tags fiber       (Fiber v2)
//	// build with: -tags fiber_v3    (Fiber v3)
//
// The blank import alone pulls nothing into the binary. Each version's
// adapter sits behind its own build tag in a sibling register_*.go file.
// This file has no imports so the package always exists for blank-imports
// even when no Fiber tag is set.
package fiber
