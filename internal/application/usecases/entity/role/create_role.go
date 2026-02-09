package role

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// CreateRoleRepositories groups all repository dependencies
type CreateRoleRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// CreateRoleServices groups all business service dependencies
type CreateRoleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateRoleUseCase handles the business logic for creating roles
type CreateRoleUseCase struct {
	repositories CreateRoleRepositories
	services     CreateRoleServices
}

// NewCreateRoleUseCase creates use case with grouped dependencies
func NewCreateRoleUseCase(
	repositories CreateRoleRepositories,
	services CreateRoleServices,
) *CreateRoleUseCase {
	return &CreateRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateRoleUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateRoleUseCase with grouped parameters instead
func NewCreateRoleUseCaseUngrouped(roleRepo rolepb.RoleDomainServiceServer) *CreateRoleUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateRoleRepositories{
		Role: roleRepo,
	}

	services := CreateRoleServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateRoleUseCase(repositories, services)
}

// Execute performs the create role operation
func (uc *CreateRoleUseCase) Execute(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityRole, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role creation within a transaction
func (uc *CreateRoleUseCase) executeWithTransaction(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
	var result *rolepb.CreateRoleResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "role.errors.creation_failed", "Role creation failed [DEFAULT]")
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
func (uc *CreateRoleUseCase) executeCore(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
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
	return uc.repositories.Role.CreateRole(ctx, req)
}

// validateInput validates the input request
func (uc *CreateRoleUseCase) validateInput(ctx context.Context, req *rolepb.CreateRoleRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.request_required", "Request is required for roles [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.data_required", "Role data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.name_required", "Role name is required [DEFAULT]"))
	}
	return nil
}

// enrichRoleData adds generated fields and audit information
func (uc *CreateRoleUseCase) enrichRoleData(role *rolepb.Role) error {
	now := time.Now()

	// Generate Role ID if not provided
	if role.Id == "" {
		role.Id = uc.services.IDService.GenerateID()
	}

	// Set default color if not provided
	if role.Color == "" {
		role.Color = "#3B82F6" // Default blue color
	}

	// Set role audit fields
	role.DateCreated = &[]int64{now.Unix()}[0]
	role.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	role.DateModified = &[]int64{now.Unix()}[0]
	role.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	role.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateRoleUseCase) validateBusinessRules(ctx context.Context, role *rolepb.Role) error {
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
func (uc *CreateRoleUseCase) validateHexColor(color string) error {
	hexColorRegex := regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
	if !hexColorRegex.MatchString(color) {
		return errors.New("color must be a valid hex color (e.g., #FF0000 or #F00)")
	}
	return nil
}
