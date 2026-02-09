package invoice

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// ListInvoicesRepositories groups all repository dependencies
type ListInvoicesRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// ListInvoicesServices groups all business service dependencies
type ListInvoicesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService   // Current: Text translation and localization
}

// ListInvoicesUseCase handles the business logic for listing invoices
type ListInvoicesUseCase struct {
	repositories ListInvoicesRepositories
	services     ListInvoicesServices
}

// NewListInvoicesUseCase creates a new ListInvoicesUseCase
func NewListInvoicesUseCase(
	repositories ListInvoicesRepositories,
	services ListInvoicesServices,
) *ListInvoicesUseCase {
	return &ListInvoicesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list invoices operation
func (uc *ListInvoicesUseCase) Execute(ctx context.Context, req *invoicepb.ListInvoicesRequest) (*invoicepb.ListInvoicesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityInvoice, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityInvoice, ports.ActionList)
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

	// Call repository with error handling
	response, err := uc.repositories.Invoice.ListInvoices(ctx, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateInput validates the input request
func (uc *ListInvoicesUseCase) validateInput(ctx context.Context, req *invoicepb.ListInvoicesRequest) error {
	if req == nil {
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "invoice.validation.request_required", "request is required")
		return errors.New(errorMsg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for invoice listing
func (uc *ListInvoicesUseCase) validateBusinessRules(ctx context.Context, req *invoicepb.ListInvoicesRequest) error {
	// No specific business rules for listing invoices
	return nil
}
