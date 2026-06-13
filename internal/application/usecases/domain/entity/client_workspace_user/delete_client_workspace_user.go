package client_workspace_user

import (
	"context"
	"errors"
	"fmt"

	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// DeleteClientWorkspaceUserRepositories groups all repository dependencies
type DeleteClientWorkspaceUserRepositories struct {
	ClientWorkspaceUser clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer
}

// DeleteClientWorkspaceUserServices groups all business service dependencies
type DeleteClientWorkspaceUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteClientWorkspaceUserUseCase handles the business logic for deleting client workspace users (soft-delete)
type DeleteClientWorkspaceUserUseCase struct {
	repositories DeleteClientWorkspaceUserRepositories
	services     DeleteClientWorkspaceUserServices
}

// NewDeleteClientWorkspaceUserUseCase creates a new DeleteClientWorkspaceUserUseCase
func NewDeleteClientWorkspaceUserUseCase(
	repositories DeleteClientWorkspaceUserRepositories,
	services DeleteClientWorkspaceUserServices,
) *DeleteClientWorkspaceUserUseCase {
	return &DeleteClientWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete client workspace user operation (soft-delete: active=false).
func (uc *DeleteClientWorkspaceUserUseCase) Execute(ctx context.Context, req *clientworkspaceuserpb.DeleteClientWorkspaceUserRequest) (*clientworkspaceuserpb.DeleteClientWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ClientWorkspaceUser,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ClientWorkspaceUser.DeleteClientWorkspaceUser(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.errors.deletion_failed", "Client workspace user deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *DeleteClientWorkspaceUserUseCase) validateInput(ctx context.Context, req *clientworkspaceuserpb.DeleteClientWorkspaceUserRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.id_required", "Client workspace user ID is required [DEFAULT]"))
	}
	return nil
}
