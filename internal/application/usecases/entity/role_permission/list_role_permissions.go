package role_permission

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// ListRolePermissionsRepositories groups all repository dependencies
type ListRolePermissionsRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// ListRolePermissionsServices groups all business service dependencies
type ListRolePermissionsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListRolePermissionsUseCase handles the business logic for listing role permissions
type ListRolePermissionsUseCase struct {
	repositories ListRolePermissionsRepositories
	services     ListRolePermissionsServices
}

// NewListRolePermissionsUseCase creates use case with grouped dependencies
func NewListRolePermissionsUseCase(
	repositories ListRolePermissionsRepositories,
	services ListRolePermissionsServices,
) *ListRolePermissionsUseCase {
	return &ListRolePermissionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListRolePermissionsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListRolePermissionsUseCase with grouped parameters instead
func NewListRolePermissionsUseCaseUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *ListRolePermissionsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListRolePermissionsRepositories{
		RolePermission: rolePermissionRepo,
		Role:           nil, // Not needed for list operations
		Permission:     nil, // Not needed for list operations
	}

	services := ListRolePermissionsServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListRolePermissionsUseCase(repositories, services)
}

func (uc *ListRolePermissionsUseCase) Execute(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) (*rolepermissionpb.ListRolePermissionsResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.user_not_authenticated", "User not authenticated"))
		}

		// Check permission to list role permissions
		permission := ports.EntityPermission(ports.EntityRolePermission, ports.ActionRead)
		authorized, err := uc.services.AuthorizationService.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.authorization_check_failed", "Authorization check failed")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !authorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.access_denied", "Access denied")
			return nil, errors.New(translatedError)
		}
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
	resp, err := uc.repositories.RolePermission.ListRolePermissions(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.list_failed", "Failed to retrieve role-permissions")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListRolePermissionsUseCase) validateInput(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.request_required", "Request is required for role-permissions"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListRolePermissionsUseCase) validateBusinessRules(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) error {
	// Add any business rules for filtering or access control
	// For example, ensure user has permission to view role permissions
	return nil
}

// Helper functions
