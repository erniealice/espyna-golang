//go:build http

// security_headers.go
//
// Security-header middleware: CSP (with per-request nonce), X-Frame-Options,
// HSTS, and other defense-in-depth response headers.
//
// CSP MODE IS GATED: the enforce/report-only choice is read from
// CONFIG_SECURITY_CSP_ENFORCE, DEFAULT REPORT-ONLY (OFF). This mirrors the
// sibling AUTHZ_ENFORCE / SCHEMA_BOOTSHOT_ENFORCE flags.
//
// Per-request CSP nonce: SecurityHeaders is the natural place to mint the
// per-request nonce (outermost middleware). It generates a fresh 128-bit
// base64 nonce, stores it on r.Context() via render.WithNonce (pyeza render is
// the canonical home of the nonce ctx-key — espyna writes, pyeza reads), writes
// the header, and threads the enriched context inward so templates can read it
// back via render.NonceFromContext in the render pipeline.
package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"net/http"

	"github.com/erniealice/pyeza-golang/render"
)

// --- Nonce context key ---
//
// The CSP nonce ctx-key is owned by pyeza render (render.WithNonce /
// render.NonceFromContext), the canonical reader in the render pipeline. The
// espyna→pyeza import arrow makes render the single source of this key: this
// writer (SecurityHeaders) writes render's key; the render pipeline reads it.
// Do NOT re-declare a private nonce key here — that fork is exactly the
// silent ctx-key drift the leaf convergence (W3) eliminated.

// --- Constants ---

const (
	// EnvKeyHSTSEnabled gates the Strict-Transport-Security header. HSTS MUST
	// only be sent over https / in production. Dev opt-out: OFF unless
	// SECURE_HSTS_ENABLED is explicitly set to "true"/"1".
	EnvKeyHSTSEnabled = "SECURE_HSTS_ENABLED"

	// cspReportPath is the sink endpoint for CSP violation reports.
	cspReportPath = "/csp-report"

	// cspReportToGroup is the named reporting group used by the modern
	// report-to directive and the Reporting-Endpoints header.
	cspReportToGroup = "csp-endpoint"

	// EnvKeyCSPEnforce gates the CSP enforce/report-only choice. DEFAULT is
	// REPORT-ONLY (OFF). Only explicit truthy values flip to ENFORCING.
	EnvKeyCSPEnforce = "CONFIG_SECURITY_CSP_ENFORCE"

	// cspNonceBytes is the entropy of the per-request CSP nonce before base64
	// encoding. 16 bytes (128 bits) is the OWASP-recommended floor.
	cspNonceBytes = 16
)

// buildCSP builds the Content-Security-Policy value for a single request,
// appending the per-request nonce to the script-src directive.
//
// fbAuthDomain, when non-empty (firebase auth provider), widens script-src /
// connect-src / frame-src to permit the Firebase JS SDK (gstatic) and the
// Identity-Platform sign-in popup/iframe (the authDomain + the Google/Microsoft
// OAuth endpoints). Empty = no Firebase widening (password/mock builds keep the
// strict baseline). Additive only — every other directive is unchanged.
func buildCSP(nonce, fbAuthDomain string) string {
	scriptSrc := "script-src 'self' 'nonce-" + nonce + "'"
	connectSrc := "connect-src 'self'"
	frameSrc := "frame-src 'self'"
	if fbAuthDomain != "" {
		// Firebase JS SDK host + Identity Platform sign-in (popup uses a hidden
		// iframe to <authDomain>/__/auth/iframe; the federated OAuth providers
		// for Microsoft/Google live on their own origins).
		scriptSrc += " https://www.gstatic.com https://apis.google.com"
		connectSrc += " https://identitytoolkit.googleapis.com https://securetoken.googleapis.com https://www.googleapis.com https://" + fbAuthDomain
		frameSrc += " https://" + fbAuthDomain + " https://apis.google.com https://accounts.google.com https://login.microsoftonline.com https://login.live.com"
	}
	return "default-src 'self'; " +
		scriptSrc + "; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self'; " +
		connectSrc + "; " +
		frameSrc + "; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"object-src 'none'; " +
		"report-uri " + cspReportPath + "; " +
		"report-to " + cspReportToGroup
}

// newNonce returns a fresh base64-encoded 128-bit random nonce. The two return
// values are (nonce, ok); ok is false only if the OS CSPRNG fails.
func newNonce() (string, bool) {
	b := make([]byte, cspNonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", false
	}
	return base64.StdEncoding.EncodeToString(b), true
}

// SecurityHeadersConfig configures the security-header middleware.
type SecurityHeadersConfig struct {
	// HSTSEnabled, when true, emits Strict-Transport-Security. Leave false for
	// local http dev. Resolve from the environment via SecurityHeadersConfigFromEnv.
	HSTSEnabled bool

	// CSPEnforce flips the CSP from REPORT-ONLY (default, false) to ENFORCING
	// (true). DEFAULT IS REPORT-ONLY.
	CSPEnforce bool

	// FirebaseAuthDomain, when non-empty, widens the CSP to permit the Firebase
	// JS SDK + sign-in popup/iframe. Resolved from FIREBASE_AUTH_DOMAIN only when
	// CONFIG_AUTH_PROVIDER=firebase. Empty for password/mock builds.
	FirebaseAuthDomain string
}

// cspEnforceEnabled returns true only for an explicit truthy value.
func cspEnforceEnabled(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	default:
		return false
	}
}

// SecurityHeadersConfigFromEnv builds the config from environment variables.
// HSTS is OFF by default. CSP is REPORT-ONLY by default.
func SecurityHeadersConfigFromEnv(getenv func(string) string) SecurityHeadersConfig {
	v := getenv(EnvKeyHSTSEnabled)
	fbAuthDomain := ""
	if getenv("CONFIG_AUTH_PROVIDER") == "firebase" {
		fbAuthDomain = getenv("FIREBASE_AUTH_DOMAIN")
	}
	return SecurityHeadersConfig{
		HSTSEnabled:        v == "true" || v == "1",
		CSPEnforce:         cspEnforceEnabled(getenv(EnvKeyCSPEnforce)),
		FirebaseAuthDomain: fbAuthDomain,
	}
}

// SecurityHeaders returns middleware that sets defense-in-depth response headers
// on EVERY response. Must be mounted OUTERMOST in the chain, ahead of
// Gzip/Logger/Recovery, so the headers are written before any inner handler
// writes the body.
//
// Always-enforced:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: SAMEORIGIN
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: camera=(), microphone=(), geolocation=()
//   - Content-Security-Policy (or Report-Only): per-request nonce
//   - Reporting-Endpoints: csp-endpoint="/csp-report"
//
// Conditional:
//   - Strict-Transport-Security -- only when cfg.HSTSEnabled (https/prod)
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// Always-safe headers (enforced now).
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "SAMEORIGIN")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			// Mint a per-request nonce and stash it on the context.
			nonce, ok := newNonce()
			if !ok {
				log.Printf("[security-headers] CSPRNG failure: crypto/rand.Read returned an error; responding 500")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			ctx := render.WithNonce(r.Context(), nonce)

			// CSP enforce/report-only gated behind CONFIG_SECURITY_CSP_ENFORCE.
			if cfg.CSPEnforce {
				h.Set("Content-Security-Policy", buildCSP(nonce, cfg.FirebaseAuthDomain))
			} else {
				h.Set("Content-Security-Policy-Report-Only", buildCSP(nonce, cfg.FirebaseAuthDomain))
			}
			// Declare the named reporting endpoint (both modes).
			h.Set("Reporting-Endpoints", cspReportToGroup+`="`+cspReportPath+`"`)

			// HSTS only over https / prod.
			if cfg.HSTSEnabled {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSPReportPath is the public path of the CSP violation report sink. Exported so
// the composition layer can register the handler on the mux.
func CSPReportPath() string { return cspReportPath }

// NewCSPReportHandler returns the minimal CSP-report sink. Browsers POST the
// JSON violation report here; the handler logs a bounded prefix of the body and
// returns 204.
func NewCSPReportHandler() http.HandlerFunc {
	const maxReportBytes = 16 << 10 // 16 KiB
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(io.LimitReader(r.Body, maxReportBytes))
		if len(body) > 0 {
			log.Printf("[csp-report] %s", body)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
