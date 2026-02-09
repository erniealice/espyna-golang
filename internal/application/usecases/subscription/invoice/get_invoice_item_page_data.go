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

// GetInvoiceItemPageDataRepositories groups all repository dependencies
type GetInvoiceItemPageDataRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// GetInvoiceItemPageDataServices groups all business service dependencies
type GetInvoiceItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetInvoiceItemPageDataUseCase handles the business logic for getting invoice item page data
type GetInvoiceItemPageDataUseCase struct {
	repositories GetInvoiceItemPageDataRepositories
	services     GetInvoiceItemPageDataServices
}

// NewGetInvoiceItemPageDataUseCase creates use case with grouped dependencies
func NewGetInvoiceItemPageDataUseCase(
	repositories GetInvoiceItemPageDataRepositories,
	services GetInvoiceItemPageDataServices,
) *GetInvoiceItemPageDataUseCase {
	return &GetInvoiceItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get invoice item page data operation
func (uc *GetInvoiceItemPageDataUseCase) Execute(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) (*invoicepb.GetInvoiceItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInvoice, ports.ActionList); err != nil {
		return nil, err
	}


	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.request_required", "Request is required for invoice item page data"))
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

// executeWithTransaction executes invoice item page data retrieval within a transaction
func (uc *GetInvoiceItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) (*invoicepb.GetInvoiceItemPageDataResponse, error) {
	var result *invoicepb.GetInvoiceItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "invoice.errors.get_item_page_data_failed", "")
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

// executeCore contains the core business logic for getting invoice item page data
func (uc *GetInvoiceItemPageDataUseCase) executeCore(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) (*invoicepb.GetInvoiceItemPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.Invoice.GetInvoiceItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetInvoiceItemPageDataUseCase) validateInput(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.request_required", ""))
	}

	if req.InvoiceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.id_required", "Invoice ID is required"))
	}

	// Validate ID format (basic validation)
	if len(req.InvoiceId) > 255 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.id_too_long", "Invoice ID cannot exceed 255 characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting invoice item page data
func (uc *GetInvoiceItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *invoicepb.GetInvoiceItemPageDataRequest) error {
	// Financial security: Ensure proper access control for invoice item data
	// Additional authorization checks would be implemented here in a real system
	// For example, certain invoices might be restricted based on user roles or client ownership

	// Business rule: Validate invoice access permissions
	// This would typically check if the current user has permission to view this specific invoice
	// In a real system, this might involve checking client ownership or admin privileges

	// Business rule: Ensure financial data integrity
	// Validate that only appropriate users can access detailed invoice information
	// This is critical for billing systems where invoices contain sensitive financial data

	return nil
}
