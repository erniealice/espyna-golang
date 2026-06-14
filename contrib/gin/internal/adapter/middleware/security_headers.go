//go:build gin

package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Security-header middleware for Gin. Faithfully mirrors the vanilla net/http
// implementation (apps/service-admin/internal/infrastructure/input/http/
// middleware/security_headers.go): same headers, same CSP nonce flow, same
// HSTS/CSP-enforce gating, same CSPRNG-failure → 500 behavior.
//
// This middleware must be mounted OUTERMOST in the chain so headers are set
// before any inner handler writes the body.

const (
	// EnvKeyHSTSEnabled gates the Strict-Transport-Security header. HSTS is OFF
	// unless SECURE_HSTS_ENABLED is explicitly "true"/"1" — sending HSTS over
	// plain http would brick local development.
	EnvKeyHSTSEnabled = "SECURE_HSTS_ENABLED"

	// EnvKeyCSPEnforce gates the CSP enforce/report-only choice. DEFAULT is
	// REPORT-ONLY (OFF). Same shape as AUTHZ_ENFORCE / SCHEMA_BOOTSHOT_ENFORCE.
	EnvKeyCSPEnforce = "CONFIG_SECURITY_CSP_ENFORCE"

	cspReportPath    = "/csp-report"
	cspReportToGroup = "csp-endpoint"
	cspNonceBytes    = 16
)

// SecurityHeadersConfig configures the security-header middleware.
type SecurityHeadersConfig struct {
	// HSTSEnabled, when true, emits Strict-Transport-Security. Leave false for
	// local http dev (HSTS over http bricks the browser).
	HSTSEnabled bool

	// CSPEnforce flips the CSP from REPORT-ONLY (default, false) to ENFORCING
	// (true). Default is report-only: violations gathered at /csp-report but
	// never blocked.
	CSPEnforce bool
}

// SecurityHeadersConfigFromEnv builds the config from environment variables.
// HSTS is OFF by default; CSP enforce is OFF by default (report-only).
func SecurityHeadersConfigFromEnv(getenv func(string) string) SecurityHeadersConfig {
	v := getenv(EnvKeyHSTSEnabled)
	return SecurityHeadersConfig{
		HSTSEnabled: v == "true" || v == "1",
		CSPEnforce:  cspEnforceEnabled(getenv(EnvKeyCSPEnforce)),
	}
}

// cspEnforceEnabled returns true only for an explicit truthy value. Anything
// else keeps the safe REPORT-ONLY posture.
func cspEnforceEnabled(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	default:
		return false
	}
}

// buildCSP builds the Content-Security-Policy value for a single request,
// appending the per-request nonce to script-src.
func buildCSP(nonce string) string {
	scriptSrc := "script-src 'self' 'nonce-" + nonce + "'"
	return "default-src 'self'; " +
		scriptSrc + "; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self'; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"object-src 'none'; " +
		"report-uri " + cspReportPath + "; " +
		"report-to " + cspReportToGroup
}

// newCSPNonce returns a fresh base64-encoded 128-bit random nonce.
// Returns (nonce, ok); ok is false only if the OS CSPRNG fails.
func newCSPNonce() (string, bool) {
	b := make([]byte, cspNonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", false
	}
	return base64.StdEncoding.EncodeToString(b), true
}

// SecurityHeaders returns a Gin middleware that sets defense-in-depth response
// headers on every response. Must be mounted OUTERMOST in the chain so headers
// are set before any inner handler writes the body.
//
// Always-enforced:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: SAMEORIGIN
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: camera=(), microphone=(), geolocation=()
//   - Content-Security-Policy (or -Report-Only): with per-request nonce
//   - Reporting-Endpoints: for CSP report-to target
//
// Conditional:
//   - Strict-Transport-Security: only when cfg.HSTSEnabled (https/prod)
//
// The per-request CSP nonce is stored on the gin.Context under the key "csp_nonce"
// so downstream handlers/templates can read it with c.GetString("csp_nonce") or
// via the request context.
func SecurityHeaders(cfg SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Always-safe headers.
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// Mint a per-request CSP nonce.
		nonce, ok := newCSPNonce()
		if !ok {
			log.Printf("[security-headers] CSPRNG failure: crypto/rand.Read returned an error; responding 500")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Stash the nonce on the gin context for downstream handlers/templates.
		c.Set("csp_nonce", nonce)

		// CSP enforce/report-only gated behind config flag.
		if cfg.CSPEnforce {
			c.Header("Content-Security-Policy", buildCSP(nonce))
		} else {
			c.Header("Content-Security-Policy-Report-Only", buildCSP(nonce))
		}
		// Reporting endpoint for both modes.
		c.Header("Reporting-Endpoints", cspReportToGroup+`="`+cspReportPath+`"`)

		// HSTS only over https / prod.
		if cfg.HSTSEnabled {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// CSPReportPath returns the public path of the CSP violation report sink.
func CSPReportPath() string { return cspReportPath }

// NewCSPReportHandler returns a gin.HandlerFunc that accepts CSP violation
// reports (POST only, JSON body capped at 16 KiB, returns 204).
func NewCSPReportHandler() gin.HandlerFunc {
	const maxReportBytes = 16 << 10 // 16 KiB
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Status(http.StatusMethodNotAllowed)
			return
		}
		body := make([]byte, maxReportBytes)
		n, _ := c.Request.Body.Read(body)
		if n > 0 {
			log.Printf("[csp-report] %s", body[:n])
		}
		c.Status(http.StatusNoContent)
	}
}
