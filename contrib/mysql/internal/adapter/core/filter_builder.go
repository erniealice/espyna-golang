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
// Returns (clauses, args, nextParamIndex, error). The returned nextIdx is a
// simple counter; since MySQL's "?" does not embed an index, it is used only to
// keep parity with the postgres API and to let callers reserve positions for
// preceding args (e.g., workspace_id occupies position 1 and passes startIdx=2).
//
// Caller joins clauses with " AND " and prepends them to an existing WHERE.
//
// Injection guard: filter VALUES are always bound via ? placeholders, but the
// filter FIELD names are interpolated into the query text. Every field is
// therefore validated through ValidateSQLIdent and the whole call fails closed
// on the first non-identifier field (callers must propagate the error — a
// silently dropped filter would widen the result set). Belt-and-braces, each
// validated identifier is additionally backtick-quoted per dot-component via
// quoteSortIdent (`alias.col` → `alias`.`col`) before interpolation.
//
// KNOWN GAP — the fail-closed guarantee above covers INVALID IDENTIFIERS ONLY, not
// every possible dropped filter. Four inputs still fall through and emit no clause,
// which widens the result set (fail-open):
//
//   - *commonpb.TypedFilter_RangeFilter — a real oneof variant (esqyma
//     pkg/schema/v1/domain/common/filter.pb.go:552) with no case in the type switch
//     below. It IS handled by internal/application/shared/listdata/filter.go:64 and by
//     this package's operations.go buildRangeFilter (:1026), so the drop is specific to
//     this list-page path.
//   - DATE_BETWEEN with a nil/empty RangeEnd.
//   - StatusFilter / ListFilter with zero Values.
//
// These are NOT a mysql regression: contrib/postgres and contrib/sqlserver
// filter_builder.go have the identical seven cases and the identical empty-value
// guards, so closing the gap must be done across all three dialects at once or it
// becomes the cross-dialect drift this port exists to remove. Latent as of 2026-07-29
// — repo-wide there are only consumers of TypedFilter_RangeFilter, no producer
// constructs one. Tracked in
// docs/plan/20260728-sql-injection-dialect-parity/progress.md § "Known parity gap —
// silent filter drops (W2 → W3/W4)". Do not restate this as "no silent drop anywhere".
func BuildFilterWhere(
	filters *commonpb.FilterRequest,
	search *commonpb.SearchRequest,
	searchFields []string,
	startIdx int,
) (clauses []string, args []any, nextIdx int, err error) {
	nextIdx = startIdx

	// Search — LIKE OR block across declared search fields. searchFields is an
	// author-controlled literal slice at every call site (never request-derived),
	// but validate each anyway so the function stays self-defending if a future
	// caller ever wires a dynamic column in. The search TEXT is bound as ?.
	// MySQL uses LIKE; the surrounding "%" makes it substring-match.
	if search != nil && search.Query != "" && len(searchFields) > 0 {
		query := "%" + search.Query + "%"
		var likeClauses []string
		for _, col := range searchFields {
			if err := ValidateSQLIdent(col); err != nil {
				return nil, nil, startIdx, fmt.Errorf("search field: %w", err)
			}
			args = append(args, query)
			likeClauses = append(likeClauses, fmt.Sprintf("%s LIKE ?", quoteSortIdent(col)))
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

	return clauses, args, nextIdx, nil
}
