package client

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
)

// UpdateClientRepositories groups all repository dependencies
type UpdateClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
}

// UpdateClientServices groups all business service dependencies
type UpdateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateClientUseCase handles the business logic for updating a client
type UpdateClientUseCase struct {
	repositories UpdateClientRepositories
	services     UpdateClientServices
}

// NewUpdateClientUseCase creates use case with grouped dependencies
func NewUpdateClientUseCase(
	repositories UpdateClientRepositories,
	services UpdateClientServices,
) *UpdateClientUseCase {
	return &UpdateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateClientUseCase with grouped parameters instead
func NewUpdateClientUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer) *UpdateClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateClientRepositories{
		Client: clientRepo,
	}

	services := UpdateClientServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateClientUseCase(repositories, services)
}

// Execute performs the update client operation
func (uc *UpdateClientUseCase) Execute(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.request_required", "Request is required for clients [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.id_required", "Client ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.User != nil && req.Data.User.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.email_required", "Client email is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Client.UpdateClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.errors.update_failed", "Client update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
