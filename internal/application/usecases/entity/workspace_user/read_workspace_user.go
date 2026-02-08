package workspace_user

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

// ReadWorkspaceUserRepositories groups all repository dependencies
type ReadWorkspaceUserRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // Primary entity repository
	Workspace     workspacepb.WorkspaceDomainServiceServer         // Entity reference validation
	User          userpb.UserDomainServiceServer                   // Entity reference validation
}

// ReadWorkspaceUserServices groups all business service dependencies
type ReadWorkspaceUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadWorkspaceUserUseCase handles the business logic for reading a workspace user
type ReadWorkspaceUserUseCase struct {
	repositories ReadWorkspaceUserRepositories
	services     ReadWorkspaceUserServices
}

// NewReadWorkspaceUserUseCase creates use case with grouped dependencies
func NewReadWorkspaceUserUseCase(
	repositories ReadWorkspaceUserRepositories,
	services ReadWorkspaceUserServices,
) *ReadWorkspaceUserUseCase {
	return &ReadWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadWorkspaceUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadWorkspaceUserUseCase with grouped parameters instead
func NewReadWorkspaceUserUseCaseUngrouped(
	workspaceUserRepo workspaceuserpb.WorkspaceUserDomainServiceServer,
) *ReadWorkspaceUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadWorkspaceUserRepositories{
		WorkspaceUser: workspaceUserRepo,
		Workspace:     nil,
		User:          nil,
	}

	services := ReadWorkspaceUserServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadWorkspaceUserUseCase(repositories, services)
}

// Execute performs the read workspace user operation
func (uc *ReadWorkspaceUserUseCase) Execute(ctx context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
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

		// For read operations, we might need to read the entity first to get workspace context
		// Or we can require workspace ID to be passed in the request
		// For simplicity, we'll check if user has read permission (workspace context would be determined by the repository)

		// TODO: Implement more sophisticated authorization for read operations
		// This might involve reading the entity first to get workspace context
	}

	// Call repository with intelligent error handling
	resp, err := uc.repositories.WorkspaceUser.ReadWorkspaceUser(ctx, req)
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

	// Return successful response
	return resp, nil
}
