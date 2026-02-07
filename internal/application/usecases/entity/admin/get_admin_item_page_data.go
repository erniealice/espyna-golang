package admin

import (
	"context"

	"leapfor.xyz/espyna/internal/application/ports"
	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
)

// GetAdminItemPageDataRepositories groups repository dependencies for GetAdminItemPageData use case
type GetAdminItemPageDataRepositories struct {
	Admin adminpb.AdminDomainServiceServer
}

// GetAdminItemPageDataServices groups service dependencies for GetAdminItemPageData use case
type GetAdminItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetAdminItemPageDataUseCase handles getting individual admin item data
type GetAdminItemPageDataUseCase struct {
	adminpb.UnimplementedAdminDomainServiceServer
	repos    GetAdminItemPageDataRepositories
	services GetAdminItemPageDataServices
}

// NewGetAdminItemPageDataUseCase creates a new GetAdminItemPageData use case
func NewGetAdminItemPageDataUseCase(
	repos GetAdminItemPageDataRepositories,
	services GetAdminItemPageDataServices,
) *GetAdminItemPageDataUseCase {
	return &GetAdminItemPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetAdminItemPageData use case
func (uc *GetAdminItemPageDataUseCase) Execute(
	ctx context.Context,
	req *adminpb.GetAdminItemPageDataRequest,
) (*adminpb.GetAdminItemPageDataResponse, error) {
	// For now, delegate to the repository layer
	// In the future, this could include business logic like:
	// - Permission checking
	// - Data transformation
	// - Related data loading
	// - Metrics/logging
	return uc.repos.Admin.GetAdminItemPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ adminpb.AdminDomainServiceServer = (*GetAdminItemPageDataUseCase)(nil)

// Required AdminDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetAdminItemPageDataUseCase) CreateAdmin(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	return uc.repos.Admin.CreateAdmin(ctx, req)
}

func (uc *GetAdminItemPageDataUseCase) ReadAdmin(ctx context.Context, req *adminpb.ReadAdminRequest) (*adminpb.ReadAdminResponse, error) {
	return uc.repos.Admin.ReadAdmin(ctx, req)
}

func (uc *GetAdminItemPageDataUseCase) UpdateAdmin(ctx context.Context, req *adminpb.UpdateAdminRequest) (*adminpb.UpdateAdminResponse, error) {
	return uc.repos.Admin.UpdateAdmin(ctx, req)
}

func (uc *GetAdminItemPageDataUseCase) DeleteAdmin(ctx context.Context, req *adminpb.DeleteAdminRequest) (*adminpb.DeleteAdminResponse, error) {
	return uc.repos.Admin.DeleteAdmin(ctx, req)
}

func (uc *GetAdminItemPageDataUseCase) ListAdmins(ctx context.Context, req *adminpb.ListAdminsRequest) (*adminpb.ListAdminsResponse, error) {
	return uc.repos.Admin.ListAdmins(ctx, req)
}

func (uc *GetAdminItemPageDataUseCase) GetAdminListPageData(ctx context.Context, req *adminpb.GetAdminListPageDataRequest) (*adminpb.GetAdminListPageDataResponse, error) {
	return uc.repos.Admin.GetAdminListPageData(ctx, req)
}

func (uc *GetAdminItemPageDataUseCase) GetAdminItemPageData(ctx context.Context, req *adminpb.GetAdminItemPageDataRequest) (*adminpb.GetAdminItemPageDataResponse, error) {
	return uc.repos.Admin.GetAdminItemPageData(ctx, req)
}
