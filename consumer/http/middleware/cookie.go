// Package middleware — cookie.go (AGNOSTIC surface, NO build tag).
//
// Single source of truth for the byte-level attributes of every cookie espyna's
// HTTP chain writes. These are pure BUILDER functions returning *http.Cookie;
// the caller writes them via http.SetCookie (or the WriteCookie sugar below).
//
// net/http is INTENTIONALLY allowed here: the architecture doc defines the
// agnostic surface as "pure stdlib, no build tags"
// (docs/wiki/articles/http-middleware-architecture.md:24,38) — NOT net/http-free.
// *http.Request, http.Handler, and the renderers (adapter.go) already use
// net/http in consumer/http; http.Cookie / http.SameSite are the same stdlib.
// There is deliberately NO CookieSink interface, NO SameSite translation enum,
// and NO build-tag dispatch — the builders own the attributes ONCE and every
// framework rides its own net/http adapter (gin.WrapH etc.) for the write.
//
// `secure` is an EXPLICIT bool parameter on every builder — the single source of
// truth resolved by the CALLER from the chain's Secure policy (Preset.CookieSecure()
// / SessionMiddleware.CookieSecure / the contrib secureCookies var), NOT a
// process-global read inside the builder. Default fails CLOSED to true: a caller
// that cannot resolve a policy passes true. The dev mock alone passes false to
// preserve its historical Secure-absent bytes.
//
// Each builder is BYTE-IDENTICAL to the live writer it replaces — see the
// citations on each function (and resequence.md A.2.1).
package middleware

import "net/http"

// SessionCookieSpec builds the login / dev session cookie. SameSite=Lax,
// HttpOnly. maxAge is m.CookieMaxAge (604800) at login or 86400*365 for the dev
// mock. Byte-identical to consumer/middleware_session.go:139-147 (login) and
// consumer/middleware_mock_session.go:100-107 (mock, secure=false).
func SessionCookieSpec(name, token string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// SessionRotateCookieSpec builds the URL-rotation session-upgrade cookie. The
// Lax→Strict upgrade on /w/{slug} navigation. SameSite=Strict, MaxAge=86400*7.
// Byte-identical to contrib workspace_path.go writeStrictSessionCookie.
func SessionRotateCookieSpec(name, token string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// SessionClearCookieSpec builds the INVALID-SESSION tombstone written by
// SessionMiddleware.clearSessionCookie (the middleware's "saw a bad/expired
// token" path). SameSite=**Lax**, MaxAge=-1, HttpOnly=true. Byte-identical to
// consumer/middleware_session.go:207-215. This is NOT the logout clear — keeping
// them separate avoids a Strict→Lax SameSite downgrade on logout.
func SessionClearCookieSpec(name string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// SessionLogoutClearCookieSpec builds the LOGOUT session tombstone. SameSite=
// **Strict** — the deliberate Q-SEC-3 (2026-05-31) "pin logout SameSite=Strict"
// posture (entydad service/auth/handlers.go:660-678, with its explicit security
// comment). MaxAge=-1, HttpOnly=true. KEPT DISTINCT from SessionClearCookieSpec
// (Lax): collapsing the two would downgrade logout Strict→Lax (a real security
// regression AND a byte-identity break). Byte-identical to handlers.go:670-678.
func SessionLogoutClearCookieSpec(name string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// WorkspaceCSRFCookieSpec builds the ws_csrf double-submit cookie. HttpOnly=
// FALSE (HTMX configRequest reads ws_csrf into X-Ws-Csrf-Token — load-bearing),
// SameSite=Lax, MaxAge=3600. Byte-identical to contrib csrf.go:166-178.
func WorkspaceCSRFCookieSpec(token string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:   WorkspaceCSRFCookieName,
		Value:  token,
		Path:   "/",
		MaxAge: 3600,
		// Double-submit CSRF cookie: it MUST be readable by JS so the htmx
		// configRequest hook can mirror it into the X-Ws-Csrf-Token header on
		// every non-GET request. HttpOnly would silently break that.
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// WorkspaceCSRFClearCookieSpec builds the logout ws_csrf tombstone (entydad
// service/auth/handlers.go:683-690). NOTE HttpOnly=TRUE here — the CLEAR is
// asymmetric with the ISSUE (HttpOnly=false); preserved byte-for-byte. MaxAge=-1,
// SameSite=Lax.
func WorkspaceCSRFClearCookieSpec(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     WorkspaceCSRFCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// WriteCookie is OPTIONAL sugar over http.SetCookie(w, c). Call sites may use
// either; there is no interface, no dispatch, no enum.
func WriteCookie(w http.ResponseWriter, c *http.Cookie) {
	http.SetCookie(w, c)
}
