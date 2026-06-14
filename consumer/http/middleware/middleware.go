// Package middleware is the AGNOSTIC public API surface for espyna HTTP
// middleware: declarative config types + the opaque Preset. It carries NO
// //go:build tag and imports NO framework impl. The framework-native middleware
// bodies live in contrib/{http,gin,fiber}/internal/adapter/middleware and are
// assembled by the build-tagged contrib chain (provider.BuildChain), which is
// the ONE seam that owns chain assembly — there are deliberately no per-middleware
// re-export forwarders here (those drift; the assembler calls the impls directly).
package middleware

import "net/http"

// MiddlewareFunc is a function that wraps an http.Handler with middleware
// behaviour — the standard Go middleware signature. It is carried on the Preset
// (the `extras` slot) and reverse-applied OUTSIDE the fixed security core by the
// chain assembler.
type MiddlewareFunc func(http.Handler) http.Handler
