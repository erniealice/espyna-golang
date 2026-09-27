//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

type keysetToken struct {
	page      int32 // destination page
	direction string
	id        string
}

func uuidShaped(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if id[i] != '-' {
				return false
			}
			continue
		}
		c := id[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func decodeKeysetToken(raw string, limit int32) (keysetToken, bool) {
	var token keysetToken
	if len(raw) > maxQueryCursorRunes {
		return token, false
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 4 || parts[0] != "k1" || !uuidShaped(parts[3]) {
		return token, false
	}
	page, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil || page < 1 || limit < 1 || (page-1)*int64(limit) > maxQueryOffset {
		return token, false
	}
	if parts[2] != "next" && parts[2] != "prev" {
		return token, false
	}
	token = keysetToken{int32(page), parts[2], parts[3]}
	return token, true
}

func encodeKeysetToken(page int32, direction, id string) string {
	return fmt.Sprintf("k1:%d:%s:%s", page, direction, id)
}

type listSortKey struct {
	column     string
	desc       bool
	nullsFirst bool
}

// The sort specification mirrors buildListOrderByClause. PostgreSQL's implicit
// order is NULLS FIRST for DESC and NULLS LAST for ASC; make both explicit here.
func listSortKeys(params *interfaces.ListParams) []listSortKey {
	if params == nil || params.Sort == nil || len(params.Sort.Fields) == 0 {
		return []listSortKey{{"date_created", true, true}, {"id", false, false}}
	}
	keys := make([]listSortKey, 0, len(params.Sort.Fields)+1)
	hasID := false
	for _, field := range params.Sort.Fields {
		name := strings.ToLower(field.GetField())
		if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
			name = name[dot+1:]
		}
		if name == "id" {
			hasID = true
		}
		keys = append(keys, listSortKey{name, field.GetDirection() == commonpb.SortDirection_DESC,
			field.GetNullOrder() == commonpb.NullOrder_NULLS_FIRST})
	}
	if !hasID {
		keys = append(keys, listSortKey{"id", false, false})
	}
	return keys
}

func listKeyColumns(keys []listSortKey) string {
	seen := map[string]bool{}
	columns := make([]string, 0, len(keys))
	for _, key := range keys {
		if !seen[key.column] {
			columns = append(columns, key.column)
			seen[key.column] = true
		}
	}
	return strings.Join(columns, ", ")
}

func listKeyProjection(keys []listSortKey) string {
	columns := strings.Split(listKeyColumns(keys), ", ")
	for i, column := range columns {
		columns[i] = `"` + column + `"`
	}
	return strings.Join(columns, ", ")
}

func listKeyOrder(keys []listSortKey, alias string, reverse bool) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		desc, nullsFirst := key.desc, key.nullsFirst
		if reverse {
			desc, nullsFirst = !desc, !nullsFirst
		}
		direction, nullOrder := "ASC", "NULLS LAST"
		if desc {
			direction = "DESC"
		}
		if nullsFirst {
			nullOrder = "NULLS FIRST"
		}
		parts = append(parts, fmt.Sprintf("\"%s\".\"%s\" %s %s", alias, key.column, direction, nullOrder))
	}
	return "ORDER BY " + strings.Join(parts, ", ")
}

// keysetWhere builds strict lexicographic comparison with NULL equality and
// ordering spelled out. Every boundary value is a bound parameter.
func keysetWhere(keys []listSortKey, boundary []any, firstParam int, reverse bool) (string, []any) {
	var arms []string
	var prefix []string
	args := make([]any, len(keys))
	copy(args, boundary)
	for i, key := range keys {
		col := `"` + key.column + `"`
		param := fmt.Sprintf("$%d", firstParam+i)
		desc, nullsFirst := key.desc, key.nullsFirst
		if reverse {
			desc, nullsFirst = !desc, !nullsFirst
		}
		comparison := " > "
		if desc {
			comparison = " < "
		}
		var after string
		if boundary[i] == nil {
			if nullsFirst {
				after = col + " IS NOT NULL"
			}
		} else {
			after = col + comparison + param
			if !nullsFirst {
				after = "(" + after + " OR " + col + " IS NULL)"
			}
		}
		if after != "" {
			arms = append(arms, "("+strings.Join(append(append([]string{}, prefix...), after), " AND ")+")")
		}
		prefix = append(prefix, col+" IS NOT DISTINCT FROM "+param)
	}
	if len(arms) == 0 {
		return "FALSE", args
	}
	return "(" + strings.Join(arms, " OR ") + ")", args
}

// scopedBoundary reads only the selected sort keys from a row that still
// satisfies the list's exact filter, workspace and principal predicates.
func (p *PostgresOperations) scopedBoundary(ctx context.Context, tableName, where string, values []any,
	keys []listSortKey, id string) ([]any, bool, error) {
	columns := strings.Split(listKeyColumns(keys), ", ")
	query := fmt.Sprintf("SELECT %s FROM \"%s\" WHERE %s AND id = $%d", listKeyProjection(keys), tableName, where, len(values)+1)
	bound := append(append([]any{}, values...), id)
	dest := make([]any, len(columns))
	scans := make([]any, len(columns))
	for i := range dest {
		scans[i] = &dest[i]
	}
	if err := p.getExecutor(ctx).QueryRowContext(ctx, query, bound...).Scan(scans...); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	byColumn := make(map[string]any, len(columns))
	for i, col := range columns {
		byColumn[col] = dest[i]
	}
	result := make([]any, len(keys))
	for i, key := range keys {
		result[i] = byColumn[key.column]
	}
	return result, true, nil
}
