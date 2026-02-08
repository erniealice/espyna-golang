package price_list

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
)

type GetPriceListListPageDataRepositories struct {
	PriceList pricelistpb.PriceListDomainServiceServer
}

type GetPriceListListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetPriceListListPageDataUseCase handles the business logic for getting price list list page data
type GetPriceListListPageDataUseCase struct {
	repositories GetPriceListListPageDataRepositories
	services     GetPriceListListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetPriceListListPageDataUseCase creates a new GetPriceListListPageDataUseCase
func NewGetPriceListListPageDataUseCase(
	repositories GetPriceListListPageDataRepositories,
	services GetPriceListListPageDataServices,
) *GetPriceListListPageDataUseCase {
	return &GetPriceListListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get price list list page data operation
func (uc *GetPriceListListPageDataUseCase) Execute(
	ctx context.Context,
	req *pricelistpb.GetPriceListListPageDataRequest,
) (*pricelistpb.GetPriceListListPageDataResponse, error) {
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

// executeWithTransaction executes price list list page data retrieval within a transaction
func (uc *GetPriceListListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pricelistpb.GetPriceListListPageDataRequest,
) (*pricelistpb.GetPriceListListPageDataResponse, error) {
	var result *pricelistpb.GetPriceListListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"price_list.errors.list_page_data_failed",
				"price list list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting price list list page data
func (uc *GetPriceListListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pricelistpb.GetPriceListListPageDataRequest,
) (*pricelistpb.GetPriceListListPageDataResponse, error) {
	// First, get all price lists from the repository
	listReq := &pricelistpb.ListPriceListsRequest{}
	listResp, err := uc.repositories.PriceList.ListPriceLists(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.errors.list_failed",
			"failed to retrieve price lists: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &pricelistpb.GetPriceListListPageDataResponse{
			PriceListList: []*pricelistpb.PriceList{},
			Pagination:    emptyPagination,
			SearchResults: []*commonpb.SearchResult{},
			Success:       true,
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
			"price_list.errors.processing_failed",
			"failed to process price list list data: %w",
		), err)
	}

	// Convert processed items back to price list protobuf format
	priceLists := make([]*pricelistpb.PriceList, len(result.Items))
	for i, item := range result.Items {
		if priceList, ok := item.(*pricelistpb.PriceList); ok {
			priceLists[i] = priceList
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.errors.type_conversion_failed",
				"failed to convert item to price list type",
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

	return &pricelistpb.GetPriceListListPageDataResponse{
		PriceListList: priceLists,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// validateInput validates the input request
func (uc *GetPriceListListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pricelistpb.GetPriceListListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.request_required",
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
func (uc *GetPriceListListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	switch method := pagination.Method.(type) {
	case *commonpb.PaginationRequest_Offset:
		if method.Offset.Page < 1 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.validation.invalid_page",
				"page number must be greater than 0",
			))
		}
	case *commonpb.PaginationRequest_Cursor:
		if method.Cursor.Token == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.validation.invalid_cursor",
				"cursor token cannot be empty",
			))
		}
	}

	return nil
}

// validateFilters validates filter parameters
func (uc *GetPriceListListPageDataUseCase) validateFilters(
	ctx context.Context,
	filters *commonpb.FilterRequest,
) error {
	if len(filters.Filters) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.empty_filters",
			"filters cannot be empty when filter request is provided",
		))
	}

	for i, filter := range filters.Filters {
		if filter.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.validation.filter_field_required",
				"filter field is required for filter %d",
			), i)
		}

		if !uc.isValidPriceListField(filter.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.validation.invalid_filter_field",
				"invalid filter field: %s",
			), filter.Field)
		}
	}

	return nil
}

// validateSort validates sort parameters
func (uc *GetPriceListListPageDataUseCase) validateSort(
	ctx context.Context,
	sort *commonpb.SortRequest,
) error {
	if len(sort.Fields) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.empty_sort_fields",
			"sort fields cannot be empty when sort request is provided",
		))
	}

	for i, sortField := range sort.Fields {
		if sortField.Field == "" {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.validation.sort_field_required",
				"sort field is required for sort field %d",
			), i)
		}

		if !uc.isValidPriceListField(sortField.Field) {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.validation.invalid_sort_field",
				"invalid sort field: %s",
			), sortField.Field)
		}
	}

	return nil
}

// validateSearch validates search parameters
func (uc *GetPriceListListPageDataUseCase) validateSearch(
	ctx context.Context,
	search *commonpb.SearchRequest,
) error {
	if search.Query == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.empty_search_query",
			"search query cannot be empty when search request is provided",
		))
	}

	if search.Options != nil {
		for _, field := range search.Options.SearchFields {
			if !uc.isValidPriceListField(field) {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
					ctx,
					uc.services.TranslationService,
					"price_list.validation.invalid_search_field",
					"invalid search field: %s",
				), field)
			}
		}

		if search.Options.MaxResults < 0 || search.Options.MaxResults > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"price_list.validation.invalid_max_results",
				"max results must be between 0 and 1000",
			))
		}
	}

	return nil
}

// isValidPriceListField checks if a field name is valid for price list filtering/sorting/searching
func (uc *GetPriceListListPageDataUseCase) isValidPriceListField(field string) bool {
	validFields := map[string]bool{
		"id":                   true,
		"name":                 true,
		"description":          true,
		"active":               true,
		"date_start":           true,
		"date_start_string":    true,
		"date_end":             true,
		"date_end_string":      true,
		"location_id":          true,
		"date_created":         true,
		"date_created_string":  true,
		"date_modified":        true,
		"date_modified_string": true,
	}

	return validFields[field]
}
