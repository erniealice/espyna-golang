package client_workspace_user

import (
	"context"
	"errors"

	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// ListClientWorkspaceUsersRepositories groups all repository dependencies
type ListClientWorkspaceUsersRepositories struct {
	ClientWorkspaceUser clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer
}

// ListClientWorkspaceUsersServices groups all business service dependencies
type ListClientWorkspaceUsersServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListClientWorkspaceUsersUseCase handles the business logic for listing client workspace users
type ListClientWorkspaceUsersUseCase struct {
	repositories ListClientWorkspaceUsersRepositories
	services     ListClientWorkspaceUsersServices
}

// NewListClientWorkspaceUsersUseCase creates a new ListClientWorkspaceUsersUseCase
func NewListClientWorkspaceUsersUseCase(
	repositories ListClientWorkspaceUsersRepositories,
	services ListClientWorkspaceUsersServices,
) *ListClientWorkspaceUsersUseCase {
	return &ListClientWorkspaceUsersUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list client workspace users operation. Filters by
// client_id and workspace_user_id ride on req.Filters.
func (uc *ListClientWorkspaceUsersUseCase) Execute(ctx context.Context, req *clientworkspaceuserpb.ListClientWorkspaceUsersRequest) (*clientworkspaceuserpb.ListClientWorkspaceUsersResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ClientWorkspaceUser,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ClientWorkspaceUser.ListClientWorkspaceUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ListClientWorkspaceUsersUseCase) validateInput(ctx context.Context, req *clientworkspaceuserpb.ListClientWorkspaceUsersRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
