package collection

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// ReadCollectionRepositories groups all repository dependencies
type ReadCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// ReadCollectionServices groups all business service dependencies
type ReadCollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadCollectionUseCase handles the business logic for reading a collection
type ReadCollectionUseCase struct {
	repositories ReadCollectionRepositories
	services     ReadCollectionServices
}

// NewReadCollectionUseCase creates use case with grouped dependencies
func NewReadCollectionUseCase(
	repositories ReadCollectionRepositories,
	services ReadCollectionServices,
) *ReadCollectionUseCase {
	return &ReadCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read collection operation
func (uc *ReadCollectionUseCase) Execute(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityCollection, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.id_required", "Collection ID is required [DEFAULT]"))
	}

	return uc.repositories.Collection.ReadCollection(ctx, req)
}
