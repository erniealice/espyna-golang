package workspace

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// ListUserWorkspacesRepositories groups all repository dependencies
type ListUserWorkspacesRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// ListUserWorkspacesServices groups all business service dependencies
type ListUserWorkspacesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListUserWorkspacesUseCase handles the business logic for listing user workspaces
type ListUserWorkspacesUseCase struct {
	repositories ListUserWorkspacesRepositories
	services     ListUserWorkspacesServices
}

// NewListUserWorkspacesUseCase creates use case with grouped dependencies
func NewListUserWorkspacesUseCase(
	repositories ListUserWorkspacesRepositories,
	services ListUserWorkspacesServices,
) *ListUserWorkspacesUseCase {
	return &ListUserWorkspacesUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ListUserWorkspacesUseCase) Execute(ctx context.Context, req *workspacepb.ListUserWorkspacesRequest) (*workspacepb.ListUserWorkspacesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Workspace, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &workspacepb.ListUserWorkspacesRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Workspace.ListUserWorkspaces(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.errors.list_user_workspaces_failed", "Failed to list user workspaces [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
