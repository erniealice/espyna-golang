//go:build http

// chain_http.go
//
// The fixed-order net/http middleware chain ASSEMBLER for the `http` server
// provider. This is the ONE seam the W2 plan collapses the W1 dispatch trio
// (buildWorkspacePath/buildCSRF/buildActionGuard) AND the inline 11-middleware
// hand-assembly in consumer/http/server.go into.
//
// It lives in package provider (rooted at contrib/http) because the real
// net/http middleware impls are under contrib/http/internal/adapter/middleware
// (Go-`internal/` visibility means ONLY packages rooted at contrib/http may
// import them). It must NOT import consumer or consumer/http (that would close
// the consumer -> contrib/http import cycle); every per-request dependency
// arrives as a closure on the agnostic consumer/http/middleware Preset, which is
// a dep-free leaf this package may import.
//
// SECURITY-CRITICAL: BuildChain realizes the EXACT fixed order from the
// pre-W2 server.go (SecurityHeaders → Gzip → Logger → Recovery → LoginRateLimit
// → Session → WorkspacePath → CSRF → ActionGuard → Timezone → businessType →
// inner). The assembler only RE-ORDERS the calls — it must not alter the impl
// behaviour. See docs/plan/20260614-composition-model-a/w2-plan.md §7.
package provider

import (
	"context"
	"net/http"
	"os"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	cmw "github.com/erniealice/espyna-golang/contrib/http/internal/adapter/middleware"
)

// BuildChain assembles the full fixed-order net/http middleware chain from the
// agnostic Preset and wraps the given inner handler (the businessType→mux core,
// or just the mux when businessType is slotted here). It is the single seam that
// replaces server.go's hand-rolled 11-middleware nest and subsumes the W1
// buildWorkspacePath/buildCSRF/buildActionGuard dispatch.
//
// FIXED ORDER (outermost→innermost) — SECURITY-CRITICAL, MUST equal the pre-W2
// server.go hand-assembly:
//
//	SecurityHeaders → Gzip → Logger → Recovery → LoginRateLimit →
//	  Session → WorkspacePath → CSRF → ActionGuard → Timezone →
//	  businessType → inner
//
// extras (WithMiddleware additions carried on the Preset) wrap OUTSIDE the fixed
// core (reverse-applied in the tail) — never spliced between security layers.
func BuildChain(p consumermw.Preset, inner http.Handler) http.Handler {
	// ── Boot prelude: resolve the process-wide Secure-cookie policy ONCE,
	// before any cookie writer (CSRF / WorkspacePath rotation) is constructed
	// below. Secure-by-default; COOKIE_SECURE=false opts out for local http dev.
	// (W2 §5.5: moved out of the app container into the chain assembler so the
	// app no longer owns it.)
	cmw.SetSecureCookies(p.CookieSecure())

	// ── businessType slot (innermost wrapper before the mux) ────────────────
	// Pin the default business type into ctx. Retire the hand-rolled bare
	// string-key version from server.go by using the contrib middleware, which
	// writes the SAME "businessType" ctx key the downstream readers expect.
	handler := cmw.NewBusinessTypeMiddleware(p.BusinessType()).SetBusinessType(inner)

	// ── Timezone slot ───────────────────────────────────────────────────────
	// The agnostic TimezoneConfig carries two closures (GetUserID +
	// LookupTimezone). Translate them into the contrib NewTimezoneMiddleware
	// shape: the impl's UserTimezoneLookupFunc returns (string, error), so wrap
	// the agnostic (ctx,uid)->string lookup to swallow/no-op the error.
	tz := p.Timezone()
	tzMw := cmw.NewTimezoneMiddleware(
		userIDFromCtx(tz.GetUserID),
		timezoneLookup(tz.LookupTimezone),
	)
	handler = tzMw.Handle(handler)

	// ── ActionGuard slot ────────────────────────────────────────────────────
	// REUSE the W1 bridge (BuildActionGuard, middleware_http.go). Empty secret →
	// pass-through (the consumer/http boot guard fatals first for a real auth
	// provider, so the pass-through is only reached when no real provider needs
	// protection — fail-closed ordering preserved).
	handler = BuildActionGuard(p.ActionGuardConfig())(handler)

	// ── CSRF slot ───────────────────────────────────────────────────────────
	// REUSE the W1 bridge (BuildCSRF). Empty secret → opaque-token pass-through.
	handler = BuildCSRF(p.CSRFConfig())(handler)

	// ── WorkspacePath slot ──────────────────────────────────────────────────
	// REUSE the W1 bridge (BuildWorkspacePath). Sits AFTER Session, BEFORE the
	// guards so the URL-canonical workspace_id is pinned into ctx before CSRF /
	// ActionGuard read it. Preserves ErrAmbiguousBinding→picker (no auto-elect),
	// rotation rate-limit, Strict session cookie, fresh CSRF cookie on rotation.
	handler = BuildWorkspacePath(p.Workspace())(handler)

	// ── Session slot ────────────────────────────────────────────────────────
	// nil handler → pass-through (boot-time / no auth provider).
	handler = cmw.Session(sessionHandler(p.Session()))(handler)

	// ── LoginRateLimit slot ─────────────────────────────────────────────────
	// The contrib impl returns (mw, stop). In production the sweeper goroutine
	// lives for the process lifetime; the server is never torn down, so the stop
	// fn is intentionally not retained (no leak — the process exit reclaims it).
	if p.LoginRateLimitActive() {
		rlMw, _ := cmw.NewLoginRateLimitMiddlewareFromEnv(os.Getenv)
		handler = rlMw(handler)
	}

	// ── Recovery → Logger → Gzip slots (free-func impls, trivial wrap) ───────
	handler = cmw.Recovery(handler)
	handler = cmw.Logger(handler)
	handler = cmw.Gzip(handler)

	// ── SecurityHeaders slot (outermost) ────────────────────────────────────
	// Writes defense-in-depth headers + mints the per-request CSP nonce on every
	// response, including error responses. MUST be outermost. The agnostic and
	// contrib SecurityHeadersConfig are structurally identical
	// ({HSTSEnabled,CSPEnforce} bools) but distinct named types, so convert.
	handler = cmw.SecurityHeaders(cmw.SecurityHeadersConfig(p.Security()))(handler)

	// ── extras: wrap OUTSIDE the fixed core ─────────────────────────────────
	// Reverse-apply so the first-registered extra is the outermost wrapper,
	// matching the pre-W2 server.go WithMiddleware fail-safe. Extras never splice
	// between the security layers above.
	extras := p.Extras()
	for i := len(extras) - 1; i >= 0; i-- {
		if extras[i] != nil {
			handler = extras[i](handler)
		}
	}

	return handler
}

// userIDFromCtx adapts the agnostic GetUserID closure to the impl's
// UserIDFromContextFunc. A nil closure yields a func that returns "" (the impl
// then skips the timezone lookup and falls back to the default zone).
func userIDFromCtx(fn func(ctx context.Context) string) cmw.UserIDFromContextFunc {
	if fn == nil {
		return func(context.Context) string { return "" }
	}
	return cmw.UserIDFromContextFunc(fn)
}

// timezoneLookup adapts the agnostic LookupTimezone closure (ctx,uid)->string to
// the impl's UserTimezoneLookupFunc (ctx,uid)->(string,error). A nil closure
// yields nil so the impl skips resolution and uses the default zone.
func timezoneLookup(fn func(ctx context.Context, userID string) string) cmw.UserTimezoneLookupFunc {
	if fn == nil {
		return nil
	}
	return func(ctx context.Context, userID string) (string, error) {
		return fn(ctx, userID), nil
	}
}

// sessionHandler adapts the agnostic SessionHandler (carried on the Preset) to
// the impl's SessionHandler interface. The agnostic surface type is an alias of
// the shared espyna-internal SessionHandler interface; the contrib Session impl
// declares its own structurally-identical SessionHandler interface, so the value
// satisfies it directly via the interface method set. A nil handler is passed
// through (the impl treats nil as a pass-through slot).
func sessionHandler(h consumermw.SessionHandler) cmw.SessionHandler {
	if h == nil {
		return nil
	}
	return h
}
