package role_permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// ListRolePermissionsRepositories groups all repository dependencies
type ListRolePermissionsRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// ListRolePermissionsServices groups all business service dependencies
type ListRolePermissionsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	authorizationService ports.Authorizer,
) *ListRolePermissionsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListRolePermissionsRepositories{
		RolePermission: rolePermissionRepo,
		Role:           nil, // Not needed for list operations
		Permission:     nil, // Not needed for list operations
	}

	services := ListRolePermissionsServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListRolePermissionsUseCase(repositories, services)
}

func (uc *ListRolePermissionsUseCase) Execute(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) (*rolepermissionpb.ListRolePermissionsResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.RolePermission,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.input_validation_failed", "Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.business_rule_validation_failed", "Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RolePermission.ListRolePermissions(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.list_failed", "Failed to retrieve role-permissions")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListRolePermissionsUseCase) validateInput(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.request_required", "Request is required for role-permissions"))
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
