//go:build postgresql

package core

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// These limits are an adapter-level backstop. Typed use cases may choose lower
// entity-specific limits, but no caller may make the generic PostgreSQL list
// path allocate or execute unbounded request-controlled work.
const (
	maxQueryFilters          = 32
	maxQuerySortFields       = 4
	maxQuerySearchFields     = 8
	maxQuerySearchRunes      = 256
	maxQueryFilterValueRunes = 512
	maxQueryListValues       = 100
	maxQueryPageSize         = int32(100)
	maxQueryOffset           = int64(1_000_000)
	maxQueryCursorRunes      = 64
)

// BoundedQueryIDs validates and de-duplicates an adapter request's identifier
// set before it becomes an ANY/IN predicate. The raw input length is checked
// before allocation/de-duplication so a caller cannot bypass the finite-work
// budget with repeated values. Empty values and oversized identifiers fail
// closed; output order follows the first occurrence for deterministic tests and
// query plans.
func BoundedQueryIDs(values []string) ([]string, error) {
	if len(values) > maxQueryListValues {
		return nil, fmt.Errorf("too many query IDs: got %d, maximum is %d", len(values), maxQueryListValues)
	}
	bounded := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		if value == "" {
			return nil, fmt.Errorf("query ID %d is empty", i)
		}
		if runeLen(value) > maxQueryFilterValueRunes {
			return nil, fmt.Errorf("query ID %d exceeds %d characters", i, maxQueryFilterValueRunes)
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		bounded = append(bounded, value)
	}
	return bounded, nil
}

// BoundedContainsSearchPattern validates text for a hand-written PostgreSQL
// list query whose searched columns are fixed in source. It intentionally
// ignores request search-field identifiers because those columns are not
// request-selected. Callers bind the returned scalar to their LIKE/ILIKE
// placeholder.
func BoundedContainsSearchPattern(search *commonpb.SearchRequest) (string, error) {
	if search == nil || search.GetQuery() == "" {
		return "", nil
	}
	return BoundedContainsPattern(search.GetQuery())
}

// BoundedContainsPattern validates a literal LIKE/ILIKE contains term. The
// returned value is safe to bind to a query that declares ESCAPE '\\'.
func BoundedContainsPattern(query string) (string, error) {
	if query == "" {
		return "", nil
	}
	if runeLen(query) > maxQuerySearchRunes {
		return "", fmt.Errorf("search query exceeds %d characters", maxQuerySearchRunes)
	}
	return "%" + escapeLikeLiteral(query) + "%", nil
}

// BoundedPrefixPattern validates a literal LIKE/ILIKE prefix term. The
// returned value is safe to bind to a query that declares ESCAPE '\\'.
func BoundedPrefixPattern(query string) (string, error) {
	if query == "" {
		return "", nil
	}
	if runeLen(query) > maxQuerySearchRunes {
		return "", fmt.Errorf("search query exceeds %d characters", maxQuerySearchRunes)
	}
	return escapeLikeLiteral(query) + "%", nil
}

// BoundedQueryLimit normalizes a request limit to an adapter-owned finite
// budget. Non-positive requests intentionally select the adapter default.
func BoundedQueryLimit(requested, defaultLimit, maximum int32) (int32, error) {
	if defaultLimit <= 0 || maximum <= 0 || defaultLimit > maximum {
		return 0, fmt.Errorf("invalid query limit configuration: default %d, maximum %d", defaultLimit, maximum)
	}
	if requested <= 0 {
		return defaultLimit, nil
	}
	if requested > maximum {
		return 0, fmt.Errorf("query limit %d exceeds maximum %d", requested, maximum)
	}
	return requested, nil
}

// BoundedQueryOffset normalizes a raw offset to the global finite-work
// backstop. Negative offsets become zero for compatibility with legacy callers.
func BoundedQueryOffset(requested int32) (int32, error) {
	if requested < 0 {
		return 0, nil
	}
	if int64(requested) > maxQueryOffset {
		return 0, fmt.Errorf("query offset %d exceeds maximum %d", requested, maxQueryOffset)
	}
	return requested, nil
}

// validateListRequest validates finite work budgets and restricts every
// request-supplied identifier to the persisted columns of the selected entity.
// allowedColumns is descriptor- or catalog-derived by the caller; it is never
// request-derived.
func validateListRequest(params *interfaces.ListParams, tableName string, allowedColumns map[string]struct{}) error {
	if params == nil {
		return nil
	}
	if err := validateFilterBudget(params.Filters); err != nil {
		return err
	}
	if err := validateSearchBudget(params.Search); err != nil {
		return err
	}
	if err := validateSortBudget(params.Sort); err != nil {
		return err
	}
	if _, _, _, err := listPaginationBounds(params.Pagination); err != nil {
		return err
	}

	if params.Filters != nil {
		for _, filter := range params.Filters.Filters {
			if err := validateEntityColumn(filter.GetField(), tableName, allowedColumns); err != nil {
				return fmt.Errorf("filter field: %w", err)
			}
		}
	}
	if params.Search != nil && params.Search.GetQuery() != "" {
		for _, field := range params.Search.GetOptions().GetSearchFields() {
			if err := validateEntityColumn(field, tableName, allowedColumns); err != nil {
				return fmt.Errorf("search field: %w", err)
			}
		}
	}
	if params.Sort != nil {
		for _, field := range params.Sort.Fields {
			if err := validateEntityColumn(field.GetField(), tableName, allowedColumns); err != nil {
				return fmt.Errorf("sort field: %w", err)
			}
		}
	}
	return nil
}

// validateFilterAndSearchBudget is shared by hand-written CTE adapters through
// BuildFilterWhere. Those adapters still own their projected-column allowlists;
// this helper supplies the transport-independent finite-work backstop.
func validateFilterAndSearchBudget(filters *commonpb.FilterRequest, search *commonpb.SearchRequest, searchFieldCount int) error {
	if err := validateFilterBudget(filters); err != nil {
		return err
	}
	if err := validateSearchText(search, searchFieldCount); err != nil {
		return err
	}
	return nil
}

// BoundedOffsetPagination validates an offset-only list-page request and
// returns its finite SQL bounds. Hand-written CTE adapters use this instead of
// independently multiplying unchecked request values.
func BoundedOffsetPagination(p *commonpb.PaginationRequest, defaultLimit int32) (limit, offset, page int32, err error) {
	if defaultLimit <= 0 || defaultLimit > maxQueryPageSize {
		return 0, 0, 0, fmt.Errorf("invalid adapter default page size %d", defaultLimit)
	}
	if p == nil {
		return defaultLimit, 0, 1, nil
	}
	if p.GetLimit() == 0 {
		withDefault := *p
		withDefault.Limit = defaultLimit
		p = &withDefault
	}
	limit, offset, cursorMode, err := listPaginationBounds(p)
	if err != nil {
		return 0, 0, 0, err
	}
	if cursorMode {
		return 0, 0, 0, fmt.Errorf("cursor pagination is not supported by this offset query")
	}
	page = 1
	if offsetPage := p.GetOffset(); offsetPage != nil {
		page = offsetPage.GetPage()
	}
	return limit, offset, page, nil
}

func validateFilterBudget(filters *commonpb.FilterRequest) error {
	if filters == nil {
		return nil
	}
	if err := validateFilterLogic(filters.GetLogic()); err != nil {
		return err
	}
	if len(filters.Filters) > maxQueryFilters {
		return fmt.Errorf("too many filters: got %d, maximum is %d", len(filters.Filters), maxQueryFilters)
	}
	for i, filter := range filters.Filters {
		if filter == nil {
			return fmt.Errorf("filter %d is nil", i)
		}
		if err := ValidateSQLIdent(filter.GetField()); err != nil {
			return fmt.Errorf("filter %d field: %w", i, err)
		}
		switch typed := filter.FilterType.(type) {
		case *commonpb.TypedFilter_StringFilter:
			if typed.StringFilter == nil {
				return fmt.Errorf("filter %d has a nil string filter", i)
			}
			if runeLen(typed.StringFilter.GetValue()) > maxQueryFilterValueRunes {
				return fmt.Errorf("filter %d string value exceeds %d characters", i, maxQueryFilterValueRunes)
			}
			switch typed.StringFilter.GetOperator() {
			case commonpb.StringOperator_STRING_EQUALS,
				commonpb.StringOperator_STRING_NOT_EQUALS,
				commonpb.StringOperator_STRING_CONTAINS,
				commonpb.StringOperator_STRING_STARTS_WITH,
				commonpb.StringOperator_STRING_ENDS_WITH,
				commonpb.StringOperator_STRING_REGEX:
			default:
				return fmt.Errorf("filter %d has unsupported string operator %d", i, typed.StringFilter.GetOperator())
			}
		case *commonpb.TypedFilter_NumberFilter:
			if typed.NumberFilter == nil {
				return fmt.Errorf("filter %d has a nil number filter", i)
			}
			switch typed.NumberFilter.GetOperator() {
			case commonpb.NumberOperator_NUMBER_EQUALS,
				commonpb.NumberOperator_NUMBER_NOT_EQUALS,
				commonpb.NumberOperator_NUMBER_GREATER_THAN,
				commonpb.NumberOperator_NUMBER_GREATER_THAN_OR_EQUAL,
				commonpb.NumberOperator_NUMBER_LESS_THAN,
				commonpb.NumberOperator_NUMBER_LESS_THAN_OR_EQUAL:
			default:
				return fmt.Errorf("filter %d has unsupported number operator %d", i, typed.NumberFilter.GetOperator())
			}
		case *commonpb.TypedFilter_DateFilter:
			if typed.DateFilter == nil {
				return fmt.Errorf("filter %d has a nil date filter", i)
			}
			if runeLen(typed.DateFilter.GetValue()) > maxQueryFilterValueRunes || runeLen(typed.DateFilter.GetRangeEnd()) > maxQueryFilterValueRunes {
				return fmt.Errorf("filter %d date value exceeds %d characters", i, maxQueryFilterValueRunes)
			}
			switch typed.DateFilter.GetOperator() {
			case commonpb.DateOperator_DATE_EQUALS,
				commonpb.DateOperator_DATE_BEFORE,
				commonpb.DateOperator_DATE_AFTER:
			case commonpb.DateOperator_DATE_BETWEEN:
				if typed.DateFilter.RangeEnd == nil || typed.DateFilter.GetRangeEnd() == "" {
					return fmt.Errorf("filter %d DATE_BETWEEN requires range_end", i)
				}
			default:
				return fmt.Errorf("filter %d has unsupported date operator %d", i, typed.DateFilter.GetOperator())
			}
		case *commonpb.TypedFilter_ListFilter:
			if typed.ListFilter == nil {
				return fmt.Errorf("filter %d has a nil list filter", i)
			}
			if len(typed.ListFilter.GetValues()) == 0 {
				return fmt.Errorf("filter %d list requires at least one value", i)
			}
			if err := validateStringValues(i, "list", typed.ListFilter.GetValues()); err != nil {
				return err
			}
			switch typed.ListFilter.GetOperator() {
			case commonpb.ListOperator_LIST_IN, commonpb.ListOperator_LIST_NOT_IN:
			default:
				return fmt.Errorf("filter %d has unsupported list operator %d", i, typed.ListFilter.GetOperator())
			}
		case *commonpb.TypedFilter_RangeFilter:
			if typed.RangeFilter == nil {
				return fmt.Errorf("filter %d has a nil range filter", i)
			}
			if typed.RangeFilter.GetMin() > typed.RangeFilter.GetMax() {
				return fmt.Errorf("filter %d range min must not exceed max", i)
			}
		case *commonpb.TypedFilter_BooleanFilter:
			if typed.BooleanFilter == nil {
				return fmt.Errorf("filter %d has a nil boolean filter", i)
			}
		case *commonpb.TypedFilter_MoneyFilter:
			if typed.MoneyFilter == nil {
				return fmt.Errorf("filter %d has a nil money filter", i)
			}
			switch typed.MoneyFilter.GetOperator() {
			case commonpb.MoneyOperator_MONEY_EQUALS,
				commonpb.MoneyOperator_MONEY_LESS_THAN,
				commonpb.MoneyOperator_MONEY_GREATER_THAN,
				commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL,
				commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
			case commonpb.MoneyOperator_MONEY_BETWEEN:
				if typed.MoneyFilter.GetAmount() > typed.MoneyFilter.GetAmountTo() {
					return fmt.Errorf("filter %d money range start must not exceed end", i)
				}
			default:
				return fmt.Errorf("filter %d has unsupported money operator %d", i, typed.MoneyFilter.GetOperator())
			}
		case *commonpb.TypedFilter_StatusFilter:
			if typed.StatusFilter == nil {
				return fmt.Errorf("filter %d has a nil status filter", i)
			}
			if len(typed.StatusFilter.GetValues()) == 0 {
				return fmt.Errorf("filter %d status requires at least one value", i)
			}
			if err := validateStringValues(i, "status", typed.StatusFilter.GetValues()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("filter %d has no supported filter type", i)
		}
	}
	return nil
}

func validateFilterLogic(logic commonpb.FilterLogic) error {
	switch logic {
	case commonpb.FilterLogic_AND, commonpb.FilterLogic_OR:
		return nil
	default:
		return fmt.Errorf("unsupported filter logic %d", logic)
	}
}

func validateStringValues(filterIndex int, kind string, values []string) error {
	if len(values) > maxQueryListValues {
		return fmt.Errorf("filter %d %s contains %d values, maximum is %d", filterIndex, kind, len(values), maxQueryListValues)
	}
	for valueIndex, value := range values {
		if runeLen(value) > maxQueryFilterValueRunes {
			return fmt.Errorf("filter %d %s value %d exceeds %d characters", filterIndex, kind, valueIndex, maxQueryFilterValueRunes)
		}
	}
	return nil
}

func validateSearchBudget(search *commonpb.SearchRequest) error {
	fieldCount := 0
	if search != nil {
		fieldCount = len(search.GetOptions().GetSearchFields())
	}
	return validateSearchText(search, fieldCount)
}

func validateSearchText(search *commonpb.SearchRequest, fieldCount int) error {
	if search == nil || search.GetQuery() == "" {
		return nil
	}
	if runeLen(search.GetQuery()) > maxQuerySearchRunes {
		return fmt.Errorf("search query exceeds %d characters", maxQuerySearchRunes)
	}
	if fieldCount == 0 {
		return fmt.Errorf("search requires at least one search field")
	}
	if fieldCount > maxQuerySearchFields {
		return fmt.Errorf("too many search fields: got %d, maximum is %d", fieldCount, maxQuerySearchFields)
	}
	return nil
}

func validateSortBudget(sortReq *commonpb.SortRequest) error {
	if sortReq == nil {
		return nil
	}
	if len(sortReq.Fields) > maxQuerySortFields {
		return fmt.Errorf("too many sort fields: got %d, maximum is %d", len(sortReq.Fields), maxQuerySortFields)
	}
	for i, field := range sortReq.Fields {
		if field == nil {
			return fmt.Errorf("sort field %d is nil", i)
		}
		if err := ValidateSQLIdent(field.GetField()); err != nil {
			return fmt.Errorf("sort field %d: %w", i, err)
		}
		if err := validateSortDirection(field.GetDirection()); err != nil {
			return fmt.Errorf("sort field %d: %w", i, err)
		}
		if err := validateNullOrder(field.GetNullOrder()); err != nil {
			return fmt.Errorf("sort field %d: %w", i, err)
		}
	}
	return nil
}

func validateSortDirection(direction commonpb.SortDirection) error {
	switch direction {
	case commonpb.SortDirection_ASC, commonpb.SortDirection_DESC:
		return nil
	default:
		return fmt.Errorf("unsupported sort direction %d", direction)
	}
}

func validateNullOrder(nullOrder commonpb.NullOrder) error {
	switch nullOrder {
	case commonpb.NullOrder_NULLS_FIRST, commonpb.NullOrder_NULLS_LAST:
		return nil
	default:
		return fmt.Errorf("unsupported null order %d", nullOrder)
	}
}

// listPaginationBounds validates and normalizes offset/cursor pagination.
// cursor tokens intentionally support only the repository's existing
// "offset:<n>" contract; malformed/oversized cursors fail closed instead of
// silently restarting at page one.
func listPaginationBounds(p *commonpb.PaginationRequest) (limit int32, offset int32, cursorMode bool, err error) {
	limit = maxQueryPageSize
	if p == nil {
		return limit, 0, false, nil
	}
	if p.GetLimit() < 0 || p.GetLimit() > maxQueryPageSize {
		return 0, 0, false, fmt.Errorf("pagination limit must be between 1 and %d", maxQueryPageSize)
	}
	if p.GetLimit() > 0 {
		limit = p.GetLimit()
	}

	switch method := p.Method.(type) {
	case nil:
		return limit, 0, false, nil
	case *commonpb.PaginationRequest_Offset:
		if method.Offset == nil || method.Offset.GetPage() < 1 {
			return 0, 0, false, fmt.Errorf("pagination page must be greater than zero")
		}
		calculated := int64(method.Offset.GetPage()-1) * int64(limit)
		if calculated > maxQueryOffset {
			return 0, 0, false, fmt.Errorf("pagination offset exceeds maximum %d", maxQueryOffset)
		}
		return limit, int32(calculated), false, nil
	case *commonpb.PaginationRequest_Cursor:
		if method.Cursor == nil {
			return 0, 0, false, fmt.Errorf("pagination cursor is required")
		}
		token := method.Cursor.GetToken()
		if runeLen(token) > maxQueryCursorRunes {
			return 0, 0, false, fmt.Errorf("pagination cursor exceeds %d characters", maxQueryCursorRunes)
		}
		if token == "" {
			return limit, 0, true, nil
		}
		prefix, rawOffset, ok := strings.Cut(token, ":")
		if !ok || prefix != "offset" || rawOffset == "" {
			return 0, 0, false, fmt.Errorf("invalid pagination cursor")
		}
		parsed, parseErr := strconv.ParseInt(rawOffset, 10, 32)
		if parseErr != nil || parsed < 0 || parsed > maxQueryOffset {
			return 0, 0, false, fmt.Errorf("invalid pagination cursor offset")
		}
		return limit, int32(parsed), true, nil
	default:
		return 0, 0, false, fmt.Errorf("unsupported pagination method")
	}
}

func validateEntityColumn(field, tableName string, allowedColumns map[string]struct{}) error {
	if err := ValidateSQLIdent(field); err != nil {
		return err
	}
	parts := strings.Split(field, ".")
	column := parts[len(parts)-1]
	if len(parts) == 2 && parts[0] != tableName {
		return fmt.Errorf("column qualifier %q does not match entity %q", parts[0], tableName)
	}
	if _, ok := allowedColumns[column]; !ok {
		return fmt.Errorf("column %q is not allowed for entity %q", column, tableName)
	}
	return nil
}

// escapeLikeLiteral makes user text literal inside a LIKE/ILIKE pattern. The
// caller adds only the wildcards implied by the declared operator.
func escapeLikeLiteral(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func runeLen(value string) int {
	return utf8.RuneCountInString(value)
}
