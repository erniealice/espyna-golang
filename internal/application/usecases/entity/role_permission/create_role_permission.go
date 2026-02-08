package role_permission

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// CreateRolePermissionRepositories groups all repository dependencies
type CreateRolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// CreateRolePermissionServices groups all business service dependencies
type CreateRolePermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateRolePermissionUseCase handles the business logic for creating role permissions
type CreateRolePermissionUseCase struct {
	repositories CreateRolePermissionRepositories
	services     CreateRolePermissionServices
}

// NewCreateRolePermissionUseCase creates use case with grouped dependencies
func NewCreateRolePermissionUseCase(
	repositories CreateRolePermissionRepositories,
	services CreateRolePermissionServices,
) *CreateRolePermissionUseCase {
	return &CreateRolePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateRolePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateRolePermissionUseCase with grouped parameters instead
func NewCreateRolePermissionUseCaseUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
	roleRepo rolepb.RoleDomainServiceServer,
	permissionRepo permissionpb.PermissionDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateRolePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateRolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           roleRepo,
		Permission:     permissionRepo,
	}

	services := CreateRolePermissionServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateRolePermissionUseCase(repositories, services)
}

func (uc *CreateRolePermissionUseCase) Execute(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) (*rolepermissionpb.CreateRolePermissionResponse, error) {
	// Authorization check - CRITICAL: This manages role permissions (high security)
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.user_not_authenticated", "User not authenticated"))
		}

		// For role-permission relationships, we need global permission to manage role permissions
		// This is a highly sensitive operation that affects system security
		permission := ports.EntityPermission(ports.EntityRolePermission, ports.ActionCreate)
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

	// Business logic and enrichment
	if err := uc.enrichRolePermissionData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.enrichment_failed", "Business logic enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.business_rule_validation_failed", "Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.reference_validation_failed", "Entity reference validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RolePermission.CreateRolePermission(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.creation_failed", "Role-Permission creation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateRolePermissionUseCase) validateInput(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.request_required", "Request is required for role-permissions"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.data_required", "Role-Permission data is required"))
	}
	if req.Data.RoleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.role_id_required", "Role ID is required"))
	}
	if req.Data.PermissionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.permission_id_required", "Permission ID is required"))
	}
	return nil
}

// enrichRolePermissionData adds generated fields and audit information
func (uc *CreateRolePermissionUseCase) enrichRolePermissionData(rolePermission *rolepermissionpb.RolePermission) error {
	now := time.Now()

	// Generate RolePermission ID if not provided
	if rolePermission.Id == "" {
		rolePermission.Id = uc.services.IDService.GenerateID()
	}

	// Set default permission type if not specified
	if rolePermission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		rolePermission.PermissionType = permissionpb.PermissionType_PERMISSION_TYPE_ALLOW
	}

	// Set audit fields
	rolePermission.DateCreated = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	rolePermission.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	rolePermission.DateModified = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	rolePermission.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	rolePermission.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateRolePermissionUseCase) validateBusinessRules(ctx context.Context, rolePermission *rolepermissionpb.RolePermission) error {
	// Validate permission type
	if rolePermission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.permission_type_unspecified", "Permission type must be specified"))
	}

	// Validate role and permission relationship
	if rolePermission.RoleId == rolePermission.PermissionId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.same_id", "Role ID and permission ID cannot be the same"))
	}

	// Business rule: Prevent duplicate role-permission relationships
	// This validation should be checked at the repository level to ensure uniqueness
	// The repository implementation should check if a relationship already exists
	// between the role and permission before creating a new one

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateRolePermissionUseCase) validateEntityReferences(ctx context.Context, rolePermission *rolepermissionpb.RolePermission) error {
	// Validate Role entity reference
	if rolePermission.RoleId != "" {
		role, err := uc.repositories.Role.ReadRole(ctx, &rolepb.ReadRoleRequest{
			Data: &rolepb.Role{Id: rolePermission.RoleId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.role_reference_validation_failed", "Failed to validate role entity reference")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if role == nil || !role.Success || role.Data == nil || len(role.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.role_not_found", "Referenced role with ID '{roleId}' does not exist")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", rolePermission.RoleId)
			return errors.New(translatedError)
		}
		if !role.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.role_not_active", "Referenced role with ID '{roleId}' is not active")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", rolePermission.RoleId)
			return errors.New(translatedError)
		}
	}

	// Validate Permission entity reference
	if rolePermission.PermissionId != "" {
		permission, err := uc.repositories.Permission.ReadPermission(ctx, &permissionpb.ReadPermissionRequest{
			Data: &permissionpb.Permission{Id: rolePermission.PermissionId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.permission_reference_validation_failed", "Failed to validate permission entity reference")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if permission == nil || !permission.Success || permission.Data == nil || len(permission.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.permission_not_found", "Referenced permission with ID '{permissionId}' does not exist")
			translatedError = strings.ReplaceAll(translatedError, "{permissionId}", rolePermission.PermissionId)
			return errors.New(translatedError)
		}
		if !permission.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.permission_not_active", "Referenced permission with ID '{permissionId}' is not active")
			translatedError = strings.ReplaceAll(translatedError, "{permissionId}", rolePermission.PermissionId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// Helper functions
