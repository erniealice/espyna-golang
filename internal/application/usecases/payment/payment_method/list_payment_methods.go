package payment_method

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
)

// ListPaymentMethodsRepositories groups all repository dependencies
type ListPaymentMethodsRepositories struct {
	PaymentMethod paymentmethodpb.PaymentMethodDomainServiceServer // Primary entity repository
}

// ListPaymentMethodsServices groups all business service dependencies
type ListPaymentMethodsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPaymentMethodsUseCase handles the business logic for listing payment methods
type ListPaymentMethodsUseCase struct {
	repositories ListPaymentMethodsRepositories
	services     ListPaymentMethodsServices
}

// NewListPaymentMethodsUseCase creates a new ListPaymentMethodsUseCase
func NewListPaymentMethodsUseCase(
	repositories ListPaymentMethodsRepositories,
	services ListPaymentMethodsServices,
) *ListPaymentMethodsUseCase {
	return &ListPaymentMethodsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list payment methods operation
func (uc *ListPaymentMethodsUseCase) Execute(ctx context.Context, req *paymentmethodpb.ListPaymentMethodsRequest) (*paymentmethodpb.ListPaymentMethodsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPaymentMethod, ports.ActionList); err != nil {
		return nil, err
	}


	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic
	// No complex business logic needed for list operations

	// Transaction handling
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Core execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment method list within a transaction
func (uc *ListPaymentMethodsUseCase) executeWithTransaction(ctx context.Context, req *paymentmethodpb.ListPaymentMethodsRequest) (*paymentmethodpb.ListPaymentMethodsResponse, error) {
	var result *paymentmethodpb.ListPaymentMethodsResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("payment method list failed: %w", err)
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
func (uc *ListPaymentMethodsUseCase) executeCore(ctx context.Context, req *paymentmethodpb.ListPaymentMethodsRequest) (*paymentmethodpb.ListPaymentMethodsResponse, error) {
	// Business rule validation
	if err := uc.validateBusinessRules(ctx); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.business_rule_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Delegate to repository
	resp, err := uc.repositories.PaymentMethod.ListPaymentMethods(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.list_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListPaymentMethodsUseCase) validateInput(ctx context.Context, req *paymentmethodpb.ListPaymentMethodsRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.request_required", ""))
	}
	// ListPaymentMethodsRequest is typically empty or contains pagination/filtering parameters
	// Basic validation can be added here if needed
	return nil
}

// validateBusinessRules enforces business constraints for listing payment methods
func (uc *ListPaymentMethodsUseCase) validateBusinessRules(ctx context.Context) error {

	// Financial security: Ensure proper access control for payment method listing
	// Additional authorization checks would be implemented here in a real system
	// Users should only see their own payment methods unless they have admin privileges

	// Business rule: Apply data filtering based on user permissions
	// This would typically filter results based on user role and permissions

	// Financial security: Mask sensitive payment method information in list views
	// Only show last 4 digits of cards/accounts, not full details

	return nil
}

// Additional validation methods can be added here as needed
