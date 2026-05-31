// Package appcontext is the dependency-free public leaf for espyna's
// session/identity context helpers.
//
// It re-exports ONLY the stdlib-backed helpers from
// internal/application/shared/context (user.go / business.go / errors.go) plus
// the default session cookie name. It deliberately does NOT re-export the
// translation helpers (translation.go pulls internal/application/ports), so
// this package never gains a container/adapter/ports edge and stays a clean,
// acyclic leaf: appcontext -> internal/application/shared/context -> stdlib only.
//
// Block/view layers should import this package instead of the consumer package
// for context access. The consumer package keeps its own copies of these
// helpers (server + middleware still depend on them); appcontext is additive.
package appcontext

import (
	"context"

	internalctx "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// ErrUserNotFoundInContext indicates no valid user ID was found in the context.
// Re-points to the internal canonical error so errors.Is comparisons made
// through either package match.
var ErrUserNotFoundInContext = internalctx.ErrUserNotFoundInContext

// DefaultSessionCookieName is the default cookie name for session tokens.
// Relocated copy of the consumer package const; coordinate the canonical home
// with the 20260518 Q-D work (decisions.md Q-GATE-2). consumer keeps its own.
const DefaultSessionCookieName = "ichizen_session"

// --- Writers (set identity on a context) ---

// WithUserID creates a new context with the specified user ID.
// Useful for service-to-service calls (e.g. webhooks) that bypass user
// authentication but still require a user context for authorization.
func WithUserID(ctx context.Context, userID string) context.Context {
	return internalctx.WithUserID(ctx, userID)
}

// WithWorkspaceID creates a new context with the specified workspace ID.
func WithWorkspaceID(ctx context.Context, wsID string) context.Context {
	return internalctx.WithWorkspaceID(ctx, wsID)
}

// WithWorkspaceUserID creates a new context with the specified workspace user ID.
func WithWorkspaceUserID(ctx context.Context, wsUserID string) context.Context {
	return internalctx.WithWorkspaceUserID(ctx, wsUserID)
}

// WithSessionIdentity creates a new context carrying the full session identity.
// Empty workspace/workspace-user/email values are skipped.
func WithSessionIdentity(ctx context.Context, userID, workspaceID, workspaceUserID, email string) context.Context {
	return internalctx.WithSessionIdentity(ctx, userID, workspaceID, workspaceUserID, email)
}

// --- Readers (extract identity from a context) ---

// ExtractUserIDFromContext extracts the user ID from context (set by auth
// middleware). Returns empty string if no valid user ID is found.
func ExtractUserIDFromContext(ctx context.Context) string {
	return internalctx.ExtractUserIDFromContext(ctx)
}

// RequireUserIDFromContext extracts the user ID and returns an error if absent.
// Convenience for use cases that require an authenticated user.
func RequireUserIDFromContext(ctx context.Context) (string, error) {
	return internalctx.RequireUserIDFromContext(ctx)
}

// HasUserInContext reports whether a valid user ID exists in the context.
func HasUserInContext(ctx context.Context) bool {
	return internalctx.HasUserInContext(ctx)
}

// GetWorkspaceIDFromContext returns the workspace ID from context, or "".
// Wraps the internal ExtractWorkspaceIDFromContext (public name mirrors consumer).
func GetWorkspaceIDFromContext(ctx context.Context) string {
	return internalctx.ExtractWorkspaceIDFromContext(ctx)
}

// GetWorkspaceUserIDFromContext returns the workspace user ID from context, or "".
// Wraps the internal ExtractWorkspaceUserIDFromContext (public name mirrors consumer).
func GetWorkspaceUserIDFromContext(ctx context.Context) string {
	return internalctx.ExtractWorkspaceUserIDFromContext(ctx)
}
