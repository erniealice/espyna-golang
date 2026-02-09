package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
)

// ListPaymentsRepositories groups all repository dependencies
type ListPaymentsRepositories struct {
	Payment paymentpb.PaymentDomainServiceServer // Primary entity repository
}

// ListPaymentsServices groups all business service dependencies
type ListPaymentsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPaymentsUseCase handles the business logic for listing payments
type ListPaymentsUseCase struct {
	repositories ListPaymentsRepositories
	services     ListPaymentsServices
}

// NewListPaymentsUseCase creates use case with grouped dependencies
func NewListPaymentsUseCase(
	repositories ListPaymentsRepositories,
	services ListPaymentsServices,
) *ListPaymentsUseCase {
	return &ListPaymentsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list payments operation
func (uc *ListPaymentsUseCase) Execute(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPayment, ports.ActionList); err != nil {
		return nil, err
	}


	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", "Request is required for payments"))
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment listing within a transaction
func (uc *ListPaymentsUseCase) executeWithTransaction(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
	var result *paymentpb.ListPaymentsResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment.errors.list_failed", "")
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

// executeCore contains the core business logic for listing payments
func (uc *ListPaymentsUseCase) executeCore(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
	// Delegate to repository
	return uc.repositories.Payment.ListPayments(ctx, req)
}

// validateInput validates the input request
func (uc *ListPaymentsUseCase) validateInput(ctx context.Context, req *paymentpb.ListPaymentsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", ""))
	}
	// ListPaymentsRequest is typically empty or contains pagination/filtering parameters
	// Basic validation can be added here if needed
	return nil
}

// validateBusinessRules enforces business constraints for listing payments
func (uc *ListPaymentsUseCase) validateBusinessRules(ctx context.Context) error {
	// Financial security: Ensure proper access control for payment listing
	// Additional authorization checks would be implemented here in a real system
	// For example, users should only see their own payments, admins can see all

	// Business rule: Apply data filtering based on user permissions
	// This would typically filter results based on user role and permissions

	return nil
}

// Additional validation methods can be added here as needed
