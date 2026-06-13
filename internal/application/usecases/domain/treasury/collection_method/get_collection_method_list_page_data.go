package collection_method

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// GetCollectionMethodListPageDataRepositories groups all repository dependencies
type GetCollectionMethodListPageDataRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// GetCollectionMethodListPageDataServices groups all business service dependencies
type GetCollectionMethodListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetCollectionMethodListPageDataUseCase handles the business logic for getting collection method list page data with pagination, filtering, sorting, and search
type GetCollectionMethodListPageDataUseCase struct {
	repositories GetCollectionMethodListPageDataRepositories
	services     GetCollectionMethodListPageDataServices
}

// NewGetCollectionMethodListPageDataUseCase creates use case with grouped dependencies
func NewGetCollectionMethodListPageDataUseCase(
	repositories GetCollectionMethodListPageDataRepositories,
	services GetCollectionMethodListPageDataServices,
) *GetCollectionMethodListPageDataUseCase {
	return &GetCollectionMethodListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get collection method list page data operation
func (uc *GetCollectionMethodListPageDataUseCase) Execute(ctx context.Context, req *collectionmethodpb.GetCollectionMethodListPageDataRequest) (*collectionmethodpb.GetCollectionMethodListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.CollectionMethod, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionMethod.GetCollectionMethodListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load collection method list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetCollectionMethodListPageDataUseCase) validateInput(ctx context.Context, req *collectionmethodpb.GetCollectionMethodListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetCollectionMethodListPageDataUseCase) validateBusinessRules(ctx context.Context, req *collectionmethodpb.GetCollectionMethodListPageDataRequest) error {
	// For now, we'll allow all authenticated users to view collection method lists
	return nil
}
