package payment_method

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
)

// DeletePaymentMethodRepositories groups all repository dependencies
type DeletePaymentMethodRepositories struct {
	PaymentMethod paymentmethodpb.PaymentMethodDomainServiceServer // Primary entity repository
}

// DeletePaymentMethodServices groups all business service dependencies
type DeletePaymentMethodServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeletePaymentMethodUseCase handles the business logic for deleting payment methods
type DeletePaymentMethodUseCase struct {
	repositories DeletePaymentMethodRepositories
	services     DeletePaymentMethodServices
}

// NewDeletePaymentMethodUseCase creates use case with grouped dependencies
func NewDeletePaymentMethodUseCase(
	repositories DeletePaymentMethodRepositories,
	services DeletePaymentMethodServices,
) *DeletePaymentMethodUseCase {
	return &DeletePaymentMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete payment method operation
func (uc *DeletePaymentMethodUseCase) Execute(ctx context.Context, req *paymentmethodpb.DeletePaymentMethodRequest) (*paymentmethodpb.DeletePaymentMethodResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPaymentMethod, ports.ActionDelete)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.authorization_failed", ""))
			}
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
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

// executeWithTransaction executes payment method deletion within a transaction
func (uc *DeletePaymentMethodUseCase) executeWithTransaction(ctx context.Context, req *paymentmethodpb.DeletePaymentMethodRequest) (*paymentmethodpb.DeletePaymentMethodResponse, error) {
	var result *paymentmethodpb.DeletePaymentMethodResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment_method.errors.deletion_failed", "")
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

// executeCore contains the core business logic for deleting a payment method
func (uc *DeletePaymentMethodUseCase) executeCore(ctx context.Context, req *paymentmethodpb.DeletePaymentMethodRequest) (*paymentmethodpb.DeletePaymentMethodResponse, error) {
	// Delegate to repository
	return uc.repositories.PaymentMethod.DeletePaymentMethod(ctx, req)
}

// validateInput validates the input request
func (uc *DeletePaymentMethodUseCase) validateInput(ctx context.Context, req *paymentmethodpb.DeletePaymentMethodRequest) error {
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

// validateBusinessRules enforces business constraints for payment method deletion
func (uc *DeletePaymentMethodUseCase) validateBusinessRules(ctx context.Context, paymentMethod *paymentmethodpb.PaymentMethod) error {
	// Validate payment method ID format
	if len(paymentMethod.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.id_too_short", ""))
	}

	// Financial constraint: Cannot delete payment methods that are referenced by active payments
	// In a real system, this would check for active payment profiles or transactions using this method

	// Financial constraint: Cannot delete payment methods with pending transactions
	// This would typically check for any pending payments or subscriptions using this method

	// Financial constraint: Ensure proper authorization for payment method deletion
	// Additional authorization checks would be implemented here in a real system
	// Users should only be able to delete their own payment methods

	// Business rule: For compliance reasons, payment methods should typically be marked as inactive
	// rather than physically deleted to maintain audit trails

	return nil
}

// Additional validation methods can be added here as needed
