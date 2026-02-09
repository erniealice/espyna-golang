package client

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// GetClientItemPageDataRepositories groups repository dependencies for GetClientItemPageData use case
type GetClientItemPageDataRepositories struct {
	Client clientpb.ClientDomainServiceServer
}

// GetClientItemPageDataServices groups service dependencies for GetClientItemPageData use case
type GetClientItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetClientItemPageDataUseCase handles getting individual client item data
type GetClientItemPageDataUseCase struct {
	clientpb.UnimplementedClientDomainServiceServer
	repos    GetClientItemPageDataRepositories
	services GetClientItemPageDataServices
}

// NewGetClientItemPageDataUseCase creates a new GetClientItemPageData use case
func NewGetClientItemPageDataUseCase(
	repos GetClientItemPageDataRepositories,
	services GetClientItemPageDataServices,
) *GetClientItemPageDataUseCase {
	return &GetClientItemPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetClientItemPageData use case
func (uc *GetClientItemPageDataUseCase) Execute(
	ctx context.Context,
	req *clientpb.GetClientItemPageDataRequest,
) (*clientpb.GetClientItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClient, ports.ActionList); err != nil {
		return nil, err
	}

	// For now, delegate to the repository layer
	// In the future, this could include business logic like:
	// - Permission checking
	// - Data transformation
	// - Related data loading
	// - Metrics/logging
	return uc.repos.Client.GetClientItemPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ clientpb.ClientDomainServiceServer = (*GetClientItemPageDataUseCase)(nil)

// Required ClientDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetClientItemPageDataUseCase) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	return uc.repos.Client.CreateClient(ctx, req)
}

func (uc *GetClientItemPageDataUseCase) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	return uc.repos.Client.ReadClient(ctx, req)
}

func (uc *GetClientItemPageDataUseCase) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	return uc.repos.Client.UpdateClient(ctx, req)
}

func (uc *GetClientItemPageDataUseCase) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	return uc.repos.Client.DeleteClient(ctx, req)
}

func (uc *GetClientItemPageDataUseCase) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
	return uc.repos.Client.ListClients(ctx, req)
}

func (uc *GetClientItemPageDataUseCase) GetClientListPageData(ctx context.Context, req *clientpb.GetClientListPageDataRequest) (*clientpb.GetClientListPageDataResponse, error) {
	return uc.repos.Client.GetClientListPageData(ctx, req)
}

func (uc *GetClientItemPageDataUseCase) GetClientItemPageData(ctx context.Context, req *clientpb.GetClientItemPageDataRequest) (*clientpb.GetClientItemPageDataResponse, error) {
	return uc.repos.Client.GetClientItemPageData(ctx, req)
}
