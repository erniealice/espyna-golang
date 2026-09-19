package middleware

import (
	"context"
	"time"
)

// TimezoneConfig supplies calendar lookup and context storage hooks to the
// HTTP provider. Workspace dates take precedence over personal preferences;
// unscoped requests retain the user preference and provider fallback.
type TimezoneConfig struct {
	// Workspace timezone takes precedence for workspace business dates.
	LookupWorkspaceTimezone func(context.Context) (string, error)
	// WithLocation bridges the resolved zone into the UI context.
	WithLocation func(context.Context, *time.Location) context.Context
	// GetUserID extracts the authenticated user ID from the request context.
	GetUserID func(ctx context.Context) string

	// LookupTimezone fetches the user's IANA timezone preference given their
	// user ID and request context. Return "" to fall back to the impl default.
	LookupTimezone func(ctx context.Context, userID string) string
}
