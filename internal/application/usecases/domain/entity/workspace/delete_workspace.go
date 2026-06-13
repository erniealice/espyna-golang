package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// DeleteWorkspaceRepositories groups all repository dependencies
type DeleteWorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// DeleteWorkspaceServices groups all business service dependencies
type DeleteWorkspaceServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewDeleteWorkspaceUseCase(repositories, services)
}

func (uc *DeleteWorkspaceUseCase) Execute(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Workspace,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.request_required", "Request is required for workspaces [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.id_required", "Workspace ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Workspace.DeleteWorkspace(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.errors.deletion_failed", "Workspace deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
