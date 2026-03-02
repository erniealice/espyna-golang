package supplier

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// ListSuppliersRepositories groups all repository dependencies
type ListSuppliersRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
}

// ListSuppliersServices groups all business service dependencies
type ListSuppliersServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListSuppliersUseCase(repositories, services)
}

// Execute performs the list suppliers operation
func (uc *ListSuppliersUseCase) Execute(ctx context.Context, req *supplierpb.ListSuppliersRequest) (*supplierpb.ListSuppliersResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier", ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &supplierpb.ListSuppliersRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Supplier.ListSuppliers(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.errors.list_failed", "Failed to retrieve suppliers [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
