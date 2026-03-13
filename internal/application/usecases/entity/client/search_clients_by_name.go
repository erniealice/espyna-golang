package client

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// SearchClientsByNameRepositories groups all repository dependencies
type SearchClientsByNameRepositories struct {
	Client clientpb.ClientDomainServiceServer
}

// SearchClientsByNameServices groups all business service dependencies
type SearchClientsByNameServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClient, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		req = &clientpb.SearchClientsByNameRequest{}
	}

	resp, err := uc.repositories.Client.SearchClientsByName(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.errors.search_failed", "Failed to search clients [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
