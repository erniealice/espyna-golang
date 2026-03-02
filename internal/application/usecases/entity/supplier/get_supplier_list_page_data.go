package supplier

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// GetSupplierListPageDataRepositories groups repository dependencies for GetSupplierListPageData use case
type GetSupplierListPageDataRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer
}

// GetSupplierListPageDataServices groups service dependencies for GetSupplierListPageData use case
type GetSupplierListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetSupplierListPageDataUseCase handles getting paginated supplier list data with search, filtering, and sorting
type GetSupplierListPageDataUseCase struct {
	supplierpb.UnimplementedSupplierDomainServiceServer
	repos    GetSupplierListPageDataRepositories
	services GetSupplierListPageDataServices
}

// NewGetSupplierListPageDataUseCase creates a new GetSupplierListPageData use case
func NewGetSupplierListPageDataUseCase(
	repos GetSupplierListPageDataRepositories,
	services GetSupplierListPageDataServices,
) *GetSupplierListPageDataUseCase {
	return &GetSupplierListPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetSupplierListPageData use case
func (uc *GetSupplierListPageDataUseCase) Execute(
	ctx context.Context,
	req *supplierpb.GetSupplierListPageDataRequest,
) (*supplierpb.GetSupplierListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier", ports.ActionList); err != nil {
		return nil, err
	}

	// Delegate to the repository layer
	return uc.repos.Supplier.GetSupplierListPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ supplierpb.SupplierDomainServiceServer = (*GetSupplierListPageDataUseCase)(nil)

// Required SupplierDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetSupplierListPageDataUseCase) CreateSupplier(ctx context.Context, req *supplierpb.CreateSupplierRequest) (*supplierpb.CreateSupplierResponse, error) {
	return uc.repos.Supplier.CreateSupplier(ctx, req)
}

func (uc *GetSupplierListPageDataUseCase) ReadSupplier(ctx context.Context, req *supplierpb.ReadSupplierRequest) (*supplierpb.ReadSupplierResponse, error) {
	return uc.repos.Supplier.ReadSupplier(ctx, req)
}

func (uc *GetSupplierListPageDataUseCase) UpdateSupplier(ctx context.Context, req *supplierpb.UpdateSupplierRequest) (*supplierpb.UpdateSupplierResponse, error) {
	return uc.repos.Supplier.UpdateSupplier(ctx, req)
}

func (uc *GetSupplierListPageDataUseCase) DeleteSupplier(ctx context.Context, req *supplierpb.DeleteSupplierRequest) (*supplierpb.DeleteSupplierResponse, error) {
	return uc.repos.Supplier.DeleteSupplier(ctx, req)
}

func (uc *GetSupplierListPageDataUseCase) ListSuppliers(ctx context.Context, req *supplierpb.ListSuppliersRequest) (*supplierpb.ListSuppliersResponse, error) {
	return uc.repos.Supplier.ListSuppliers(ctx, req)
}

func (uc *GetSupplierListPageDataUseCase) GetSupplierItemPageData(ctx context.Context, req *supplierpb.GetSupplierItemPageDataRequest) (*supplierpb.GetSupplierItemPageDataResponse, error) {
	return uc.repos.Supplier.GetSupplierItemPageData(ctx, req)
}

func (uc *GetSupplierListPageDataUseCase) GetSupplierListPageData(ctx context.Context, req *supplierpb.GetSupplierListPageDataRequest) (*supplierpb.GetSupplierListPageDataResponse, error) {
	return uc.repos.Supplier.GetSupplierListPageData(ctx, req)
}
