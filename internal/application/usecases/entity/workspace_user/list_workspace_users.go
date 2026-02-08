package workspace_user

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

// ListWorkspaceUsersRepositories groups all repository dependencies
type ListWorkspaceUsersRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // Primary entity repository
	Workspace     workspacepb.WorkspaceDomainServiceServer         // Entity reference validation
	User          userpb.UserDomainServiceServer                   // Entity reference validation
}

// ListWorkspaceUsersServices groups all business service dependencies
type ListWorkspaceUsersServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListWorkspaceUsersUseCase handles the business logic for listing workspace users
type ListWorkspaceUsersUseCase struct {
	repositories ListWorkspaceUsersRepositories
	services     ListWorkspaceUsersServices
}

// NewListWorkspaceUsersUseCase creates use case with grouped dependencies
func NewListWorkspaceUsersUseCase(
	repositories ListWorkspaceUsersRepositories,
	services ListWorkspaceUsersServices,
) *ListWorkspaceUsersUseCase {
	return &ListWorkspaceUsersUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListWorkspaceUsersUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListWorkspaceUsersUseCase with grouped parameters instead
func NewListWorkspaceUsersUseCaseUngrouped(
	workspaceUserRepo workspaceuserpb.WorkspaceUserDomainServiceServer,
) *ListWorkspaceUsersUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListWorkspaceUsersRepositories{
		WorkspaceUser: workspaceUserRepo,
		Workspace:     nil,
		User:          nil,
	}

	services := ListWorkspaceUsersServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListWorkspaceUsersUseCase(repositories, services)
}

// Execute performs the list workspace users operation
func (uc *ListWorkspaceUsersUseCase) Execute(ctx context.Context, req *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	// Input validation
	if req == nil {
		req = &workspaceuserpb.ListWorkspaceUsersRequest{}
	}

	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, ports.ErrUserNotAuthenticated()
		}

		// For list operations, we might need workspace-specific filtering
		// This is a simplified approach - in practice, you might want to filter results
		// based on workspaces the user has access to
		// For now, we'll require a global list permission or workspace-specific context

		// If there's a workspace filter in the request, check permission for that workspace
		// Note: This assumes the protobuf has filtering capabilities - adjust as needed
		// For now, we'll check if user has general list permission

		// TODO: Implement workspace-specific filtering based on user permissions
		// This is a placeholder for more sophisticated permission checking
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUser.ListWorkspaceUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	// Business logic post-processing (if needed)
	// Currently no additional business rules for list operation

	return resp, nil
}
