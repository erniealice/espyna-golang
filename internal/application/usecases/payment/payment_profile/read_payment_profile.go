package payment_profile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymentprofilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
)

// ReadPaymentProfileRepositories groups all repository dependencies
type ReadPaymentProfileRepositories struct {
	PaymentProfile paymentprofilepb.PaymentProfileDomainServiceServer // Primary entity repository
}

// ReadPaymentProfileServices groups all business service dependencies
type ReadPaymentProfileServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPaymentProfileUseCase handles the business logic for reading payment profiles
type ReadPaymentProfileUseCase struct {
	repositories ReadPaymentProfileRepositories
	services     ReadPaymentProfileServices
}

// NewReadPaymentProfileUseCase creates a new ReadPaymentProfileUseCase
func NewReadPaymentProfileUseCase(
	repositories ReadPaymentProfileRepositories,
	services ReadPaymentProfileServices,
) *ReadPaymentProfileUseCase {
	return &ReadPaymentProfileUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read payment profile operation
func (uc *ReadPaymentProfileUseCase) Execute(ctx context.Context, req *paymentprofilepb.ReadPaymentProfileRequest) (*paymentprofilepb.ReadPaymentProfileResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPaymentProfile, ports.ActionRead)); err != nil || !authorized {
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

// executeWithTransaction executes payment profile read within a transaction
func (uc *ReadPaymentProfileUseCase) executeWithTransaction(ctx context.Context, req *paymentprofilepb.ReadPaymentProfileRequest) (*paymentprofilepb.ReadPaymentProfileResponse, error) {
	var result *paymentprofilepb.ReadPaymentProfileResponse

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
func (uc *ReadPaymentProfileUseCase) executeCore(ctx context.Context, req *paymentprofilepb.ReadPaymentProfileRequest) (*paymentprofilepb.ReadPaymentProfileResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.input_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.business_rule_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Delegate to repository
	resp, err := uc.repositories.PaymentProfile.ReadPaymentProfile(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.read_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" { // Assuming resp.Data will be nil or have empty ID if not found
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.not_found", "")
		translatedError = strings.ReplaceAll(translatedError, "{paymentProfileId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPaymentProfileUseCase) validateInput(ctx context.Context, req *paymentprofilepb.ReadPaymentProfileRequest) error {
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

// validateBusinessRules enforces business constraints for reading payment profiles
func (uc *ReadPaymentProfileUseCase) validateBusinessRules(ctx context.Context, paymentProfile *paymentprofilepb.PaymentProfile) error {
	// Validate payment profile ID format
	if len(paymentProfile.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.id_too_short", ""))
	}

	// Financial security: Ensure proper access control for payment profile data
	// Additional authorization checks would be implemented here in a real system
	// Users should only access payment profiles for their own clients unless they have admin privileges

	// Business rule: Verify user has permission to access this payment profile
	// This would typically check user permissions and client associations

	return nil
}

// Additional validation methods can be added here as needed
