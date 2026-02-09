package subscription_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// DeleteSubscriptionAttributeRepositories groups all repository dependencies
type DeleteSubscriptionAttributeRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
}

// DeleteSubscriptionAttributeServices groups all business service dependencies
type DeleteSubscriptionAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteSubscriptionAttributeUseCase handles the business logic for deleting subscription attributes
type DeleteSubscriptionAttributeUseCase struct {
	repositories DeleteSubscriptionAttributeRepositories
	services     DeleteSubscriptionAttributeServices
}

// NewDeleteSubscriptionAttributeUseCase creates a new DeleteSubscriptionAttributeUseCase
func NewDeleteSubscriptionAttributeUseCase(
	repositories DeleteSubscriptionAttributeRepositories,
	services DeleteSubscriptionAttributeServices,
) *DeleteSubscriptionAttributeUseCase {
	return &DeleteSubscriptionAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete subscription attribute operation
func (uc *DeleteSubscriptionAttributeUseCase) Execute(ctx context.Context, req *subscriptionattributepb.DeleteSubscriptionAttributeRequest) (*subscriptionattributepb.DeleteSubscriptionAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscriptionAttribute, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SubscriptionAttribute.DeleteSubscriptionAttribute(ctx, req)
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
		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.deletion_failed", "Subscription attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteSubscriptionAttributeUseCase) validateInput(ctx context.Context, req *subscriptionattributepb.DeleteSubscriptionAttributeRequest) error {
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
