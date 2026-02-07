package payment_method

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentmethodpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_method"
)

// ReadPaymentMethodRepositories groups all repository dependencies
type ReadPaymentMethodRepositories struct {
	PaymentMethod paymentmethodpb.PaymentMethodDomainServiceServer // Primary entity repository
}

// ReadPaymentMethodServices groups all business service dependencies
type ReadPaymentMethodServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPaymentMethodUseCase handles the business logic for reading payment methods
type ReadPaymentMethodUseCase struct {
	repositories ReadPaymentMethodRepositories
	services     ReadPaymentMethodServices
}

// NewReadPaymentMethodUseCase creates use case with grouped dependencies
func NewReadPaymentMethodUseCase(
	repositories ReadPaymentMethodRepositories,
	services ReadPaymentMethodServices,
) *ReadPaymentMethodUseCase {
	return &ReadPaymentMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read payment method operation
func (uc *ReadPaymentMethodUseCase) Execute(ctx context.Context, req *paymentmethodpb.ReadPaymentMethodRequest) (*paymentmethodpb.ReadPaymentMethodResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPaymentMethod, ports.ActionRead)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.authorization_failed", ""))
			}
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic
	// No complex business logic needed for read operations

	// Transaction handling
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Core execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment method read within a transaction
func (uc *ReadPaymentMethodUseCase) executeWithTransaction(ctx context.Context, req *paymentmethodpb.ReadPaymentMethodRequest) (*paymentmethodpb.ReadPaymentMethodResponse, error) {
	var result *paymentmethodpb.ReadPaymentMethodResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("payment method read failed: %w", err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *ReadPaymentMethodUseCase) executeCore(ctx context.Context, req *paymentmethodpb.ReadPaymentMethodRequest) (*paymentmethodpb.ReadPaymentMethodResponse, error) {
	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.business_rule_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Delegate to repository
	resp, err := uc.repositories.PaymentMethod.ReadPaymentMethod(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "payment_method.errors.not_found", map[string]interface{}{"paymentMethodId": req.Data.Id}, "Payment method not found")
			return nil, errors.New(translatedError)
		}
		// Other error handling
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.read_failed", "Failed to read payment method")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Check if result is empty
	if len(resp.Data) == 0 || resp.Data[0].Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "payment_method.errors.not_found", map[string]interface{}{"paymentMethodId": req.Data.Id}, "Payment method not found")
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPaymentMethodUseCase) validateInput(ctx context.Context, req *paymentmethodpb.ReadPaymentMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.id_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints for reading payment methods
func (uc *ReadPaymentMethodUseCase) validateBusinessRules(ctx context.Context, paymentMethod *paymentmethodpb.PaymentMethod) error {
	// Validate payment method ID format
	if len(paymentMethod.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.id_too_short", ""))
	}

	// Financial security: Ensure proper access control for payment method data
	// Additional authorization checks would be implemented here in a real system
	// Users should only access their own payment methods unless they have admin privileges

	return nil
}

// Additional validation methods can be added here as needed
