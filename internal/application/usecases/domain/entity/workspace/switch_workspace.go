package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// SwitchWorkspaceRepositories groups all repository dependencies
type SwitchWorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// SwitchWorkspaceServices groups all business service dependencies
type SwitchWorkspaceServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// SwitchWorkspaceUseCase handles the business logic for switching workspaces
type SwitchWorkspaceUseCase struct {
	repositories SwitchWorkspaceRepositories
	services     SwitchWorkspaceServices
}

// NewSwitchWorkspaceUseCase creates use case with grouped dependencies
func NewSwitchWorkspaceUseCase(
	repositories SwitchWorkspaceRepositories,
	services SwitchWorkspaceServices,
) *SwitchWorkspaceUseCase {
	return &SwitchWorkspaceUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *SwitchWorkspaceUseCase) Execute(ctx context.Context, req *workspacepb.SwitchWorkspaceRequest) (*workspacepb.SwitchWorkspaceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityWorkspace, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.errors.switch_request_required", "switch workspace request is required"))
	}

	// Call repository
	resp, err := uc.repositories.Workspace.SwitchWorkspace(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.errors.switch_failed", "Failed to switch workspace [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
