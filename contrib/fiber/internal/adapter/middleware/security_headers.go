//go:build fiber

package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	// securityNonceBytes is the number of random bytes for the per-request CSP
	// nonce (128 bits). Mirrors the vanilla cspNonceBytes constant.
	securityNonceBytes = 16
)

// SecurityHeadersConfig configures the security-header middleware.
// Mirrors the vanilla SecurityHeadersConfig (consumer/http/middleware/security_headers.go).
type SecurityHeadersConfig struct {
	// HSTSEnabled emits Strict-Transport-Security when true. Leave false
	// for local HTTP dev.
	HSTSEnabled bool

	// CSPEnforce switches from Content-Security-Policy-Report-Only (false)
	// to Content-Security-Policy (true).
	CSPEnforce bool
}

// SecurityHeaders returns a Fiber middleware that sets defense-in-depth security
// response headers on every response. Must be mounted OUTERMOST in the chain
// so headers are written before any downstream handler writes the body.
//
// Mirrors the vanilla net/http reference implementation
// (consumer/http/middleware/security_headers.go): same CSP policy with
// per-request nonce, same HSTS conditional, same always-safe headers.
func SecurityHeaders(cfg ...SecurityHeadersConfig) fiber.Handler {
	var c SecurityHeadersConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	return func(ctx *fiber.Ctx) error {
		// Always-safe headers.
		ctx.Set("X-Content-Type-Options", "nosniff")
		ctx.Set("X-Frame-Options", "SAMEORIGIN")
		ctx.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		ctx.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// Per-request CSP nonce.
		nonce, ok := generateSecurityNonce()
		if !ok {
			log.Printf("[security-headers] CSPRNG failure - returning 500")
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal Server Error",
			})
		}

		csp := buildSecurityCSP(nonce)
		if c.CSPEnforce {
			ctx.Set("Content-Security-Policy", csp)
		} else {
			ctx.Set("Content-Security-Policy-Report-Only", csp)
		}

		// Reporting endpoint (both modes).
		ctx.Set("Reporting-Endpoints", `csp-endpoint="/csp-report"`)

		// HSTS (conditional).
		if c.HSTSEnabled {
			ctx.Set("Strict-Transport-Security",
				"max-age=63072000; includeSubDomains; preload")
		}

		// Store nonce on user context for downstream template rendering.
		userCtx := contextWithValue(ctx.UserContext(), ctxKeyCSPNonce, nonce)
		ctx.SetUserContext(userCtx)

		return ctx.Next()
	}
}

// CSPNonceFromContext retrieves the per-request CSP nonce from the Fiber user
// context. Returns "" if the security headers middleware did not run.
func CSPNonceFromContext(ctx *fiber.Ctx) string {
	if v, ok := ctx.UserContext().Value(ctxKeyCSPNonce).(string); ok {
		return v
	}
	return ""
}

func generateSecurityNonce() (string, bool) {
	b := make([]byte, securityNonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", false
	}
	return base64.StdEncoding.EncodeToString(b), true
}

// buildSecurityCSP constructs the Content-Security-Policy value with the given
// nonce. Mirrors the vanilla buildCSP function.
func buildSecurityCSP(nonce string) string {
	var b strings.Builder
	fmt.Fprintf(&b,
		"default-src 'self'; "+
			"script-src 'self' 'nonce-%s'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data:; "+
			"font-src 'self'; "+
			"connect-src 'self'; "+
			"frame-ancestors 'self'; "+
			"form-action 'self'; "+
			"base-uri 'self'; "+
			"report-uri /csp-report; "+
			"report-to csp-endpoint",
		nonce,
	)
	return b.String()
}
