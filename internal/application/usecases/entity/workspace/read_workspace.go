package workspace

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// ReadWorkspaceRepositories groups all repository dependencies
type ReadWorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// ReadWorkspaceServices groups all business service dependencies
type ReadWorkspaceServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadWorkspaceUseCase handles the business logic for reading a workspace
type ReadWorkspaceUseCase struct {
	repositories ReadWorkspaceRepositories
	services     ReadWorkspaceServices
}

// NewReadWorkspaceUseCase creates use case with grouped dependencies
func NewReadWorkspaceUseCase(
	repositories ReadWorkspaceRepositories,
	services ReadWorkspaceServices,
) *ReadWorkspaceUseCase {
	return &ReadWorkspaceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadWorkspaceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadWorkspaceUseCase with grouped parameters instead
func NewReadWorkspaceUseCaseUngrouped(workspaceRepo workspacepb.WorkspaceDomainServiceServer) *ReadWorkspaceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadWorkspaceRepositories{
		Workspace: workspaceRepo,
	}

	services := ReadWorkspaceServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadWorkspaceUseCase(repositories, services)
}

// Execute performs the read workspace operation
func (uc *ReadWorkspaceUseCase) Execute(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Workspace.ReadWorkspace(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadWorkspaceUseCase) validateInput(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.request_required", "Request is required for workspaces [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.data_required", "Workspace data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.id_required", "Workspace ID is required [DEFAULT]"))
	}
	return nil
}
