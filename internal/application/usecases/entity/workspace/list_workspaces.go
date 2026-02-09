package workspace

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// ListWorkspacesRepositories groups all repository dependencies
type ListWorkspacesRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// ListWorkspacesServices groups all business service dependencies
type ListWorkspacesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListWorkspacesUseCase(repositories, services)
}

func (uc *ListWorkspacesUseCase) Execute(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspace, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &workspacepb.ListWorkspacesRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Workspace.ListWorkspaces(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.errors.list_failed", "Failed to retrieve workspaces [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
