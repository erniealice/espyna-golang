package collection_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
)

type GetCollectionPlanListPageDataRepositories struct {
	CollectionPlan collectionplanpb.CollectionPlanDomainServiceServer
}

type GetCollectionPlanListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetCollectionPlanListPageDataUseCase handles the business logic for getting collection plan list page data
type GetCollectionPlanListPageDataUseCase struct {
	repositories GetCollectionPlanListPageDataRepositories
	services     GetCollectionPlanListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetCollectionPlanListPageDataUseCase creates a new GetCollectionPlanListPageDataUseCase
func NewGetCollectionPlanListPageDataUseCase(
	repositories GetCollectionPlanListPageDataRepositories,
	services GetCollectionPlanListPageDataServices,
) *GetCollectionPlanListPageDataUseCase {
	return &GetCollectionPlanListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get collection plan list page data operation
func (uc *GetCollectionPlanListPageDataUseCase) Execute(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanListPageDataRequest,
) (*collectionplanpb.GetCollectionPlanListPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection plan list page data retrieval within a transaction
func (uc *GetCollectionPlanListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanListPageDataRequest,
) (*collectionplanpb.GetCollectionPlanListPageDataResponse, error) {
	var result *collectionplanpb.GetCollectionPlanListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"collection_plan.errors.list_page_data_failed",
				"collection plan list page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting collection plan list page data
func (uc *GetCollectionPlanListPageDataUseCase) executeCore(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanListPageDataRequest,
) (*collectionplanpb.GetCollectionPlanListPageDataResponse, error) {
	// First, get all collection plans from the repository
	listReq := &collectionplanpb.ListCollectionPlansRequest{}
	listResp, err := uc.repositories.CollectionPlan.ListCollectionPlans(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.errors.list_failed",
			"failed to retrieve collection plans: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &collectionplanpb.GetCollectionPlanListPageDataResponse{
			CollectionPlanList: []*collectionplanpb.CollectionPlan{},
			Pagination:         emptyPagination,
			SearchResults:      []*commonpb.SearchResult{},
			Success:            true,
		}, nil
	}

	// Process the data with filtering, sorting, searching, and pagination
	result, err := uc.processor.ProcessListRequest(
		listResp.Data,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.errors.processing_failed",
			"failed to process collection plan list data: %w",
		), err)
	}

	// Convert processed items back to collection plan protobuf format
	collectionPlans := make([]*collectionplanpb.CollectionPlan, len(result.Items))
	for i, item := range result.Items {
		if collectionPlan, ok := item.(*collectionplanpb.CollectionPlan); ok {
			collectionPlans[i] = collectionPlan
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.errors.type_conversion_failed",
				"failed to convert item to collection plan type",
			))
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &collectionplanpb.GetCollectionPlanListPageDataResponse{
		CollectionPlanList: collectionPlans,
		Pagination:         result.PaginationResponse,
		SearchResults:      searchResults,
		Success:            true,
	}, nil
}

// validateInput validates the input request
func (uc *GetCollectionPlanListPageDataUseCase) validateInput(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.request_required",
			"request is required",
		))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	// Validate filters if provided
	if req.Filters != nil {
		if err := uc.validateFilters(ctx, req.Filters); err != nil {
			return err
		}
	}

	// Validate sort if provided
	if req.Sort != nil {
		if err := uc.validateSort(ctx, req.Sort); err != nil {
			return err
		}
	}

	// Validate search if provided
	if req.Search != nil {
		if err := uc.validateSearch(ctx, req.Search); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetCollectionPlanListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	// Validate specific pagination method
	switch method := pagination.Method.(type) {
	case *commonpb.PaginationRequest_Offset:
		if method.Offset.Page < 1 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.validation.invalid_page",
				"page number must be greater than 0",
			))
		}
	case *commonpb.PaginationRequest_Cursor:
		// Cursor validation could be more sophisticated
		if method.Cursor.Token == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.validation.invalid_cursor",
				"cursor token cannot be empty",
			))
		}
	}

	return nil
}

// validateFilters validates filter parameters
func (uc *GetCollectionPlanListPageDataUseCase) validateFilters(
	ctx context.Context,
	filters *commonpb.FilterRequest,
) error {
	if len(filters.Filters) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.empty_filters",
			"filters cannot be empty when filter request is provided",
		))
	}

	// Validate individual filters
	for i, filter := range filters.Filters {
		if filter.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.validation.filter_field_required",
				"filter field is required for filter %d",
			), i)
		}

		// Validate that the field exists on the collection plan entity
		if !uc.isValidCollectionPlanField(filter.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.validation.invalid_filter_field",
				"invalid filter field: %s",
			), filter.Field)
		}
	}

	return nil
}

// validateSort validates sort parameters
func (uc *GetCollectionPlanListPageDataUseCase) validateSort(
	ctx context.Context,
	sort *commonpb.SortRequest,
) error {
	if len(sort.Fields) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.empty_sort_fields",
			"sort fields cannot be empty when sort request is provided",
		))
	}

	// Validate individual sort fields
	for i, sortField := range sort.Fields {
		if sortField.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.validation.sort_field_required",
				"sort field is required for sort field %d",
			), i)
		}

		// Validate that the field exists on the collection plan entity
		if !uc.isValidCollectionPlanField(sortField.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.validation.invalid_sort_field",
				"invalid sort field: %s",
			), sortField.Field)
		}
	}

	return nil
}

// validateSearch validates search parameters
func (uc *GetCollectionPlanListPageDataUseCase) validateSearch(
	ctx context.Context,
	search *commonpb.SearchRequest,
) error {
	if search.Query == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.empty_search_query",
			"search query cannot be empty when search request is provided",
		))
	}

	if search.Options != nil {
		// Validate search fields if specified
		for _, field := range search.Options.SearchFields {
			if !uc.isValidCollectionPlanField(field) {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
					ctx,
					uc.services.TranslationService,
					"collection_plan.validation.invalid_search_field",
					"invalid search field: %s",
				), field)
			}
		}

		// Validate max results
		if search.Options.MaxResults < 0 || search.Options.MaxResults > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"collection_plan.validation.invalid_max_results",
				"max results must be between 0 and 1000",
			))
		}
	}

	return nil
}

// isValidCollectionPlanField checks if a field name is valid for collection plan filtering/sorting/searching
func (uc *GetCollectionPlanListPageDataUseCase) isValidCollectionPlanField(field string) bool {
	validFields := map[string]bool{
		"id":                   true,
		"name":                 true,
		"collection_id":        true,
		"plan_id":              true,
		"active":               true,
		"date_created":         true,
		"date_created_string":  true,
		"date_modified":        true,
		"date_modified_string": true,
		// Nested fields
		"collection.name": true,
		"collection.id":   true,
		"plan.name":       true,
		"plan.id":         true,
	}

	return validFields[field]
}
