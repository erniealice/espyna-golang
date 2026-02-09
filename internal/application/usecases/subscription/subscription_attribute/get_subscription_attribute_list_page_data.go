package subscription_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// GetSubscriptionAttributeListPageDataRepositories groups all repository dependencies
type GetSubscriptionAttributeListPageDataRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
}

// GetSubscriptionAttributeListPageDataServices groups all business service dependencies
type GetSubscriptionAttributeListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscriptionAttribute, ports.ActionList); err != nil {
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
