package invoice_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
)

// GetInvoiceAttributeListPageDataRepositories groups all repository dependencies
type GetInvoiceAttributeListPageDataRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
}

// GetInvoiceAttributeListPageDataServices groups all business service dependencies
type GetInvoiceAttributeListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetInvoiceAttributeListPageDataUseCase handles the business logic for getting invoice attribute list page data
type GetInvoiceAttributeListPageDataUseCase struct {
	repositories GetInvoiceAttributeListPageDataRepositories
	services     GetInvoiceAttributeListPageDataServices
}

// NewGetInvoiceAttributeListPageDataUseCase creates a new GetInvoiceAttributeListPageDataUseCase
func NewGetInvoiceAttributeListPageDataUseCase(
	repositories GetInvoiceAttributeListPageDataRepositories,
	services GetInvoiceAttributeListPageDataServices,
) *GetInvoiceAttributeListPageDataUseCase {
	return &GetInvoiceAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get invoice attribute list page data operation
func (uc *GetInvoiceAttributeListPageDataUseCase) Execute(ctx context.Context, req *invoiceattributepb.GetInvoiceAttributeListPageDataRequest) (*invoiceattributepb.GetInvoiceAttributeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.InvoiceAttribute, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.InvoiceAttribute.GetInvoiceAttributeListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetInvoiceAttributeListPageDataUseCase) validateInput(ctx context.Context, req *invoiceattributepb.GetInvoiceAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "invoice_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
