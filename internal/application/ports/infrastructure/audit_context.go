package infrastructure

import "context"

// AuditContext carries actor metadata for audit logging.
// Populated by provider-specific middleware on every HTTP request.
type AuditContext struct {
	ActorID   string // from auth context "uid" key
	ActorType string // "user" | "system" | "api_key"
	IP        string // from r.RemoteAddr or X-Forwarded-For
	UserAgent string // from User-Agent header
	RequestID string // from X-Request-ID header or generated UUID
}

type auditCtxKey struct{}

// WithAuditContext stores an AuditContext in the given context.
func WithAuditContext(ctx context.Context, ac AuditContext) context.Context {
	return context.WithValue(ctx, auditCtxKey{}, ac)
}

// GetAuditContext retrieves the AuditContext from context.
// Returns zero value and false if not set.
func GetAuditContext(ctx context.Context) (AuditContext, bool) {
	ac, ok := ctx.Value(auditCtxKey{}).(AuditContext)
	return ac, ok
}
