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
	authService ports.AuthorizationService,
	translationService ports.TranslationService,
	entity string,
	action string,
) error {
	if authService == nil {
		log.Println("WARNING: AuthorizationService is nil — denying by default")
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
