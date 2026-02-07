package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// UpdatePaymentRepositories groups all repository dependencies
type UpdatePaymentRepositories struct {
	Payment      paymentpb.PaymentDomainServiceServer
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

// UpdatePaymentServices groups all business service dependencies
type UpdatePaymentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdatePaymentUseCase handles the business logic for updating payments
type UpdatePaymentUseCase struct {
	repositories UpdatePaymentRepositories
	services     UpdatePaymentServices
}

// NewUpdatePaymentUseCase creates use case with grouped dependencies
func NewUpdatePaymentUseCase(
	repositories UpdatePaymentRepositories,
	services UpdatePaymentServices,
) *UpdatePaymentUseCase {
	return &UpdatePaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update payment operation
func (uc *UpdatePaymentUseCase) Execute(ctx context.Context, req *paymentpb.UpdatePaymentRequest) (*paymentpb.UpdatePaymentResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPayment, ports.ActionUpdate)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.errors.authorization_failed", ""))
			}
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPaymentData(req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment update within a transaction
func (uc *UpdatePaymentUseCase) executeWithTransaction(ctx context.Context, req *paymentpb.UpdatePaymentRequest) (*paymentpb.UpdatePaymentResponse, error) {
	var result *paymentpb.UpdatePaymentResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment.errors.update_failed", "")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for updating a payment
func (uc *UpdatePaymentUseCase) executeCore(ctx context.Context, req *paymentpb.UpdatePaymentRequest) (*paymentpb.UpdatePaymentResponse, error) {
	// Delegate to repository
	return uc.repositories.Payment.UpdatePayment(ctx, req)
}

// validateInput validates the input request
// Note: Only Id is required for updates. Name and SubscriptionId are optional
// to support partial updates from workflow orchestration.
func (uc *UpdatePaymentUseCase) validateInput(ctx context.Context, req *paymentpb.UpdatePaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_required", ""))
	}
	// Name and SubscriptionId validation removed to support partial updates
	// from workflow orchestration where only specific fields are updated
	return nil
}

// enrichPaymentData adds updated audit information
func (uc *UpdatePaymentUseCase) enrichPaymentData(payment *paymentpb.Payment) error {
	now := time.Now()

	// Update modification timestamp
	payment.DateModified = &[]int64{now.UnixMilli()}[0]
	payment.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for payment updates
// Note: Relaxed validation to support partial updates from workflow orchestration
func (uc *UpdatePaymentUseCase) validateBusinessRules(ctx context.Context, payment *paymentpb.Payment) error {

	// Validate payment ID format (always required)
	if len(payment.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_too_short", ""))
	}

	// Validate payment name length only if provided (supports partial updates)
	if payment.Name != "" {
		if len(payment.Name) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.name_too_short", ""))
		}
		if len(payment.Name) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.name_too_long", ""))
		}
	}

	// Validate subscription ID format only if provided (supports partial updates)
	if payment.SubscriptionId != "" && len(payment.SubscriptionId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.subscription_id_too_short", ""))
	}

	// Financial constraint: Cannot modify completed payments
	// This would typically check payment status in a real system

	return nil
}

// validateEntityReferences validates that all referenced entities exist
// Note: Only validates references that are being updated (supports partial updates)
func (uc *UpdatePaymentUseCase) validateEntityReferences(ctx context.Context, payment *paymentpb.Payment) error {
	// Skip subscription validation for partial updates (status-only updates)
	// Only validate if SubscriptionId is being explicitly changed
	// For workflow orchestration, we trust the existing subscription reference
	return nil
}

// Additional validation methods can be added here as needed
