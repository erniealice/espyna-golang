package collection

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// ListCollectionsRepositories groups all repository dependencies
type ListCollectionsRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// ListCollectionsServices groups all business service dependencies
type ListCollectionsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListCollectionsUseCase handles the business logic for listing collections
type ListCollectionsUseCase struct {
	repositories ListCollectionsRepositories
	services     ListCollectionsServices
}

// NewListCollectionsUseCase creates a new ListCollectionsUseCase
func NewListCollectionsUseCase(
	repositories ListCollectionsRepositories,
	services ListCollectionsServices,
) *ListCollectionsUseCase {
	return &ListCollectionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list collections operation
func (uc *ListCollectionsUseCase) Execute(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityCollection, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Collection.ListCollections(ctx, req)
}
