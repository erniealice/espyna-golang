package workspace_user_role

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
	workspaceuserrolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user_role"
)

// ListWorkspaceUserRolesRepositories groups all repository dependencies
type ListWorkspaceUserRolesRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer // Primary entity repository
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer         // Entity reference validation
	Role              rolepb.RoleDomainServiceServer                           // Entity reference validation
}

// ListWorkspaceUserRolesServices groups all business service dependencies
type ListWorkspaceUserRolesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListWorkspaceUserRolesUseCase handles the business logic for listing workspace user roles
type ListWorkspaceUserRolesUseCase struct {
	repositories ListWorkspaceUserRolesRepositories
	services     ListWorkspaceUserRolesServices
}

// NewListWorkspaceUserRolesUseCase creates use case with grouped dependencies
func NewListWorkspaceUserRolesUseCase(
	repositories ListWorkspaceUserRolesRepositories,
	services ListWorkspaceUserRolesServices,
) *ListWorkspaceUserRolesUseCase {
	return &ListWorkspaceUserRolesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list workspace user roles operation
func (uc *ListWorkspaceUserRolesUseCase) Execute(ctx context.Context, req *workspaceuserrolepb.ListWorkspaceUserRolesRequest) (*workspaceuserrolepb.ListWorkspaceUserRolesResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.user_not_authenticated", "User not authenticated "))
		}

		// Check permission to list workspace user roles
		permission := ports.EntityPermission(ports.EntityWorkspaceUserRole, ports.ActionRead)
		authorized, err := uc.services.AuthorizationService.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.authorization_check_failed", "Authorization check failed ")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !authorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.access_denied", "Access denied ")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.business_rule_validation_failed", "Business rule validation failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUserRole.ListWorkspaceUserRoles(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.list_failed", "Failed to retrieve workspace user roles ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListWorkspaceUserRolesUseCase) validateInput(ctx context.Context, req *workspaceuserrolepb.ListWorkspaceUserRolesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.request_required", "Request is required for workspace user roles "))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListWorkspaceUserRolesUseCase) validateBusinessRules(ctx context.Context, req *workspaceuserrolepb.ListWorkspaceUserRolesRequest) error {
	// Add any business rules for filtering or access control
	// For example, ensure user has permission to view workspace user roles
	return nil
}

// Helper functions
