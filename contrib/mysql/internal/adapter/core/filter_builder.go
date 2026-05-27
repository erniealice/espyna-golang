//go:build mysql

package core

import (
	"fmt"
	"strings"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// BuildFilterWhere constructs parameterized WHERE clauses from proto
// filter/search requests for MySQL. Mirrors
// contrib/postgres/internal/adapter/core/filter_builder.go with two
// mechanical differences:
//   - Placeholders are "?" (MySQL positional) instead of "$N" (postgres).
//   - ILIKE → LIKE  (MySQL's default utf8mb4_unicode_ci collation provides
//     case-insensitive matching without a distinct keyword).
//
// Returns (clauses, args, nextParamIndex). The returned nextIdx is a simple
// counter; since MySQL's "?" does not embed an index, it is used only to keep
// parity with the postgres API and to let callers reserve positions for
// preceding args (e.g., workspace_id occupies position 1 and passes startIdx=2).
//
// Caller joins clauses with " AND " and prepends them to an existing WHERE.
func BuildFilterWhere(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int) {
	nextIdx = startIdx

	// Search — LIKE OR block across declared search fields.
	// MySQL uses LIKE; the surrounding "%" makes it substring-match.
	if search != nil && search.Query != "" && len(searchFields) > 0 {
		query := "%" + search.Query + "%"
		var likeClauses []string
		for _, col := range searchFields {
			args = append(args, query)
			likeClauses = append(likeClauses, fmt.Sprintf("%s LIKE ?", col))
			nextIdx++
		}
		clauses = append(clauses, "("+strings.Join(likeClauses, " OR ")+")")
	}

	// Typed filters.
	if filters != nil {
		for _, filter := range filters.Filters {
			field := filter.Field

			switch ft := filter.FilterType.(type) {
			case *commonpb.TypedFilter_StringFilter:
				sf := ft.StringFilter
				switch sf.Operator {
				case commonpb.StringOperator_STRING_CONTAINS:
					args = append(args, "%"+sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s LIKE ?", field))
					nextIdx++
				case commonpb.StringOperator_STRING_EQUALS:
					args = append(args, sf.Value)
					clauses = append(clauses, fmt.Sprintf("%s = ?", field))
					nextIdx++
				case commonpb.StringOperator_STRING_STARTS_WITH:
					args = append(args, sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s LIKE ?", field))
					nextIdx++
				case commonpb.StringOperator_STRING_ENDS_WITH:
					args = append(args, "%"+sf.Value)
					clauses = append(clauses, fmt.Sprintf("%s LIKE ?", field))
					nextIdx++
				default:
					args = append(args, "%"+sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s LIKE ?", field))
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
				clauses = append(clauses, fmt.Sprintf("%s %s ?", field, op))
				nextIdx++

			case *commonpb.TypedFilter_BooleanFilter:
				args = append(args, ft.BooleanFilter.Value)
				clauses = append(clauses, fmt.Sprintf("%s = ?", field))
				nextIdx++

			case *commonpb.TypedFilter_DateFilter:
				df := ft.DateFilter
				switch df.Operator {
				case commonpb.DateOperator_DATE_EQUALS:
					// MySQL date cast: DATE(col)
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("DATE(%s) = DATE(?)", field))
					nextIdx++
				case commonpb.DateOperator_DATE_BEFORE:
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("%s < ?", field))
					nextIdx++
				case commonpb.DateOperator_DATE_AFTER:
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("%s >= ?", field))
					nextIdx++
				case commonpb.DateOperator_DATE_BETWEEN:
					if df.RangeEnd != nil && *df.RangeEnd != "" {
						args = append(args, df.Value, *df.RangeEnd)
						// Half-open range: [from, to)
						clauses = append(clauses, fmt.Sprintf("%s >= ? AND %s < ?", field, field))
						nextIdx += 2
					}
				}

			case *commonpb.TypedFilter_MoneyFilter:
				mf := ft.MoneyFilter
				switch mf.Operator {
				case commonpb.MoneyOperator_MONEY_EQUALS:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s = ?", field))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s < ?", field))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s > ?", field))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s <= ?", field))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s >= ?", field))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_BETWEEN:
					args = append(args, mf.Amount, mf.AmountTo)
					clauses = append(clauses, fmt.Sprintf("%s BETWEEN ? AND ?", field))
					nextIdx += 2
				}

			case *commonpb.TypedFilter_StatusFilter:
				sf := ft.StatusFilter
				if len(sf.Values) > 0 {
					placeholders := make([]string, len(sf.Values))
					for i, v := range sf.Values {
						placeholders[i] = "?"
						args = append(args, v)
						nextIdx++
					}
					clauses = append(clauses, fmt.Sprintf(
						"%s IN (%s)", field, strings.Join(placeholders, ", "),
					))
				}

			case *commonpb.TypedFilter_ListFilter:
				lf := ft.ListFilter
				if len(lf.Values) > 0 {
					placeholders := make([]string, len(lf.Values))
					for i, v := range lf.Values {
						placeholders[i] = "?"
						args = append(args, v)
						nextIdx++
					}
					op := "IN"
					if lf.Operator == commonpb.ListOperator_LIST_NOT_IN {
						op = "NOT IN"
					}
					clauses = append(clauses, fmt.Sprintf(
						"%s %s (%s)", field, op, strings.Join(placeholders, ", "),
					))
				}
			}
		}
	}

	return clauses, args, nextIdx
}
