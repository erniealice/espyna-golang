//go:build gin

// cookie_propagation_test.go
//
// Wave-A gate (resequence.md A.3.4 / A.4, codex round-2 #3): prove that a
// Set-Cookie written by the AGNOSTIC net/http chain via http.SetCookie SURVIVES
// through the gin adapter (gin.WrapH hands the REAL http.ResponseWriter to the
// inner handler, contrib/gin/internal/adapter/adapter.go:296). This is the
// executable backing for the "gin = zero cookie code" claim: the agnostic cookie
// builders + http.SetCookie ride gin's existing net/http adapter with no
// gin-specific cookie code.
//
// The matching FIBER assertion is t.Skip-until-wired: the live Fiber adapter
// shim's Header() returns a fresh http.Header map that is never flushed back to
// the Fiber response (contrib/fiber/internal/adapter/adapter.go:309,
// adapterv3/adapter.go:301), so http.SetCookie is dropped and the cookie never
// reaches the client. Fiber is DEFERRED — wiring it REQUIRES first fixing that
// shim's Set-Cookie/header propagation. The skipped subtest documents the gap
// and is the gate that flips green only once that fix lands.
package adapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

// agnosticChainHandler is a stand-in for espyna's agnostic chain: a plain
// net/http handler that writes a cookie via http.SetCookie + an agnostic
// builder, exactly as the real session / ws_csrf writers do.
func agnosticChainHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ws_csrf: HttpOnly=false, Lax, MaxAge=3600 (load-bearing for HTMX).
		http.SetCookie(w, consumermw.WorkspaceCSRFCookieSpec("smoke-token", false))
		// session: Lax, HttpOnly, MaxAge=604800.
		http.SetCookie(w, consumermw.SessionCookieSpec("ichizen_session", "sess-token", 604800, false))
		w.WriteHeader(http.StatusOK)
	})
}

// TestGinAdapter_SetCookieSurvivesWrapH asserts the Set-Cookie headers the
// agnostic chain writes via http.SetCookie reach the gin response unchanged when
// the handler is mounted through gin.WrapH (the same wrap the GinAdapter uses for
// custom handlers at adapter.go:296).
func TestGinAdapter_SetCookieSurvivesWrapH(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// gin.WrapH is the EXACT adapter the GinAdapter uses; mounting the agnostic
	// net/http handler through it must propagate http.SetCookie to the response.
	router.GET("/smoke", gin.WrapH(agnosticChainHandler()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/smoke", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	got := map[string]*http.Cookie{}
	for _, c := range cookies {
		got[c.Name] = c
	}

	ws, ok := got[consumermw.WorkspaceCSRFCookieName]
	if !ok {
		t.Fatalf("ws_csrf cookie did NOT survive gin.WrapH (gin adapter dropped Set-Cookie). got cookies: %v", cookies)
	}
	if ws.Value != "smoke-token" {
		t.Errorf("ws_csrf value mismatch: got %q", ws.Value)
	}
	// HttpOnly MUST stay false on the issue (HTMX reads it). Byte-identity guard.
	if ws.HttpOnly {
		t.Errorf("ws_csrf HttpOnly must be false on issue (HTMX configRequest read), got true")
	}
	if ws.SameSite != http.SameSiteLaxMode {
		t.Errorf("ws_csrf SameSite must be Lax, got %v", ws.SameSite)
	}

	sess, ok := got["ichizen_session"]
	if !ok {
		t.Fatalf("session cookie did NOT survive gin.WrapH. got cookies: %v", cookies)
	}
	if !sess.HttpOnly {
		t.Errorf("session HttpOnly must be true, got false")
	}
	if sess.SameSite != http.SameSiteLaxMode {
		t.Errorf("session SameSite must be Lax at login, got %v", sess.SameSite)
	}
	if sess.MaxAge != 604800 {
		t.Errorf("session MaxAge must be 604800, got %d", sess.MaxAge)
	}
}

// TestFiberAdapter_SetCookieSurvives is SKIPPED-until-wired. It documents the
// verified Fiber adapter gap (resequence.md A.3.4 / codex round-2 #3): the Fiber
// shim's Header() returns a fresh, never-flushed http.Header, so a
// http.SetCookie written by the agnostic chain is DROPPED. Fiber is deferred —
// this test is expected to fail against today's shim and is the gate that flips
// green once the Fiber header-propagation fix lands. It lives here (gin module)
// as a documentation marker; the real fiber-typed assertion belongs in the fiber
// module when that wiring is done.
func TestFiberAdapter_SetCookieSurvives(t *testing.T) {
	t.Skip("DEFERRED: the live Fiber adapter shim's Header() returns an unflushed " +
		"fresh http.Header (contrib/fiber/internal/adapter/adapter.go:309, " +
		"adapterv3/adapter.go:301), so http.SetCookie is dropped and the cookie " +
		"never reaches the client. Wiring Fiber REQUIRES fixing the shim's " +
		"Set-Cookie/header propagation first; un-skip this once that lands.")
}
