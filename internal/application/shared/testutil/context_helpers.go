//go:build mock_db && mock_auth

// Package testutil provides test-only utilities for configurable context
// creation — it builds *context.Context values pre-seeded with a user ID and
// business type (honoring TEST_USER_ID / TEST_BUSINESS_TYPE env vars) and
// exposes the canonical test-error sentinels. Every file here except errors.go
// is guarded by `//go:build mock_db && mock_auth`; these helpers must NEVER be
// called from production code paths.
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...)
//   - DB drivers or adapter packages
//   - anything under internal/application/usecases/...
//
// Depends only on the Go standard library plus
// internal/application/shared/context (it seeds the same context keys the
// use case layer reads).
//
// Consumers (keep in sync):
//   - usecases/domain/entity/**/*_test.go ONLY — admin, client, client_attribute,
//     delegate_client, group, location, location_attribute, permission, role,
//     role_permission, workspace, workspace_user, workspace_user_role test files.
//
// Q5 — REVIEWER OVERRIDE (sanctioned test-only cross-domain consumer):
// testutil currently has exactly ONE cross-domain consumer root (the `entity`
// domain), so it does NOT meet the Rule of Three (hexagonal-rules.md §4). It is
// retained at `shared/` by explicit reviewer override rather than demoted to a
// domain-local `entity/testhelper` package, because:
//  1. It is an unambiguously pure leaf utility (context seeding + error sentinels,
//     std-lib + shared/context only).
//  2. It is genuinely cross-cutting test INFRASTRUCTURE — every domain's CRUD
//     test suite will adopt it as those suites are written; entity is simply the
//     first/largest adopter today, not the intended sole owner.
//  3. It is build-tag-gated to test builds (`mock_db && mock_auth`), so it adds
//     zero weight to production binaries and cannot leak upward into a use case.
//
// Re-evaluate when a 2nd domain's tests adopt it (expected) — at ≥3 cross-domain
// roots the override is moot and the Rule of Three is satisfied outright. This
// override is the audit-recommended disposition (structure 20260527144541
// report.md rec #5: "genuine test infra").
package testutil

import (
	"context"
	"os"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
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
