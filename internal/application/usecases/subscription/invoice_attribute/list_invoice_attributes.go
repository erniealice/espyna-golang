package invoice_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
)

// ListInvoiceAttributesRepositories groups all repository dependencies
type ListInvoiceAttributesRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
}

// ListInvoiceAttributesServices groups all business service dependencies
type ListInvoiceAttributesServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListInvoiceAttributesUseCase handles the business logic for listing invoice attributes
type ListInvoiceAttributesUseCase struct {
	repositories ListInvoiceAttributesRepositories
	services     ListInvoiceAttributesServices
}

// NewListInvoiceAttributesUseCase creates a new ListInvoiceAttributesUseCase
func NewListInvoiceAttributesUseCase(
	repositories ListInvoiceAttributesRepositories,
	services ListInvoiceAttributesServices,
) *ListInvoiceAttributesUseCase {
	return &ListInvoiceAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list invoice attributes operation
func (uc *ListInvoiceAttributesUseCase) Execute(ctx context.Context, req *invoiceattributepb.ListInvoiceAttributesRequest) (*invoiceattributepb.ListInvoiceAttributesResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.InvoiceAttribute.ListInvoiceAttributes(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListInvoiceAttributesUseCase) validateInput(ctx context.Context, req *invoiceattributepb.ListInvoiceAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
