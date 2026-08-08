//go:build postgresql

package core

import (
	"fmt"
	"slices"
	"strings"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// BuildFilterWhereAllowed applies an adapter-owned filter-field allowlist before
// delegating to BuildFilterWhere. searchFields already form an author-owned
// search allowlist; this companion closes the equivalent filter identifier seam.
func BuildFilterWhereAllowed(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	allowedFilterFields []string,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int, err error) {
	for _, allowed := range allowedFilterFields {
		if err := ValidateSQLIdent(allowed); err != nil {
			return nil, nil, startIdx, fmt.Errorf("configured filter field: %w", err)
		}
	}
	if filters != nil {
		for _, filter := range filters.Filters {
			if filter == nil {
				return nil, nil, startIdx, fmt.Errorf("filter is nil")
			}
			if !slices.Contains(allowedFilterFields, filter.GetField()) {
				return nil, nil, startIdx, fmt.Errorf("filter field %q is not allowed (allowed: %v)", filter.GetField(), allowedFilterFields)
			}
		}
	}
	return BuildFilterWhere(filters, search, searchFields, startIdx)
}

// BuildFilterWhereMapped translates public/entity filter field names to the
// concrete SQL identifiers that are valid inside a hand-written query, and
// rejects every field absent from the adapter-owned map. The request is copied
// before translation so one adapter cannot mutate a protobuf request observed
// by another decorator/use case.
func BuildFilterWhereMapped(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	filterFieldMap map[string]string,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int, err error) {
	return buildFilterWhereMapped(filters, search, filterFieldMap, nil, searchFields, startIdx)
}

// BuildFilterWhereMappedISODateText is the legacy-schema companion for fields
// that persist strict ISO YYYY-MM-DD values in TEXT columns. Those declared
// public fields retain adapter-owned identifier mapping, but DateFilter
// comparisons use validated lexical ISO-date predicates instead of comparing a
// TEXT column to a PostgreSQL timestamp (which has no matching operator).
//
// Only fields named in isoDateTextFields receive this behavior. Other date
// fields retain the ordinary timestamp/date predicates. Caller values are
// parsed with time.Parse before any SQL runs, so malformed dates fail closed.
func BuildFilterWhereMappedISODateText(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	filterFieldMap map[string]string,
	isoDateTextFields []string,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int, err error) {
	isoPublicFields := make(map[string]struct{}, len(isoDateTextFields))
	for _, field := range isoDateTextFields {
		if _, ok := filterFieldMap[field]; !ok {
			return nil, nil, startIdx, fmt.Errorf("ISO date text field %q has no configured SQL mapping", field)
		}
		isoPublicFields[field] = struct{}{}
	}
	return buildFilterWhereMapped(filters, search, filterFieldMap, isoPublicFields, searchFields, startIdx)
}

func buildFilterWhereMapped(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	filterFieldMap map[string]string,
	isoDateTextPublicFields map[string]struct{},
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int, err error) {
	for publicField, sqlField := range filterFieldMap {
		if publicField == "" {
			return nil, nil, startIdx, fmt.Errorf("configured public filter field is empty")
		}
		if err := ValidateSQLIdent(sqlField); err != nil {
			return nil, nil, startIdx, fmt.Errorf("configured SQL filter field for %q: %w", publicField, err)
		}
	}

	isoDateTextSQLFields := make(map[string]struct{}, len(isoDateTextPublicFields))
	for publicField := range isoDateTextPublicFields {
		sqlField, ok := filterFieldMap[publicField]
		if !ok {
			return nil, nil, startIdx, fmt.Errorf("ISO date text field %q has no configured SQL mapping", publicField)
		}
		isoDateTextSQLFields[sqlField] = struct{}{}
	}

	if filters == nil {
		return buildFilterWhere(nil, search, searchFields, startIdx, isoDateTextSQLFields)
	}

	mapped := &commonpb.FilterRequest{
		Logic:   filters.GetLogic(),
		Filters: make([]*commonpb.TypedFilter, 0, len(filters.Filters)),
	}
	for _, filter := range filters.Filters {
		if filter == nil {
			return nil, nil, startIdx, fmt.Errorf("filter is nil")
		}
		sqlField, ok := filterFieldMap[filter.GetField()]
		if !ok {
			return nil, nil, startIdx, fmt.Errorf("filter field %q is not allowed", filter.GetField())
		}
		mappedFilter := *filter
		mappedFilter.Field = sqlField
		mapped.Filters = append(mapped.Filters, &mappedFilter)
	}

	return buildFilterWhere(mapped, search, searchFields, startIdx, isoDateTextSQLFields)
}

// BuildFilterWhere constructs parameterized WHERE clauses from proto filter/search requests.
// Returns (clauses, args, nextParamIndex, error). Caller joins clauses with " AND ".
// searchFields specifies which columns to ILIKE search against.
// This function is used by entity CTE adapters to avoid duplicating filter logic.
//
// Injection guard: filter VALUES are always bound via $N placeholders, but the
// filter FIELD names are interpolated into the query text. Every field is
// therefore validated through ValidateSQLIdent and the whole call fails closed
// on the first non-identifier field (callers must propagate the error — a
// silently dropped filter would widen the result set).
func BuildFilterWhere(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int, err error) {
	return buildFilterWhere(filters, search, searchFields, startIdx, nil)
}

func buildFilterWhere(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	searchFields []string,
	startIdx int,
	isoDateTextFields map[string]struct{},
) (clauses []string, args []any, nextIdx int, err error) {
	nextIdx = startIdx
	if err := validateFilterAndSearchBudget(filters, search, len(searchFields)); err != nil {
		return nil, nil, startIdx, err
	}

	// Search — ILIKE OR block across declared search fields. searchFields is an
	// author-controlled literal slice at every call site (never request-derived),
	// but validate each anyway so the function stays self-defending if a future
	// caller ever wires a dynamic column in. The search TEXT is bound as $N.
	if search != nil && search.Query != "" && len(searchFields) > 0 {
		query := "%" + escapeLikeLiteral(search.Query) + "%"
		var likeClauses []string
		for _, col := range searchFields {
			if err := ValidateSQLIdent(col); err != nil {
				return nil, nil, startIdx, fmt.Errorf("search field: %w", err)
			}
			args = append(args, query)
			likeClauses = append(likeClauses, fmt.Sprintf("%s ILIKE $%d ESCAPE '\\'", col, nextIdx))
			nextIdx++
		}
		clauses = append(clauses, "("+strings.Join(likeClauses, " OR ")+")")
	}

	// Typed filters. Keep each TypedFilter as one logical clause so OR requests
	// cannot split a range/date-between filter into two independently true arms.
	var filterClauses []string
	if filters != nil {
		for _, filter := range filters.Filters {
			field := filter.Field
			if err := ValidateSQLIdent(field); err != nil {
				return nil, nil, startIdx, fmt.Errorf("filter field: %w", err)
			}

			switch ft := filter.FilterType.(type) {
			case *commonpb.TypedFilter_StringFilter:
				sf := ft.StringFilter
				matchOp := "ILIKE"
				regexOp := "~*"
				value := sf.Value
				if sf.CaseSensitive {
					matchOp = "LIKE"
					regexOp = "~"
				}
				switch sf.Operator {
				case commonpb.StringOperator_STRING_CONTAINS:
					args = append(args, "%"+escapeLikeLiteral(value)+"%")
					filterClauses = append(filterClauses, fmt.Sprintf("%s %s $%d ESCAPE '\\'", field, matchOp, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_EQUALS:
					if sf.CaseSensitive {
						args = append(args, value)
						filterClauses = append(filterClauses, fmt.Sprintf("%s = $%d", field, nextIdx))
					} else {
						args = append(args, strings.ToLower(value))
						filterClauses = append(filterClauses, fmt.Sprintf("LOWER(%s) = $%d", field, nextIdx))
					}
					nextIdx++
				case commonpb.StringOperator_STRING_NOT_EQUALS:
					if sf.CaseSensitive {
						args = append(args, value)
						filterClauses = append(filterClauses, fmt.Sprintf("%s != $%d", field, nextIdx))
					} else {
						args = append(args, strings.ToLower(value))
						filterClauses = append(filterClauses, fmt.Sprintf("LOWER(%s) != $%d", field, nextIdx))
					}
					nextIdx++
				case commonpb.StringOperator_STRING_STARTS_WITH:
					args = append(args, escapeLikeLiteral(value)+"%")
					filterClauses = append(filterClauses, fmt.Sprintf("%s %s $%d ESCAPE '\\'", field, matchOp, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_ENDS_WITH:
					args = append(args, "%"+escapeLikeLiteral(value))
					filterClauses = append(filterClauses, fmt.Sprintf("%s %s $%d ESCAPE '\\'", field, matchOp, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_REGEX:
					args = append(args, value)
					filterClauses = append(filterClauses, fmt.Sprintf("%s %s $%d", field, regexOp, nextIdx))
					nextIdx++
				}

			case *commonpb.TypedFilter_NumberFilter:
				nf := ft.NumberFilter
				op := "="
				switch nf.Operator {
				case commonpb.NumberOperator_NUMBER_GREATER_THAN:
					op = ">"
				case commonpb.NumberOperator_NUMBER_GREATER_THAN_OR_EQUAL:
					op = ">="
				case commonpb.NumberOperator_NUMBER_LESS_THAN:
					op = "<"
				case commonpb.NumberOperator_NUMBER_LESS_THAN_OR_EQUAL:
					op = "<="
				case commonpb.NumberOperator_NUMBER_NOT_EQUALS:
					op = "!="
				}
				args = append(args, nf.Value)
				filterClauses = append(filterClauses, fmt.Sprintf("%s %s $%d", field, op, nextIdx))
				nextIdx++

			case *commonpb.TypedFilter_BooleanFilter:
				args = append(args, ft.BooleanFilter.Value)
				filterClauses = append(filterClauses, fmt.Sprintf("%s = $%d", field, nextIdx))
				nextIdx++

			case *commonpb.TypedFilter_DateFilter:
				df := ft.DateFilter
				if _, isoDateText := isoDateTextFields[field]; isoDateText {
					if err := validateISODate(df.Value); err != nil {
						return nil, nil, startIdx, fmt.Errorf("date filter field %q: %w", field, err)
					}
					switch df.Operator {
					case commonpb.DateOperator_DATE_EQUALS:
						args = append(args, df.Value)
						filterClauses = append(filterClauses, fmt.Sprintf("%s = $%d", field, nextIdx))
						nextIdx++
					case commonpb.DateOperator_DATE_BEFORE:
						args = append(args, df.Value)
						filterClauses = append(filterClauses, fmt.Sprintf("%s < $%d", field, nextIdx))
						nextIdx++
					case commonpb.DateOperator_DATE_AFTER:
						args = append(args, df.Value)
						filterClauses = append(filterClauses, fmt.Sprintf("%s >= $%d", field, nextIdx))
						nextIdx++
					case commonpb.DateOperator_DATE_BETWEEN:
						if err := validateISODate(df.GetRangeEnd()); err != nil {
							return nil, nil, startIdx, fmt.Errorf("date filter field %q range_end: %w", field, err)
						}
						args = append(args, df.Value, df.GetRangeEnd())
						filterClauses = append(filterClauses, fmt.Sprintf("(%s >= $%d AND %s < $%d)", field, nextIdx, field, nextIdx+1))
						nextIdx += 2
					}
					continue
				}
				switch df.Operator {
				case commonpb.DateOperator_DATE_EQUALS:
					args = append(args, df.Value)
					filterClauses = append(filterClauses, fmt.Sprintf("%s::date = $%d::date", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_BEFORE:
					args = append(args, df.Value)
					filterClauses = append(filterClauses, fmt.Sprintf("%s < $%d::timestamp", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_AFTER:
					args = append(args, df.Value)
					filterClauses = append(filterClauses, fmt.Sprintf("%s >= $%d::timestamp", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_BETWEEN:
					args = append(args, df.Value, df.GetRangeEnd())
					filterClauses = append(filterClauses, fmt.Sprintf("(%s >= $%d::timestamp AND %s < $%d::timestamp)", field, nextIdx, field, nextIdx+1))
					nextIdx += 2
				}

			case *commonpb.TypedFilter_RangeFilter:
				rf := ft.RangeFilter
				minOp := ">"
				if rf.IncludeMin {
					minOp = ">="
				}
				maxOp := "<"
				if rf.IncludeMax {
					maxOp = "<="
				}
				args = append(args, rf.Min, rf.Max)
				filterClauses = append(filterClauses, fmt.Sprintf("(%s %s $%d AND %s %s $%d)", field, minOp, nextIdx, field, maxOp, nextIdx+1))
				nextIdx += 2

			case *commonpb.TypedFilter_MoneyFilter:
				mf := ft.MoneyFilter
				switch mf.Operator {
				case commonpb.MoneyOperator_MONEY_EQUALS:
					args = append(args, mf.Amount)
					filterClauses = append(filterClauses, fmt.Sprintf("%s = $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN:
					args = append(args, mf.Amount)
					filterClauses = append(filterClauses, fmt.Sprintf("%s < $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN:
					args = append(args, mf.Amount)
					filterClauses = append(filterClauses, fmt.Sprintf("%s > $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					filterClauses = append(filterClauses, fmt.Sprintf("%s <= $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					filterClauses = append(filterClauses, fmt.Sprintf("%s >= $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_BETWEEN:
					args = append(args, mf.Amount, mf.AmountTo)
					filterClauses = append(filterClauses, fmt.Sprintf("%s BETWEEN $%d AND $%d", field, nextIdx, nextIdx+1))
					nextIdx += 2
				}

			case *commonpb.TypedFilter_StatusFilter:
				sf := ft.StatusFilter
				placeholders := make([]string, len(sf.Values))
				for i, v := range sf.Values {
					placeholders[i] = fmt.Sprintf("$%d", nextIdx)
					args = append(args, v)
					nextIdx++
				}
				filterClauses = append(filterClauses, fmt.Sprintf(
					"%s IN (%s)", field, strings.Join(placeholders, ", "),
				))

			case *commonpb.TypedFilter_ListFilter:
				lf := ft.ListFilter
				placeholders := make([]string, len(lf.Values))
				for i, v := range lf.Values {
					placeholders[i] = fmt.Sprintf("$%d", nextIdx)
					args = append(args, v)
					nextIdx++
				}
				op := "IN"
				if lf.Operator == commonpb.ListOperator_LIST_NOT_IN {
					op = "NOT IN"
				}
				filterClauses = append(filterClauses, fmt.Sprintf(
					"%s %s (%s)", field, op, strings.Join(placeholders, ", "),
				))
			}
		}
	}
	clauses = append(clauses, groupFilterClauses(filters.GetLogic(), filterClauses)...)

	return clauses, args, nextIdx, nil
}

func validateISODate(value string) error {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return fmt.Errorf("value %q must be a valid YYYY-MM-DD date", value)
	}
	return nil
}

// groupFilterClauses preserves the FilterRequest contract while allowing callers
// to keep joining the returned top-level clauses with AND. Search remains a
// separate AND condition; only the typed filter set is grouped by its declared
// logic.
func groupFilterClauses(logic commonpb.FilterLogic, filterClauses []string) []string {
	if len(filterClauses) == 0 {
		return nil
	}
	if logic == commonpb.FilterLogic_OR {
		return []string{"(" + strings.Join(filterClauses, " OR ") + ")"}
	}
	return filterClauses
}
