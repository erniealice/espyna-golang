package collection

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// DeleteCollectionRepositories groups all repository dependencies
type DeleteCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// DeleteCollectionServices groups all business service dependencies
type DeleteCollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteCollectionUseCase handles the business logic for deleting collections
type DeleteCollectionUseCase struct {
	repositories DeleteCollectionRepositories
	services     DeleteCollectionServices
}

// NewDeleteCollectionUseCase creates a new DeleteCollectionUseCase
func NewDeleteCollectionUseCase(
	repositories DeleteCollectionRepositories,
	services DeleteCollectionServices,
) *DeleteCollectionUseCase {
	return &DeleteCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete collection operation
func (uc *DeleteCollectionUseCase) Execute(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityCollection, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.id_required", "Collection ID is required [DEFAULT]"))
	}

	return uc.repositories.Collection.DeleteCollection(ctx, req)
}
