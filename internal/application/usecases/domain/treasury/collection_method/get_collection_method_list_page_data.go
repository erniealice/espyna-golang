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

// GetCollectionMethodListPageDataRepositories groups all repository dependencies.
type GetCollectionMethodListPageDataRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// GetCollectionMethodListPageDataServices groups all business service dependencies.
type GetCollectionMethodListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCollectionMethodListPageDataUseCase handles fetching paginated, searchable list data.
type GetCollectionMethodListPageDataUseCase struct {
	repositories GetCollectionMethodListPageDataRepositories
	services     GetCollectionMethodListPageDataServices
}

// NewGetCollectionMethodListPageDataUseCase creates use case with grouped dependencies.
func NewGetCollectionMethodListPageDataUseCase(
	repositories GetCollectionMethodListPageDataRepositories,
	services GetCollectionMethodListPageDataServices,
) *GetCollectionMethodListPageDataUseCase {
	return &GetCollectionMethodListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get collection method list page data operation.
func (uc *GetCollectionMethodListPageDataUseCase) Execute(ctx context.Context, req *collectionmethodpb.GetCollectionMethodListPageDataRequest) (*collectionmethodpb.GetCollectionMethodListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethod, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.CollectionMethod == nil {
		return nil, errors.New("collection method repository is not available")
	}
	resp, err := uc.repositories.CollectionMethod.GetCollectionMethodListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load collection method list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetCollectionMethodListPageDataUseCase) validateInput(ctx context.Context, req *collectionmethodpb.GetCollectionMethodListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
