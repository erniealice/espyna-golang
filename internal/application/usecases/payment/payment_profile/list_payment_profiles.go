package payment_profile

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"
)

// ListPaymentProfilesRepositories groups all repository dependencies
type ListPaymentProfilesRepositories struct {
	PaymentProfile paymentprofilepb.PaymentProfileDomainServiceServer // Primary entity repository
}

// ListPaymentProfilesServices groups all business service dependencies
type ListPaymentProfilesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPaymentProfilesUseCase handles the business logic for listing payment profiles
type ListPaymentProfilesUseCase struct {
	repositories ListPaymentProfilesRepositories
	services     ListPaymentProfilesServices
}

// NewListPaymentProfilesUseCase creates a new ListPaymentProfilesUseCase
func NewListPaymentProfilesUseCase(
	repositories ListPaymentProfilesRepositories,
	services ListPaymentProfilesServices,
) *ListPaymentProfilesUseCase {
	return &ListPaymentProfilesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list payment profiles operation
func (uc *ListPaymentProfilesUseCase) Execute(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) (*paymentprofilepb.ListPaymentProfilesResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPaymentProfile, ports.ActionList)); err != nil || !authorized {
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

// executeWithTransaction executes payment profile listing within a transaction
func (uc *ListPaymentProfilesUseCase) executeWithTransaction(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) (*paymentprofilepb.ListPaymentProfilesResponse, error) {
	var result *paymentprofilepb.ListPaymentProfilesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment_profile.errors.list_failed", "")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.transaction_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic
func (uc *ListPaymentProfilesUseCase) executeCore(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) (*paymentprofilepb.ListPaymentProfilesResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.input_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.business_rule_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Delegate to repository
	resp, err := uc.repositories.PaymentProfile.ListPaymentProfiles(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.list_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListPaymentProfilesUseCase) validateInput(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.request_required", ""))
	}
	// if req.ClientId == "" {
	// 	return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.client_id_required", ""))
	// }
	return nil
}

// validateBusinessRules enforces business constraints for payment profile listing
func (uc *ListPaymentProfilesUseCase) validateBusinessRules(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) error {
	// Security constraint: Ensure user has permission to view profiles for this client
	// In a real system, this would verify the requesting user has appropriate permissions

	// Business rule: Validate client ID format
	// if len(req.ClientId) < 5 {
	// 	return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.client_id_too_short", ""))
	// }

	// Privacy constraint: Ensure client exists and is accessible
	// This would typically verify the client exists and the user has access rights

	return nil
}
