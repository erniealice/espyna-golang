// Package context re-exports internal application context utilities for use
// by contrib sub-modules. Contrib packages (which are separate Go modules)
// should not import internal/ directly; this package provides stable public
// aliases — same pattern as the root-level `ports` and `composition/contracts`
// re-exports.
package context

import (
	"context"

	internal "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// Setters
func WithUserID(ctx context.Context, userID string) context.Context {
	return internal.WithUserID(ctx, userID)
}
func WithWorkspaceID(ctx context.Context, wsID string) context.Context {
	return internal.WithWorkspaceID(ctx, wsID)
}
func WithWorkspaceUserID(ctx context.Context, wsUserID string) context.Context {
	return internal.WithWorkspaceUserID(ctx, wsUserID)
}
func WithEmail(ctx context.Context, email string) context.Context {
	return internal.WithEmail(ctx, email)
}
func WithSessionIdentity(ctx context.Context, userID, workspaceID, workspaceUserID, email string) context.Context {
	return internal.WithSessionIdentity(ctx, userID, workspaceID, workspaceUserID, email)
}

// Extractors
func ExtractUserIDFromContext(ctx context.Context) string {
	return internal.ExtractUserIDFromContext(ctx)
}
func ExtractWorkspaceIDFromContext(ctx context.Context) string {
	return internal.ExtractWorkspaceIDFromContext(ctx)
}
func ExtractWorkspaceUserIDFromContext(ctx context.Context) string {
	return internal.ExtractWorkspaceUserIDFromContext(ctx)
}
func ExtractEmailFromContext(ctx context.Context) string {
	return internal.ExtractEmailFromContext(ctx)
}
func RequireUserIDFromContext(ctx context.Context) (string, error) {
	return internal.RequireUserIDFromContext(ctx)
}
func HasUserInContext(ctx context.Context) bool {
	return internal.HasUserInContext(ctx)
}
