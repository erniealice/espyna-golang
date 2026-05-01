// Package gin registers the Gin HTTP framework adapter with espyna's
// registry. Import this package with a blank identifier to enable Gin
// support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/gin"
//	// build with: -tags gin
//
// The blank import alone pulls nothing into the binary. The Gin adapter
// sits behind a build-tagged sibling register file. This file has no
// imports so the package always exists for blank-imports even when the
// gin tag isn't set.
package gin
