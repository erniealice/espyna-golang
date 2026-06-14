//go:build http

// csrf.go
//
// WorkspaceCSRFMiddleware extends the base CSRF cookie-vs-header check with
// a workspace_id claim embedded in the token itself.
//
// Token format:
//
//	v1.<base64url(sessionToken|workspaceID|nonce)>.<base64url(HMAC-SHA256)>
//
// The HMAC key follows the same env-var hierarchy as action_workspace_guard:
// WORKSPACE_FORM_HMAC_KEY -> PASSWORD_AUTH_RESET_TOKEN_SECRET.
// An empty key disables claim validation (legacy opaque-token mode only).
//
// Middleware chain position:
//
//	session -> workspace_path -> CSRF -> action_workspace_guard -> timezone -> mux
//
// After workspace rotation the HTTP layer MUST call IssueWorkspaceCSRFCookie
// alongside SetSessionCookie.
package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

// --- Constants (the canonical values now live on the agnostic surface,
//     consumer/http/middleware/csrf.go — ONE source of cookie-name truth, A.2.3).
//     These package-local aliases keep the impl + its tests referencing a short
//     local name while pointing at the single agnostic definition. ---

const (
	// WorkspaceCSRFCookieName is the cookie that carries the signed CSRF token.
	WorkspaceCSRFCookieName = consumermw.WorkspaceCSRFCookieName

	// WorkspaceCSRFHeaderName is the request header that mirrors the cookie value.
	WorkspaceCSRFHeaderName = consumermw.WorkspaceCSRFHeaderName
)

// --- WorkspaceCSRFConfig ---

// WorkspaceCSRFConfig holds the middleware configuration.
type WorkspaceCSRFConfig struct {
	// Secret is the HMAC-SHA256 signing key. Use SecretFromEnv to populate.
	// When empty the middleware logs "CSRF: DISABLED" and issues/verifies
	// opaque random tokens (no workspace/session claim validation).
	Secret []byte

	// SessionToken extracts the current session token from the request.
	// Typically wired to consumer.GetSessionTokenFromContext.
	// When nil, returns "" (no session claim validation).
	SessionToken func(r *http.Request) string

	// WorkspaceID extracts the session's current workspace_id from the request.
	// Typically wired to consumer.GetWorkspaceIDFromContext.
	// When nil, returns "" (no workspace claim validation).
	WorkspaceID func(r *http.Request) string

	// PathPrefix scopes validation to mutation endpoints. Defaults to
	// "/action/" -- matches action_workspace_guard. Cookie issuance still
	// happens on every GET regardless of prefix.
	PathPrefix string
}

// NewWorkspaceCSRFMiddleware constructs the CSRF middleware. Returns nil when
// Secret is empty -- callers should skip wiring it in that case.
func NewWorkspaceCSRFMiddleware(cfg WorkspaceCSRFConfig) func(http.Handler) http.Handler {
	if cfg.SessionToken == nil {
		cfg.SessionToken = func(r *http.Request) string { return "" }
	}
	if cfg.WorkspaceID == nil {
		cfg.WorkspaceID = func(r *http.Request) string { return "" }
	}
	if cfg.PathPrefix == "" {
		cfg.PathPrefix = "/action/"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods: issue/refresh the CSRF cookie on GET so the next
			// POST has a fresh token. HEAD/OPTIONS pass through silently.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				if r.Method == http.MethodGet {
					// GET refresh (A.3.3 trigger #1): issue/refresh ws_csrf via
					// the agnostic no-tag writer. secure = this package's
					// process-wide policy (single source of truth).
					consumermw.IssueWorkspaceCSRFCookie(w, cfg.Secret,
						cfg.SessionToken(r), cfg.WorkspaceID(r), secureCookies)
				}
				next.ServeHTTP(w, r)
				return
			}

			// 0. Path scope -- only mutation endpoints under PathPrefix go
			//    through validation.
			if !strings.HasPrefix(r.URL.Path, cfg.PathPrefix) {
				next.ServeHTTP(w, r)
				return
			}

			// 0a. /action/auth/* is the LOGIN FLOW (pre-workspace-binding).
			//     No session token to claim against -- exempt.
			if strings.HasPrefix(r.URL.Path, cfg.PathPrefix+"auth/") {
				next.ServeHTTP(w, r)
				return
			}

			// 0b. /action/admin/switch-workspace is the workspace-switch
			//     handler -- by definition it crosses a workspace boundary,
			//     so a workspace-claim CSRF token would always fail.
			if r.URL.Path == cfg.PathPrefix+"admin/switch-workspace" {
				next.ServeHTTP(w, r)
				return
			}

			// Unsafe methods: validate.
			cookieTok := ""
			if c, err := r.Cookie(WorkspaceCSRFCookieName); err == nil {
				cookieTok = c.Value
			}
			if cookieTok == "" {
				writeWorkspaceCSRFError(w, "CSRF token missing from cookie")
				return
			}
			headerTok := r.Header.Get(WorkspaceCSRFHeaderName)
			if headerTok == "" {
				writeWorkspaceCSRFError(w, "CSRF token missing from header")
				return
			}
			if cookieTok != headerTok {
				writeWorkspaceCSRFError(w, "CSRF token validation failed")
				return
			}

			// Workspace+session claim check (only when Secret is configured).
			if len(cfg.Secret) > 0 {
				wantSession := cfg.SessionToken(r)
				wantWorkspace := cfg.WorkspaceID(r)
				if err := verifyWorkspaceCSRFToken(cfg.Secret, cookieTok, wantSession, wantWorkspace); err != nil {
					log.Printf("[csrf_workspace] claim mismatch path=%s: %v", r.URL.Path, err)
					writeWorkspaceCSRFError(w, fmt.Sprintf("CSRF claim mismatch: %v", err))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IssueWorkspaceCSRFCookie generates a fresh workspace-claim CSRF token and
// writes it as a Set-Cookie response header via the AGNOSTIC no-tag writer (the
// attribute source of truth, consumer/http/middleware/cookie.go). Returns the
// token so the caller can embed it in a page response header if needed. secure
// follows this package's process-wide COOKIE_SECURE policy (secureCookies).
//
// Call this alongside SetSessionCookie whenever the session is rotated.
func IssueWorkspaceCSRFCookie(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string {
	return consumermw.IssueWorkspaceCSRFCookie(w, secret, sessionToken, workspaceID, secureCookies)
}

// --- internal token helpers (forwarders to the agnostic no-tag crypto leaf,
//     consumer/http/middleware/csrf.go — A.3.0). Kept as package-local names so
//     the impl's claim check + the existing csrf_test.go reference one short
//     symbol; the crypto itself is single-sourced agnostically. ---

// issueWorkspaceCSRFToken produces a v1 workspace-claim CSRF token.
func issueWorkspaceCSRFToken(secret []byte, sessionToken, workspaceID string) string {
	return consumermw.IssueWorkspaceCSRFToken(secret, sessionToken, workspaceID)
}

// verifyWorkspaceCSRFToken parses and verifies a v1 workspace-claim CSRF token.
func verifyWorkspaceCSRFToken(secret []byte, token, wantSessionToken, wantWorkspaceID string) error {
	return consumermw.VerifyWorkspaceCSRFToken(secret, token, wantSessionToken, wantWorkspaceID)
}

func writeWorkspaceCSRFError(w http.ResponseWriter, message string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{"error":%q}`, message)
}
