//go:build sqlserver

package core

import (
	"fmt"
	"strings"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// BuildFilterWhere constructs parameterized WHERE clauses from proto filter/search requests.
// Returns (clauses, args, nextParamIndex, error). Caller joins clauses with " AND ".
// searchFields specifies which columns to LIKE-search against.
//
// SQL Server differences from the postgres gold standard (filter_builder.go):
//   - Placeholders: @p1, @p2, … (not $1, $2, …). Use Placeholder(n) to format.
//   - Case-insensitive search: LIKE instead of ILIKE. SQL Server's default
//     collation (Latin1_General_CI_AS) is case-insensitive, so plain LIKE
//     behaves identically to postgres ILIKE on most instances. Add a COLLATE
//     clause only when targeting a case-sensitive collation.
//   - Date casts: CAST(... AS date) / CAST(... AS datetime2) instead of
//     postgres ::date / ::timestamp casts.
//
// This function is used by entity CTE adapters to avoid duplicating filter logic.
//
// Injection guard: filter VALUES are always bound via @pN placeholders, but the
// filter FIELD names are interpolated into the query text. Every field is
// therefore validated through ValidateSQLIdent and the whole call fails closed
// on the first non-identifier field (callers must propagate the error — a
// silently dropped filter would widen the result set). Belt-and-braces, each
// validated identifier is additionally bracket-quoted per dot-component via
// quoteSortIdent (`alias.col` → [alias].[col]) before interpolation.
//
// KNOWN GAP — the fail-closed guarantee above covers INVALID IDENTIFIERS ONLY, not
// every possible dropped filter. Four inputs still fall through and emit no clause,
// which widens the result set (fail-open):
//
//   - *commonpb.TypedFilter_RangeFilter — a real oneof variant (esqyma
//     pkg/schema/v1/domain/common/filter.pb.go) with no case in the type switch
//     below.
//   - DATE_BETWEEN with a nil/empty RangeEnd.
//   - StatusFilter / ListFilter with zero Values.
//
// These are NOT a sqlserver regression: contrib/postgres and contrib/mysql
// filter_builder.go have the identical seven cases and the identical empty-value
// guards, so closing the gap must be done across all three dialects at once or it
// becomes the cross-dialect drift this port exists to remove. Tracked in
// docs/plan/20260728-sql-injection-dialect-parity/progress.md § "Known parity gap —
// silent filter drops (W2 → W3/W4)". Do not restate this as "no silent drop anywhere".
func BuildFilterWhere(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int, err error) {
	nextIdx = startIdx

	// Search — LIKE OR block across declared search fields.
	// SQL Server's default CI collation makes plain LIKE case-insensitive, matching
	// postgres ILIKE behaviour without an explicit COLLATE clause.
	//
	// searchFields is an author-controlled literal slice at every call site (never
	// request-derived), but validate each anyway so the function stays
	// self-defending if a future caller ever wires a dynamic column in. The search
	// TEXT is bound as @pN.
	if search != nil && search.Query != "" && len(searchFields) > 0 {
		query := "%" + search.Query + "%"
		var likeClauses []string
		for _, col := range searchFields {
			if err := ValidateSQLIdent(col); err != nil {
				return nil, nil, startIdx, fmt.Errorf("search field: %w", err)
			}
			args = append(args, query)
			likeClauses = append(likeClauses, fmt.Sprintf("%s LIKE @p%d", quoteSortIdent(col), nextIdx))
			nextIdx++
		}
		clauses = append(clauses, "("+strings.Join(likeClauses, " OR ")+")")
	}

	// Typed filters.
	if filters != nil {
		for _, filter := range filters.Filters {
			field := filter.Field
			if err := ValidateSQLIdent(field); err != nil {
				return nil, nil, startIdx, fmt.Errorf("filter field: %w", err)
			}
			// Belt-and-braces: regex-validated above AND quoted per component here.
			field = quoteSortIdent(field)

			switch ft := filter.FilterType.(type) {
			case *commonpb.TypedFilter_StringFilter:
				sf := ft.StringFilter
				switch sf.Operator {
				case commonpb.StringOperator_STRING_CONTAINS:
					args = append(args, "%"+sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s LIKE @p%d", field, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_EQUALS:
					args = append(args, sf.Value)
					clauses = append(clauses, fmt.Sprintf("%s = @p%d", field, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_STARTS_WITH:
					args = append(args, sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s LIKE @p%d", field, nextIdx))
					nextIdx++
				case commonpb.StringOperator_STRING_ENDS_WITH:
					args = append(args, "%"+sf.Value)
					clauses = append(clauses, fmt.Sprintf("%s LIKE @p%d", field, nextIdx))
					nextIdx++
				default:
					args = append(args, "%"+sf.Value+"%")
					clauses = append(clauses, fmt.Sprintf("%s LIKE @p%d", field, nextIdx))
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
				clauses = append(clauses, fmt.Sprintf("%s %s @p%d", field, op, nextIdx))
				nextIdx++

			case *commonpb.TypedFilter_BooleanFilter:
				args = append(args, ft.BooleanFilter.Value)
				clauses = append(clauses, fmt.Sprintf("%s = @p%d", field, nextIdx))
				nextIdx++

			case *commonpb.TypedFilter_DateFilter:
				df := ft.DateFilter
				switch df.Operator {
				case commonpb.DateOperator_DATE_EQUALS:
					args = append(args, df.Value)
					// SQL Server: CAST(col AS date) = CAST(@pN AS date)
					clauses = append(clauses, fmt.Sprintf("CAST(%s AS date) = CAST(@p%d AS date)", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_BEFORE:
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("%s < CAST(@p%d AS datetime2)", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_AFTER:
					args = append(args, df.Value)
					clauses = append(clauses, fmt.Sprintf("%s >= CAST(@p%d AS datetime2)", field, nextIdx))
					nextIdx++
				case commonpb.DateOperator_DATE_BETWEEN:
					if df.RangeEnd != nil && *df.RangeEnd != "" {
						args = append(args, df.Value, *df.RangeEnd)
						clauses = append(clauses, fmt.Sprintf(
							"%s >= CAST(@p%d AS datetime2) AND %s < CAST(@p%d AS datetime2)",
							field, nextIdx, field, nextIdx+1,
						))
						nextIdx += 2
					}
				}

			case *commonpb.TypedFilter_MoneyFilter:
				mf := ft.MoneyFilter
				switch mf.Operator {
				case commonpb.MoneyOperator_MONEY_EQUALS:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s = @p%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s < @p%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s > @p%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_LESS_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s <= @p%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_GREATER_THAN_OR_EQUAL:
					args = append(args, mf.Amount)
					clauses = append(clauses, fmt.Sprintf("%s >= @p%d", field, nextIdx))
					nextIdx++
				case commonpb.MoneyOperator_MONEY_BETWEEN:
					args = append(args, mf.Amount, mf.AmountTo)
					clauses = append(clauses, fmt.Sprintf("%s BETWEEN @p%d AND @p%d", field, nextIdx, nextIdx+1))
					nextIdx += 2
				}

			case *commonpb.TypedFilter_StatusFilter:
				sf := ft.StatusFilter
				if len(sf.Values) > 0 {
					placeholders := make([]string, len(sf.Values))
					for i, v := range sf.Values {
						placeholders[i] = fmt.Sprintf("@p%d", nextIdx)
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
						placeholders[i] = fmt.Sprintf("@p%d", nextIdx)
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

	return clauses, args, nextIdx, nil
}
