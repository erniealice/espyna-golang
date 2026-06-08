package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// DeleteClientRepositories groups all repository dependencies
type DeleteClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
}

// DeleteClientServices groups all business service dependencies
type DeleteClientServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewDeleteClientUseCase(repositories, services)
}

// Execute performs the delete client operation
func (uc *DeleteClientUseCase) Execute(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Client, entityid.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.request_required", "Request is required for clients [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.id_required", "Client ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Client.DeleteClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.errors.deletion_failed", "Client deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
