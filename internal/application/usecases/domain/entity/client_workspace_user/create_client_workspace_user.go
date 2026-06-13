package client_workspace_user

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// CreateClientWorkspaceUserRepositories groups all repository dependencies
type CreateClientWorkspaceUserRepositories struct {
	ClientWorkspaceUser clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer // Primary entity repository
	Client              clientpb.ClientDomainServiceServer                           // FK validation
}

// CreateClientWorkspaceUserServices groups all business service dependencies
type CreateClientWorkspaceUserServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateClientWorkspaceUserUseCase handles the business logic for creating client workspace users
type CreateClientWorkspaceUserUseCase struct {
	repositories CreateClientWorkspaceUserRepositories
	services     CreateClientWorkspaceUserServices
}

// NewCreateClientWorkspaceUserUseCase creates use case with grouped dependencies
func NewCreateClientWorkspaceUserUseCase(
	repositories CreateClientWorkspaceUserRepositories,
	services CreateClientWorkspaceUserServices,
) *CreateClientWorkspaceUserUseCase {
	return &CreateClientWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create client workspace user operation.
//
// Single-owner-per-client invariant: when is_owner=true, no OTHER active row for
// the same client may already be is_owner=true. Enforced transactionally here;
// the DB partial-unique (client_id) WHERE is_owner is the backstop (this surfaces
// a clean domain error before the raw constraint fires).
func (uc *CreateClientWorkspaceUserUseCase) Execute(ctx context.Context, req *clientworkspaceuserpb.CreateClientWorkspaceUserRequest) (*clientworkspaceuserpb.CreateClientWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ClientWorkspaceUser,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	uc.enrich(req.Data)

	var resp *clientworkspaceuserpb.CreateClientWorkspaceUserResponse
	run := func(txCtx context.Context) error {
		if req.Data.IsOwner {
			if err := ensureNoOtherOwner(txCtx, uc.repositories.ClientWorkspaceUser, uc.services.Translator, req.Data.ClientId, ""); err != nil {
				return err
			}
		}
		r, err := uc.repositories.ClientWorkspaceUser.CreateClientWorkspaceUser(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "client_workspace_user.errors.creation_failed", "Client workspace user creation failed [DEFAULT]")
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

func (uc *CreateClientWorkspaceUserUseCase) validateInput(ctx context.Context, req *clientworkspaceuserpb.CreateClientWorkspaceUserRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.client_id_required", "Client ID is required [DEFAULT]"))
	}
	if req.Data.WorkspaceUserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_workspace_user.validation.workspace_user_id_required", "Workspace user ID is required [DEFAULT]"))
	}
	return nil
}

func (uc *CreateClientWorkspaceUserUseCase) enrich(cwu *clientworkspaceuserpb.ClientWorkspaceUser) {
	now := time.Now()
	if cwu.Id == "" {
		cwu.Id = uc.services.IDGenerator.GenerateID()
	}
	cwu.Active = true
	cwu.DateCreated = &[]int64{now.UnixMilli()}[0]
	cwu.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	cwu.DateModified = &[]int64{now.UnixMilli()}[0]
	cwu.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}

// ensureNoOtherOwner returns a clean domain error if any active client_workspace_user
// row for clientID (other than excludeID) already has is_owner=true.
func ensureNoOtherOwner(ctx context.Context, repo clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer, tr ports.Translator, clientID, excludeID string) error {
	listResp, err := repo.ListClientWorkspaceUsers(ctx, &clientworkspaceuserpb.ListClientWorkspaceUsersRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{cwuStringEq("client_id", clientID)},
		},
	})
	if err != nil {
		return err
	}
	if listResp == nil {
		return nil
	}
	for _, row := range listResp.Data {
		if row.Id == excludeID {
			continue
		}
		if row.Active && row.IsOwner {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr, "client_workspace_user.errors.owner_already_exists", "This client already has an owner [DEFAULT]"))
		}
	}
	return nil
}

// cwuStringEq builds a STRING_EQUALS TypedFilter for the given field.
func cwuStringEq(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    value,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}
}
