//go:build postgresql

package ledger

import commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"

// andScopeFilter ANDs a request-scope equality (e.g. charge_policy_version_id) onto the caller's
// filters so a scope field is never silently ignored by the generic list. The result is built
// fresh (the request is not mutated); OR logic on the caller's filters is refused downstream by
// ScopedListParams.
func andScopeFilter(existing *commonpb.FilterRequest, field string, value *string) *commonpb.FilterRequest {
	if value == nil || *value == "" {
		return existing
	}
	scope := &commonpb.TypedFilter{Field: field, FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
		Value: *value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
	}}}
	out := &commonpb.FilterRequest{Logic: existing.GetLogic()}
	out.Filters = append(out.Filters, existing.GetFilters()...)
	out.Filters = append(out.Filters, scope)
	return out
}
