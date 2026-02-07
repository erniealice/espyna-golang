//go:build mock_db && mock_auth

package testutil

import (
	"context"
)

// CreateStandardTestContext creates a standard test context using the default test user ID
func CreateStandardTestContext() context.Context {
	return CreateTestContext()
}

// CreateStandardTestContextWithUser creates a test context with a specific user ID
func CreateStandardTestContextWithUser(userID string) context.Context {
	return CreateTestContextWithUser(userID)
}
