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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- Constants ---

const (
	// WorkspaceCSRFCookieName is the cookie that carries the signed CSRF token.
	WorkspaceCSRFCookieName = "ws_csrf"

	// WorkspaceCSRFHeaderName is the request header that mirrors the cookie value.
	WorkspaceCSRFHeaderName = "X-Ws-Csrf-Token"

	// workspaceCSRFTokenVersion is the token format discriminator.
	workspaceCSRFTokenVersion = "v1"

	// workspaceCSRFNonceBytes is the per-token nonce length.
	workspaceCSRFNonceBytes = 16
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
					IssueWorkspaceCSRFCookie(w, cfg.Secret,
						cfg.SessionToken(r), cfg.WorkspaceID(r))
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
// writes it as a Set-Cookie response header. Returns the token so the caller
// can embed it in a page response header if needed.
//
// Call this alongside SetSessionCookie whenever the session is rotated.
func IssueWorkspaceCSRFCookie(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string {
	tok := issueWorkspaceCSRFToken(secret, sessionToken, workspaceID)
	http.SetCookie(w, &http.Cookie{
		Name:     WorkspaceCSRFCookieName,
		Value:    tok,
		Path:     "/",
		MaxAge:   3600,
		// Double-submit CSRF cookie: it MUST be readable by JS so the htmx
		// configRequest hook can mirror it into the X-Ws-Csrf-Token header
		// on every non-GET request. HttpOnly would silently break that.
		HttpOnly: false,
		// Secure follows the process-wide COOKIE_SECURE policy.
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	return tok
}

// --- internal token helpers ---

// issueWorkspaceCSRFToken produces a v1 workspace-claim CSRF token.
// Falls back to a random opaque token when secret is empty (legacy/disabled mode).
func issueWorkspaceCSRFToken(secret []byte, sessionToken, workspaceID string) string {
	if len(secret) == 0 {
		return generateWorkspaceCSRFToken()
	}
	nonce := make([]byte, workspaceCSRFNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		nonce = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	nonceB64 := base64.RawURLEncoding.EncodeToString(nonce)
	payload := sessionToken + "|" + workspaceID + "|" + nonceB64
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payload))
	sig := computeWorkspaceCSRFHMAC(secret, payloadB64)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return workspaceCSRFTokenVersion + "." + payloadB64 + "." + sigB64
}

// verifyWorkspaceCSRFToken parses and verifies a v1 workspace-claim CSRF token.
func verifyWorkspaceCSRFToken(secret []byte, token, wantSessionToken, wantWorkspaceID string) error {
	if len(secret) == 0 {
		return nil // no-op in disabled mode
	}
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 || parts[0] != workspaceCSRFTokenVersion {
		return fmt.Errorf("unrecognized token format")
	}
	payloadB64 := parts[1]
	sigB64 := parts[2]

	// Verify HMAC (constant-time comparison).
	expectedSig := computeWorkspaceCSRFHMAC(secret, payloadB64)
	gotSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("sig decode: %w", err)
	}
	if !hmac.Equal(expectedSig, gotSig) {
		return fmt.Errorf("HMAC verification failed")
	}

	// Decode payload: sessionToken|workspaceID|nonce
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return fmt.Errorf("payload decode: %w", err)
	}
	payloadStr := string(payloadBytes)
	firstPipe := strings.Index(payloadStr, "|")
	if firstPipe < 0 {
		return fmt.Errorf("malformed payload: missing first pipe")
	}
	rest := payloadStr[firstPipe+1:]
	secondPipe := strings.Index(rest, "|")
	if secondPipe < 0 {
		return fmt.Errorf("malformed payload: missing second pipe")
	}
	claimSession := payloadStr[:firstPipe]
	claimWorkspace := rest[:secondPipe]

	if wantSessionToken != "" && claimSession != wantSessionToken {
		return fmt.Errorf("session claim mismatch")
	}
	if wantWorkspaceID != "" && claimWorkspace != wantWorkspaceID {
		return fmt.Errorf("workspace claim mismatch")
	}
	return nil
}

func computeWorkspaceCSRFHMAC(secret []byte, payload string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func generateWorkspaceCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func writeWorkspaceCSRFError(w http.ResponseWriter, message string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{"error":%q}`, message)
}
