//go:build !http

// chain_stub.go
//
// Pass-through twin of chain_http.go's BuildChain for builds where the `http`
// server provider is NOT compiled in (plain `go build ./...`, `go vet`, tests,
// or a future gin/fiber/grpc provider build before its own chain assembler
// lands). It returns the inner handler unwrapped so the consumer/http dispatcher
// (middleware_dispatch_stub.go) compiles under any provider tag.
//
// Fail-safety: a bare pass-through means a `!http` build serves the inner
// handler with NO SecurityHeaders/Session/WorkspacePath/CSRF/ActionGuard. That
// matches the pre-W2 middleware_dispatch_stub.go behaviour for the W1 trio
// (already pass-through), WIDENED to all middleware. Since scripts/build.sh
// always selects -tags http (CONFIG_SERVER_PROVIDER=http), NO shipped binary
// hits this stub — it exists purely for `go build ./...` / `go vet` / tests. A
// future non-http provider MUST ship its own chain_<tag>.go with the real fixed
// order; it must NOT rely on this bare pass-through.
package provider

import (
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

// BuildChain (no-http stub): returns the inner handler unwrapped.
func BuildChain(_ consumermw.Preset, inner http.Handler) http.Handler { return inner }
