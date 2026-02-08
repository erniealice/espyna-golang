package invoice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// UpdateInvoiceRepositories groups all repository dependencies
type UpdateInvoiceRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// UpdateInvoiceServices groups all business service dependencies
type UpdateInvoiceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateInvoiceUseCase handles the business logic for updating invoices
type UpdateInvoiceUseCase struct {
	repositories UpdateInvoiceRepositories
	services     UpdateInvoiceServices
}

// NewUpdateInvoiceUseCase creates a new UpdateInvoiceUseCase
func NewUpdateInvoiceUseCase(
	repositories UpdateInvoiceRepositories,
	services UpdateInvoiceServices,
) *UpdateInvoiceUseCase {
	return &UpdateInvoiceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update invoice operation
func (uc *UpdateInvoiceUseCase) Execute(ctx context.Context, req *invoicepb.UpdateInvoiceRequest) (*invoicepb.UpdateInvoiceResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInvoice, ports.ActionUpdate)
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
	response, err := uc.repositories.Invoice.UpdateInvoice(ctx, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateInput validates the input request
func (uc *UpdateInvoiceUseCase) validateInput(ctx context.Context, req *invoicepb.UpdateInvoiceRequest) error {
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

// enrichInvoiceData adds audit information for updates
func (uc *UpdateInvoiceUseCase) enrichInvoiceData(invoice *invoicepb.Invoice) error {
	now := time.Now()

	// Update modification timestamp
	invoice.DateModified = &[]int64{now.UnixMilli()}[0]
	invoice.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for invoice updates
func (uc *UpdateInvoiceUseCase) validateBusinessRules(ctx context.Context, invoice *invoicepb.Invoice) error {
	// Validate invoice ID format
	if len(invoice.Id) < 3 {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.id_format", "invoice ID must be at least 3 characters long")
		return errors.New(errorMsg)
	}

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
