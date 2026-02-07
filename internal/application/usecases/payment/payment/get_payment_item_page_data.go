package payment

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment"
)

// GetPaymentItemPageDataRepositories groups all repository dependencies
type GetPaymentItemPageDataRepositories struct {
	Payment paymentpb.PaymentDomainServiceServer // Primary entity repository
}

// GetPaymentItemPageDataServices groups all business service dependencies
type GetPaymentItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPaymentItemPageDataUseCase handles the business logic for getting payment item page data
type GetPaymentItemPageDataUseCase struct {
	repositories GetPaymentItemPageDataRepositories
	services     GetPaymentItemPageDataServices
}

// NewGetPaymentItemPageDataUseCase creates use case with grouped dependencies
func NewGetPaymentItemPageDataUseCase(
	repositories GetPaymentItemPageDataRepositories,
	services GetPaymentItemPageDataServices,
) *GetPaymentItemPageDataUseCase {
	return &GetPaymentItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get payment item page data operation
func (uc *GetPaymentItemPageDataUseCase) Execute(ctx context.Context, req *paymentpb.GetPaymentItemPageDataRequest) (*paymentpb.GetPaymentItemPageDataResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPayment, ports.ActionRead)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.errors.authorization_failed", ""))
			}
		}
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", "Request is required for payment item page data"))
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment item page data retrieval within a transaction
func (uc *GetPaymentItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *paymentpb.GetPaymentItemPageDataRequest) (*paymentpb.GetPaymentItemPageDataResponse, error) {
	var result *paymentpb.GetPaymentItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment.errors.get_item_page_data_failed", "")
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

// executeCore contains the core business logic for getting payment item page data
func (uc *GetPaymentItemPageDataUseCase) executeCore(ctx context.Context, req *paymentpb.GetPaymentItemPageDataRequest) (*paymentpb.GetPaymentItemPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.Payment.GetPaymentItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPaymentItemPageDataUseCase) validateInput(ctx context.Context, req *paymentpb.GetPaymentItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", ""))
	}

	if req.PaymentId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_required", "Payment ID is required"))
	}

	// Validate ID format (basic validation)
	if len(req.PaymentId) > 255 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_too_long", "Payment ID cannot exceed 255 characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting payment item page data
func (uc *GetPaymentItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *paymentpb.GetPaymentItemPageDataRequest) error {
	// Financial security: Ensure proper access control for payment item data
	// Additional authorization checks would be implemented here in a real system
	// For example, users should only see their own payments, admins can see all

	// Business rule: Validate payment ownership or admin access
	// This would typically check if the current user has permission to view this specific payment
	// In a real system, this would involve checking payment ownership or admin privileges

	return nil
}
