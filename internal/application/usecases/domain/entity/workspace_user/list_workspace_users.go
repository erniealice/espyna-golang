package workspace_user

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
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
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkspaceUser,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
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
