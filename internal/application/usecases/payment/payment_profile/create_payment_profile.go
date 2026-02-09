package payment_profile

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymentMethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
	paymentprofilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
)

// CreatePaymentProfileRepositories groups all repository dependencies
type CreatePaymentProfileRepositories struct {
	PaymentProfile paymentprofilepb.PaymentProfileDomainServiceServer
	Client         clientpb.ClientDomainServiceServer
	PaymentMethod  paymentMethodpb.PaymentMethodDomainServiceServer
}

// CreatePaymentProfileServices groups all business service dependencies
type CreatePaymentProfileServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePaymentProfileUseCase handles the business logic for creating payment profiles
type CreatePaymentProfileUseCase struct {
	repositories CreatePaymentProfileRepositories
	services     CreatePaymentProfileServices
}

// NewCreatePaymentProfileUseCase creates a new CreatePaymentProfileUseCase
func NewCreatePaymentProfileUseCase(
	repositories CreatePaymentProfileRepositories,
	services CreatePaymentProfileServices,
) *CreatePaymentProfileUseCase {
	return &CreatePaymentProfileUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create payment profile operation
func (uc *CreatePaymentProfileUseCase) Execute(ctx context.Context, req *paymentprofilepb.CreatePaymentProfileRequest) (*paymentprofilepb.CreatePaymentProfileResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPaymentProfile, ports.ActionCreate); err != nil {
		return nil, err
	}


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

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment profile creation within a transaction
func (uc *CreatePaymentProfileUseCase) executeWithTransaction(ctx context.Context, req *paymentprofilepb.CreatePaymentProfileRequest) (*paymentprofilepb.CreatePaymentProfileResponse, error) {
	var result *paymentprofilepb.CreatePaymentProfileResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment_profile.errors.creation_failed", "")
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

// executeCore contains the core business logic for creating a payment profile
func (uc *CreatePaymentProfileUseCase) executeCore(ctx context.Context, req *paymentprofilepb.CreatePaymentProfileRequest) (*paymentprofilepb.CreatePaymentProfileResponse, error) {
	// Delegate to repository
	return uc.repositories.PaymentProfile.CreatePaymentProfile(ctx, req)
}

// validateInput validates the input request
func (uc *CreatePaymentProfileUseCase) validateInput(ctx context.Context, req *paymentprofilepb.CreatePaymentProfileRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.data_required", ""))
	}
	if req.Data.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.client_id_required", ""))
	}
	if req.Data.PaymentMethodId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.payment_method_id_required", ""))
	}
	return nil
}

// enrichPaymentProfileData adds generated fields and audit information
func (uc *CreatePaymentProfileUseCase) enrichPaymentProfileData(paymentProfile *paymentprofilepb.PaymentProfile) error {
	now := time.Now()

	// Generate Payment Profile ID if not provided
	if paymentProfile.Id == "" {
		paymentProfile.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	paymentProfile.DateCreated = &[]int64{now.UnixMilli()}[0]
	paymentProfile.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	paymentProfile.DateModified = &[]int64{now.UnixMilli()}[0]
	paymentProfile.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	paymentProfile.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for payment profiles
func (uc *CreatePaymentProfileUseCase) validateBusinessRules(ctx context.Context, paymentProfile *paymentprofilepb.PaymentProfile) error {
	// Validate client ID format
	if len(paymentProfile.ClientId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.client_id_too_short", ""))
	}

	// Validate payment method ID format
	if len(paymentProfile.PaymentMethodId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_profile.validation.payment_method_id_too_short", ""))
	}

	// Financial constraint: Ensure client exists and is active
	// In a real system, this would verify the client exists and has appropriate permissions

	// Financial constraint: Ensure payment method exists and is valid
	// In a real system, this would verify the payment method exists and belongs to the client

	// Business rule: One client should not have multiple profiles for the same payment method
	// This would typically check for existing profiles with the same client_id and payment_method_id

	// Financial security: Ensure proper authorization
	// Additional checks to ensure the user has permission to create payment profiles for this client

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreatePaymentProfileUseCase) validateEntityReferences(ctx context.Context, paymentProfile *paymentprofilepb.PaymentProfile) error {

	// Validate ClientId entity reference
	if paymentProfile.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: paymentProfile.ClientId},
		})
		if err != nil {
			return fmt.Errorf("failed to validate client entity reference: %w", err)
		}
		if client == nil || !client.Success || client.Data == nil || len(client.Data) == 0 {
			return fmt.Errorf("referenced client with ID '%s' does not exist", paymentProfile.ClientId)
		}
		if !client.Data[0].Active {
			return fmt.Errorf("referenced client with ID '%s' is not active", paymentProfile.ClientId)
		}
	}

	// Validate PaymentMethodId entity reference
	if paymentProfile.PaymentMethodId != "" {
		paymentMethod, err := uc.repositories.PaymentMethod.ReadPaymentMethod(ctx, &paymentMethodpb.ReadPaymentMethodRequest{
			Data: &paymentMethodpb.PaymentMethod{Id: paymentProfile.PaymentMethodId},
		})
		if err != nil {
			return fmt.Errorf("failed to validate payment method entity reference: %w", err)
		}
		if paymentMethod == nil || !paymentMethod.Success || paymentMethod.Data == nil || len(paymentMethod.Data) == 0 {
			return fmt.Errorf("referenced payment method with ID '%s' does not exist", paymentProfile.PaymentMethodId)
		}
		if !paymentMethod.Data[0].Active {
			return fmt.Errorf("referenced payment method with ID '%s' is not active", paymentProfile.PaymentMethodId)
		}
	}

	return nil
}

// Additional validation methods can be added here as needed
