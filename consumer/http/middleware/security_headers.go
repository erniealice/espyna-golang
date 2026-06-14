package middleware

import (
	"io"
	"log"
	"net/http"
)

const (
	// EnvKeyHSTSEnabled gates the Strict-Transport-Security header.
	EnvKeyHSTSEnabled = "SECURE_HSTS_ENABLED"

	// EnvKeyCSPEnforce gates the CSP enforce/report-only choice (DEFAULT
	// report-only / OFF).
	EnvKeyCSPEnforce = "CONFIG_SECURITY_CSP_ENFORCE"

	// cspReportPath is the sink endpoint for CSP violation reports. Mounted on
	// the mux OUTSIDE the chain (server.go), so it lives on the agnostic surface.
	cspReportPath = "/csp-report"
)

// SecurityHeadersConfig configures the security-header middleware. It is the
// declarative slot the Preset carries; the contrib SecurityHeaders impl takes a
// structurally-identical type, which the chain assembler converts to. Keeping
// these two in lockstep is a contract: {HSTSEnabled, CSPEnforce}.
type SecurityHeadersConfig struct {
	// HSTSEnabled, when true, emits Strict-Transport-Security. Leave false for
	// local http dev.
	HSTSEnabled bool
	// CSPEnforce flips the CSP from REPORT-ONLY (default, false) to ENFORCING.
	CSPEnforce bool
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
	return SecurityHeadersConfig{
		HSTSEnabled: v == "true" || v == "1",
		CSPEnforce:  cspEnforceEnabled(getenv(EnvKeyCSPEnforce)),
	}
}

// CSPReportPath is the public path of the CSP violation report sink. Mounted on
// the mux directly by server.go (NOT through the chain), so it is an agnostic,
// stdlib-only helper. Kept byte-identical to the contrib impl's cspReportPath.
func CSPReportPath() string { return cspReportPath }

// NewCSPReportHandler returns the minimal CSP-report sink. Browsers POST the
// JSON violation report here; the handler logs a bounded prefix of the body and
// returns 204. Pure stdlib — mounted outside the chain, so it lives here.
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
