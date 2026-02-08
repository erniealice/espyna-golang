package invoice_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
)

// ReadInvoiceAttributeRepositories groups all repository dependencies
type ReadInvoiceAttributeRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
}

// ReadInvoiceAttributeServices groups all business service dependencies
type ReadInvoiceAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadInvoiceAttributeUseCase handles the business logic for reading invoice attributes
type ReadInvoiceAttributeUseCase struct {
	repositories ReadInvoiceAttributeRepositories
	services     ReadInvoiceAttributeServices
}

// NewReadInvoiceAttributeUseCase creates a new ReadInvoiceAttributeUseCase
func NewReadInvoiceAttributeUseCase(
	repositories ReadInvoiceAttributeRepositories,
	services ReadInvoiceAttributeServices,
) *ReadInvoiceAttributeUseCase {
	return &ReadInvoiceAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read invoice attribute operation
func (uc *ReadInvoiceAttributeUseCase) Execute(ctx context.Context, req *invoiceattributepb.ReadInvoiceAttributeRequest) (*invoiceattributepb.ReadInvoiceAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.InvoiceAttribute.ReadInvoiceAttribute(ctx, req)
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
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadInvoiceAttributeUseCase) validateInput(ctx context.Context, req *invoiceattributepb.ReadInvoiceAttributeRequest) error {
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
