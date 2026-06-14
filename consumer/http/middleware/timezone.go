package middleware

import "context"

// TimezoneConfig is the agnostic declarative slot the Preset carries for the
// Timezone middleware. It holds ONLY the two request-scoped closures the chain
// assembler needs; the chain translates them into the contrib
// NewTimezoneMiddleware(uidFn, lookupFn) form (which owns its own default-zone +
// store-location behaviour internally). No impl re-export, no build tag.
type TimezoneConfig struct {
	// GetUserID extracts the authenticated user ID from the request context.
	GetUserID func(ctx context.Context) string

	// LookupTimezone fetches the user's IANA timezone preference given their
	// user ID and request context. Return "" to fall back to the impl default.
	LookupTimezone func(ctx context.Context, userID string) string
}
