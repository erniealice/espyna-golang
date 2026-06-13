package delegate_client

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// ReadDelegateClientRepositories groups all repository dependencies
type ReadDelegateClientRepositories struct {
	DelegateClient delegateclientpb.DelegateClientDomainServiceServer // Primary entity repository
	Delegate       delegatepb.DelegateDomainServiceServer             // Entity reference validation
	Client         clientpb.ClientDomainServiceServer                 // Entity reference validation
}

// ReadDelegateClientServices groups all business service dependencies
type ReadDelegateClientServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadDelegateClientUseCase handles the business logic for reading a delegate client
type ReadDelegateClientUseCase struct {
	repositories ReadDelegateClientRepositories
	services     ReadDelegateClientServices
}

// NewReadDelegateClientUseCase creates use case with grouped dependencies
func NewReadDelegateClientUseCase(
	repositories ReadDelegateClientRepositories,
	services ReadDelegateClientServices,
) *ReadDelegateClientUseCase {
	return &ReadDelegateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadDelegateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadDelegateClientUseCase with grouped parameters instead
func NewReadDelegateClientUseCaseUngrouped(
	delegateClientRepo delegateclientpb.DelegateClientDomainServiceServer,
) *ReadDelegateClientUseCase {
	repositories := ReadDelegateClientRepositories{
		DelegateClient: delegateClientRepo,
		Delegate:       nil,
		Client:         nil,
	}
	services := ReadDelegateClientServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewReadDelegateClientUseCase(repositories, services)
}

// Execute performs the read delegate client operation
func (uc *ReadDelegateClientUseCase) Execute(ctx context.Context, req *delegateclientpb.ReadDelegateClientRequest) (*delegateclientpb.ReadDelegateClientResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate_client.validation.id_required", "Delegate-Client relationship ID is required [DEFAULT]"))
	}

	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.DelegateClient,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.DelegateClient.ReadDelegateClient(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return empty result for missing entity (following established pattern)
	if len(resp.Data) == 0 {
		return &delegateclientpb.ReadDelegateClientResponse{
			Data:    []*delegateclientpb.DelegateClient{},
			Success: true,
		}, nil
	}

	return resp, nil
}

// Helper functions
