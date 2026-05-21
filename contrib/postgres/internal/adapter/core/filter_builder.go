//go:build postgresql

package core

import (
	"fmt"
	"strings"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// BuildFilterWhere constructs parameterized WHERE clauses from proto filter/search requests.
// Returns (clauses, args, nextParamIndex). Caller joins clauses with " AND ".
// searchFields specifies which columns to ILIKE search against.
// This function is used by entity CTE adapters to avoid duplicating filter logic.
func BuildFilterWhere(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int) {
	nextIdx = startIdx

	// Search — ILIKE OR block across declared search fields
	if search != nil && search.Query != "" && len(searchFields) > 0 {
		query := "%" + search.Query + "%"
		var likeClauses []string
		for _, col := range searchFields {
			args = append(args, query)
			likeClauses = append(likeClauses, fmt.Sprintf("%s ILIKE $%d", col, nextIdx))
			nextIdx++
		}
		clauses = append(clauses, "("+strings.Join(likeClauses, " OR ")+")")
	}

	// Typed filters
	if filters != nil {
		for _, filter := range filters.Filters {
			field := filter.Field

			switch ft := filter.FilterType.(type) {
			case *commonpb.TypedFilter_StringFilter:
				sf := ft.StringFilter
				switch sf.Operator {
				case commonpb.StringOperator_STRING_CONTAINS:
					args = append(args, "%"+sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s ILIKE $%d", field, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_EQUALS:
					args = append(args, sf.Value)
					clauses = append(clauses, fmt.Sprintf("%s = $%d", field, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_STARTS_WITH:
					args = append(args, sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s ILIKE $%d", field, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_ENDS_WITH:
					args = append(args, "%"+sf.Value)
					clauses = append(clauses, fmt.Sprintf("%s ILIKE $%d", field, nextIdx))
					nextIdx++
				default:
					args = append(args, "%"+sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s ILIKE $%d", field, nextIdx))
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
				clauses = append(clauses, fmt.Sprintf("%s %s $%d", field, op, nextIdx))
				nextIdx++

			case *commonpb.TypedFilter_BooleanFilter:
				args = append(args, ft.BooleanFilter.Value)
				clauses = append(clauses, fmt.Sprintf("%s = $%d", field, nextIdx))
				nextIdx++

			case *commonpb.TypedFilter_DateFilter:
				df := ft.DateFilter
				switch df.Operator {
				case commonpb.DateOperator_DATE_EQUALS:
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("%s::date = $%d::date", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_BEFORE:
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("%s < $%d::timestamp", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_AFTER:
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("%s >= $%d::timestamp", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_BETWEEN:
					if df.RangeEnd != nil && *df.RangeEnd != "" {
						args = append(args, df.Value, *df.RangeEnd)
						// Half-open range: [from, to)
						clauses = append(clauses, fmt.Sprintf("%s >= $%d::timestamp AND %s < $%d::timestamp", field, nextIdx, field, nextIdx+1))
						nextIdx += 2
					}
				}

			case *commonpb.TypedFilter_MoneyFilter:
				mf := ft.MoneyFilter
				switch mf.Operator {
				case commonpb.MoneyOperator_MONEY_EQUALS:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s = $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s < $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s > $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s <= $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s >= $%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_BETWEEN:
					args = append(args, mf.Amount, mf.AmountTo)
					clauses = append(clauses, fmt.Sprintf("%s BETWEEN $%d AND $%d", field, nextIdx, nextIdx+1))
					nextIdx += 2
				}

			case *commonpb.TypedFilter_StatusFilter:
				sf := ft.StatusFilter
				if len(sf.Values) > 0 {
					placeholders := make([]string, len(sf.Values))
					for i, v := range sf.Values {
						placeholders[i] = fmt.Sprintf("$%d", nextIdx)
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
						placeholders[i] = fmt.Sprintf("$%d", nextIdx)
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
