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

// CreateInvoiceAttributeRepositories groups all repository dependencies
type CreateInvoiceAttributeRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
	Invoice          invoicepb.InvoiceDomainServiceServer                   // Entity reference validation
	Attribute        attributepb.AttributeDomainServiceServer               // Entity reference validation
}

// CreateInvoiceAttributeServices groups all business service dependencies
type CreateInvoiceAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateInvoiceAttributeUseCase handles the business logic for creating invoice attributes
type CreateInvoiceAttributeUseCase struct {
	repositories CreateInvoiceAttributeRepositories
	services     CreateInvoiceAttributeServices
}

// NewCreateInvoiceAttributeUseCase creates use case with grouped dependencies
func NewCreateInvoiceAttributeUseCase(
	repositories CreateInvoiceAttributeRepositories,
	services CreateInvoiceAttributeServices,
) *CreateInvoiceAttributeUseCase {
	return &CreateInvoiceAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create invoice attribute operation
func (uc *CreateInvoiceAttributeUseCase) Execute(ctx context.Context, req *invoiceattributepb.CreateInvoiceAttributeRequest) (*invoiceattributepb.CreateInvoiceAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// TODO: Re-enable workspace-scoped authorization check once Invoice.WorkspaceId is available
	// if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
	// 	userID := contextutil.ExtractUserIDFromContext(ctx)
	// 	if userID == "" {
	// 		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.user_not_authenticated", "User not authenticated [DEFAULT]"))
	// 	}
	//
	// 	permission := ports.EntityPermission(ports.EntityInvoiceAttribute, ports.ActionCreate)
	// 	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	// 	if err != nil {
	// 		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.authorization_failed", "Authorization failed [DEFAULT]")
	// 		return nil, fmt.Errorf("%s: %w", translatedError, err)
	// 	}
	// 	if !hasPerm {
	// 		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.authorization_failed", "Authorization failed [DEFAULT]")
	// 		return nil, errors.New(translatedError)
	// 	}
	// }

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
	resp, err := uc.repositories.InvoiceAttribute.CreateInvoiceAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.errors.creation_failed", "Invoice attribute creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateInvoiceAttributeUseCase) validateInput(ctx context.Context, req *invoiceattributepb.CreateInvoiceAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.InvoiceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.invoice_id_required", "Invoice ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice_attribute.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichInvoiceAttributeData adds generated fields and audit information
func (uc *CreateInvoiceAttributeUseCase) enrichInvoiceAttributeData(invoiceAttribute *invoiceattributepb.InvoiceAttribute) error {
	now := time.Now()

	// Generate InvoiceAttribute ID
	if invoiceAttribute.Id == "" {
		invoiceAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set invoice attribute audit fields
	invoiceAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	invoiceAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	invoiceAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	invoiceAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	invoiceAttribute.Active = true

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateInvoiceAttributeUseCase) validateEntityReferences(ctx context.Context, invoiceAttribute *invoiceattributepb.InvoiceAttribute) error {
	// Validate Invoice entity reference
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

	// Validate Attribute entity reference
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
