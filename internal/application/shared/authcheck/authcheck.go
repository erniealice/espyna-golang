// Package authcheck centralises the "is the actor authorized for this
// (entity, action) pair?" guard called at the top of every use case Execute
// method. It is Layer 3 infrastructure beneath the use case layer (see Q-PSV5
// in docs/plan/20260518-hexagonal-strict-adherence/proto-service.md) — a
// keycard reader at every room's door, not a room itself.
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...)
//   - DB drivers or adapter packages
//   - anything under internal/application/usecases/...
//
// Depends only on the Go standard library plus
// internal/application/ports + internal/application/shared/context.
//
// Consumer surface: 957 .go files across espyna-golang call
// authcheck.Check(ctx, authSvc, i18nSvc, "<entity>", ports.Action<X>) as the
// FIRST line of every use case Execute body. Adding a new caller is the
// canonical signal that you are at the use case layer; needing to call it
// from a non-use-case file is a smell.
package authcheck

import (
	"context"
	"errors"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// Check verifies that the user in context has the given permission.
// Returns nil if authorized, or an error if denied/missing.
// SECURE DEFAULT: If authService is nil, returns an error (deny-by-default).
// If authService.IsEnabled() returns false, allows all (for dev/mock mode).
func Check(
	ctx context.Context,
	authService ports.Authorizer,
	translationService ports.Translator,
	entity string,
	action string,
) error {
	if authService == nil {
		log.Println("WARNING: Authorizer is nil — denying by default")
		msg := contextutil.GetTranslatedMessageWithContext(
			ctx, translationService, "common.errors.authorization_failed", "Authorization not configured")
		return errors.New(msg)
	}

	if !authService.IsEnabled() {
		return nil // Dev/mock mode: allow all
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(
			ctx, translationService, "common.errors.authorization_failed", "Authorization failed")
		return errors.New(msg)
	}

	permission := ports.EntityPermission(entity, action)
	hasPerm, err := authService.HasPermission(ctx, userID, permission)
	if err != nil {
		log.Printf("AUTHZ_ERROR | user=%s | permission=%s | error=%v", userID, permission, err)
		msg := contextutil.GetTranslatedMessageWithContext(
			ctx, translationService, "common.errors.authorization_failed", "Authorization failed")
		return errors.New(msg)
	}

	if !hasPerm {
		log.Printf("AUTHZ_DENIED | user=%s | permission=%s", userID, permission)
		msg := contextutil.GetTranslatedMessageWithContext(
			ctx, translationService, "common.errors.permission_denied", "Permission denied")
		return errors.New(msg)
	}

	return nil
}
