package event

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
)

type GetEventListPageDataRepositories struct {
	Event eventpb.EventDomainServiceServer
}

type GetEventListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetEventListPageDataUseCase handles the business logic for getting event list page data
// with specialized time-based filtering and calendar optimization
type GetEventListPageDataUseCase struct {
	repositories GetEventListPageDataRepositories
	services     GetEventListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetEventListPageDataUseCase creates a new GetEventListPageDataUseCase
func NewGetEventListPageDataUseCase(
	repositories GetEventListPageDataRepositories,
	services GetEventListPageDataServices,
) *GetEventListPageDataUseCase {
	return &GetEventListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get event list page data operation with time-based optimization
func (uc *GetEventListPageDataUseCase) Execute(
	ctx context.Context,
	req *eventpb.GetEventListPageDataRequest,
) (*eventpb.GetEventListPageDataResponse, error) {
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

// executeWithTransaction executes event list page data retrieval within a transaction
func (uc *GetEventListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *eventpb.GetEventListPageDataRequest,
) (*eventpb.GetEventListPageDataResponse, error) {
	var result *eventpb.GetEventListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"event.errors.list_page_data_failed",
				"event list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting event list page data
func (uc *GetEventListPageDataUseCase) executeCore(
	ctx context.Context,
	req *eventpb.GetEventListPageDataRequest,
) (*eventpb.GetEventListPageDataResponse, error) {
	// First, get all events from the repository
	listReq := &eventpb.ListEventsRequest{}
	listResp, err := uc.repositories.Event.ListEvents(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.errors.list_failed",
			"failed to retrieve events: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &eventpb.GetEventListPageDataResponse{
			EventList:     []*eventpb.Event{},
			Pagination:    emptyPagination,
			SearchResults: []*commonpb.SearchResult{},
			Success:       true,
		}, nil
	}

	// Process the data with filtering, sorting, searching, and pagination
	// Events require special handling for time-based queries
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
			"event.errors.processing_failed",
			"failed to process event list data: %w",
		), err)
	}

	// Convert processed items back to event protobuf format
	events := make([]*eventpb.Event, len(result.Items))
	for i, item := range result.Items {
		if event, ok := item.(*eventpb.Event); ok {
			events[i] = event
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"event.errors.type_conversion_failed",
				"failed to convert item to event type",
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

	return &eventpb.GetEventListPageDataResponse{
		EventList:     events,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// validateInput validates the input request
func (uc *GetEventListPageDataUseCase) validateInput(
	ctx context.Context,
	req *eventpb.GetEventListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.request_required",
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
func (uc *GetEventListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.invalid_limit",
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
				"event.validation.invalid_page",
				"page number must be greater than 0",
			))
		}
	case *commonpb.PaginationRequest_Cursor:
		// Cursor validation could be more sophisticated
		if method.Cursor.Token == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"event.validation.invalid_cursor",
				"cursor token cannot be empty",
			))
		}
	}

	return nil
}

// validateFilters validates filter parameters with time-based considerations
func (uc *GetEventListPageDataUseCase) validateFilters(
	ctx context.Context,
	filters *commonpb.FilterRequest,
) error {
	if len(filters.Filters) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.empty_filters",
			"filters cannot be empty when filter request is provided",
		))
	}

	// Basic validation - detailed filter validation is handled by the listdata processor
	for i, filter := range filters.Filters {
		if filter == nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"event.validation.filter_required",
				"filter %d cannot be nil",
			), i)
		}

		// Special validation for time-based fields
		if uc.isTimeBasedFilterType(filter) {
			if err := uc.validateTimeBasedFilter(ctx, filter); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateSort validates sort parameters with time-based considerations
func (uc *GetEventListPageDataUseCase) validateSort(
	ctx context.Context,
	sort *commonpb.SortRequest,
) error {
	if len(sort.Fields) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.empty_sort_fields",
			"sort fields cannot be empty when sort request is provided",
		))
	}

	// Basic validation - detailed field validation is handled by the listdata processor
	for i, sortField := range sort.Fields {
		if sortField == nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"event.validation.sort_field_required",
				"sort field %d cannot be nil",
			), i)
		}
	}

	return nil
}

// validateSearch validates search parameters with event-specific fields
func (uc *GetEventListPageDataUseCase) validateSearch(
	ctx context.Context,
	search *commonpb.SearchRequest,
) error {
	if search.Query == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.empty_search_query",
			"search query cannot be empty when search request is provided",
		))
	}

	if search.Options != nil {
		// Validate search fields if specified
		for _, field := range search.Options.SearchFields {
			if !uc.isValidEventField(field) {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
					ctx,
					uc.services.TranslationService,
					"event.validation.invalid_search_field",
					"invalid search field: %s",
				), field)
			}
		}

		// Validate max results
		if search.Options.MaxResults < 0 || search.Options.MaxResults > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"event.validation.invalid_max_results",
				"max results must be between 0 and 1000",
			))
		}
	}

	return nil
}

// isValidEventField checks if a field name is valid for event filtering/sorting/searching
func (uc *GetEventListPageDataUseCase) isValidEventField(field string) bool {
	validFields := map[string]bool{
		// Basic event fields
		"id":                   true,
		"name":                 true,
		"description":          true,
		"active":               true,
		"date_created":         true,
		"date_created_string":  true,
		"date_modified":        true,
		"date_modified_string": true,
		// Time-based fields (crucial for scheduling)
		"start_date_time_utc":        true,
		"end_date_time_utc":          true,
		"start_date_time_utc_string": true,
		"end_date_time_utc_string":   true,
		"timezone":                   true,
		// Virtual/computed fields for time-based queries
		"duration":      true,
		"is_past":       true,
		"is_future":     true,
		"is_today":      true,
		"is_this_week":  true,
		"is_this_month": true,
		"weekday":       true,
		"hour":          true,
	}

	return validFields[field]
}

// isTimeBasedField checks if a field is time-related and needs special validation
func (uc *GetEventListPageDataUseCase) isTimeBasedField(field string) bool {
	timeFields := map[string]bool{
		"start_date_time_utc":        true,
		"end_date_time_utc":          true,
		"start_date_time_utc_string": true,
		"end_date_time_utc_string":   true,
		"date_created":               true,
		"date_modified":              true,
		"date_created_string":        true,
		"date_modified_string":       true,
		"duration":                   true,
		"is_past":                    true,
		"is_future":                  true,
		"is_today":                   true,
		"is_this_week":               true,
		"is_this_month":              true,
		"weekday":                    true,
		"hour":                       true,
	}

	return timeFields[field]
}

// isTimeBasedFilterType checks if a filter involves time-related data
func (uc *GetEventListPageDataUseCase) isTimeBasedFilterType(filter *commonpb.TypedFilter) bool {
	// TODO: Implement based on filter type and field analysis
	// For now, assume all filters could potentially be time-based
	return true
}

// validateTimeBasedFilter performs specialized validation for time-based filter fields
func (uc *GetEventListPageDataUseCase) validateTimeBasedFilter(
	ctx context.Context,
	filter *commonpb.TypedFilter,
) error {
	// TODO: Implement specialized time-based filter validation
	// For now, we'll rely on the listdata processor for basic filtering
	return nil
}
