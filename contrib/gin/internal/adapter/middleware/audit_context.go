//go:build gin

package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	infraports "github.com/erniealice/espyna-golang/ports"
)

// AuditContext populates infraports.AuditContext on every request with ActorID,
// ActorType, IP, UserAgent, RequestID. Must run AFTER authentication middleware
// so that the "uid" key is already present in the Gin context.
// Mirrors vanilla contrib/http/internal/adapter/middleware/audit_context.go.
func AuditContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Actor ID: prefer the "uid" key set by authentication middleware,
		// fall back to "user_id" for consumer-app compatibility.
		actorID := c.GetString("uid")
		if actorID == "" {
			actorID = c.GetString("user_id")
		}
		actorType := "user"
		if actorID == "" {
			actorID = "system"
			actorType = "system"
		}

		// Request ID: use incoming header or generate one.
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// IP address: prefer X-Forwarded-For first entry, else RemoteAddr.
		ip := c.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = c.Request.RemoteAddr
			// Strip port from "host:port".
			if i := strings.LastIndex(ip, ":"); i > 0 {
				ip = ip[:i]
			}
		} else {
			// X-Forwarded-For may be comma-separated — take first (client) IP.
			if i := strings.Index(ip, ","); i > 0 {
				ip = strings.TrimSpace(ip[:i])
			}
		}

		ac := infraports.AuditContext{
			ActorID:   actorID,
			ActorType: actorType,
			IP:        ip,
			UserAgent: c.GetHeader("User-Agent"),
			RequestID: requestID,
		}

		// Store audit context in the request's Go context so downstream
		// use cases and infrastructure adapters can retrieve it.
		ctx := infraports.WithAuditContext(c.Request.Context(), ac)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
