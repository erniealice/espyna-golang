package workspace_user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

// DeleteWorkspaceUserRepositories groups all repository dependencies
type DeleteWorkspaceUserRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // Primary entity repository
	Workspace     workspacepb.WorkspaceDomainServiceServer         // Entity reference validation
	User          userpb.UserDomainServiceServer                   // Entity reference validation
}

// DeleteWorkspaceUserServices groups all business service dependencies
type DeleteWorkspaceUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteWorkspaceUserUseCase handles the business logic for deleting a workspace user
type DeleteWorkspaceUserUseCase struct {
	repositories DeleteWorkspaceUserRepositories
	services     DeleteWorkspaceUserServices
}

// NewDeleteWorkspaceUserUseCase creates use case with grouped dependencies
func NewDeleteWorkspaceUserUseCase(
	repositories DeleteWorkspaceUserRepositories,
	services DeleteWorkspaceUserServices,
) *DeleteWorkspaceUserUseCase {
	return &DeleteWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteWorkspaceUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteWorkspaceUserUseCase with grouped parameters instead
func NewDeleteWorkspaceUserUseCaseUngrouped(
	workspaceUserRepo workspaceuserpb.WorkspaceUserDomainServiceServer,
) *DeleteWorkspaceUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteWorkspaceUserRepositories{
		WorkspaceUser: workspaceUserRepo,
		Workspace:     nil,
		User:          nil,
	}

	services := DeleteWorkspaceUserServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteWorkspaceUserUseCase(repositories, services)
}

// Execute performs the delete workspace user operation
func (uc *DeleteWorkspaceUserUseCase) Execute(ctx context.Context, req *workspaceuserpb.DeleteWorkspaceUserRequest) (*workspaceuserpb.DeleteWorkspaceUserResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.request_required", "Request is required for workspace users")
		return nil, errors.New(translatedError)
	}

	if req.Data.Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.id_required", "Workspace-User ID is required")
		return nil, errors.New(translatedError)
	}

	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, ports.ErrUserNotAuthenticated()
		}

		// For delete operations, we need to read the existing entity first to get workspace context
		existingEntity, err := uc.repositories.WorkspaceUser.ReadWorkspaceUser(ctx, &workspaceuserpb.ReadWorkspaceUserRequest{
			Data: &workspaceuserpb.WorkspaceUser{Id: req.Data.Id},
		})
		if err != nil {
			// If we can't read the workspace user for authorization, return authorization failed
			// This covers both "not found" errors and other read errors
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.authorization_failed", "Authorization failed for workspace users")
			return nil, errors.New(translatedError)
		}
		if existingEntity == nil || existingEntity.Data == nil || len(existingEntity.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.authorization_failed", "Authorization failed for workspace users")
			return nil, errors.New(translatedError)
		}

		workspaceID := existingEntity.Data[0].WorkspaceId
		if workspaceID == "" {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.workspace_id_required", "Workspace ID is required for authorization check")
			return nil, errors.New(translatedError)
		}

		permission := ports.EntityPermission(ports.EntityWorkspaceUser, ports.ActionDelete)
		authorized, err := uc.services.AuthorizationService.HasPermissionInWorkspace(ctx, userID, workspaceID, permission)
		if err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}

		if !authorized {
			return nil, ports.ErrWorkspaceAccessDenied(userID, workspaceID).WithDetails("permission", permission)
		}
	}

	// Call repository with intelligent error handling
	resp, err := uc.repositories.WorkspaceUser.DeleteWorkspaceUser(ctx, req)
	if err != nil {
		// Check if this is a "not found" error - if so, create our own translated message
		// Otherwise, return the error as-is for the calling function to handle
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.not_found", "Workspace-User with ID \"{workspaceUserId}\" not found")
			translatedError = strings.ReplaceAll(translatedError, "{workspaceUserId}", req.Data.Id)
			return nil, errors.New(translatedError)
		}
		// For other errors, return as-is
		return nil, err
	}

	// Business logic post-processing (if needed)
	// Currently no additional business rules for delete operation

	return resp, nil
}
