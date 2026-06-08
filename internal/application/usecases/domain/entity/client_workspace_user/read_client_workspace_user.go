package client_workspace_user

import (
	"context"
	"errors"

	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// ReadClientWorkspaceUserRepositories groups all repository dependencies
type ReadClientWorkspaceUserRepositories struct {
	ClientWorkspaceUser clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer
}

// ReadClientWorkspaceUserServices groups all business service dependencies
type ReadClientWorkspaceUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadClientWorkspaceUserUseCase handles the business logic for reading client workspace users
type ReadClientWorkspaceUserUseCase struct {
	repositories ReadClientWorkspaceUserRepositories
	services     ReadClientWorkspaceUserServices
}

// NewReadClientWorkspaceUserUseCase creates a new ReadClientWorkspaceUserUseCase
func NewReadClientWorkspaceUserUseCase(
	repositories ReadClientWorkspaceUserRepositories,
	services ReadClientWorkspaceUserServices,
) *ReadClientWorkspaceUserUseCase {
	return &ReadClientWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read client workspace user operation
func (uc *ReadClientWorkspaceUserUseCase) Execute(ctx context.Context, req *clientworkspaceuserpb.ReadClientWorkspaceUserRequest) (*clientworkspaceuserpb.ReadClientWorkspaceUserResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.ClientWorkspaceUser, entityid.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ClientWorkspaceUser.ReadClientWorkspaceUser(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ReadClientWorkspaceUserUseCase) validateInput(ctx context.Context, req *clientworkspaceuserpb.ReadClientWorkspaceUserRequest) error {
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
