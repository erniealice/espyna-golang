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

// GetPaymentListPageDataRepositories groups all repository dependencies
type GetPaymentListPageDataRepositories struct {
	Payment paymentpb.PaymentDomainServiceServer // Primary entity repository
}

// GetPaymentListPageDataServices groups all business service dependencies
type GetPaymentListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPaymentListPageDataUseCase handles the business logic for getting payment list page data
type GetPaymentListPageDataUseCase struct {
	repositories GetPaymentListPageDataRepositories
	services     GetPaymentListPageDataServices
}

// NewGetPaymentListPageDataUseCase creates use case with grouped dependencies
func NewGetPaymentListPageDataUseCase(
	repositories GetPaymentListPageDataRepositories,
	services GetPaymentListPageDataServices,
) *GetPaymentListPageDataUseCase {
	return &GetPaymentListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get payment list page data operation
func (uc *GetPaymentListPageDataUseCase) Execute(ctx context.Context, req *paymentpb.GetPaymentListPageDataRequest) (*paymentpb.GetPaymentListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPayment, ports.ActionList); err != nil {
		return nil, err
	}


	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", "Request is required for payment list page data"))
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

// executeWithTransaction executes payment list page data retrieval within a transaction
func (uc *GetPaymentListPageDataUseCase) executeWithTransaction(ctx context.Context, req *paymentpb.GetPaymentListPageDataRequest) (*paymentpb.GetPaymentListPageDataResponse, error) {
	var result *paymentpb.GetPaymentListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment.errors.get_list_page_data_failed", "")
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

// executeCore contains the core business logic for getting payment list page data
func (uc *GetPaymentListPageDataUseCase) executeCore(ctx context.Context, req *paymentpb.GetPaymentListPageDataRequest) (*paymentpb.GetPaymentListPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.Payment.GetPaymentListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPaymentListPageDataUseCase) validateInput(ctx context.Context, req *paymentpb.GetPaymentListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", ""))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.invalid_limit", "Pagination limit must be non-negative"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.limit_too_large", "Pagination limit cannot exceed 1000"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting payment list page data
func (uc *GetPaymentListPageDataUseCase) validateBusinessRules(ctx context.Context, req *paymentpb.GetPaymentListPageDataRequest) error {
	// Financial security: Ensure proper access control for payment list data
	// Additional authorization checks would be implemented here in a real system
	// For example, users should only see their own payments, admins can see all

	// Business rule: Apply data filtering based on user permissions
	// This would typically filter results based on user role and permissions

	// Business rule: Validate search and filter parameters for security
	if req.Search != nil && req.Search.Query != "" {
		// Prevent SQL injection and other malicious queries
		// In a real system, implement proper query sanitization
	}

	return nil
}
