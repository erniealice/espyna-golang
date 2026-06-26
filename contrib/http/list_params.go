package espynahttp

import (
	"time"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/tableparams"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ToListParams converts TableQueryParams into the espyna ListParams
// used by PostgresOperations.List(). searchFields drives ILIKE search;
// pass nil or empty slice when the caller does not support search.
// Timezone is applied to date filter string parsing.
func ToListParams(p tableparams.TableQueryParams, searchFields []string) interfaces.ListParams {
	// Sort with stable tie-breaker
	dir := commonpb.SortDirection_ASC
	if p.SortDir == "desc" {
		dir = commonpb.SortDirection_DESC
	}
	sort := &commonpb.SortRequest{
		Fields: []*commonpb.SortField{
			{Field: p.SortColumn, Direction: dir},
			{Field: "id", Direction: commonpb.SortDirection_ASC}, // stable tie-breaker
		},
	}

	// Pagination
	pagination := &commonpb.PaginationRequest{
		Limit: int32(p.PageSize),
		Method: &commonpb.PaginationRequest_Offset{
			Offset: &commonpb.OffsetPagination{Page: int32(p.Page)},
		},
	}

	// Search (only when a query is provided and search fields are declared)
	var search *commonpb.SearchRequest
	if p.Search != "" && len(searchFields) > 0 {
		search = &commonpb.SearchRequest{
			Query: p.Search,
			Options: &commonpb.SearchOptions{
				SearchFields: searchFields,
			},
		}
	}

	// Apply timezone to date filters in the parsed filter list
	if len(p.Filters) > 0 && p.Timezone != "" && p.Timezone != "UTC" {
		loc, err := time.LoadLocation(p.Timezone)
		if err == nil {
			applyTimezoneToFilters(p.Filters, loc)
		}
	}

	return interfaces.ListParams{
		Filters:    &commonpb.FilterRequest{Filters: p.Filters},
		Sort:       sort,
		Pagination: pagination,
		Search:     search,
	}
}

// applyTimezoneToFilters converts date filter string values from local time to UTC.
// Date strings arrive from the browser as "2006-01-02"; treated as midnight in the
// user's timezone and converted to UTC ISO strings. "to" date uses half-open range:
// advanced by one day (exclusive end-of-day) before UTC conversion.
func applyTimezoneToFilters(filters []*commonpb.TypedFilter, loc *time.Location) {
	for _, f := range filters {
		df, ok := f.FilterType.(*commonpb.TypedFilter_DateFilter)
		if !ok {
			continue
		}
		if df.DateFilter.Value != "" {
			t, err := time.ParseInLocation("2006-01-02", df.DateFilter.Value, loc)
			if err == nil {
				df.DateFilter.Value = t.UTC().Format(time.RFC3339)
			}
		}
		if df.DateFilter.RangeEnd != nil && *df.DateFilter.RangeEnd != "" {
			t, err := time.ParseInLocation("2006-01-02", *df.DateFilter.RangeEnd, loc)
			if err == nil {
				endStr := t.AddDate(0, 0, 1).UTC().Format(time.RFC3339)
				df.DateFilter.RangeEnd = &endStr
			}
		}
	}
}
