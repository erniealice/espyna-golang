package collectionmethodgrant

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// GetCollectionMethodGrantListPageDataRepositories groups all repository dependencies.
type GetCollectionMethodGrantListPageDataRepositories struct {
	CollectionMethodGrant grantpb.CollectionMethodGrantDomainServiceServer
}

// GetCollectionMethodGrantListPageDataServices groups all business service dependencies.
type GetCollectionMethodGrantListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCollectionMethodGrantListPageDataUseCase handles fetching paginated, searchable list data.
type GetCollectionMethodGrantListPageDataUseCase struct {
	repositories GetCollectionMethodGrantListPageDataRepositories
	services     GetCollectionMethodGrantListPageDataServices
}

// NewGetCollectionMethodGrantListPageDataUseCase creates use case with grouped dependencies.
func NewGetCollectionMethodGrantListPageDataUseCase(
	repositories GetCollectionMethodGrantListPageDataRepositories,
	services GetCollectionMethodGrantListPageDataServices,
) *GetCollectionMethodGrantListPageDataUseCase {
	return &GetCollectionMethodGrantListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get grant list page data operation.
func (uc *GetCollectionMethodGrantListPageDataUseCase) Execute(ctx context.Context, req *grantpb.GetCollectionMethodGrantListPageDataRequest) (*grantpb.GetCollectionMethodGrantListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityCollectionMethodGrant, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.CollectionMethodGrant == nil {
		return nil, errors.New("collection method grant repository is not available")
	}
	resp, err := uc.repositories.CollectionMethodGrant.GetCollectionMethodGrantListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load collection method grant list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetCollectionMethodGrantListPageDataUseCase) validateInput(ctx context.Context, req *grantpb.GetCollectionMethodGrantListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method_grant.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
