// Package consumer provides the public API for the espyna package.
// This file exposes context utilities for external packages to use.
package consumer

import (
	"context"
	"errors"

	internalctx "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// Context-specific errors (re-exported from internal package)
var (
	// ErrUserNotFoundInContext indicates no valid user ID was found in the context
	ErrUserNotFoundInContext = errors.New("user not found in context")
)

// WithUserID creates a new context with the specified user ID.
// This is useful for service-to-service calls (like webhooks) that need to
// bypass user authentication but still require a user context for authorization.
//
// Example usage:
//
//	ctx := consumer.WithUserID(context.Background(), "webhook-service")
//	payment, err := paymentuc.GetPaymentDocument(ctx, container, paymentID)
func WithUserID(ctx context.Context, userID string) context.Context {
	return internalctx.WithUserID(ctx, userID)
}

// ExtractUserIDFromContext extracts user ID from context (set by authentication middleware).
// It checks for both "uid" and "user_id" context keys for backward compatibility.
// Returns empty string if no valid user ID is found.
func ExtractUserIDFromContext(ctx context.Context) string {
	return internalctx.ExtractUserIDFromContext(ctx)
}

// RequireUserIDFromContext extracts user ID from context and returns an error if not found.
// This is a convenience method for use cases that require authenticated users.
func RequireUserIDFromContext(ctx context.Context) (string, error) {
	return internalctx.RequireUserIDFromContext(ctx)
}

// HasUserInContext checks if a valid user ID exists in the context without extracting it.
func HasUserInContext(ctx context.Context) bool {
	return internalctx.HasUserInContext(ctx)
}
