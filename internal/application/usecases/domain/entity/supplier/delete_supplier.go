package supplier

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// DeleteSupplierRepositories groups all repository dependencies
type DeleteSupplierRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
}

// DeleteSupplierServices groups all business service dependencies
type DeleteSupplierServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteSupplierUseCase handles the business logic for deleting a supplier
type DeleteSupplierUseCase struct {
	repositories DeleteSupplierRepositories
	services     DeleteSupplierServices
}

// NewDeleteSupplierUseCase creates use case with grouped dependencies
func NewDeleteSupplierUseCase(
	repositories DeleteSupplierRepositories,
	services DeleteSupplierServices,
) *DeleteSupplierUseCase {
	return &DeleteSupplierUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteSupplierUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteSupplierUseCase with grouped parameters instead
func NewDeleteSupplierUseCaseUngrouped(supplierRepo supplierpb.SupplierDomainServiceServer) *DeleteSupplierUseCase {
	repositories := DeleteSupplierRepositories{
		Supplier: supplierRepo,
	}

	services := DeleteSupplierServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewDeleteSupplierUseCase(repositories, services)
}

// Execute performs the delete supplier operation
func (uc *DeleteSupplierUseCase) Execute(ctx context.Context, req *supplierpb.DeleteSupplierRequest) (*supplierpb.DeleteSupplierResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"supplier", entityid.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.validation.request_required", "Request is required for suppliers [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.validation.id_required", "Supplier ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Supplier.DeleteSupplier(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.errors.deletion_failed", "Supplier deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
