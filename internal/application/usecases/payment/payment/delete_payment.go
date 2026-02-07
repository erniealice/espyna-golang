package payment

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment"
)

// DeletePaymentRepositories groups all repository dependencies
type DeletePaymentRepositories struct {
	Payment paymentpb.PaymentDomainServiceServer // Primary entity repository
}

// DeletePaymentServices groups all business service dependencies
type DeletePaymentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeletePaymentUseCase handles the business logic for deleting payments
type DeletePaymentUseCase struct {
	repositories DeletePaymentRepositories
	services     DeletePaymentServices
}

// NewDeletePaymentUseCase creates use case with grouped dependencies
func NewDeletePaymentUseCase(
	repositories DeletePaymentRepositories,
	services DeletePaymentServices,
) *DeletePaymentUseCase {
	return &DeletePaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete payment operation
func (uc *DeletePaymentUseCase) Execute(ctx context.Context, req *paymentpb.DeletePaymentRequest) (*paymentpb.DeletePaymentResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPayment, ports.ActionDelete)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.errors.authorization_failed", ""))
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

// executeWithTransaction executes payment deletion within a transaction
func (uc *DeletePaymentUseCase) executeWithTransaction(ctx context.Context, req *paymentpb.DeletePaymentRequest) (*paymentpb.DeletePaymentResponse, error) {
	var result *paymentpb.DeletePaymentResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment.errors.deletion_failed", "")
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

// executeCore contains the core business logic for deleting a payment
func (uc *DeletePaymentUseCase) executeCore(ctx context.Context, req *paymentpb.DeletePaymentRequest) (*paymentpb.DeletePaymentResponse, error) {
	// Delegate to repository
	return uc.repositories.Payment.DeletePayment(ctx, req)
}

// validateInput validates the input request
func (uc *DeletePaymentUseCase) validateInput(ctx context.Context, req *paymentpb.DeletePaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints for payment deletion
func (uc *DeletePaymentUseCase) validateBusinessRules(ctx context.Context, payment *paymentpb.Payment) error {
	// Validate payment ID format
	if len(payment.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_too_short", ""))
	}

	// Financial constraint: Cannot delete processed payments
	// In a real system, this would check payment status and prevent deletion of completed transactions
	// For audit compliance, payments should typically be marked as inactive rather than physically deleted

	// Financial constraint: Ensure proper authorization for payment deletion
	// Additional authorization checks would be implemented here in a real system

	// Business rule: Only allow deletion of pending/draft payments
	// This would typically check payment status before allowing deletion

	return nil
}

// Additional validation methods can be added here as needed
