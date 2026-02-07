package workspace

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	workspacepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace"
)

// DeleteWorkspaceRepositories groups all repository dependencies
type DeleteWorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// DeleteWorkspaceServices groups all business service dependencies
type DeleteWorkspaceServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteWorkspaceUseCase handles the business logic for deleting a workspace
type DeleteWorkspaceUseCase struct {
	repositories DeleteWorkspaceRepositories
	services     DeleteWorkspaceServices
}

// NewDeleteWorkspaceUseCase creates use case with grouped dependencies
func NewDeleteWorkspaceUseCase(
	repositories DeleteWorkspaceRepositories,
	services DeleteWorkspaceServices,
) *DeleteWorkspaceUseCase {
	return &DeleteWorkspaceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteWorkspaceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteWorkspaceUseCase with grouped parameters instead
func NewDeleteWorkspaceUseCaseUngrouped(workspaceRepo workspacepb.WorkspaceDomainServiceServer) *DeleteWorkspaceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteWorkspaceRepositories{
		Workspace: workspaceRepo,
	}

	services := DeleteWorkspaceServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteWorkspaceUseCase(repositories, services)
}

func (uc *DeleteWorkspaceUseCase) Execute(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.request_required", "Request is required for workspaces [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.id_required", "Workspace ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Workspace.DeleteWorkspace(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.errors.deletion_failed", "Workspace deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
