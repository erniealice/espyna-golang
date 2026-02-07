package payment_profile

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"
)

type GetPaymentProfileItemPageDataRepositories struct {
	PaymentProfile paymentprofilepb.PaymentProfileDomainServiceServer
}

type GetPaymentProfileItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetPaymentProfileItemPageDataUseCase handles the business logic for getting payment profile item page data
type GetPaymentProfileItemPageDataUseCase struct {
	repositories GetPaymentProfileItemPageDataRepositories
	services     GetPaymentProfileItemPageDataServices
}

// NewGetPaymentProfileItemPageDataUseCase creates a new GetPaymentProfileItemPageDataUseCase
func NewGetPaymentProfileItemPageDataUseCase(
	repositories GetPaymentProfileItemPageDataRepositories,
	services GetPaymentProfileItemPageDataServices,
) *GetPaymentProfileItemPageDataUseCase {
	return &GetPaymentProfileItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get payment profile item page data operation
func (uc *GetPaymentProfileItemPageDataUseCase) Execute(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileItemPageDataRequest,
) (*paymentprofilepb.GetPaymentProfileItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.PaymentProfileId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment profile item page data retrieval within a transaction
func (uc *GetPaymentProfileItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileItemPageDataRequest,
) (*paymentprofilepb.GetPaymentProfileItemPageDataResponse, error) {
	var result *paymentprofilepb.GetPaymentProfileItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"payment_profile.errors.item_page_data_failed",
				"payment profile item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting payment profile item page data
func (uc *GetPaymentProfileItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileItemPageDataRequest,
) (*paymentprofilepb.GetPaymentProfileItemPageDataResponse, error) {
	// Create read request for the payment profile
	readReq := &paymentprofilepb.ReadPaymentProfileRequest{
		Data: &paymentprofilepb.PaymentProfile{
			Id: req.PaymentProfileId,
		},
	}

	// Retrieve the payment profile
	readResp, err := uc.repositories.PaymentProfile.ReadPaymentProfile(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_profile.errors.read_failed",
			"failed to retrieve payment profile: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_profile.errors.not_found",
			"payment profile not found",
		))
	}

	// Get the payment profile (should be only one)
	paymentProfile := readResp.Data[0]

	// Validate that we got the expected payment profile
	if paymentProfile.Id != req.PaymentProfileId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_profile.errors.id_mismatch",
			"retrieved payment profile ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (client details, payment methods, billing history) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format sensitive payment data based on user permissions
	// 4. Add audit logging for payment profile access
	// 5. Mask sensitive financial information appropriately
	// 6. Load associated payment methods and their status

	// For now, return the payment profile as-is
	return &paymentprofilepb.GetPaymentProfileItemPageDataResponse{
		PaymentProfile: paymentProfile,
		Success:        true,
	}, nil
}

// validateInput validates the input request
func (uc *GetPaymentProfileItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_profile.validation.request_required",
			"request is required",
		))
	}

	if req.PaymentProfileId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_profile.validation.id_required",
			"payment profile ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading payment profile item page data
func (uc *GetPaymentProfileItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	paymentProfileId string,
) error {
	// Validate payment profile ID format
	if len(paymentProfileId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_profile.validation.id_too_short",
			"payment profile ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this payment profile
	// - Validate payment profile belongs to the current user's organization
	// - Check if payment profile is in a state that allows viewing
	// - Rate limiting for payment profile access
	// - Audit logging requirements for financial data access
	// - PCI compliance checks for sensitive payment data viewing
	// - Validate client relationship and access permissions

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like client and payment method details
// This would be called from executeCore if needed
func (uc *GetPaymentProfileItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	paymentProfile *paymentprofilepb.PaymentProfile,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to client and payment method repositories
	// to populate the nested client object and payment methods if they're not already loaded

	// Example implementation would be:
	// if paymentProfile.Client == nil && paymentProfile.ClientId != "" {
	//     // Load client data
	// }
	// if paymentProfile.PaymentMethods == nil {
	//     // Load associated payment methods
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetPaymentProfileItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	paymentProfile *paymentprofilepb.PaymentProfile,
) *paymentprofilepb.PaymentProfile {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Masking sensitive payment information
	// - Formatting currency and monetary values
	// - Converting dates to user's preferred timezone
	// - Applying localization for currency display
	// - Sanitizing sensitive financial data based on user permissions
	// - Adding calculated fields for payment history summaries

	return paymentProfile
}

// checkAccessPermissions validates user has permission to access this payment profile
func (uc *GetPaymentProfileItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	paymentProfileId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating payment profile belongs to user's organization
	// - Applying multi-tenant access controls
	// - Checking PCI compliance requirements
	// - Validating client relationship permissions
	// - Audit logging for sensitive financial data access

	return nil
}

// maskSensitiveData applies appropriate data masking for payment information
func (uc *GetPaymentProfileItemPageDataUseCase) maskSensitiveData(
	ctx context.Context,
	paymentProfile *paymentprofilepb.PaymentProfile,
) error {
	// TODO: Implement proper data masking
	// This could include:
	// - Masking credit card numbers
	// - Hiding sensitive billing information
	// - Redacting payment method details based on user permissions
	// - Applying PCI DSS compliance requirements
	// - Conditional data exposure based on user role

	return nil
}

// loadPaymentHistory loads associated payment and billing history
func (uc *GetPaymentProfileItemPageDataUseCase) loadPaymentHistory(
	ctx context.Context,
	paymentProfile *paymentprofilepb.PaymentProfile,
) error {
	// TODO: Load payment history data
	// This could include:
	// - Recent payment transactions
	// - Billing history
	// - Failed payment attempts
	// - Payment method usage statistics
	// - Subscription billing cycles

	return nil
}
