package invoice

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// ReadInvoiceRepositories groups all repository dependencies
type ReadInvoiceRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// ReadInvoiceServices groups all business service dependencies
type ReadInvoiceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadInvoiceUseCase handles the business logic for reading invoices
type ReadInvoiceUseCase struct {
	repositories ReadInvoiceRepositories
	services     ReadInvoiceServices
}

// NewReadInvoiceUseCase creates a new ReadInvoiceUseCase
func NewReadInvoiceUseCase(
	repositories ReadInvoiceRepositories,
	services ReadInvoiceServices,
) *ReadInvoiceUseCase {
	return &ReadInvoiceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read invoice operation
func (uc *ReadInvoiceUseCase) Execute(ctx context.Context, req *invoicepb.ReadInvoiceRequest) (*invoicepb.ReadInvoiceResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInvoice, ports.ActionRead)
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

	// Call repository
	response, err := uc.repositories.Invoice.ReadInvoice(ctx, req)
	if err != nil {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.not_found", "")
		errorMsg = strings.ReplaceAll(errorMsg, "{invoiceId}", req.Data.Id)
		return nil, errors.New(errorMsg)
	}

	// Not found error
	if response == nil || len(response.Data) == 0 {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.not_found", "")
		errorMsg = strings.ReplaceAll(errorMsg, "{invoiceId}", req.Data.Id)
		return nil, errors.New(errorMsg)
	}

	return response, nil
}

// validateInput validates the input request
func (uc *ReadInvoiceUseCase) validateInput(ctx context.Context, req *invoicepb.ReadInvoiceRequest) error {
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

// validateBusinessRules enforces business constraints for invoice reading
func (uc *ReadInvoiceUseCase) validateBusinessRules(ctx context.Context, req *invoicepb.ReadInvoiceRequest) error {
	// Validate invoice ID format
	if len(req.Data.Id) < 3 {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.id_format", "invoice ID must be at least 3 characters long")
		return errors.New(errorMsg)
	}

	return nil
}
