// Package middleware — csrf.go (AGNOSTIC surface, NO build tag).
//
// Framework-independent contract for the workspace-claim CSRF middleware, PLUS
// the workspace-claim CSRF token CRYPTO (HMAC-SHA256) and the agnostic cookie
// writer. The token crypto has ZERO net/http dependency — only crypto/* and
// encoding/base64 — so it lives here as a no-tag agnostic LEAF (the only reason
// it previously lived behind //go:build http was file placement). The net/http
// MIDDLEWARE BODY still lives in contrib/http/internal/adapter/middleware/csrf.go
// (//go:build http) and is assembled through provider.BuildCSRF; this file owns
// the config type, the token crypto, and the cookie writer.
//
// net/http is allowed here (doc-legal: the agnostic surface is "pure stdlib, no
// build tags" — http-middleware-architecture.md:24,38). There is NO build-tag
// dispatch and NO "" stub for the token path: IssueWorkspaceCSRFCookie computes
// the token inline and writes via http.SetCookie + the WorkspaceCSRFCookieSpec
// builder, so the workspace-CSRF cookie is written under EVERY build (fail-loud).
package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// --- Constants (relocated from contrib/http/internal/adapter/middleware/csrf.go
//     so the agnostic cookie builders + every framework reference one source) ---

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

// IssueWorkspaceCSRFCookie computes the workspace-claim HMAC token agnostically
// and writes the ws_csrf cookie via http.SetCookie + WorkspaceCSRFCookieSpec.
// Returns the token. There is NO "" return and NO build-tag dispatch — the
// crypto is right here, so the cookie is written under any build (fail-loud).
//
// Call this alongside the session cookie whenever the session is rotated, on the
// GET refresh, and on the auth login / sidebar-switch paths. `secure` is the
// caller-resolved COOKIE_SECURE policy (single source of truth — see cookie.go).
//
// Token format: v1.<base64url(sessionToken|workspaceID|nonce)>.<base64url(HMAC)>.
// An empty secret falls back to a random opaque token (claim validation disabled);
// the non-mock empty-secret BOOT guard in consumer/http/server.go fatals before a
// production binary can reach that fallback.
func IssueWorkspaceCSRFCookie(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string, secure bool) string {
	tok := IssueWorkspaceCSRFToken(secret, sessionToken, workspaceID)
	http.SetCookie(w, WorkspaceCSRFCookieSpec(tok, secure))
	return tok
}

// IssueWorkspaceCSRFToken produces a v1 workspace-claim CSRF token. Falls back to
// a random opaque token when secret is empty (legacy/disabled mode). No net/http.
func IssueWorkspaceCSRFToken(secret []byte, sessionToken, workspaceID string) string {
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

// VerifyWorkspaceCSRFToken parses and verifies a v1 workspace-claim CSRF token.
// No net/http.
func VerifyWorkspaceCSRFToken(secret []byte, token, wantSessionToken, wantWorkspaceID string) error {
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

// CSRFConfig configures the workspace-claim CSRF middleware.
type CSRFConfig struct {
	// Secret is the HMAC-SHA256 signing key for workspace-claim CSRF tokens.
	// Use SecretFromEnv to populate. When empty the impl issues/verifies
	// opaque random tokens (no workspace/session claim validation).
	Secret []byte

	// SessionToken extracts the current session token from the request.
	// Typically wired to consumer.GetSessionTokenFromContext. When nil the
	// impl skips the session claim.
	SessionToken func(r *http.Request) string

	// WorkspaceID extracts the session's current workspace_id from the request.
	// Typically wired to consumer.GetWorkspaceIDFromContext. When nil the impl
	// skips the workspace claim.
	WorkspaceID func(r *http.Request) string

	// PathPrefix scopes validation to mutation endpoints. Defaults to
	// "/action/". Cookie issuance still happens on every GET regardless.
	PathPrefix string
}

// CSRF returns a MiddlewareFunc that validates workspace-claim CSRF tokens on
// mutating requests and refreshes the double-submit cookie on GET.
//
// This agnostic entry point is a pass-through; consumer/http overrides it via
// the build-tagged buildCSRF when the `http` server provider is compiled in.
func CSRF(cfg CSRFConfig) MiddlewareFunc {
	return func(next http.Handler) http.Handler { return next }
}
