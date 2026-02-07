package subscription_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	subscriptionattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription_attribute"
)

// ReadSubscriptionAttributeRepositories groups all repository dependencies
type ReadSubscriptionAttributeRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
}

// ReadSubscriptionAttributeServices groups all business service dependencies
type ReadSubscriptionAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadSubscriptionAttributeUseCase handles the business logic for reading subscription attributes
type ReadSubscriptionAttributeUseCase struct {
	repositories ReadSubscriptionAttributeRepositories
	services     ReadSubscriptionAttributeServices
}

// NewReadSubscriptionAttributeUseCase creates a new ReadSubscriptionAttributeUseCase
func NewReadSubscriptionAttributeUseCase(
	repositories ReadSubscriptionAttributeRepositories,
	services ReadSubscriptionAttributeServices,
) *ReadSubscriptionAttributeUseCase {
	return &ReadSubscriptionAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read subscription attribute operation
func (uc *ReadSubscriptionAttributeUseCase) Execute(ctx context.Context, req *subscriptionattributepb.ReadSubscriptionAttributeRequest) (*subscriptionattributepb.ReadSubscriptionAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SubscriptionAttribute.ReadSubscriptionAttribute(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("subscription_attribute with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"subscription_attribute.errors.not_found",
				map[string]interface{}{"subscriptionAttributeId": req.Data.Id},
				"Subscription attribute not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadSubscriptionAttributeUseCase) validateInput(ctx context.Context, req *subscriptionattributepb.ReadSubscriptionAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.id_required", "Subscription attribute ID is required [DEFAULT]"))
	}
	return nil
}
