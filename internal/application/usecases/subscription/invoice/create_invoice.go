package invoice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// CreateInvoiceRepositories groups all repository dependencies
type CreateInvoiceRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// CreateInvoiceServices groups all business service dependencies
type CreateInvoiceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateInvoiceUseCase handles the business logic for creating invoices
type CreateInvoiceUseCase struct {
	repositories CreateInvoiceRepositories
	services     CreateInvoiceServices
}

// NewCreateInvoiceUseCase creates a new CreateInvoiceUseCase
func NewCreateInvoiceUseCase(
	repositories CreateInvoiceRepositories,
	services CreateInvoiceServices,
) *CreateInvoiceUseCase {
	return &CreateInvoiceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create invoice operation
func (uc *CreateInvoiceUseCase) Execute(ctx context.Context, req *invoicepb.CreateInvoiceRequest) (*invoicepb.CreateInvoiceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInvoice, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInvoice, ports.ActionCreate)
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

	// Business logic and enrichment
	if err := uc.enrichInvoiceData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository with error handling
	response, err := uc.repositories.Invoice.CreateInvoice(ctx, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateInput validates the input request
func (uc *CreateInvoiceUseCase) validateInput(ctx context.Context, req *invoicepb.CreateInvoiceRequest) error {
	if req == nil {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.request_required", "request is required")
		return errors.New(errorMsg)
	}
	if req.Data == nil {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.data_required", "invoice data is required")
		return errors.New(errorMsg)
	}
	return nil
}

// enrichInvoiceData adds generated fields and audit information
func (uc *CreateInvoiceUseCase) enrichInvoiceData(invoice *invoicepb.Invoice) error {
	now := time.Now()

	// Generate Invoice ID if not provided
	if invoice.Id == "" {
		invoice.Id = uc.services.IDService.GenerateID()
	}

	// Generate Invoice Number if not provided
	if invoice.InvoiceNumber == "" {
		invoice.InvoiceNumber = fmt.Sprintf("INV-%d", now.UnixNano())
	}

	// Set audit fields
	invoice.DateCreated = &[]int64{now.UnixMilli()}[0]
	invoice.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	invoice.DateModified = &[]int64{now.UnixMilli()}[0]
	invoice.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	invoice.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for invoices
func (uc *CreateInvoiceUseCase) validateBusinessRules(ctx context.Context, invoice *invoicepb.Invoice) error {
	// Validate invoice number uniqueness (this would typically involve checking the repository)
	if strings.TrimSpace(invoice.InvoiceNumber) == "" {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.number_required", "invoice number cannot be empty")
		return errors.New(errorMsg)
	}

	// Validate amount constraints
	if invoice.Amount <= 0 {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.amount_positive", "invoice amount must be greater than 0")
		return errors.New(errorMsg)
	}

	// Note: Currency field does not exist in Invoice protobuf

	// Note: SubscriptionId field does not exist in Invoice protobuf

	// Note: ClientId field does not exist in Invoice protobuf

	return nil
}
