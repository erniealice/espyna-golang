package subscription_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// ListSubscriptionAttributesRepositories groups all repository dependencies
type ListSubscriptionAttributesRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
}

// ListSubscriptionAttributesServices groups all business service dependencies
type ListSubscriptionAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListSubscriptionAttributesUseCase handles the business logic for listing subscription attributes
type ListSubscriptionAttributesUseCase struct {
	repositories ListSubscriptionAttributesRepositories
	services     ListSubscriptionAttributesServices
}

// NewListSubscriptionAttributesUseCase creates a new ListSubscriptionAttributesUseCase
func NewListSubscriptionAttributesUseCase(
	repositories ListSubscriptionAttributesRepositories,
	services ListSubscriptionAttributesServices,
) *ListSubscriptionAttributesUseCase {
	return &ListSubscriptionAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list subscription attributes operation
func (uc *ListSubscriptionAttributesUseCase) Execute(ctx context.Context, req *subscriptionattributepb.ListSubscriptionAttributesRequest) (*subscriptionattributepb.ListSubscriptionAttributesResponse, error) {
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
	resp, err := uc.repositories.SubscriptionAttribute.ListSubscriptionAttributes(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListSubscriptionAttributesUseCase) validateInput(ctx context.Context, req *subscriptionattributepb.ListSubscriptionAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
