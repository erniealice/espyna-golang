package invoice_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	invoiceattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice_attribute"
)

// DeleteInvoiceAttributeRepositories groups all repository dependencies
type DeleteInvoiceAttributeRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
}

// DeleteInvoiceAttributeServices groups all business service dependencies
type DeleteInvoiceAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteInvoiceAttributeUseCase handles the business logic for deleting invoice attributes
type DeleteInvoiceAttributeUseCase struct {
	repositories DeleteInvoiceAttributeRepositories
	services     DeleteInvoiceAttributeServices
}

// NewDeleteInvoiceAttributeUseCase creates a new DeleteInvoiceAttributeUseCase
func NewDeleteInvoiceAttributeUseCase(
	repositories DeleteInvoiceAttributeRepositories,
	services DeleteInvoiceAttributeServices,
) *DeleteInvoiceAttributeUseCase {
	return &DeleteInvoiceAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete invoice attribute operation
func (uc *DeleteInvoiceAttributeUseCase) Execute(ctx context.Context, req *invoiceattributepb.DeleteInvoiceAttributeRequest) (*invoiceattributepb.DeleteInvoiceAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.InvoiceAttribute.DeleteInvoiceAttribute(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("invoice_attribute with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"invoice_attribute.errors.not_found",
				map[string]interface{}{"invoiceAttributeId": req.Data.Id},
				"Invoice attribute not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.deletion_failed", "Invoice attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteInvoiceAttributeUseCase) validateInput(ctx context.Context, req *invoiceattributepb.DeleteInvoiceAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.id_required", "Invoice attribute ID is required [DEFAULT]"))
	}
	return nil
}
