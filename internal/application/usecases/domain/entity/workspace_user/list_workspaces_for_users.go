package workspace_user

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

// ListWorkspacesForUsersRepositories groups all repository dependencies
type ListWorkspacesForUsersRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // Primary entity repository
}

// ListWorkspacesForUsersServices groups all business service dependencies
type ListWorkspacesForUsersServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListWorkspacesForUsersUseCase handles the business logic for listing
// workspace memberships grouped by user.
type ListWorkspacesForUsersUseCase struct {
	repositories ListWorkspacesForUsersRepositories
	services     ListWorkspacesForUsersServices
}

// NewListWorkspacesForUsersUseCase creates use case with grouped dependencies
func NewListWorkspacesForUsersUseCase(
	repositories ListWorkspacesForUsersRepositories,
	services ListWorkspacesForUsersServices,
) *ListWorkspacesForUsersUseCase {
	return &ListWorkspacesForUsersUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list-workspaces-for-users operation.
func (uc *ListWorkspacesForUsersUseCase) Execute(
	ctx context.Context,
	req *workspaceuserpb.ListWorkspacesForUsersRequest,
) (*workspaceuserpb.ListWorkspacesForUsersResponse, error) {
	if req == nil {
		req = &workspaceuserpb.ListWorkspacesForUsersRequest{}
	}

	// Authorization check — reuses the WorkspaceUser entity + List action
	// because this query is a read-only projection of workspace_user data.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkspaceUser,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	return uc.repositories.WorkspaceUser.ListWorkspacesForUsers(ctx, req)
}
