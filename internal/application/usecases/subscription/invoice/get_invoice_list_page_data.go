package invoice

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// GetInvoiceListPageDataRepositories groups all repository dependencies
type GetInvoiceListPageDataRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// GetInvoiceListPageDataServices groups all business service dependencies
type GetInvoiceListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetInvoiceListPageDataUseCase handles the business logic for getting invoice list page data
type GetInvoiceListPageDataUseCase struct {
	repositories GetInvoiceListPageDataRepositories
	services     GetInvoiceListPageDataServices
}

// NewGetInvoiceListPageDataUseCase creates use case with grouped dependencies
func NewGetInvoiceListPageDataUseCase(
	repositories GetInvoiceListPageDataRepositories,
	services GetInvoiceListPageDataServices,
) *GetInvoiceListPageDataUseCase {
	return &GetInvoiceListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get invoice list page data operation
func (uc *GetInvoiceListPageDataUseCase) Execute(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) (*invoicepb.GetInvoiceListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInvoice, ports.ActionList); err != nil {
		return nil, err
	}


	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.request_required", "Request is required for invoice list page data"))
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

// executeWithTransaction executes invoice list page data retrieval within a transaction
func (uc *GetInvoiceListPageDataUseCase) executeWithTransaction(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) (*invoicepb.GetInvoiceListPageDataResponse, error) {
	var result *invoicepb.GetInvoiceListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "invoice.errors.get_list_page_data_failed", "")
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

// executeCore contains the core business logic for getting invoice list page data
func (uc *GetInvoiceListPageDataUseCase) executeCore(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) (*invoicepb.GetInvoiceListPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.Invoice.GetInvoiceListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetInvoiceListPageDataUseCase) validateInput(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.request_required", ""))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.invalid_limit", "Pagination limit must be non-negative"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.limit_too_large", "Pagination limit cannot exceed 1000"))
		}
	}

	// Validate filter parameters for financial data
	if req.Filters != nil {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "amount" {
				// Validate amount filter to prevent invalid monetary operations
				if filter.GetNumberFilter() != nil && filter.GetNumberFilter().Value < 0 {
					return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.invalid_amount_filter", "Amount filter cannot be negative"))
				}
			}
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting invoice list page data
func (uc *GetInvoiceListPageDataUseCase) validateBusinessRules(ctx context.Context, req *invoicepb.GetInvoiceListPageDataRequest) error {
	// Financial security: Ensure proper access control for invoice list data
	// Additional authorization checks would be implemented here in a real system
	// For example, certain invoices might be restricted based on user roles or client ownership

	// Business rule: Apply data filtering based on user permissions
	// This would typically filter results based on user role and permissions
	// For example, clients should only see their own invoices, while admins can see all

	// Business rule: Validate search and filter parameters for security
	if req.Search != nil && req.Search.Query != "" {
		// Prevent SQL injection and other malicious queries
		// In a real system, implement proper query sanitization
	}

	// Business rule: Ensure financial data integrity
	// Validate that only appropriate users can access invoice information
	// This is critical for billing systems where invoices contain sensitive financial data

	return nil
}
