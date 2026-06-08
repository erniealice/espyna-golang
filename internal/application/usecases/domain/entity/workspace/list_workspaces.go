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

// ListWorkspacesRepositories groups all repository dependencies
type ListWorkspacesRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// ListWorkspacesServices groups all business service dependencies
type ListWorkspacesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListWorkspacesUseCase handles the business logic for listing workspaces
type ListWorkspacesUseCase struct {
	repositories ListWorkspacesRepositories
	services     ListWorkspacesServices
}

// NewListWorkspacesUseCase creates use case with grouped dependencies
func NewListWorkspacesUseCase(
	repositories ListWorkspacesRepositories,
	services ListWorkspacesServices,
) *ListWorkspacesUseCase {
	return &ListWorkspacesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListWorkspacesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListWorkspacesUseCase with grouped parameters instead
func NewListWorkspacesUseCaseUngrouped(workspaceRepo workspacepb.WorkspaceDomainServiceServer) *ListWorkspacesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListWorkspacesRepositories{
		Workspace: workspaceRepo,
	}

	services := ListWorkspacesServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListWorkspacesUseCase(repositories, services)
}

func (uc *ListWorkspacesUseCase) Execute(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Workspace, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &workspacepb.ListWorkspacesRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Workspace.ListWorkspaces(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.errors.list_failed", "Failed to retrieve workspaces [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
