package client

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// ListClientsRepositories groups all repository dependencies
type ListClientsRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
}

// ListClientsServices groups all business service dependencies
type ListClientsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListClientsUseCase handles the business logic for listing clients
type ListClientsUseCase struct {
	repositories ListClientsRepositories
	services     ListClientsServices
}

// NewListClientsUseCase creates use case with grouped dependencies
func NewListClientsUseCase(
	repositories ListClientsRepositories,
	services ListClientsServices,
) *ListClientsUseCase {
	return &ListClientsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListClientsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListClientsUseCase with grouped parameters instead
func NewListClientsUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer) *ListClientsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListClientsRepositories{
		Client: clientRepo,
	}

	services := ListClientsServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListClientsUseCase(repositories, services)
}

// Execute performs the list clients operation
func (uc *ListClientsUseCase) Execute(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClient, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &clientpb.ListClientsRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Client.ListClients(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.errors.list_failed", "Failed to retrieve clients [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
