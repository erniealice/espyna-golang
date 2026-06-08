package invoice

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// ListInvoicesRepositories groups all repository dependencies
type ListInvoicesRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// ListInvoicesServices groups all business service dependencies
type ListInvoicesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator // Current: Text translation and localization
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Invoice, entityid.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.Invoice, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "invoice.errors.authorization_failed", "Authorization failed for billing statements [DEFAULT]")
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
		errorMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "invoice.validation.request_required", "request is required")
		return errors.New(errorMsg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for invoice listing
func (uc *ListInvoicesUseCase) validateBusinessRules(ctx context.Context, req *invoicepb.ListInvoicesRequest) error {
	// No specific business rules for listing invoices
	return nil
}
