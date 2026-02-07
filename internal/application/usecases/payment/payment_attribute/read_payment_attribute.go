package payment_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentattributepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_attribute"
)

// ReadPaymentAttributeUseCase handles the business logic for reading a payment attribute
// ReadPaymentAttributeRepositories groups all repository dependencies
type ReadPaymentAttributeRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer // Primary entity repository
}

// ReadPaymentAttributeServices groups all business service dependencies
type ReadPaymentAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadPaymentAttributeUseCase handles the business logic for reading a payment attribute
type ReadPaymentAttributeUseCase struct {
	repositories ReadPaymentAttributeRepositories
	services     ReadPaymentAttributeServices
}

// NewReadPaymentAttributeUseCase creates a new ReadPaymentAttributeUseCase
func NewReadPaymentAttributeUseCase(
	repositories ReadPaymentAttributeRepositories,
	services ReadPaymentAttributeServices,
) *ReadPaymentAttributeUseCase {
	return &ReadPaymentAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read payment attribute operation
func (uc *ReadPaymentAttributeUseCase) Execute(ctx context.Context, req *paymentattributepb.ReadPaymentAttributeRequest) (*paymentattributepb.ReadPaymentAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPaymentAttribute, ports.ActionRead)
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
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.PaymentAttribute.ReadPaymentAttribute(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "payment_attribute.errors.not_found", map[string]interface{}{"paymentAttributeId": req.Data.Id}, "Payment attribute not found")
			return nil, errors.New(translatedError)
		}
		// Other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.read_failed", "Failed to read payment attribute")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPaymentAttributeUseCase) validateInput(ctx context.Context, req *paymentattributepb.ReadPaymentAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.data_required", "payment attribute data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.id_required", "payment attribute ID is required"))
	}
	return nil
}
