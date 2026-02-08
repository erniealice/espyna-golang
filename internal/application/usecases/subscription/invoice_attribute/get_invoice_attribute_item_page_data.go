package invoice_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
)

// GetInvoiceAttributeItemPageDataRepositories groups all repository dependencies
type GetInvoiceAttributeItemPageDataRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
}

// GetInvoiceAttributeItemPageDataServices groups all business service dependencies
type GetInvoiceAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetInvoiceAttributeItemPageDataUseCase handles the business logic for getting invoice attribute item page data
type GetInvoiceAttributeItemPageDataUseCase struct {
	repositories GetInvoiceAttributeItemPageDataRepositories
	services     GetInvoiceAttributeItemPageDataServices
}

// NewGetInvoiceAttributeItemPageDataUseCase creates a new GetInvoiceAttributeItemPageDataUseCase
func NewGetInvoiceAttributeItemPageDataUseCase(
	repositories GetInvoiceAttributeItemPageDataRepositories,
	services GetInvoiceAttributeItemPageDataServices,
) *GetInvoiceAttributeItemPageDataUseCase {
	return &GetInvoiceAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get invoice attribute item page data operation
func (uc *GetInvoiceAttributeItemPageDataUseCase) Execute(ctx context.Context, req *invoiceattributepb.GetInvoiceAttributeItemPageDataRequest) (*invoiceattributepb.GetInvoiceAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.InvoiceAttribute.GetInvoiceAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.item_page_data_failed", "Failed to retrieve invoice attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetInvoiceAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *invoiceattributepb.GetInvoiceAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.request_required", "Request is required for invoice attributes [DEFAULT]"))
	}

	// Validate invoice attribute ID - uses direct field req.InvoiceAttributeId
	if strings.TrimSpace(req.InvoiceAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.id_required", "Invoice attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.InvoiceAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.id_too_short", "Invoice attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
