package subscription_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// UpdateSubscriptionAttributeRepositories groups all repository dependencies
type UpdateSubscriptionAttributeRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer // Primary entity repository
	Subscription          subscriptionpb.SubscriptionDomainServiceServer                   // Entity reference validation
	Attribute             attributepb.AttributeDomainServiceServer                         // Entity reference validation
}

// UpdateSubscriptionAttributeServices groups all business service dependencies
type UpdateSubscriptionAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateSubscriptionAttributeUseCase handles the business logic for updating subscription attributes
type UpdateSubscriptionAttributeUseCase struct {
	repositories UpdateSubscriptionAttributeRepositories
	services     UpdateSubscriptionAttributeServices
}

// NewUpdateSubscriptionAttributeUseCase creates a new UpdateSubscriptionAttributeUseCase
func NewUpdateSubscriptionAttributeUseCase(
	repositories UpdateSubscriptionAttributeRepositories,
	services UpdateSubscriptionAttributeServices,
) *UpdateSubscriptionAttributeUseCase {
	return &UpdateSubscriptionAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update subscription attribute operation
func (uc *UpdateSubscriptionAttributeUseCase) Execute(ctx context.Context, req *subscriptionattributepb.UpdateSubscriptionAttributeRequest) (*subscriptionattributepb.UpdateSubscriptionAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscriptionAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichSubscriptionAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SubscriptionAttribute.UpdateSubscriptionAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.update_failed", "Subscription attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateSubscriptionAttributeUseCase) validateInput(ctx context.Context, req *subscriptionattributepb.UpdateSubscriptionAttributeRequest) error {
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

// enrichSubscriptionAttributeData updates audit information
func (uc *UpdateSubscriptionAttributeUseCase) enrichSubscriptionAttributeData(subscriptionAttribute *subscriptionattributepb.SubscriptionAttribute) error {
	now := time.Now()

	// Update audit fields
	subscriptionAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	subscriptionAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateSubscriptionAttributeUseCase) validateEntityReferences(ctx context.Context, subscriptionAttribute *subscriptionattributepb.SubscriptionAttribute) error {
	// Validate Subscription entity reference (if being updated)
	if subscriptionAttribute.SubscriptionId != "" {
		subscription, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
			Data: &subscriptionpb.Subscription{Id: subscriptionAttribute.SubscriptionId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.subscription_reference_validation_failed", "Failed to validate subscription entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if subscription == nil || subscription.Data == nil || len(subscription.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.subscription_not_found", "Subscription not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{subscriptionId}", subscriptionAttribute.SubscriptionId)
			return errors.New(translatedError)
		}
		if !subscription.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.subscription_not_active", "Referenced subscription with ID '{subscriptionId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{subscriptionId}", subscriptionAttribute.SubscriptionId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference (if being updated)
	if subscriptionAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: subscriptionAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.attribute_not_found", "Attribute not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", subscriptionAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", subscriptionAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
