package subscription_workspace_user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// CreateSubscriptionWorkspaceUserRepositories groups all repository dependencies
type CreateSubscriptionWorkspaceUserRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer // Primary entity repository
	Subscription              subscriptionpb.SubscriptionDomainServiceServer                           // client_id stamping + FK validation
	ClientWorkspaceUser       clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer             // composite-FK pre-check
}

// CreateSubscriptionWorkspaceUserServices groups all business service dependencies
type CreateSubscriptionWorkspaceUserServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateSubscriptionWorkspaceUserUseCase handles the business logic for creating subscription workspace users
type CreateSubscriptionWorkspaceUserUseCase struct {
	repositories CreateSubscriptionWorkspaceUserRepositories
	services     CreateSubscriptionWorkspaceUserServices
}

// NewCreateSubscriptionWorkspaceUserUseCase creates use case with grouped dependencies
func NewCreateSubscriptionWorkspaceUserUseCase(
	repositories CreateSubscriptionWorkspaceUserRepositories,
	services CreateSubscriptionWorkspaceUserServices,
) *CreateSubscriptionWorkspaceUserUseCase {
	return &CreateSubscriptionWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create subscription workspace user operation.
//
// Two enforced rules:
//  1. client_id is stamped from the parent subscription (single-write boundary),
//     NEVER taken from caller input.
//  2. Composite-FK pre-check: the (client_id, workspace_user_id) pair must already
//     be an ACTIVE client_workspace_user row (project servicers must be a subset
//     of the account team). A clean domain error is returned BEFORE the DB
//     composite FK fires.
func (uc *CreateSubscriptionWorkspaceUserUseCase) Execute(ctx context.Context, req *subscriptionworkspaceuserpb.CreateSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.CreateSubscriptionWorkspaceUserResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionWorkspaceUser, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Rule 1: stamp client_id from the subscription (single-write boundary).
	if err := uc.stampClientFromSubscription(ctx, req.Data); err != nil {
		return nil, err
	}

	// Rule 2: composite-FK pre-check (project servicer ⊆ account team).
	if err := uc.assertOnAccountTeam(ctx, req.Data.ClientId, req.Data.WorkspaceUserId); err != nil {
		return nil, err
	}

	uc.enrich(req.Data)

	resp, err := uc.repositories.SubscriptionWorkspaceUser.CreateSubscriptionWorkspaceUser(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.creation_failed", "Subscription workspace user creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *CreateSubscriptionWorkspaceUserUseCase) validateInput(ctx context.Context, req *subscriptionworkspaceuserpb.CreateSubscriptionWorkspaceUserRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.SubscriptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.subscription_id_required", "Subscription ID is required [DEFAULT]"))
	}
	if req.Data.WorkspaceUserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.workspace_user_id_required", "Workspace user ID is required [DEFAULT]"))
	}
	return nil
}

// stampClientFromSubscription reads the parent subscription, validates it exists +
// is active, and stamps client_id from subscription.client_id.
func (uc *CreateSubscriptionWorkspaceUserUseCase) stampClientFromSubscription(ctx context.Context, swu *subscriptionworkspaceuserpb.SubscriptionWorkspaceUser) error {
	subscription, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: swu.SubscriptionId},
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.subscription_reference_validation_failed", "Failed to validate subscription entity reference [DEFAULT]")
		return fmt.Errorf("%s: %w", translatedError, err)
	}
	if subscription == nil || len(subscription.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.subscription_not_found", "Subscription not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{subscriptionId}", swu.SubscriptionId)
		return errors.New(translatedError)
	}
	if !subscription.Data[0].Active {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.subscription_not_active", "Referenced subscription is not active [DEFAULT]")
		return errors.New(translatedError)
	}
	// Single-write boundary: client_id ALWAYS comes from the subscription.
	swu.ClientId = subscription.Data[0].ClientId
	return nil
}

// assertOnAccountTeam rejects the create unless (clientID, workspaceUserID) is an
// ACTIVE client_workspace_user row. This pre-empts the DB composite FK with a
// clean domain error (project servicers ⊆ account team).
func (uc *CreateSubscriptionWorkspaceUserUseCase) assertOnAccountTeam(ctx context.Context, clientID, workspaceUserID string) error {
	if clientID == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.client_unresolved", "Client could not be resolved from the subscription [DEFAULT]")
		return errors.New(translatedError)
	}
	filter := subscriptionWorkspaceUserAccountTeamFilter(clientID, workspaceUserID)
	listResp, err := uc.repositories.ClientWorkspaceUser.ListClientWorkspaceUsers(ctx, &filter)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.account_team_check_failed", "Failed to verify account-team membership [DEFAULT]")
		return fmt.Errorf("%s: %w", translatedError, err)
	}
	if listResp != nil {
		for _, row := range listResp.Data {
			if row.Active && row.ClientId == clientID && row.WorkspaceUserId == workspaceUserID {
				return nil
			}
		}
	}
	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.not_on_account_team", "This user is not on the client's account team [DEFAULT]")
	return errors.New(translatedError)
}

func (uc *CreateSubscriptionWorkspaceUserUseCase) enrich(swu *subscriptionworkspaceuserpb.SubscriptionWorkspaceUser) {
	now := time.Now()
	if swu.Id == "" {
		swu.Id = uc.services.IDGenerator.GenerateID()
	}
	swu.Active = true
	swu.DateCreated = &[]int64{now.UnixMilli()}[0]
	swu.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	swu.DateModified = &[]int64{now.UnixMilli()}[0]
	swu.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}

// subscriptionWorkspaceUserAccountTeamFilter builds a list request filtered to a
// single (client_id, workspace_user_id) candidate row.
func subscriptionWorkspaceUserAccountTeamFilter(clientID, workspaceUserID string) clientworkspaceuserpb.ListClientWorkspaceUsersRequest {
	return clientworkspaceuserpb.ListClientWorkspaceUsersRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				swuStringEq("client_id", clientID),
				swuStringEq("workspace_user_id", workspaceUserID),
			},
		},
	}
}

// swuStringEq builds a STRING_EQUALS TypedFilter for the given field.
func swuStringEq(field, value string) *commonpb.TypedFilter {
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
