package collectionmethod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// GetCollectionMethodItemPageDataRepositories groups all repository dependencies.
type GetCollectionMethodItemPageDataRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// GetCollectionMethodItemPageDataServices groups all business service dependencies.
type GetCollectionMethodItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCollectionMethodItemPageDataUseCase handles fetching a single enriched item.
type GetCollectionMethodItemPageDataUseCase struct {
	repositories GetCollectionMethodItemPageDataRepositories
	services     GetCollectionMethodItemPageDataServices
}

// NewGetCollectionMethodItemPageDataUseCase creates use case with grouped dependencies.
func NewGetCollectionMethodItemPageDataUseCase(
	repositories GetCollectionMethodItemPageDataRepositories,
	services GetCollectionMethodItemPageDataServices,
) *GetCollectionMethodItemPageDataUseCase {
	return &GetCollectionMethodItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get collection method item page data operation.
func (uc *GetCollectionMethodItemPageDataUseCase) Execute(ctx context.Context, req *collectionmethodpb.GetCollectionMethodItemPageDataRequest) (*collectionmethodpb.GetCollectionMethodItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.CollectionMethodId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.id_required", "Collection method ID is required [DEFAULT]"))
	}

	if uc.repositories.CollectionMethod == nil {
		return nil, errors.New("collection method repository is not available")
	}
	resp, err := uc.repositories.CollectionMethod.GetCollectionMethodItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load collection method")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
