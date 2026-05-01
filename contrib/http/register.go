// Package espynahttp hosts framework-specific HTTP server adapters and shared
// HTTP utilities (params, list params, sort spec) used across all frameworks.
//
// To enable the vanilla net/http adapter:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/http"
//	// build with: -tags vanilla
//
// The blank import alone pulls only the params utilities; the vanilla adapter
// registers via init() only when the `vanilla` build tag is active. This file
// has no imports so the package always exists for blank-imports regardless of
// which (if any) framework tag is set.
package espynahttp
