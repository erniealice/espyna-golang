//go:build fiber

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	infraports "github.com/erniealice/espyna-golang/ports"
	contextutil "github.com/erniealice/espyna-golang/shared/context"
)

// AuditContext populates infraports.AuditContext on every request with ActorID,
// ActorType, IP, UserAgent, RequestID. Must run AFTER authentication middleware
// so that the user ID is already present in the user context.
// Mirrors vanilla contrib/http/internal/adapter/middleware/audit_context.go.
func AuditContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Actor ID from auth context (set by authentication middleware via
		// contextutil.WithUserID on c.UserContext()).
		actorID := contextutil.ExtractUserIDFromContext(c.UserContext())
		actorType := "user"
		if actorID == "" {
			actorID = "system"
			actorType = "system"
		}

		// Request ID: use incoming header or generate one.
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// IP address: prefer X-Forwarded-For first entry, else IP().
		ip := c.Get("X-Forwarded-For")
		if ip == "" {
			ip = c.IP()
			// Strip port from "host:port" if present.
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
			UserAgent: c.Get("User-Agent"),
			RequestID: requestID,
		}

		// Store audit context in the user context so downstream use cases
		// and infrastructure adapters can retrieve it via infraports.GetAuditContext.
		ctx := infraports.WithAuditContext(c.UserContext(), ac)
		c.SetUserContext(ctx)

		return c.Next()
	}
}
