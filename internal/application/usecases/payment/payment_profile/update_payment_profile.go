package payment_profile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymentMethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
	paymentprofilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
)

// UpdatePaymentProfileRepositories groups all repository dependencies
type UpdatePaymentProfileRepositories struct {
	PaymentProfile paymentprofilepb.PaymentProfileDomainServiceServer
	Client         clientpb.ClientDomainServiceServer
	PaymentMethod  paymentMethodpb.PaymentMethodDomainServiceServer
}

// UpdatePaymentProfileServices groups all business service dependencies
type UpdatePaymentProfileServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdatePaymentProfileUseCase handles the business logic for updating payment profiles
type UpdatePaymentProfileUseCase struct {
	repositories UpdatePaymentProfileRepositories
	services     UpdatePaymentProfileServices
}

// NewUpdatePaymentProfileUseCase creates a new UpdatePaymentProfileUseCase
func NewUpdatePaymentProfileUseCase(
	repositories UpdatePaymentProfileRepositories,
	services UpdatePaymentProfileServices,
) *UpdatePaymentProfileUseCase {
	return &UpdatePaymentProfileUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update payment profile operation
func (uc *UpdatePaymentProfileUseCase) Execute(ctx context.Context, req *paymentprofilepb.UpdatePaymentProfileRequest) (*paymentprofilepb.UpdatePaymentProfileResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPaymentProfile, ports.ActionUpdate); err != nil {
		return nil, err
	}


	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment profile update within a transaction
func (uc *UpdatePaymentProfileUseCase) executeWithTransaction(ctx context.Context, req *paymentprofilepb.UpdatePaymentProfileRequest) (*paymentprofilepb.UpdatePaymentProfileResponse, error) {
	var result *paymentprofilepb.UpdatePaymentProfileResponse

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

// executeCore contains the core business logic (moved from original Execute method)
func (uc *UpdatePaymentProfileUseCase) executeCore(ctx context.Context, req *paymentprofilepb.UpdatePaymentProfileRequest) (*paymentprofilepb.UpdatePaymentProfileResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPaymentProfileData(req.Data); err != nil {
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

	// Delegate to repository
	return uc.repositories.PaymentProfile.UpdatePaymentProfile(ctx, req)
}

// validateInput validates the input request
func (uc *UpdatePaymentProfileUseCase) validateInput(ctx context.Context, req *paymentprofilepb.UpdatePaymentProfileRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.id_required", ""))
	}
	if req.Data.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.client_id_required", ""))
	}
	if req.Data.PaymentMethodId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.payment_method_id_required", ""))
	}
	return nil
}

// enrichPaymentProfileData adds updated audit information
func (uc *UpdatePaymentProfileUseCase) enrichPaymentProfileData(paymentProfile *paymentprofilepb.PaymentProfile) error {
	now := time.Now()

	// Update modification timestamp
	paymentProfile.DateModified = &[]int64{now.UnixMilli()}[0]
	paymentProfile.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for payment profile updates
func (uc *UpdatePaymentProfileUseCase) validateBusinessRules(ctx context.Context, paymentProfile *paymentprofilepb.PaymentProfile) error {
	// Validate payment profile ID format
	if len(paymentProfile.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.id_too_short", ""))
	}

	// Validate client ID format
	if len(paymentProfile.ClientId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.client_id_too_short", ""))
	}

	// Validate payment method ID format
	if len(paymentProfile.PaymentMethodId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.payment_method_id_too_short", ""))
	}

	// Financial constraint: Cannot modify payment profiles that are currently being used in active transactions
	// This would typically check for active payments using this profile

	// Financial constraint: Ensure client exists and is active
	// In a real system, this would verify the client exists and has appropriate permissions

	// Financial constraint: Ensure payment method exists and is valid
	// In a real system, this would verify the payment method exists and belongs to the client

	// Business rule: One client should not have multiple profiles for the same payment method
	// This would check for existing profiles with the same client_id and payment_method_id (excluding current profile)

	// Financial security: Ensure proper authorization
	// Additional checks to ensure the user has permission to update payment profiles for this client

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdatePaymentProfileUseCase) validateEntityReferences(ctx context.Context, paymentProfile *paymentprofilepb.PaymentProfile) error {
	// Validate ClientId entity reference
	if paymentProfile.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: paymentProfile.ClientId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.client_reference_validation_failed", "")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if client == nil || len(client.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.client_not_found", "")
			translatedError = strings.ReplaceAll(translatedError, "{clientId}", paymentProfile.ClientId)
			return errors.New(translatedError)
		}
		if !client.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.client_not_active", "")
			translatedError = strings.ReplaceAll(translatedError, "{clientId}", paymentProfile.ClientId)
			return errors.New(translatedError)
		}
	}

	// Validate PaymentMethodId entity reference
	if paymentProfile.PaymentMethodId != "" {
		paymentMethod, err := uc.repositories.PaymentMethod.ReadPaymentMethod(ctx, &paymentMethodpb.ReadPaymentMethodRequest{
			Data: &paymentMethodpb.PaymentMethod{Id: paymentProfile.PaymentMethodId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.payment_method_reference_validation_failed", "")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if paymentMethod == nil || len(paymentMethod.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.payment_method_not_found", "")
			translatedError = strings.ReplaceAll(translatedError, "{paymentMethodId}", paymentProfile.PaymentMethodId)
			return errors.New(translatedError)
		}
		if !paymentMethod.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.errors.payment_method_not_active", "")
			translatedError = strings.ReplaceAll(translatedError, "{paymentMethodId}", paymentProfile.PaymentMethodId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// Additional validation methods can be added here as needed
