// Package espynahttp hosts framework-specific HTTP server adapters and shared
// HTTP utilities (params, list params, sort spec) used across all frameworks.
//
// To enable the net/http server adapter:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/http"
//	// build with: -tags http  (selected at runtime by CONFIG_SERVER_PROVIDER=http)
//
// The blank import alone pulls only the params utilities; the net/http adapter
// registers via init() only when the `http` build tag is active (register_http.go).
// This file has no imports so the package always exists for blank-imports
// regardless of which (if any) framework tag is set.
package espynahttp
