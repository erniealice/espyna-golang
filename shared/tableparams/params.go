// Package tableparams holds transport-neutral data carriers for list-table
// queries (page, sort, filter, search).
//
// This package has zero dependency on net/http or any HTTP framework so the
// same struct can flow through HTTP form parsing, gRPC requests, CLI
// invocations, or batch jobs without binding callers to a transport choice.
//
// Transport-specific parsers (e.g. espynahttp.ParseTableParams) populate
// these structs from their respective inputs and pass them down through view
// helpers and use cases.
package tableparams

import (
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// TableQueryParams holds sanitized values describing one list-table request:
// pagination cursor, sort column + direction, free-text search, structured
// filters, and a timezone hint for date-filter normalization.
type TableQueryParams struct {
	Page       int
	PageSize   int
	Search     string
	SortColumn string
	SortDir    string // "asc" or "desc"
	Timezone   string
	Filters    []*commonpb.TypedFilter
	FiltersRaw string // raw JSON string for round-tripping back to template
}
