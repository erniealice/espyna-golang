package http

// download_cookie.go — the completion-signal seam for the client-side download
// loading indicator (pyeza download-indicator.js).
//
// A server-generated download gives the browser no event between the click and
// the first streamed byte, so the client shows a persistent progress toast and
// waits for a signal that the response has actually started. The signal is a
// short-lived per-token cookie: when a request carries a sanitized ?dltoken=<alnum>
// query param, this decorator wraps the response writer and, at header-commit
// time, sets `lf_dl_<token>=1` ONLY when the downstream response is a successful
// (2xx) attachment (Content-Disposition: attachment). Browsers apply Set-Cookie
// even for a download navigation, so the cookie lands the moment the attachment
// response headers arrive; the client polls document.cookie for its own token's
// cookie name and dismisses the toast.
//
// The per-token cookie name lets concurrent downloads acknowledge independently
// (a single shared name would let a second response overwrite the first). Gating
// on 2xx + attachment prevents a 404/403/500, redirect, or ordinary HTML page
// from falsely signalling a completed download.
//
// This is a pure net/http decorator (agnostic surface, no build tag, no contrib
// import). It wraps the catch-all handler in finalizeHTTPAdapter, so it runs
// AFTER the full fixed-order middleware chain and covers EVERY registered route
// — view routes and raw Content-Disposition download handlers alike — without a
// per-handler edit and without touching the security-critical chain order. Only
// requests that carry a valid dltoken are wrapped; all other traffic passes
// through the raw writer untouched.

import (
	"net/http"
	"strings"
)

const (
	// downloadTokenParam is the request query param carrying the client token.
	downloadTokenParam = "dltoken"
	// downloadTokenCookiePrefix prefixes the per-token response cookie name the
	// client polls for (final name is downloadTokenCookiePrefix + token).
	downloadTokenCookiePrefix = "lf_dl_"
	// downloadTokenMaxLen caps the echoed token length (defense-in-depth; the
	// client mints 24 alnum chars).
	downloadTokenMaxLen = 64
	// downloadTokenMaxAge bounds the cookie lifetime in seconds. Matches the
	// client-side 60s indicator timeout so a stale cookie never outlives the UI.
	downloadTokenMaxAge = 60
)

// withDownloadTokenCookie wraps the response writer whenever the request carries
// a sanitized dltoken query param. The wrapper sets a short-lived, non-sensitive
// per-token completion-signal cookie at header-commit time, but only for a
// successful (2xx) attachment response; every other response is untouched.
func withDownloadTokenCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := sanitizeDownloadToken(r.URL.Query().Get(downloadTokenParam))
		if tok == "" {
			// No (valid) token: pass the raw writer through so the wrapper never
			// touches non-download traffic (preserves Flusher/etc. behavior).
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(&downloadTokenWriter{ResponseWriter: w, token: tok}, r)
	})
}

// downloadTokenWriter defers the completion-signal cookie until the response
// header is committed, so the cookie is set only for a successful attachment.
type downloadTokenWriter struct {
	http.ResponseWriter
	token       string
	wroteHeader bool
}

// commitDownloadCookie sets the per-token cookie + no-store cache directive, but
// only for a 2xx attachment response. It must run BEFORE the underlying
// WriteHeader so the header block is still mutable.
func (w *downloadTokenWriter) commitDownloadCookie(status int) {
	if status < 200 || status >= 300 {
		return
	}
	if !isAttachmentDisposition(w.Header().Get("Content-Disposition")) {
		return
	}
	http.SetCookie(w.ResponseWriter, &http.Cookie{
		Name:     downloadTokenCookiePrefix + w.token,
		Value:    "1",
		Path:     "/",
		MaxAge:   downloadTokenMaxAge,
		SameSite: http.SameSiteLaxMode,
		// Deliberately NOT HttpOnly: the client JS must read this to dismiss the
		// toast. It carries no auth material — the value is a constant "1" keyed
		// by the client-minted opaque token echoed straight back.
	})
	// A token-bearing attachment is per-user and must never be served from a
	// shared cache (which could replay another request's completion cookie).
	w.Header().Set("Cache-Control", "private, no-store")
}

func (w *downloadTokenWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.commitDownloadCookie(status)
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *downloadTokenWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.commitDownloadCookie(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer so streamed downloads keep flushing.
func (w *downloadTokenWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// isAttachmentDisposition reports whether a Content-Disposition header declares
// an attachment (the disposition-type token before any parameters).
func isAttachmentDisposition(cd string) bool {
	cd = strings.TrimSpace(cd)
	if cd == "" {
		return false
	}
	if i := strings.IndexByte(cd, ';'); i >= 0 {
		cd = cd[:i]
	}
	return strings.EqualFold(strings.TrimSpace(cd), "attachment")
}

// sanitizeDownloadToken fail-closes: it returns the token only if it is a
// non-empty, length-capped, strictly-alphanumeric string; otherwise "" (no
// cookie is set). This bounds what can be echoed into Set-Cookie and forecloses
// header-injection / cookie-shape abuse via the dltoken param.
func sanitizeDownloadToken(s string) string {
	if len(s) == 0 || len(s) > downloadTokenMaxLen {
		return ""
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return ""
		}
	}
	return s
}
