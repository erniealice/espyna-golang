//go:build fiber

package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// --- CSRF token format ---
//
// v1.<base64url(sessionToken|workspaceID|nonce)>.<base64url(HMAC-SHA256)>
//
// This is a faithful Fiber port of the vanilla reference implementation
// (contrib/http/internal/adapter/middleware/csrf.go). Identical security
// semantics: HMAC keyed on the deployment secret, constant-time verification,
// the same session/workspace claim checks, the same v1 token format, and the
// same failure modes (403 on missing/mismatched/invalid token). An empty
// secret disables the workspace-claim checks and falls back to opaque-token
// comparison (backward-compatible legacy mode).
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
// When Secret is empty the middleware runs in legacy opaque-token mode (no
// workspace claim validation). Mirrors the vanilla CSRFConfig.
type CSRFConfig struct {
	// Secret is the HMAC key for signing CSRF tokens. When empty, the middleware
	// skips HMAC and workspace/session checks (legacy mode).
	Secret []byte

	// SessionToken extracts the current session token from the request.
	// This is the opaque cookie value placed by the session middleware.
	SessionToken func(c *fiber.Ctx) string

	// WorkspaceID extracts the session's current workspace_id from the request.
	// Defaults to a no-op (empty string — workspace claim not checked).
	WorkspaceID func(c *fiber.Ctx) string
}

// NewCSRFMiddleware builds the CSRF middleware with optional workspace-claim
// support. When cfg.Secret is empty the middleware is a lightweight opaque-token
// checker (backward-compatible with the legacy CSRF function).
func NewCSRFMiddleware(cfg CSRFConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()

		// Safe methods: generate/refresh token on GET; skip verification.
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			if method == fiber.MethodGet {
				sessionTok := ""
				workspaceID := ""
				if cfg.SessionToken != nil {
					sessionTok = cfg.SessionToken(c)
				}
				if cfg.WorkspaceID != nil {
					workspaceID = cfg.WorkspaceID(c)
				}
				token := issueCSRFToken(cfg.Secret, sessionTok, workspaceID)
				c.Cookie(&fiber.Cookie{
					Name:     csrfCookieName,
					Value:    token,
					Path:     "/",
					MaxAge:   3600,
					HTTPOnly: true,
					SameSite: "Lax",
				})
				c.Set(csrfHeaderName, token)
			}
			return c.Next()
		}

		// Unsafe methods: validate.
		cookieTok := c.Cookies(csrfCookieName)
		if cookieTok == "" {
			return writeCSRFError(c, "CSRF token missing from cookie")
		}
		headerTok := c.Get(csrfHeaderName)
		if headerTok == "" {
			return writeCSRFError(c, "CSRF token missing from header")
		}

		// Cookie/header equality check (both old and new format).
		if cookieTok != headerTok {
			return writeCSRFError(c, "CSRF token validation failed")
		}

		// Extended workspace-claim check (only when Secret is configured).
		if len(cfg.Secret) > 0 {
			sessionTok := ""
			workspaceID := ""
			if cfg.SessionToken != nil {
				sessionTok = cfg.SessionToken(c)
			}
			if cfg.WorkspaceID != nil {
				workspaceID = cfg.WorkspaceID(c)
			}
			if err := verifyCSRFToken(cfg.Secret, cookieTok, sessionTok, workspaceID); err != nil {
				return writeCSRFError(c, fmt.Sprintf("CSRF claim mismatch: %v", err))
			}
		}

		return c.Next()
	}
}

// CSRF is the legacy opaque-token CSRF middleware (no workspace claim).
// Kept for parity with the vanilla CSRF function and the gin CSRF helper.
// Prefer NewCSRFMiddleware(CSRFConfig{Secret: ...}) for new wiring.
func CSRF() fiber.Handler {
	return NewCSRFMiddleware(CSRFConfig{})
}

// --- internal helpers (faithful copies of the vanilla implementation) ---

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

// writeCSRFError writes a standardized CSRF error response (403), mirroring the
// vanilla JSON error shape {"success": false, "error": message}.
func writeCSRFError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}
