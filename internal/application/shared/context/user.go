// Package context provides shared utilities for extracting information from Go contexts
// in use cases across all domains while maintaining hexagonal architecture principles.
package context

import (
	"context"
)

// ExtractUserIDFromContext extracts user ID from context (set by authentication middleware).
// It checks for both "uid" and "user_id" context keys for backward compatibility.
// Returns empty string if no valid user ID is found.
func ExtractUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("uid").(string); ok && userID != "" {
		return userID
	}
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		return userID
	}
	return ""
}

// RequireUserIDFromContext extracts user ID from context and returns an error if not found.
// This is a convenience method for use cases that require authenticated users.
func RequireUserIDFromContext(ctx context.Context) (string, error) {
	userID := ExtractUserIDFromContext(ctx)
	if userID == "" {
		return "", ErrUserNotFoundInContext
	}
	return userID, nil
}

// HasUserInContext checks if a valid user ID exists in the context without extracting it.
func HasUserInContext(ctx context.Context) bool {
	return ExtractUserIDFromContext(ctx) != ""
}

// WithUserID creates a new context with the specified user ID.
// This is useful for testing scenarios where we need to simulate authenticated users.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, "uid", userID)
}
