package invoice

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// DeleteInvoiceRepositories groups all repository dependencies
type DeleteInvoiceRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// DeleteInvoiceServices groups all business service dependencies
type DeleteInvoiceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteInvoiceUseCase handles the business logic for deleting invoices
type DeleteInvoiceUseCase struct {
	repositories DeleteInvoiceRepositories
	services     DeleteInvoiceServices
}

// NewDeleteInvoiceUseCase creates a new DeleteInvoiceUseCase
func NewDeleteInvoiceUseCase(
	repositories DeleteInvoiceRepositories,
	services DeleteInvoiceServices,
) *DeleteInvoiceUseCase {
	return &DeleteInvoiceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete invoice operation
func (uc *DeleteInvoiceUseCase) Execute(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) (*invoicepb.DeleteInvoiceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInvoice, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInvoice, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository with error handling
	response, err := uc.repositories.Invoice.DeleteInvoice(ctx, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateInput validates the input request
func (uc *DeleteInvoiceUseCase) validateInput(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) error {
	if req == nil {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.request_required", "request is required")
		return errors.New(errorMsg)
	}
	if req.Data == nil {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.data_required", "invoice data is required")
		return errors.New(errorMsg)
	}
	if req.Data.Id == "" {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.id_required", "invoice ID is required")
		return errors.New(errorMsg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for invoice deletion
func (uc *DeleteInvoiceUseCase) validateBusinessRules(ctx context.Context, req *invoicepb.DeleteInvoiceRequest) error {
	// Validate invoice ID format
	if len(req.Data.Id) < 3 {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.id_format", "invoice ID must be at least 3 characters long")
		return errors.New(errorMsg)
	}

	// Additional business rules for deletion can be added here
	// For example, preventing deletion of paid invoices or invoices with payments

	return nil
}
