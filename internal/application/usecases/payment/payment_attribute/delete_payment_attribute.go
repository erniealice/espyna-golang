package payment_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
)

// DeletePaymentAttributeUseCase handles the business logic for deleting payment attributes
// DeletePaymentAttributeRepositories groups all repository dependencies
type DeletePaymentAttributeRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer // Primary entity repository
}

// DeletePaymentAttributeServices groups all business service dependencies
type DeletePaymentAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeletePaymentAttributeUseCase handles the business logic for deleting payment attributes
type DeletePaymentAttributeUseCase struct {
	repositories DeletePaymentAttributeRepositories
	services     DeletePaymentAttributeServices
}

// NewDeletePaymentAttributeUseCase creates a new DeletePaymentAttributeUseCase
func NewDeletePaymentAttributeUseCase(
	repositories DeletePaymentAttributeRepositories,
	services DeletePaymentAttributeServices,
) *DeletePaymentAttributeUseCase {
	return &DeletePaymentAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete payment attribute operation
func (uc *DeletePaymentAttributeUseCase) Execute(ctx context.Context, req *paymentattributepb.DeletePaymentAttributeRequest) (*paymentattributepb.DeletePaymentAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPaymentAttribute, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment attribute deletion within a transaction
func (uc *DeletePaymentAttributeUseCase) executeWithTransaction(ctx context.Context, req *paymentattributepb.DeletePaymentAttributeRequest) (*paymentattributepb.DeletePaymentAttributeResponse, error) {
	var result *paymentattributepb.DeletePaymentAttributeResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a payment attribute
func (uc *DeletePaymentAttributeUseCase) executeCore(ctx context.Context, req *paymentattributepb.DeletePaymentAttributeRequest) (*paymentattributepb.DeletePaymentAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPaymentAttribute, ports.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PaymentAttribute.DeletePaymentAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.deletion_failed", "Payment attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeletePaymentAttributeUseCase) validateInput(ctx context.Context, req *paymentattributepb.DeletePaymentAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.data_required", "Payment attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.id_required", "Payment attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for payment attribute deletion
func (uc *DeletePaymentAttributeUseCase) validateBusinessRules(ctx context.Context, req *paymentattributepb.DeletePaymentAttributeRequest) error {
	// Additional business rule validation can be added here
	// For example: check if payment attribute is referenced by other entities
	if uc.isPaymentAttributeInUse(ctx, req.Data.PaymentId, req.Data.AttributeId) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.in_use", "Payment attribute is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isPaymentAttributeInUse checks if the payment attribute is referenced by other entities
func (uc *DeletePaymentAttributeUseCase) isPaymentAttributeInUse(ctx context.Context, paymentID, attributeID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for payment attribute usage
	return false
}
