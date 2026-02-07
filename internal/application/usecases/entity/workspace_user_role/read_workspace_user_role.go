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

// ReadWorkspaceUserRoleRepositories groups all repository dependencies
type ReadWorkspaceUserRoleRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer // Primary entity repository
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer         // Entity reference validation
	Role              rolepb.RoleDomainServiceServer                           // Entity reference validation
}

// ReadWorkspaceUserRoleServices groups all business service dependencies
type ReadWorkspaceUserRoleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadWorkspaceUserRoleUseCase handles the business logic for reading a workspace user role
type ReadWorkspaceUserRoleUseCase struct {
	repositories ReadWorkspaceUserRoleRepositories
	services     ReadWorkspaceUserRoleServices
}

// NewReadWorkspaceUserRoleUseCase creates use case with grouped dependencies
func NewReadWorkspaceUserRoleUseCase(
	repositories ReadWorkspaceUserRoleRepositories,
	services ReadWorkspaceUserRoleServices,
) *ReadWorkspaceUserRoleUseCase {
	return &ReadWorkspaceUserRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read workspace user role operation
func (uc *ReadWorkspaceUserRoleUseCase) Execute(ctx context.Context, req *workspaceuserrolepb.ReadWorkspaceUserRoleRequest) (*workspaceuserrolepb.ReadWorkspaceUserRoleResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.user_not_authenticated", "User not authenticated "))
		}

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
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.request_required", "Request is required for workspace user roles "))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.id_required", "Workspace-User-Role ID is required "))
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUserRole.ReadWorkspaceUserRole(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.read_failed", "Failed to read workspace user role ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// Helper functions
