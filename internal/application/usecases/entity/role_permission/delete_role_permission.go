package role_permission

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// DeleteRolePermissionRepositories groups all repository dependencies
type DeleteRolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// DeleteRolePermissionServices groups all business service dependencies
type DeleteRolePermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteRolePermissionUseCase handles the business logic for deleting role permissions
type DeleteRolePermissionUseCase struct {
	repositories DeleteRolePermissionRepositories
	services     DeleteRolePermissionServices
}

// NewDeleteRolePermissionUseCase creates use case with grouped dependencies
func NewDeleteRolePermissionUseCase(
	repositories DeleteRolePermissionRepositories,
	services DeleteRolePermissionServices,
) *DeleteRolePermissionUseCase {
	return &DeleteRolePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteRolePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteRolePermissionUseCase with grouped parameters instead
func NewDeleteRolePermissionUseCaseUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *DeleteRolePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteRolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           nil, // Not needed for delete operations
		Permission:     nil, // Not needed for delete operations
	}

	services := DeleteRolePermissionServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteRolePermissionUseCase(repositories, services)
}

// Execute performs the delete role permission operation
func (uc *DeleteRolePermissionUseCase) Execute(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) (*rolepermissionpb.DeleteRolePermissionResponse, error) {

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityRolePermission, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.input_validation_failed", "Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.business_rule_validation_failed", "Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RolePermission.DeleteRolePermission(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.deletion_failed", "Role-Permission deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	log.Printf("AUTHZ_CHANGE | action=revoke | role_permission_id=%s", req.Data.Id)

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteRolePermissionUseCase) validateInput(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.request_required", "Request is required for role-permissions"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.data_required", "Role-Permission data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.id_required", "Role-Permission ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteRolePermissionUseCase) validateBusinessRules(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) error {
	// TODO: Add business rules for role permission deletion
	// Example: Check if removing this permission would leave role without critical permissions
	// For now, allow all deletions

	return nil
}

// Helper functions
