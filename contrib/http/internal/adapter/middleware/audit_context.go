//go:build http

package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"

	"uuid"

	infraports "github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// AuditContextMiddleware extracts actor metadata from the request and stores it
// in context for downstream audit logging. Must run AFTER authentication middleware
// so that the "uid" key is already present in context.
func AuditContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Actor ID from auth context (set by authentication middleware at line ~74)
		var actorID, actorType string
		if id, ok := identity.FromContext(r.Context()); ok && id.UserID != "" {
			actorID = id.UserID
			actorType = "user"
		} else {
			actorID = "system"
			actorType = "system"
		}

		// Request ID: use incoming header or generate one
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewV7().String()
		}

		ac := infraports.AuditContext{
			ActorID:   actorID,
			ActorType: actorType,
			IP:        clientIP(r),
			UserAgent: r.Header.Get("User-Agent"),
			RequestID: requestID,
		}

		ctx := infraports.WithAuditContext(r.Context(), ac)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// clientIP resolves the forensic client IP: the first X-Forwarded-For entry
// when it parses as an IP, else the socket peer. EVERY candidate is validated
// (normalizeIP) before use — the audit sink's column is PostgreSQL inet and
// audit inserts are tx-fatal, so an unparseable value (bracketed, port-bearing
// XFF, garbage header) must degrade to the socket peer, and failing that to ""
// (the adapter stores NULL) — never fail the audited transaction over IP
// cosmetics.
//
// KNOWN LIMITATION: when XFF parses, it is trusted without a trusted-proxy
// allowlist, so a direct client can spoof its forensic IP (the socket peer is
// not cross-checked). Closing that needs an ingress-policy config surface
// (trusted proxy CIDRs + resolve-from-the-trusted-end), tracked as a follow-up
// — not silently half-fixed here.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := xff
		if i := strings.Index(first, ","); i >= 0 {
			first = first[:i]
		}
		if ip := normalizeIP(strings.TrimSpace(first)); ip != "" {
			return ip
		}
		// Malformed XFF: fall through to the socket peer rather than record
		// garbage or fail the write.
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip := normalizeIP(host); ip != "" {
			return ip
		}
	}
	return normalizeIP(r.RemoteAddr)
}

// normalizeIP canonicalizes a candidate that may be a bare IP ("::1",
// "10.0.0.1"), a bracketed IPv6 ("[::1]"), or a port-bearing form
// ("10.0.0.1:443", "[2001:db8::1]:443" — e.g. ALB client-port preservation).
// Returns the canonical bare-IP string, zone stripped (inet rejects
// "fe80::1%eth0"), or "" when the candidate is not an IP at all.
func normalizeIP(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return ""
	}
	if addr, err := netip.ParseAddr(candidate); err == nil {
		return addr.WithZone("").String()
	}
	if host, _, err := net.SplitHostPort(candidate); err == nil {
		if addr, err := netip.ParseAddr(host); err == nil {
			return addr.WithZone("").String()
		}
	}
	if addr, err := netip.ParseAddr(strings.Trim(candidate, "[]")); err == nil {
		return addr.WithZone("").String()
	}
	return ""
}
