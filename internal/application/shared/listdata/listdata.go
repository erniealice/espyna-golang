package listdata

import (
	"reflect"

	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// ListDataProcessor provides a unified interface for processing list data
// with pagination, filtering, sorting, and search capabilities
type ListDataProcessor struct {
	pagination *PaginationUtils
	filter     *FilterUtils
	sort       *SortUtils
	search     *SearchUtils
}

// NewListDataProcessor creates a new list data processor with all utilities
func NewListDataProcessor() *ListDataProcessor {
	return &ListDataProcessor{
		pagination: NewPaginationUtils(),
		filter:     NewFilterUtils(),
		sort:       NewSortUtils(),
		search:     NewSearchUtils(),
	}
}

// ProcessListRequest processes a slice of items with all requested operations
func (p *ListDataProcessor) ProcessListRequest(
	items interface{},
	pagination *commonpb.PaginationRequest,
	filters *commonpb.FilterRequest,
	sorting *commonpb.SortRequest,
	search *commonpb.SearchRequest,
) (*ListDataResult, error) {
	// Validate inputs
	sliceValue := reflect.ValueOf(items)
	if sliceValue.Kind() != reflect.Slice {
		return nil, NewListDataError("items must be a slice")
	}

	// Convert slice to []interface{} for processing
	processItems := make([]interface{}, sliceValue.Len())
	for i := 0; i < sliceValue.Len(); i++ {
		processItems[i] = sliceValue.Index(i).Interface()
	}

	// Step 1: Apply filters
	filteredItems := p.applyFilters(processItems, filters)

	// Step 2: Apply search
	searchResults, searchMetrics := p.applySearch(filteredItems, search)

	// Step 3: Apply sorting (on search results if search was performed)
	if search != nil && search.Query != "" {
		// Sort by search score first, then by requested sort fields
		p.applySortingToSearchResults(searchResults, sorting)
	} else {
		// Direct sorting on filtered items
		p.sort.SortItems(filteredItems, sorting)
		// Convert to search results format for consistency
		searchResults = p.search.convertToSearchResults(filteredItems)
	}

	// Step 4: Apply pagination
	paginatedResults, paginationResponse := p.applyPagination(searchResults, pagination)

	// Extract final items from search results
	finalItems := make([]interface{}, len(paginatedResults))
	searchResultsMetadata := make([]*commonpb.SearchResult, len(paginatedResults))

	for i, result := range paginatedResults {
		finalItems[i] = result.Item
		searchResultsMetadata[i] = &commonpb.SearchResult{
			Score:      result.Score,
			Highlights: result.Highlights,
		}
	}

	return &ListDataResult{
		Items:              finalItems,
		PaginationResponse: paginationResponse,
		SearchResults:      searchResultsMetadata,
		SearchMetrics:      searchMetrics,
	}, nil
}

// applyFilters filters items based on filter request
func (p *ListDataProcessor) applyFilters(items []interface{}, filters *commonpb.FilterRequest) []interface{} {
	if filters == nil {
		return items
	}

	var filteredItems []interface{}
	for _, item := range items {
		if p.filter.EvaluateFilters(item, filters) {
			filteredItems = append(filteredItems, item)
		}
	}

	return filteredItems
}

// applySearch performs search on items
func (p *ListDataProcessor) applySearch(items []interface{}, searchReq *commonpb.SearchRequest) ([]*SearchResult, *commonpb.SearchMetrics) {
	return p.search.SearchItems(items, searchReq)
}

// applySortingToSearchResults applies sorting to search results
func (p *ListDataProcessor) applySortingToSearchResults(results []*SearchResult, sorting *commonpb.SortRequest) {
	if sorting == nil || len(sorting.Fields) == 0 {
		return // Already sorted by search score
	}

	// Extract items for sorting
	items := make([]interface{}, len(results))
	for i, result := range results {
		items[i] = result.Item
	}

	// Sort the items
	p.sort.SortItems(items, sorting)

	// Update the search results order
	for i, item := range items {
		results[i].Item = item
	}
}

// applyPagination applies pagination to search results
func (p *ListDataProcessor) applyPagination(
	results []*SearchResult,
	pagination *commonpb.PaginationRequest,
) ([]*SearchResult, *commonpb.PaginationResponse) {
	// Validate and apply defaults to pagination request
	validatedPagination := p.pagination.ValidatePaginationRequest(pagination)

	// Calculate offset and limit
	offset, limit := p.pagination.CalculateOffsetAndLimit(validatedPagination)

	// Apply pagination
	totalItems := int32(len(results))
	start := int(offset)
	end := start + int(limit)

	if start >= len(results) {
		// No items for this page
		return []*SearchResult{}, p.pagination.CreatePaginationResponse(validatedPagination, totalItems, false)
	}

	if end > len(results) {
		end = len(results)
	}

	paginatedResults := results[start:end]
	hasNext := end < len(results)

	paginationResponse := p.pagination.CreatePaginationResponse(validatedPagination, totalItems, hasNext)

	return paginatedResults, paginationResponse
}

// ListDataResult contains the processed result
type ListDataResult struct {
	Items              []interface{}
	PaginationResponse *commonpb.PaginationResponse
	SearchResults      []*commonpb.SearchResult
	SearchMetrics      *commonpb.SearchMetrics
}

// ListDataError represents an error in list data processing
type ListDataError struct {
	Message string
}

func (e *ListDataError) Error() string {
	return e.Message
}

func NewListDataError(message string) *ListDataError {
	return &ListDataError{Message: message}
}

// Helper methods for accessing individual utilities

func (p *ListDataProcessor) GetPaginationUtils() *PaginationUtils {
	return p.pagination
}

func (p *ListDataProcessor) GetFilterUtils() *FilterUtils {
	return p.filter
}

func (p *ListDataProcessor) GetSortUtils() *SortUtils {
	return p.sort
}

func (p *ListDataProcessor) GetSearchUtils() *SearchUtils {
	return p.search
}
