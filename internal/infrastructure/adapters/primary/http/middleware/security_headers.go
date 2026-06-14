package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	// cspNonceBytes is the number of random bytes for the per-request CSP
	// nonce (128 bits).
	cspNonceBytes = 16

	// EnvKeyHSTSEnabled gates the Strict-Transport-Security header. OFF by
	// default; set SECURE_HSTS_ENABLED=true for production HTTPS deployments.
	EnvKeyHSTSEnabled = "SECURE_HSTS_ENABLED"

	// EnvKeyCSPEnforce gates the CSP enforce/report-only choice. OFF by default
	// (report-only); set CONFIG_SECURITY_CSP_ENFORCE=true to enforce.
	EnvKeyCSPEnforce = "CONFIG_SECURITY_CSP_ENFORCE"
)

// SecurityHeadersConfig configures the security-header middleware.
type SecurityHeadersConfig struct {
	// HSTSEnabled emits Strict-Transport-Security when true. Leave false
	// for local HTTP dev.
	HSTSEnabled bool

	// CSPEnforce switches from Content-Security-Policy-Report-Only (false)
	// to Content-Security-Policy (true).
	CSPEnforce bool
}

// SecurityHeadersConfigFromEnv builds the config from environment variables.
func SecurityHeadersConfigFromEnv(getenv func(string) string) SecurityHeadersConfig {
	v := getenv(EnvKeyHSTSEnabled)
	return SecurityHeadersConfig{
		HSTSEnabled: v == "true" || v == "1",
		CSPEnforce:  cspEnforceEnabled(getenv(EnvKeyCSPEnforce)),
	}
}

func cspEnforceEnabled(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	default:
		return false
	}
}

// SecurityHeaders returns a MiddlewareFunc that sets defense-in-depth security
// response headers on every response. Must be mounted OUTERMOST in the chain
// so headers are written before any downstream handler writes the body.
//
// When cfg is nil the middleware uses SecurityHeadersConfigFromEnv with
// os.Getenv. Pass a non-nil config for explicit control.
func SecurityHeaders(cfg ...SecurityHeadersConfig) MiddlewareFunc {
	var c SecurityHeadersConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// Always-safe headers.
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "SAMEORIGIN")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			// Per-request CSP nonce.
			nonce, ok := newNonce()
			if !ok {
				log.Printf("[security-headers] CSPRNG failure — returning 500")
				http.Error(w, "Internal Server Error",
					http.StatusInternalServerError)
				return
			}

			csp := buildCSP(nonce)
			if c.CSPEnforce {
				h.Set("Content-Security-Policy", csp)
			} else {
				h.Set("Content-Security-Policy-Report-Only", csp)
			}

			// Reporting endpoint (both modes).
			h.Set("Reporting-Endpoints", `csp-endpoint="/csp-report"`)

			// HSTS (conditional).
			if c.HSTSEnabled {
				h.Set("Strict-Transport-Security",
					"max-age=63072000; includeSubDomains; preload")
			}

			// Store nonce on context for downstream template rendering.
			ctx := contextWithNonce(r.Context(), nonce)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newNonce() (string, bool) {
	b := make([]byte, cspNonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", false
	}
	return base64.StdEncoding.EncodeToString(b), true
}

// buildCSP constructs the Content-Security-Policy value with the given nonce.
// CSPReportPath returns the sink endpoint that the report-only CSP points its
// report-uri / report-to directives at. Mount a handler here (see
// NewCSPReportHandler) so browsers don't get 404s for violation reports.
func CSPReportPath() string { return "/csp-report" }

// NewCSPReportHandler returns an http.HandlerFunc that accepts CSP violation
// reports (POST with application/csp-report or application/reports+json) and
// discards them with a 204. In the future this can be wired to a logging
// pipeline; for now it silences browser console noise.
func NewCSPReportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Consume body so the browser considers the report delivered.
		if r.Body != nil {
			_, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func buildCSP(nonce string) string {
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
