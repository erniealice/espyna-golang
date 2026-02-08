package subscription_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// GetSubscriptionAttributeItemPageDataRepositories groups all repository dependencies
type GetSubscriptionAttributeItemPageDataRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
}

// GetSubscriptionAttributeItemPageDataServices groups all business service dependencies
type GetSubscriptionAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetSubscriptionAttributeItemPageDataUseCase handles the business logic for getting subscription attribute item page data
type GetSubscriptionAttributeItemPageDataUseCase struct {
	repositories GetSubscriptionAttributeItemPageDataRepositories
	services     GetSubscriptionAttributeItemPageDataServices
}

// NewGetSubscriptionAttributeItemPageDataUseCase creates a new GetSubscriptionAttributeItemPageDataUseCase
func NewGetSubscriptionAttributeItemPageDataUseCase(
	repositories GetSubscriptionAttributeItemPageDataRepositories,
	services GetSubscriptionAttributeItemPageDataServices,
) *GetSubscriptionAttributeItemPageDataUseCase {
	return &GetSubscriptionAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get subscription attribute item page data operation
func (uc *GetSubscriptionAttributeItemPageDataUseCase) Execute(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeItemPageDataRequest) (*subscriptionattributepb.GetSubscriptionAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.SubscriptionAttribute.GetSubscriptionAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.item_page_data_failed", "Failed to retrieve subscription attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetSubscriptionAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *subscriptionattributepb.GetSubscriptionAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.request_required", "Request is required for subscription attributes [DEFAULT]"))
	}

	// Validate subscription attribute ID - uses direct field req.SubscriptionAttributeId
	if strings.TrimSpace(req.SubscriptionAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.id_required", "Subscription attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.SubscriptionAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.validation.id_too_short", "Subscription attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
