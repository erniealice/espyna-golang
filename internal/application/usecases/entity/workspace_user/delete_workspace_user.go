package workspace_user

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspaceUser, ports.ActionDelete); err != nil {
		return nil, err
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
