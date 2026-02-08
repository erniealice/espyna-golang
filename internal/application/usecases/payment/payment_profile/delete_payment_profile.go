package payment_profile

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymentprofilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
)

// DeletePaymentProfileRepositories groups all repository dependencies
type DeletePaymentProfileRepositories struct {
	PaymentProfile paymentprofilepb.PaymentProfileDomainServiceServer // Primary entity repository
}

// DeletePaymentProfileServices groups all business service dependencies
type DeletePaymentProfileServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeletePaymentProfileUseCase handles the business logic for deleting payment profiles
type DeletePaymentProfileUseCase struct {
	repositories DeletePaymentProfileRepositories
	services     DeletePaymentProfileServices
}

// NewDeletePaymentProfileUseCase creates a new DeletePaymentProfileUseCase
func NewDeletePaymentProfileUseCase(
	repositories DeletePaymentProfileRepositories,
	services DeletePaymentProfileServices,
) *DeletePaymentProfileUseCase {
	return &DeletePaymentProfileUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete payment profile operation
func (uc *DeletePaymentProfileUseCase) Execute(ctx context.Context, req *paymentprofilepb.DeletePaymentProfileRequest) (*paymentprofilepb.DeletePaymentProfileResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPaymentProfile, ports.ActionDelete)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.authorization_failed", ""))
			}
		}
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment profile deletion within a transaction
func (uc *DeletePaymentProfileUseCase) executeWithTransaction(ctx context.Context, req *paymentprofilepb.DeletePaymentProfileRequest) (*paymentprofilepb.DeletePaymentProfileResponse, error) {
	var result *paymentprofilepb.DeletePaymentProfileResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
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
func (uc *DeletePaymentProfileUseCase) executeCore(ctx context.Context, req *paymentprofilepb.DeletePaymentProfileRequest) (*paymentprofilepb.DeletePaymentProfileResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Delegate to repository
	return uc.repositories.PaymentProfile.DeletePaymentProfile(ctx, req)
}

// validateInput validates the input request
func (uc *DeletePaymentProfileUseCase) validateInput(ctx context.Context, req *paymentprofilepb.DeletePaymentProfileRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.id_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints for payment profile deletion
func (uc *DeletePaymentProfileUseCase) validateBusinessRules(ctx context.Context, paymentProfile *paymentprofilepb.PaymentProfile) error {
	// Validate payment profile ID format
	if len(paymentProfile.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.id_too_short", ""))
	}

	// Financial security: Ensure proper authorization
	// Additional checks to ensure the user has permission to delete payment profiles

	// Business rule: Cannot delete if profile is associated with active subscriptions
	// In a real system, this would check for active subscriptions using this payment profile

	// Business rule: Cannot delete if profile has pending payments
	// This would check for any pending or processing payments using this profile

	// Audit requirement: Ensure deletion is logged appropriately
	// This would be handled by the repository layer typically

	return nil
}
