package payment_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentattributepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_attribute"
)

// ListPaymentAttributesUseCase handles the business logic for listing payment attributes
// ListPaymentAttributesRepositories groups all repository dependencies
type ListPaymentAttributesRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer // Primary entity repository
}

// ListPaymentAttributesServices groups all business service dependencies
type ListPaymentAttributesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListPaymentAttributesUseCase handles the business logic for listing payment attributes
type ListPaymentAttributesUseCase struct {
	repositories ListPaymentAttributesRepositories
	services     ListPaymentAttributesServices
}

// NewListPaymentAttributesUseCase creates a new ListPaymentAttributesUseCase
func NewListPaymentAttributesUseCase(
	repositories ListPaymentAttributesRepositories,
	services ListPaymentAttributesServices,
) *ListPaymentAttributesUseCase {
	return &ListPaymentAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list payment attributes operation
func (uc *ListPaymentAttributesUseCase) Execute(ctx context.Context, req *paymentattributepb.ListPaymentAttributesRequest) (*paymentattributepb.ListPaymentAttributesResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPaymentAttribute, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PaymentAttribute.ListPaymentAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.list_failed", "Failed to retrieve payment attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListPaymentAttributesUseCase) validateInput(ctx context.Context, req *paymentattributepb.ListPaymentAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
