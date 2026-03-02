package supplier

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// GetSupplierItemPageDataRepositories groups repository dependencies for GetSupplierItemPageData use case
type GetSupplierItemPageDataRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer
}

// GetSupplierItemPageDataServices groups service dependencies for GetSupplierItemPageData use case
type GetSupplierItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetSupplierItemPageDataUseCase handles getting individual supplier item data
type GetSupplierItemPageDataUseCase struct {
	supplierpb.UnimplementedSupplierDomainServiceServer
	repos    GetSupplierItemPageDataRepositories
	services GetSupplierItemPageDataServices
}

// NewGetSupplierItemPageDataUseCase creates a new GetSupplierItemPageData use case
func NewGetSupplierItemPageDataUseCase(
	repos GetSupplierItemPageDataRepositories,
	services GetSupplierItemPageDataServices,
) *GetSupplierItemPageDataUseCase {
	return &GetSupplierItemPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetSupplierItemPageData use case
func (uc *GetSupplierItemPageDataUseCase) Execute(
	ctx context.Context,
	req *supplierpb.GetSupplierItemPageDataRequest,
) (*supplierpb.GetSupplierItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier", ports.ActionList); err != nil {
		return nil, err
	}

	// Delegate to the repository layer
	return uc.repos.Supplier.GetSupplierItemPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ supplierpb.SupplierDomainServiceServer = (*GetSupplierItemPageDataUseCase)(nil)

// Required SupplierDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetSupplierItemPageDataUseCase) CreateSupplier(ctx context.Context, req *supplierpb.CreateSupplierRequest) (*supplierpb.CreateSupplierResponse, error) {
	return uc.repos.Supplier.CreateSupplier(ctx, req)
}

func (uc *GetSupplierItemPageDataUseCase) ReadSupplier(ctx context.Context, req *supplierpb.ReadSupplierRequest) (*supplierpb.ReadSupplierResponse, error) {
	return uc.repos.Supplier.ReadSupplier(ctx, req)
}

func (uc *GetSupplierItemPageDataUseCase) UpdateSupplier(ctx context.Context, req *supplierpb.UpdateSupplierRequest) (*supplierpb.UpdateSupplierResponse, error) {
	return uc.repos.Supplier.UpdateSupplier(ctx, req)
}

func (uc *GetSupplierItemPageDataUseCase) DeleteSupplier(ctx context.Context, req *supplierpb.DeleteSupplierRequest) (*supplierpb.DeleteSupplierResponse, error) {
	return uc.repos.Supplier.DeleteSupplier(ctx, req)
}

func (uc *GetSupplierItemPageDataUseCase) ListSuppliers(ctx context.Context, req *supplierpb.ListSuppliersRequest) (*supplierpb.ListSuppliersResponse, error) {
	return uc.repos.Supplier.ListSuppliers(ctx, req)
}

func (uc *GetSupplierItemPageDataUseCase) GetSupplierListPageData(ctx context.Context, req *supplierpb.GetSupplierListPageDataRequest) (*supplierpb.GetSupplierListPageDataResponse, error) {
	return uc.repos.Supplier.GetSupplierListPageData(ctx, req)
}

func (uc *GetSupplierItemPageDataUseCase) GetSupplierItemPageData(ctx context.Context, req *supplierpb.GetSupplierItemPageDataRequest) (*supplierpb.GetSupplierItemPageDataResponse, error) {
	return uc.repos.Supplier.GetSupplierItemPageData(ctx, req)
}
