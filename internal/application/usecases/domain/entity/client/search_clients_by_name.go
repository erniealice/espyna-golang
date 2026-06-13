package client

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// SearchClientsByNameRepositories groups all repository dependencies
type SearchClientsByNameRepositories struct {
	Client clientpb.ClientDomainServiceServer
}

// SearchClientsByNameServices groups all business service dependencies
type SearchClientsByNameServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SearchClientsByNameUseCase handles the business logic for searching clients by name
type SearchClientsByNameUseCase struct {
	repositories SearchClientsByNameRepositories
	services     SearchClientsByNameServices
}

// NewSearchClientsByNameUseCase creates use case with grouped dependencies
func NewSearchClientsByNameUseCase(
	repositories SearchClientsByNameRepositories,
	services SearchClientsByNameServices,
) *SearchClientsByNameUseCase {
	return &SearchClientsByNameUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the search clients by name operation
func (uc *SearchClientsByNameUseCase) Execute(ctx context.Context, req *clientpb.SearchClientsByNameRequest) (*clientpb.SearchClientsByNameResponse, error) {
	// Authorization check — search is a read/list operation
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Client,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		req = &clientpb.SearchClientsByNameRequest{}
	}

	resp, err := uc.repositories.Client.SearchClientsByName(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.errors.search_failed", "Failed to search clients [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
