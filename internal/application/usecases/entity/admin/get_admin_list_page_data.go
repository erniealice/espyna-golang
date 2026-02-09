package admin

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
)

// GetAdminListPageDataRepositories groups repository dependencies for GetAdminListPageData use case
type GetAdminListPageDataRepositories struct {
	Admin adminpb.AdminDomainServiceServer
}

// GetAdminListPageDataServices groups service dependencies for GetAdminListPageData use case
type GetAdminListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetAdminListPageDataUseCase handles getting paginated admin list data with search, filtering, and sorting
type GetAdminListPageDataUseCase struct {
	adminpb.UnimplementedAdminDomainServiceServer
	repos    GetAdminListPageDataRepositories
	services GetAdminListPageDataServices
}

// NewGetAdminListPageDataUseCase creates a new GetAdminListPageData use case
func NewGetAdminListPageDataUseCase(
	repos GetAdminListPageDataRepositories,
	services GetAdminListPageDataServices,
) *GetAdminListPageDataUseCase {
	return &GetAdminListPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetAdminListPageData use case
func (uc *GetAdminListPageDataUseCase) Execute(
	ctx context.Context,
	req *adminpb.GetAdminListPageDataRequest,
) (*adminpb.GetAdminListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityAdmin, ports.ActionList); err != nil {
		return nil, err
	}

	return uc.repos.Admin.GetAdminListPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ adminpb.AdminDomainServiceServer = (*GetAdminListPageDataUseCase)(nil)

// Required AdminDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetAdminListPageDataUseCase) CreateAdmin(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	return uc.repos.Admin.CreateAdmin(ctx, req)
}

func (uc *GetAdminListPageDataUseCase) ReadAdmin(ctx context.Context, req *adminpb.ReadAdminRequest) (*adminpb.ReadAdminResponse, error) {
	return uc.repos.Admin.ReadAdmin(ctx, req)
}

func (uc *GetAdminListPageDataUseCase) UpdateAdmin(ctx context.Context, req *adminpb.UpdateAdminRequest) (*adminpb.UpdateAdminResponse, error) {
	return uc.repos.Admin.UpdateAdmin(ctx, req)
}

func (uc *GetAdminListPageDataUseCase) DeleteAdmin(ctx context.Context, req *adminpb.DeleteAdminRequest) (*adminpb.DeleteAdminResponse, error) {
	return uc.repos.Admin.DeleteAdmin(ctx, req)
}

func (uc *GetAdminListPageDataUseCase) ListAdmins(ctx context.Context, req *adminpb.ListAdminsRequest) (*adminpb.ListAdminsResponse, error) {
	return uc.repos.Admin.ListAdmins(ctx, req)
}

func (uc *GetAdminListPageDataUseCase) GetAdminItemPageData(ctx context.Context, req *adminpb.GetAdminItemPageDataRequest) (*adminpb.GetAdminItemPageDataResponse, error) {
	return uc.repos.Admin.GetAdminItemPageData(ctx, req)
}

func (uc *GetAdminListPageDataUseCase) GetAdminListPageData(ctx context.Context, req *adminpb.GetAdminListPageDataRequest) (*adminpb.GetAdminListPageDataResponse, error) {
	return uc.repos.Admin.GetAdminListPageData(ctx, req)
}
