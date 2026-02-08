package role

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// UpdateRoleRepositories groups all repository dependencies
type UpdateRoleRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// UpdateRoleServices groups all business service dependencies
type UpdateRoleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateRoleUseCase handles the business logic for updating roles
type UpdateRoleUseCase struct {
	repositories UpdateRoleRepositories
	services     UpdateRoleServices
}

// NewUpdateRoleUseCase creates use case with grouped dependencies
func NewUpdateRoleUseCase(
	repositories UpdateRoleRepositories,
	services UpdateRoleServices,
) *UpdateRoleUseCase {
	return &UpdateRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateRoleUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateRoleUseCase with grouped parameters instead
func NewUpdateRoleUseCaseUngrouped(roleRepo rolepb.RoleDomainServiceServer) *UpdateRoleUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateRoleRepositories{
		Role: roleRepo,
	}

	services := UpdateRoleServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateRoleUseCase(repositories, services)
}

// Execute performs the update role operation
func (uc *UpdateRoleUseCase) Execute(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role update within a transaction
func (uc *UpdateRoleUseCase) executeWithTransaction(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	var result *rolepb.UpdateRoleResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "role.errors.update_failed", "Role update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *UpdateRoleUseCase) executeCore(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichRoleData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Role.UpdateRole(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateRoleUseCase) validateInput(ctx context.Context, req *rolepb.UpdateRoleRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.request_required", "Request is required for roles [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.data_required", "Role data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.id_required", "Role ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.name_required", "Role name is required [DEFAULT]"))
	}
	return nil
}

// enrichRoleData adds audit information for updates
func (uc *UpdateRoleUseCase) enrichRoleData(role *rolepb.Role) error {
	now := time.Now()

	// Set role audit fields for modification
	role.DateModified = &[]int64{now.Unix()}[0]
	role.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateRoleUseCase) validateBusinessRules(ctx context.Context, role *rolepb.Role) error {
	// Validate name length
	if len(role.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.name_too_short", "Role name must be at least 2 characters long [DEFAULT]"))
	}

	if len(role.Name) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.name_too_long", "Role name cannot exceed 50 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if role.Description != "" && len(role.Description) > 255 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.description_too_long", "Role description cannot exceed 255 characters [DEFAULT]"))
	}

	// Validate color format (hex color)
	if role.Color != "" {
		if err := uc.validateHexColor(role.Color); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.color_invalid", "Color must be a valid hex color (e.g., #FF0000 or #F00) [DEFAULT]"))
		}
	}

	return nil
}

// validateHexColor validates hex color format
func (uc *UpdateRoleUseCase) validateHexColor(color string) error {
	hexColorRegex := regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
	if !hexColorRegex.MatchString(color) {
		return errors.New("color must be a valid hex color (e.g., #FF0000 or #F00)")
	}
	return nil
}
