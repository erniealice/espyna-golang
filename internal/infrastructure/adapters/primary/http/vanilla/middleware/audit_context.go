//go:build vanilla

package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
)

// AuditContextMiddleware extracts actor metadata from the request and stores it
// in context for downstream audit logging. Must run AFTER authentication middleware
// so that the "uid" key is already present in context.
func AuditContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Actor ID from auth context (set by authentication middleware at line ~74)
		actorID := contextutil.ExtractUserIDFromContext(r.Context())
		actorType := "user"
		if actorID == "" {
			actorID = "system"
			actorType = "system"
		}

		// Request ID: use incoming header or generate one
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// IP address: prefer X-Forwarded-For first entry, else RemoteAddr
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
			// Strip port from "host:port"
			if i := strings.LastIndex(ip, ":"); i > 0 {
				ip = ip[:i]
			}
		} else {
			// X-Forwarded-For may be comma-separated — take first (client) IP
			if i := strings.Index(ip, ","); i > 0 {
				ip = strings.TrimSpace(ip[:i])
			}
		}

		ac := infraports.AuditContext{
			ActorID:   actorID,
			ActorType: actorType,
			IP:        ip,
			UserAgent: r.Header.Get("User-Agent"),
			RequestID: requestID,
		}

		ctx := infraports.WithAuditContext(r.Context(), ac)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
