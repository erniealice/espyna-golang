package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// DeleteClientRepositories groups all repository dependencies
type DeleteClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
}

// DeleteClientServices groups all business service dependencies
type DeleteClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteClientUseCase handles the business logic for deleting a client
type DeleteClientUseCase struct {
	repositories DeleteClientRepositories
	services     DeleteClientServices
}

// NewDeleteClientUseCase creates use case with grouped dependencies
func NewDeleteClientUseCase(
	repositories DeleteClientRepositories,
	services DeleteClientServices,
) *DeleteClientUseCase {
	return &DeleteClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteClientUseCase with grouped parameters instead
func NewDeleteClientUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer) *DeleteClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteClientRepositories{
		Client: clientRepo,
	}

	services := DeleteClientServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteClientUseCase(repositories, services)
}

// Execute performs the delete client operation
func (uc *DeleteClientUseCase) Execute(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.request_required", "Request is required for clients [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.id_required", "Client ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Client.DeleteClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.errors.deletion_failed", "Client deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
