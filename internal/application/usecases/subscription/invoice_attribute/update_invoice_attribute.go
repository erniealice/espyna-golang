package invoice_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	invoicepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice"
	invoiceattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice_attribute"
)

// UpdateInvoiceAttributeRepositories groups all repository dependencies
type UpdateInvoiceAttributeRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
	Invoice          invoicepb.InvoiceDomainServiceServer                   // Entity reference validation
	Attribute        attributepb.AttributeDomainServiceServer               // Entity reference validation
}

// UpdateInvoiceAttributeServices groups all business service dependencies
type UpdateInvoiceAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateInvoiceAttributeUseCase handles the business logic for updating invoice attributes
type UpdateInvoiceAttributeUseCase struct {
	repositories UpdateInvoiceAttributeRepositories
	services     UpdateInvoiceAttributeServices
}

// NewUpdateInvoiceAttributeUseCase creates a new UpdateInvoiceAttributeUseCase
func NewUpdateInvoiceAttributeUseCase(
	repositories UpdateInvoiceAttributeRepositories,
	services UpdateInvoiceAttributeServices,
) *UpdateInvoiceAttributeUseCase {
	return &UpdateInvoiceAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update invoice attribute operation
func (uc *UpdateInvoiceAttributeUseCase) Execute(ctx context.Context, req *invoiceattributepb.UpdateInvoiceAttributeRequest) (*invoiceattributepb.UpdateInvoiceAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichInvoiceAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.InvoiceAttribute.UpdateInvoiceAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.update_failed", "Invoice attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateInvoiceAttributeUseCase) validateInput(ctx context.Context, req *invoiceattributepb.UpdateInvoiceAttributeRequest) error {
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

// enrichInvoiceAttributeData updates audit information
func (uc *UpdateInvoiceAttributeUseCase) enrichInvoiceAttributeData(invoiceAttribute *invoiceattributepb.InvoiceAttribute) error {
	now := time.Now()

	// Update audit fields
	invoiceAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	invoiceAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateInvoiceAttributeUseCase) validateEntityReferences(ctx context.Context, invoiceAttribute *invoiceattributepb.InvoiceAttribute) error {
	// Validate Invoice entity reference (if being updated)
	if invoiceAttribute.InvoiceId != "" {
		invoice, err := uc.repositories.Invoice.ReadInvoice(ctx, &invoicepb.ReadInvoiceRequest{
			Data: &invoicepb.Invoice{Id: invoiceAttribute.InvoiceId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.invoice_reference_validation_failed", "Failed to validate invoice entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if invoice == nil || invoice.Data == nil || len(invoice.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.invoice_not_found", "Invoice not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{invoiceId}", invoiceAttribute.InvoiceId)
			return errors.New(translatedError)
		}
		if !invoice.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.invoice_not_active", "Referenced invoice with ID '{invoiceId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{invoiceId}", invoiceAttribute.InvoiceId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference (if being updated)
	if invoiceAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: invoiceAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.attribute_not_found", "Attribute not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", invoiceAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", invoiceAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
