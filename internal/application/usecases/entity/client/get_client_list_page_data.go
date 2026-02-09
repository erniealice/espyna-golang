package client

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// GetClientListPageDataRepositories groups repository dependencies for GetClientListPageData use case
type GetClientListPageDataRepositories struct {
	Client clientpb.ClientDomainServiceServer
}

// GetClientListPageDataServices groups service dependencies for GetClientListPageData use case
type GetClientListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetClientListPageDataUseCase handles getting paginated client list data with search, filtering, and sorting
type GetClientListPageDataUseCase struct {
	clientpb.UnimplementedClientDomainServiceServer
	repos    GetClientListPageDataRepositories
	services GetClientListPageDataServices
}

// NewGetClientListPageDataUseCase creates a new GetClientListPageData use case
func NewGetClientListPageDataUseCase(
	repos GetClientListPageDataRepositories,
	services GetClientListPageDataServices,
) *GetClientListPageDataUseCase {
	return &GetClientListPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetClientListPageData use case
func (uc *GetClientListPageDataUseCase) Execute(
	ctx context.Context,
	req *clientpb.GetClientListPageDataRequest,
) (*clientpb.GetClientListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClient, ports.ActionList); err != nil {
		return nil, err
	}

	// For now, delegate to the repository layer
	// In the future, this could include business logic like:
	// - Permission checking
	// - Data transformation
	// - Caching
	// - Metrics/logging
	return uc.repos.Client.GetClientListPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ clientpb.ClientDomainServiceServer = (*GetClientListPageDataUseCase)(nil)

// Required ClientDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetClientListPageDataUseCase) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	return uc.repos.Client.CreateClient(ctx, req)
}

func (uc *GetClientListPageDataUseCase) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	return uc.repos.Client.ReadClient(ctx, req)
}

func (uc *GetClientListPageDataUseCase) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	return uc.repos.Client.UpdateClient(ctx, req)
}

func (uc *GetClientListPageDataUseCase) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	return uc.repos.Client.DeleteClient(ctx, req)
}

func (uc *GetClientListPageDataUseCase) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
	return uc.repos.Client.ListClients(ctx, req)
}

func (uc *GetClientListPageDataUseCase) GetClientItemPageData(ctx context.Context, req *clientpb.GetClientItemPageDataRequest) (*clientpb.GetClientItemPageDataResponse, error) {
	return uc.repos.Client.GetClientItemPageData(ctx, req)
}

func (uc *GetClientListPageDataUseCase) GetClientListPageData(ctx context.Context, req *clientpb.GetClientListPageDataRequest) (*clientpb.GetClientListPageDataResponse, error) {
	return uc.repos.Client.GetClientListPageData(ctx, req)
}
