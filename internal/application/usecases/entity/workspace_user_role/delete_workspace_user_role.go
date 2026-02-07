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

// DeleteWorkspaceUserRoleRepositories groups all repository dependencies
type DeleteWorkspaceUserRoleRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer // Primary entity repository
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer         // Entity reference validation
	Role              rolepb.RoleDomainServiceServer                           // Entity reference validation
}

// DeleteWorkspaceUserRoleServices groups all business service dependencies
type DeleteWorkspaceUserRoleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteWorkspaceUserRoleUseCase handles the business logic for deleting a workspace user role
type DeleteWorkspaceUserRoleUseCase struct {
	repositories DeleteWorkspaceUserRoleRepositories
	services     DeleteWorkspaceUserRoleServices
}

// NewDeleteWorkspaceUserRoleUseCase creates use case with grouped dependencies
func NewDeleteWorkspaceUserRoleUseCase(
	repositories DeleteWorkspaceUserRoleRepositories,
	services DeleteWorkspaceUserRoleServices,
) *DeleteWorkspaceUserRoleUseCase {
	return &DeleteWorkspaceUserRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete workspace user role operation
func (uc *DeleteWorkspaceUserRoleUseCase) Execute(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.user_not_authenticated", "User not authenticated "))
		}

		// Check permission to delete workspace user roles
		permission := ports.EntityPermission(ports.EntityWorkspaceUserRole, ports.ActionDelete)
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

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workspace user role deletion within a transaction
func (uc *DeleteWorkspaceUserRoleUseCase) executeWithTransaction(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	var result *workspaceuserrolepb.DeleteWorkspaceUserRoleResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workspace_user_role.errors.deletion_failed", "Workspace-User-Role deletion failed ")
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
func (uc *DeleteWorkspaceUserRoleUseCase) executeCore(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.request_required", "Request is required for workspace user roles "))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.id_required", "Workspace-User-Role ID is required "))
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUserRole.DeleteWorkspaceUserRole(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.deletion_failed", "Workspace-User-Role deletion failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// Helper functions
