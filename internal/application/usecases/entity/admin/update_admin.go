package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
)

// UpdateAdminRepositories groups all repository dependencies
type UpdateAdminRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// UpdateAdminServices groups all business service dependencies
type UpdateAdminServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateAdminUseCase handles the business logic for updating an admin
type UpdateAdminUseCase struct {
	repositories UpdateAdminRepositories
	services     UpdateAdminServices
}

// NewUpdateAdminUseCase creates use case with grouped dependencies
func NewUpdateAdminUseCase(
	repositories UpdateAdminRepositories,
	services UpdateAdminServices,
) *UpdateAdminUseCase {
	return &UpdateAdminUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update admin operation
func (uc *UpdateAdminUseCase) Execute(ctx context.Context, req *adminpb.UpdateAdminRequest) (*adminpb.UpdateAdminResponse, error) {
	// Authorization check (skip if authorization service is nil or disabled)
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID, err := contextutil.RequireUserIDFromContext(ctx)
		if err != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.authorization_failed", ""))
		}

		permission := ports.EntityPermission(ports.EntityAdmin, ports.ActionUpdate)
		hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
		if err != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.authorization_failed", ""))
		}
		if !hasPerm {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.authorization_failed", ""))
		}
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.request_required", ""))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.data_required", ""))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.id_required", ""))
	}

	// Validate email format if email is provided
	if req.Data.User != nil && req.Data.User.EmailAddress != "" {
		if err := uc.validateEmail(req.Data.User.EmailAddress); err != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.email_invalid", ""))
		}
	}

	// Call repository
	resp, err := uc.repositories.Admin.UpdateAdmin(ctx, req)
	if err != nil {
		if err.Error() == contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.not_found", "") {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.not_found", ""))
		}
		return nil, fmt.Errorf("%s: %w", contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.update_failed", ""), err)
	}

	return resp, nil
}

// validateEmail validates email format
func (uc *UpdateAdminUseCase) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}
