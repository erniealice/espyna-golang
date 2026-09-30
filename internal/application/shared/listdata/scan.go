package listdata

import (
	"fmt"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// Complete scans over a paginated repository list (20260927-usage-and-pass-through-charges
// build-spec §7c C29). The generic PostgreSQL list silently caps an unpaginated request at 100
// rows, which would understate a balance or miss a row a guard must see, so ListAll walks the
// id-ordered pages and fails closed past ScanMaxPages instead of answering from a truncated read
// (precedent: outcome_criteria/code_validation.go listAllVersionsMatching).
const (
	ScanPageLimit = int32(100)
	ScanMaxPages  = int32(200)
)

// EqFilter is a single case-sensitive string-equality filter.
func EqFilter(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
		}},
	}}}
}

// IDSort orders a list by id so offset pages are stable.
func IDSort() *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}}
}

// ListAll exhausts a paginated list RPC. fetch performs one page request with the given page and
// the id sort; a page shorter than ScanPageLimit ends the scan.
func ListAll[T any](fetch func(page *commonpb.PaginationRequest, sort *commonpb.SortRequest) ([]T, error)) ([]T, error) {
	var out []T
	for page := int32(1); ; page++ {
		if page > ScanMaxPages {
			return nil, fmt.Errorf("listdata: read exceeded the %d-page bound; refusing to answer from a truncated read", ScanMaxPages)
		}
		rows, err := fetch(&commonpb.PaginationRequest{
			Limit:  ScanPageLimit,
			Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}},
		}, IDSort())
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
		if int32(len(rows)) < ScanPageLimit {
			return out, nil
		}
	}
}
