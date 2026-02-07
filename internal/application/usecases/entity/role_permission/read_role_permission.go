package role_permission

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// ReadRolePermissionRepositories groups all repository dependencies
type ReadRolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// ReadRolePermissionServices groups all business service dependencies
type ReadRolePermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadRolePermissionUseCase handles the business logic for reading a role permission
type ReadRolePermissionUseCase struct {
	repositories ReadRolePermissionRepositories
	services     ReadRolePermissionServices
}

// NewReadRolePermissionUseCase creates use case with grouped dependencies
func NewReadRolePermissionUseCase(
	repositories ReadRolePermissionRepositories,
	services ReadRolePermissionServices,
) *ReadRolePermissionUseCase {
	return &ReadRolePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadRolePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadRolePermissionUseCase with grouped parameters instead
func NewReadRolePermissionUseCaseUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
) *ReadRolePermissionUseCase {
	repositories := ReadRolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           nil,
		Permission:     nil,
	}
	services := ReadRolePermissionServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadRolePermissionUseCase(repositories, services)
}

// Execute performs the read role permission operation
func (uc *ReadRolePermissionUseCase) Execute(ctx context.Context, req *rolepermissionpb.ReadRolePermissionRequest) (*rolepermissionpb.ReadRolePermissionResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.user_not_authenticated", "User not authenticated"))
		}

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

	// Call repository
	resp, err := uc.repositories.RolePermission.ReadRolePermission(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.read_failed", "Failed to read role permission")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" { // Assuming resp.Data will be nil or have empty ID if not found
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.not_found", "Role-Permission with ID \"{rolePermissionId}\" not found")
		translatedError = strings.ReplaceAll(translatedError, "{rolePermissionId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadRolePermissionUseCase) validateInput(ctx context.Context, req *rolepermissionpb.ReadRolePermissionRequest) error {
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

// Helper functions
