package client

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// ReadClientRepositories groups all repository dependencies
type ReadClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
}

// ReadClientServices groups all business service dependencies
type ReadClientServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadClientUseCase handles the business logic for reading a client
type ReadClientUseCase struct {
	repositories ReadClientRepositories
	services     ReadClientServices
}

// NewReadClientUseCase creates use case with grouped dependencies
func NewReadClientUseCase(
	repositories ReadClientRepositories,
	services ReadClientServices,
) *ReadClientUseCase {
	return &ReadClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadClientUseCase with grouped parameters instead
func NewReadClientUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer) *ReadClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadClientRepositories{
		Client: clientRepo,
	}

	services := ReadClientServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewReadClientUseCase(repositories, services)
}

// Execute performs the read client operation
func (uc *ReadClientUseCase) Execute(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Client,
		Action: entityid.ActionRead,
	}); err != nil {
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
	resp, err := uc.repositories.Client.ReadClient(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" { // Assuming resp.Data will be nil or have empty ID if not found
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.errors.not_found", "Client with ID \"{clientId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{clientId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}
