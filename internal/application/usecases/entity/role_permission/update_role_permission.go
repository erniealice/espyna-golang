package role_permission

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// UpdateRolePermissionRepositories groups all repository dependencies
type UpdateRolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// UpdateRolePermissionServices groups all business service dependencies
type UpdateRolePermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateRolePermissionUseCase handles the business logic for updating role permissions
type UpdateRolePermissionUseCase struct {
	repositories UpdateRolePermissionRepositories
	services     UpdateRolePermissionServices
}

// NewUpdateRolePermissionUseCase creates use case with grouped dependencies
func NewUpdateRolePermissionUseCase(
	repositories UpdateRolePermissionRepositories,
	services UpdateRolePermissionServices,
) *UpdateRolePermissionUseCase {
	return &UpdateRolePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateRolePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateRolePermissionUseCase with grouped parameters instead
func NewUpdateRolePermissionUseCaseUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
	roleRepo rolepb.RoleDomainServiceServer,
	permissionRepo permissionpb.PermissionDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *UpdateRolePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateRolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           roleRepo,
		Permission:     permissionRepo,
	}

	services := UpdateRolePermissionServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateRolePermissionUseCase(repositories, services)
}

// Execute performs the update role permission operation
func (uc *UpdateRolePermissionUseCase) Execute(ctx context.Context, req *rolepermissionpb.UpdateRolePermissionRequest) (*rolepermissionpb.UpdateRolePermissionResponse, error) {

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityRolePermission, ports.ActionUpdate); err != nil {
		return nil, err
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
	resp, err := uc.repositories.RolePermission.UpdateRolePermission(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.update_failed", "Role-Permission update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateRolePermissionUseCase) validateInput(ctx context.Context, req *rolepermissionpb.UpdateRolePermissionRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.request_required", "Request is required for role-permissions"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.data_required", "Role-Permission data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.id_required", "Role-Permission ID is required"))
	}
	if req.Data.RoleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.role_id_required", "Role ID is required"))
	}
	if req.Data.PermissionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.permission_id_required", "Permission ID is required"))
	}
	return nil
}

// enrichRolePermissionData adds audit information for updates
func (uc *UpdateRolePermissionUseCase) enrichRolePermissionData(rolePermission *rolepermissionpb.RolePermission) error {
	now := time.Now()

	// Set role permission audit fields for modification
	rolePermission.DateModified = &[]int64{now.Unix()}[0]
	rolePermission.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateRolePermissionUseCase) validateBusinessRules(ctx context.Context, rolePermission *rolepermissionpb.RolePermission) error {

	// Validate permission type
	if rolePermission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.permission_type_unspecified", "Permission type must be specified"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateRolePermissionUseCase) validateEntityReferences(ctx context.Context, rolePermission *rolepermissionpb.RolePermission) error {

	// Validate Role entity reference
	if rolePermission.RoleId != "" {
		role, err := uc.repositories.Role.ReadRole(ctx, &rolepb.ReadRoleRequest{
			Data: &rolepb.Role{Id: rolePermission.RoleId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.role_reference_validation_failed", "Failed to validate role entity reference")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if role == nil || role.Data == nil || len(role.Data) == 0 {
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
		if permission == nil || permission.Data == nil || len(permission.Data) == 0 {
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
