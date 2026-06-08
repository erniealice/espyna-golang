package subscription_workspace_user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"
)

// GetSubscriptionWorkspaceUserItemPageDataRepositories groups all repository dependencies
type GetSubscriptionWorkspaceUserItemPageDataRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
}

// GetSubscriptionWorkspaceUserItemPageDataServices groups all business service dependencies
type GetSubscriptionWorkspaceUserItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetSubscriptionWorkspaceUserItemPageDataUseCase handles the business logic for getting subscription workspace user item page data
type GetSubscriptionWorkspaceUserItemPageDataUseCase struct {
	repositories GetSubscriptionWorkspaceUserItemPageDataRepositories
	services     GetSubscriptionWorkspaceUserItemPageDataServices
}

// NewGetSubscriptionWorkspaceUserItemPageDataUseCase creates a new GetSubscriptionWorkspaceUserItemPageDataUseCase
func NewGetSubscriptionWorkspaceUserItemPageDataUseCase(
	repositories GetSubscriptionWorkspaceUserItemPageDataRepositories,
	services GetSubscriptionWorkspaceUserItemPageDataServices,
) *GetSubscriptionWorkspaceUserItemPageDataUseCase {
	return &GetSubscriptionWorkspaceUserItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get subscription workspace user item page data operation
func (uc *GetSubscriptionWorkspaceUserItemPageDataUseCase) Execute(ctx context.Context, req *subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserItemPageDataRequest) (*subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionWorkspaceUser, entityid.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.SubscriptionWorkspaceUser.GetSubscriptionWorkspaceUserItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.item_page_data_failed", "Failed to retrieve subscription workspace user item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *GetSubscriptionWorkspaceUserItemPageDataUseCase) validateInput(ctx context.Context, req *subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.SubscriptionWorkspaceUserId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.id_required", "Subscription workspace user ID is required [DEFAULT]"))
	}
	return nil
}
