package subscription_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// GetSubscriptionAttributeListPageDataRepositories groups all repository dependencies
type GetSubscriptionAttributeListPageDataRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
}

// GetSubscriptionAttributeListPageDataServices groups all business service dependencies
type GetSubscriptionAttributeListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetSubscriptionAttributeListPageDataUseCase handles the business logic for getting subscription attribute list page data
type GetSubscriptionAttributeListPageDataUseCase struct {
	repositories GetSubscriptionAttributeListPageDataRepositories
	services     GetSubscriptionAttributeListPageDataServices
}

// NewGetSubscriptionAttributeListPageDataUseCase creates a new GetSubscriptionAttributeListPageDataUseCase
func NewGetSubscriptionAttributeListPageDataUseCase(
	repositories GetSubscriptionAttributeListPageDataRepositories,
	services GetSubscriptionAttributeListPageDataServices,
) *GetSubscriptionAttributeListPageDataUseCase {
	return &GetSubscriptionAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get subscription attribute list page data operation
func (uc *GetSubscriptionAttributeListPageDataUseCase) Execute(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeListPageDataRequest) (*subscriptionattributepb.GetSubscriptionAttributeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionAttribute, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SubscriptionAttribute.GetSubscriptionAttributeListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetSubscriptionAttributeListPageDataUseCase) validateInput(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
