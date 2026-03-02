package supplier_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// DeleteSupplierAttributeRepositories groups all repository dependencies
type DeleteSupplierAttributeRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
}

// DeleteSupplierAttributeServices groups all business service dependencies
type DeleteSupplierAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteSupplierAttributeUseCase handles the business logic for deleting supplier attributes
type DeleteSupplierAttributeUseCase struct {
	repositories DeleteSupplierAttributeRepositories
	services     DeleteSupplierAttributeServices
}

// NewDeleteSupplierAttributeUseCase creates use case with grouped dependencies
func NewDeleteSupplierAttributeUseCase(
	repositories DeleteSupplierAttributeRepositories,
	services DeleteSupplierAttributeServices,
) *DeleteSupplierAttributeUseCase {
	return &DeleteSupplierAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteSupplierAttributeUseCaseUngrouped creates a new DeleteSupplierAttributeUseCase
// Deprecated: Use NewDeleteSupplierAttributeUseCase with grouped parameters instead
func NewDeleteSupplierAttributeUseCaseUngrouped(supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer) *DeleteSupplierAttributeUseCase {
	repositories := DeleteSupplierAttributeRepositories{
		SupplierAttribute: supplierAttributeRepo,
	}

	services := DeleteSupplierAttributeServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteSupplierAttributeUseCase(repositories, services)
}

// Execute performs the delete supplier attribute operation
func (uc *DeleteSupplierAttributeUseCase) Execute(ctx context.Context, req *supplierattributepb.DeleteSupplierAttributeRequest) (*supplierattributepb.DeleteSupplierAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_attribute", ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SupplierAttribute.DeleteSupplierAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.deletion_failed", "Supplier attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteSupplierAttributeUseCase) validateInput(ctx context.Context, req *supplierattributepb.DeleteSupplierAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.request_required", "Request is required for supplier attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.data_required", "Supplier attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.id_required", "Supplier attribute ID is required [DEFAULT]"))
	}
	return nil
}
