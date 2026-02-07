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

// CreatePaymentRepositories groups all repository dependencies
type CreatePaymentRepositories struct {
	Payment      paymentpb.PaymentDomainServiceServer
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

// CreatePaymentServices groups all business service dependencies
type CreatePaymentServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePaymentUseCase handles the business logic for creating payments
type CreatePaymentUseCase struct {
	repositories CreatePaymentRepositories
	services     CreatePaymentServices
}

// NewCreatePaymentUseCase creates use case with grouped dependencies
func NewCreatePaymentUseCase(
	repositories CreatePaymentRepositories,
	services CreatePaymentServices,
) *CreatePaymentUseCase {
	return &CreatePaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create payment operation
func (uc *CreatePaymentUseCase) Execute(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		uid, _ := ctx.Value("uid").(string)
		if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPayment, ports.ActionCreate)); err != nil || !authorized {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.errors.authorization_failed", ""))
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

// executeWithTransaction executes payment creation within a transaction
func (uc *CreatePaymentUseCase) executeWithTransaction(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	var result *paymentpb.CreatePaymentResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment.errors.creation_failed", "")
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

// executeCore contains the core business logic for creating a payment
func (uc *CreatePaymentUseCase) executeCore(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	// Delegate to repository
	return uc.repositories.Payment.CreatePayment(ctx, req)
}

// validateInput validates the input request
func (uc *CreatePaymentUseCase) validateInput(ctx context.Context, req *paymentpb.CreatePaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.data_required", ""))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.name_required", ""))
	}
	if req.Data.SubscriptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.subscription_id_required", ""))
	}
	return nil
}

// enrichPaymentData adds generated fields and audit information
func (uc *CreatePaymentUseCase) enrichPaymentData(payment *paymentpb.Payment) error {
	now := time.Now()

	// Generate Payment ID if not provided
	if payment.Id == "" {
		payment.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	payment.DateCreated = &[]int64{now.UnixMilli()}[0]
	payment.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	payment.DateModified = &[]int64{now.UnixMilli()}[0]
	payment.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	payment.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for payments
func (uc *CreatePaymentUseCase) validateBusinessRules(ctx context.Context, payment *paymentpb.Payment) error {
	// Validate payment name length
	if len(payment.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.name_too_short", ""))
	}

	if len(payment.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.name_too_long", ""))
	}

	// Validate subscription ID format (basic format check)
	if len(payment.SubscriptionId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.subscription_id_too_short", ""))
	}

	// Financial constraint: Payment must be associated with a valid subscription
	if payment.SubscriptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.subscription_association_required", ""))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreatePaymentUseCase) validateEntityReferences(ctx context.Context, payment *paymentpb.Payment) error {
	// Validate SubscriptionId entity reference
	if payment.SubscriptionId != "" {
		subscription, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
			Data: &subscriptionpb.Subscription{Id: payment.SubscriptionId},
		})
		if err != nil {
			return fmt.Errorf("failed to validate subscription entity reference: %w", err)
		}
		if subscription == nil || len(subscription.Data) == 0 {
			return fmt.Errorf("referenced subscription with ID '%s' does not exist", payment.SubscriptionId)
		}
		if !subscription.Data[0].Active {
			return fmt.Errorf("referenced subscription with ID '%s' is not active", payment.SubscriptionId)
		}
	}

	return nil
}

// Additional validation methods can be added here as needed
