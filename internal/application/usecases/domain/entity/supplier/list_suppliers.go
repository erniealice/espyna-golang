package supplier

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// ListSuppliersRepositories groups all repository dependencies
type ListSuppliersRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
}

// ListSuppliersServices groups all business service dependencies
type ListSuppliersServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListSuppliersUseCase handles the business logic for listing suppliers
type ListSuppliersUseCase struct {
	repositories ListSuppliersRepositories
	services     ListSuppliersServices
}

// NewListSuppliersUseCase creates use case with grouped dependencies
func NewListSuppliersUseCase(
	repositories ListSuppliersRepositories,
	services ListSuppliersServices,
) *ListSuppliersUseCase {
	return &ListSuppliersUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListSuppliersUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListSuppliersUseCase with grouped parameters instead
func NewListSuppliersUseCaseUngrouped(supplierRepo supplierpb.SupplierDomainServiceServer) *ListSuppliersUseCase {
	repositories := ListSuppliersRepositories{
		Supplier: supplierRepo,
	}

	services := ListSuppliersServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListSuppliersUseCase(repositories, services)
}

// Execute performs the list suppliers operation
func (uc *ListSuppliersUseCase) Execute(ctx context.Context, req *supplierpb.ListSuppliersRequest) (*supplierpb.ListSuppliersResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "supplier",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &supplierpb.ListSuppliersRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Supplier.ListSuppliers(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier.errors.list_failed", "Failed to retrieve suppliers [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
