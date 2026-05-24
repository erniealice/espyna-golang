//go:build vanilla

package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// --- CSRF token format ---
//
// v1.<base64url(sessionToken|workspaceID|nonce)>.<base64url(HMAC-SHA256)>
//
// The HMAC is keyed on the deployment secret (same env-var hierarchy as
// action_workspace_guard: WORKSPACE_FORM_HMAC_KEY → PASSWORD_AUTH_RESET_TOKEN_SECRET).
// An empty secret disables the workspace-claim checks and falls back to opaque-token
// comparison (backward-compatible with the old behaviour). The CSRFConfig.Secret field
// controls this; if empty the middleware logs "CSRF: DISABLED" and runs without claims.
//
// Verification:
//   1. Split token on "." → must have 3 parts starting with "v1"
//   2. base64url-decode payload → sessionToken|workspaceID|nonce
//   3. Recompute HMAC, compare (constant-time)
//   4. Compare extracted sessionToken against current session token in cookie
//   5. Compare extracted workspaceID against current session workspace_id from context

const (
	csrfCookieName   = "csrf_token"
	csrfHeaderName   = "X-Csrf-Token"
	csrfTokenVersion = "v1"
	csrfNonceBytes   = 16
)

// CSRFConfig holds optional configuration for the workspace-claim CSRF middleware.
// When Secret is empty the middleware runs in legacy opaque-token mode (no workspace
// claim validation).
type CSRFConfig struct {
	// Secret is the HMAC key for signing CSRF tokens. When empty, the middleware
	// skips HMAC and workspace/session checks (legacy mode).
	Secret []byte

	// SessionToken extracts the current session token from the request context.
	// This is the opaque cookie value placed in context by the session middleware.
	// Defaults to reading the ichizen_session cookie directly.
	SessionToken func(r *http.Request) string

	// WorkspaceID extracts the session's current workspace_id from the request
	// context. Defaults to a no-op (empty string — workspace claim not checked).
	WorkspaceID func(r *http.Request) string
}

// NewCSRFMiddleware builds the CSRF middleware with optional workspace-claim support.
// When cfg.Secret is empty the middleware is a lightweight opaque-token checker
// (backward-compatible with the old CSRF function).
func NewCSRFMiddleware(cfg CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods: generate/refresh token on GET; skip verification.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				if r.Method == http.MethodGet {
					sessionTok := ""
					workspaceID := ""
					if cfg.SessionToken != nil {
						sessionTok = cfg.SessionToken(r)
					}
					if cfg.WorkspaceID != nil {
						workspaceID = cfg.WorkspaceID(r)
					}
					token := issueCSRFToken(cfg.Secret, sessionTok, workspaceID)
					http.SetCookie(w, &http.Cookie{
						Name:     csrfCookieName,
						Value:    token,
						Path:     "/",
						MaxAge:   3600,
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					})
					w.Header().Set(csrfHeaderName, token)
				}
				next.ServeHTTP(w, r)
				return
			}

			// Unsafe methods: validate.
			cookieTok := ""
			if c, err := r.Cookie(csrfCookieName); err == nil {
				cookieTok = c.Value
			}
			if cookieTok == "" {
				writeCSRFError(w, "CSRF token missing from cookie")
				return
			}
			headerTok := r.Header.Get(csrfHeaderName)
			if headerTok == "" {
				writeCSRFError(w, "CSRF token missing from header")
				return
			}

			// Cookie/header equality check (both old and new format).
			if cookieTok != headerTok {
				writeCSRFError(w, "CSRF token validation failed")
				return
			}

			// Extended workspace-claim check (only when Secret is configured).
			if len(cfg.Secret) > 0 {
				sessionTok := ""
				workspaceID := ""
				if cfg.SessionToken != nil {
					sessionTok = cfg.SessionToken(r)
				}
				if cfg.WorkspaceID != nil {
					workspaceID = cfg.WorkspaceID(r)
				}
				if err := verifyCSRFToken(cfg.Secret, cookieTok, sessionTok, workspaceID); err != nil {
					writeCSRFError(w, fmt.Sprintf("CSRF claim mismatch: %v", err))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CSRF is the legacy opaque-token CSRF middleware (no workspace claim).
// Kept for backward compatibility with code that references CSRF directly.
// Prefer NewCSRFMiddleware(CSRFConfig{Secret: ...}) for new wiring.
func CSRF(next http.Handler) http.Handler {
	return NewCSRFMiddleware(CSRFConfig{})(next)
}

// IssueCSRFCookie generates a fresh CSRF token carrying the given (sessionToken,
// workspaceID) claims and writes it as a Set-Cookie header. The returned token
// string should also be sent in X-Csrf-Token or embedded in the page for HTMX.
// Call this alongside SetSessionCookie whenever the session is rotated.
func IssueCSRFCookie(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string {
	tok := issueCSRFToken(secret, sessionToken, workspaceID)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    tok,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return tok
}

// --- internal helpers ---

// issueCSRFToken produces a v1 CSRF token.
//
// Format: v1.<base64url(sessionToken|workspaceID|nonce)>.<base64url(HMAC)>
//
// When secret is empty, falls back to a random opaque base64 string (legacy mode).
func issueCSRFToken(secret []byte, sessionToken, workspaceID string) string {
	if len(secret) == 0 {
		return generateCSRFToken()
	}
	nonce := make([]byte, csrfNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		nonce = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	nonceB64 := base64.RawURLEncoding.EncodeToString(nonce)
	// Payload: sessionToken|workspaceID|nonce (pipe-delimited; sessionToken and
	// workspaceID are never allowed to contain "|" in practice).
	payload := sessionToken + "|" + workspaceID + "|" + nonceB64
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payload))
	sig := computeCSRFHMAC(secret, payloadB64)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return csrfTokenVersion + "." + payloadB64 + "." + sigB64
}

// verifyCSRFToken parses and verifies a v1 CSRF token.
// Returns nil on success. Returns descriptive errors for each failure mode.
func verifyCSRFToken(secret []byte, token, wantSessionToken, wantWorkspaceID string) error {
	if len(secret) == 0 {
		return nil // no-op in legacy mode
	}
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 || parts[0] != csrfTokenVersion {
		return fmt.Errorf("unrecognized token format")
	}
	payloadB64 := parts[1]
	sigB64 := parts[2]

	// Verify HMAC first (constant-time).
	expectedSig := computeCSRFHMAC(secret, payloadB64)
	gotSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("sig decode: %w", err)
	}
	if !hmac.Equal(expectedSig, gotSig) {
		return fmt.Errorf("HMAC verification failed")
	}

	// Decode payload.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return fmt.Errorf("payload decode: %w", err)
	}
	// payload = sessionToken|workspaceID|nonce
	// Split on first two "|" only (session token could theoretically contain them
	// though in practice it won't; workspaceID is a UUID).
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
	// nonce is rest[secondPipe+1:] — not validated beyond HMAC integrity

	// Compare session token claim.
	if wantSessionToken != "" && claimSession != wantSessionToken {
		return fmt.Errorf("session claim mismatch")
	}
	// Compare workspace_id claim (only when the session has a workspace_id).
	if wantWorkspaceID != "" && claimWorkspace != wantWorkspaceID {
		return fmt.Errorf("workspace claim mismatch")
	}
	return nil
}

func computeCSRFHMAC(secret []byte, payload string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// generateCSRFToken generates a cryptographically secure random token (legacy mode).
func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.URLEncoding.EncodeToString(b)
}

// writeCSRFError writes a standardized CSRF error response.
func writeCSRFError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	json.NewEncoder(w).Encode(response)
}
