//go:build mock_db && mock_auth

// Package testutil provides test-specific utilities for configurable context creation.
// These functions should only be used in test files.
package testutil

import (
	"context"
	"os"

	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
)

// GetTestUserID returns the test user ID from environment variable TEST_USER_ID.
// Falls back to "test-user" if not set, maintaining backward compatibility.
func GetTestUserID() string {
	if userID := os.Getenv("TEST_USER_ID"); userID != "" {
		return userID
	}
	return "test-user" // Default fallback for existing tests
}

// GetTestBusinessType returns the test business type from environment variable TEST_BUSINESS_TYPE.
// Falls back to "education" if not set, maintaining backward compatibility.
func GetTestBusinessType() string {
	if businessType := os.Getenv("TEST_BUSINESS_TYPE"); businessType != "" {
		return businessType
	}
	return "education" // Default fallback for existing tests
}

// CreateTestContext creates a context with configurable user ID and business type.
// This is the recommended way to create test contexts, as it respects environment variables
// while maintaining backward compatibility with existing hardcoded test values.
//
// Usage:
//
//	ctx := testutil.CreateTestContext()
//	// Uses TEST_USER_ID and TEST_BUSINESS_TYPE env vars, falls back to defaults
//
//	ctx := testutil.CreateTestContextWithUser("custom-user")
//	// Uses custom user ID, respects TEST_BUSINESS_TYPE env var
func CreateTestContext() context.Context {
	ctx := context.Background()
	ctx = contextutil.WithUserID(ctx, GetTestUserID())
	ctx = contextutil.WithBusinessType(ctx, GetTestBusinessType())
	return ctx
}

// CreateTestContextWithUser creates a test context with a specific user ID,
// while still respecting the TEST_BUSINESS_TYPE environment variable.
func CreateTestContextWithUser(userID string) context.Context {
	ctx := context.Background()
	ctx = contextutil.WithUserID(ctx, userID)
	ctx = contextutil.WithBusinessType(ctx, GetTestBusinessType())
	return ctx
}

// CreateTestContextWithBusinessType creates a test context with a specific business type,
// while still respecting the TEST_USER_ID environment variable.
func CreateTestContextWithBusinessType(businessType string) context.Context {
	ctx := context.Background()
	ctx = contextutil.WithUserID(ctx, GetTestUserID())
	ctx = contextutil.WithBusinessType(ctx, businessType)
	return ctx
}

// CreateTestContextFull creates a test context with specific user ID and business type,
// ignoring environment variables. This is useful for tests that need explicit control.
func CreateTestContextFull(userID, businessType string) context.Context {
	ctx := context.Background()
	ctx = contextutil.WithUserID(ctx, userID)
	ctx = contextutil.WithBusinessType(ctx, businessType)
	return ctx
}
