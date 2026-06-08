package client_workspace_user

import (
	"context"
	"errors"
	"fmt"
	"time"

	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// UpdateClientWorkspaceUserRepositories groups all repository dependencies
type UpdateClientWorkspaceUserRepositories struct {
	ClientWorkspaceUser clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer
}

// UpdateClientWorkspaceUserServices groups all business service dependencies
type UpdateClientWorkspaceUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateClientWorkspaceUserUseCase handles the business logic for updating client workspace users
type UpdateClientWorkspaceUserUseCase struct {
	repositories UpdateClientWorkspaceUserRepositories
	services     UpdateClientWorkspaceUserServices
}

// NewUpdateClientWorkspaceUserUseCase creates a new UpdateClientWorkspaceUserUseCase
func NewUpdateClientWorkspaceUserUseCase(
	repositories UpdateClientWorkspaceUserRepositories,
	services UpdateClientWorkspaceUserServices,
) *UpdateClientWorkspaceUserUseCase {
	return &UpdateClientWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update client workspace user operation.
//
// Single-owner-per-client invariant: when is_owner is being set true, no OTHER
// active row for the same client may already be is_owner=true (transactional
// check; DB partial-unique is the backstop).
func (uc *UpdateClientWorkspaceUserUseCase) Execute(ctx context.Context, req *clientworkspaceuserpb.UpdateClientWorkspaceUserRequest) (*clientworkspaceuserpb.UpdateClientWorkspaceUserResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.ClientWorkspaceUser, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	uc.enrich(req.Data)

	var resp *clientworkspaceuserpb.UpdateClientWorkspaceUserResponse
	run := func(txCtx context.Context) error {
		if req.Data.IsOwner && req.Data.ClientId != "" {
			if err := ensureNoOtherOwner(txCtx, uc.repositories.ClientWorkspaceUser, uc.services.Translator, req.Data.ClientId, req.Data.Id); err != nil {
				return err
			}
		}
		r, err := uc.repositories.ClientWorkspaceUser.UpdateClientWorkspaceUser(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "client_workspace_user.errors.update_failed", "Client workspace user update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		resp = r
		return nil
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, run); err != nil {
			return nil, err
		}
		return resp, nil
	}
	if err := run(ctx); err != nil {
		return nil, err
	}
	return resp, nil
}

func (uc *UpdateClientWorkspaceUserUseCase) validateInput(ctx context.Context, req *clientworkspaceuserpb.UpdateClientWorkspaceUserRequest) error {
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

func (uc *UpdateClientWorkspaceUserUseCase) enrich(cwu *clientworkspaceuserpb.ClientWorkspaceUser) {
	now := time.Now()
	cwu.DateModified = &[]int64{now.UnixMilli()}[0]
	cwu.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}
