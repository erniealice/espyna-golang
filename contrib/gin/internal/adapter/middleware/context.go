//go:build gin

package middleware

// context.go — shared context helpers for the Gin middleware package.
//
// TODO: Implement context key types and accessor functions for Gin middleware.
// This file should mirror the Fiber context.go
// (contrib/fiber/internal/adapter/middleware/context.go): typed context keys,
// contextWithValue helper, GetUserFromContext, GetIdentityFromContext, and
// GetWorkspaceFromContext.
//
// The existing authorization.go already has GetUserIDFromGinContext and
// GetWorkspaceFromContext using gin.Context.Get(); this file should unify
// those accessors under the shared identity package
// (github.com/erniealice/espyna-golang/shared/identity) for consistency with
// the Fiber implementation.
//
// Blocked on: Server API wiring to determine which context propagation
// strategy (gin.Context.Set vs request context) the Gin adapter standardizes on.
