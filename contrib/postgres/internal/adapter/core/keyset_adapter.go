//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// AdapterSortKey is a projected column in the adapter's static scoped query.
// Column must be a bare identifier; the final id tie-breaker is added here.
type AdapterSortKey struct {
	Column           string
	Desc, NullsFirst bool
}

// ScopedPageSet is the adapter's complete, already-filtered row relation.
// SQL must be a static SELECT (possibly WITH ... SELECT), project unique id
// and all sort columns, and bind contiguous placeholders $1..$len(Args).
// Workspace, principal, search, and every other filter belong in this one
// relation; both boundary and page queries replay it verbatim.
type ScopedPageSet struct {
	SQL  string
	Args []any
	Sort []AdapterSortKey
}

// ScopedPageQueries contains the selected page and its exact scoped count.
// Page is changed to offset mode when a boundary no longer belongs to the set.
type ScopedPageQueries struct {
	PageSQL, CountSQL   string
	PageArgs, CountArgs []any
	Page                Page
}

type pageRowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func adapterSortKeys(sort []AdapterSortKey) ([]listSortKey, error) {
	if len(sort) == 0 || len(sort) > maxQuerySortFields {
		return nil, fmt.Errorf("adapter sort requires 1..%d keys", maxQuerySortFields)
	}
	keys := make([]listSortKey, 0, len(sort)+1)
	seen := map[string]bool{}
	for _, key := range sort {
		if err := ValidateSQLIdent(key.Column); err != nil || strings.Contains(key.Column, ".") {
			return nil, fmt.Errorf("adapter sort column %q must be a bare identifier", key.Column)
		}
		if seen[key.Column] {
			return nil, fmt.Errorf("duplicate adapter sort column %q", key.Column)
		}
		seen[key.Column] = true
		keys = append(keys, listSortKey{key.Column, key.Desc, key.NullsFirst})
	}
	if !seen["id"] {
		keys = append(keys, listSortKey{"id", false, false})
	}
	return keys, nil
}

func validateScopedPage(set ScopedPageSet, page Page) ([]listSortKey, error) {
	if strings.TrimSpace(set.SQL) == "" || strings.HasSuffix(strings.TrimSpace(set.SQL), ";") {
		return nil, fmt.Errorf("scoped page SQL must be one SELECT without a terminator")
	}
	keys, err := adapterSortKeys(set.Sort)
	if err != nil {
		return nil, err
	}
	if page.Limit < 1 || page.Limit > maxQueryPageSize || page.Offset < 0 || int64(page.Offset) > maxQueryOffset || page.Number < 1 {
		return nil, fmt.Errorf("invalid bounded adapter page")
	}
	if page.Mode != PageModeOffset && page.Mode != PageModeKeyset {
		return nil, fmt.Errorf("invalid adapter page mode %q", page.Mode)
	}
	if page.Mode == PageModeKeyset && (!uuidShaped(page.BoundaryID) || (page.Direction != "next" && page.Direction != "prev")) {
		return nil, fmt.Errorf("invalid adapter keyset boundary")
	}
	return keys, nil
}

func scopedPageBoundarySQL(set ScopedPageSet, keys []listSortKey, id string) (string, []any) {
	columns := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		if !seen[key.column] {
			columns = append(columns, `b."`+key.column+`"`)
			seen[key.column] = true
		}
	}
	query := fmt.Sprintf("SELECT %s FROM (%s) b WHERE b.id = $%d", strings.Join(columns, ", "), set.SQL, len(set.Args)+1)
	return query, append(append([]any{}, set.Args...), id)
}

// ResolveScopedPage performs a boundary lookup inside the very same filtered
// relation used by the seek and count. A missing boundary uses that token's
// requested page through the bounded offset route.
func ResolveScopedPage(ctx context.Context, db pageRowQuerier, set ScopedPageSet, page Page) (ScopedPageQueries, error) {
	keys, err := validateScopedPage(set, page)
	if err != nil {
		return ScopedPageQueries{}, err
	}
	var boundary []any
	if page.Mode == PageModeKeyset {
		if db == nil {
			return ScopedPageQueries{}, fmt.Errorf("keyset boundary executor is required")
		}
		lookup, args := scopedPageBoundarySQL(set, keys, page.BoundaryID)
		unique := strings.Split(listKeyColumns(keys), ", ")
		values := make([]any, len(unique))
		dest := make([]any, len(unique))
		for i := range values {
			dest[i] = &values[i]
		}
		err = db.QueryRowContext(ctx, lookup, args...).Scan(dest...)
		if err == sql.ErrNoRows {
			page.Mode = PageModeOffset
		} else if err != nil {
			return ScopedPageQueries{}, fmt.Errorf("resolve scoped page boundary: %w", err)
		} else {
			byColumn := make(map[string]any, len(unique))
			for i, name := range unique {
				byColumn[name] = values[i]
			}
			boundary = make([]any, len(keys))
			for i, key := range keys {
				boundary[i] = byColumn[key.column]
			}
		}
	}
	return buildScopedPageQueries(set, keys, page, boundary), nil
}

func buildScopedPageQueries(set ScopedPageSet, keys []listSortKey, page Page, boundary []any) ScopedPageQueries {
	args := append([]any{}, set.Args...)
	reverse := page.Mode == PageModeKeyset && page.Direction == "prev"
	seek := ""
	if page.Mode == PageModeKeyset {
		predicate, values := keysetWhere(keys, boundary, len(args)+1, reverse)
		seek = "WHERE " + predicate
		args = append(args, values...)
	}
	limit := fmt.Sprintf("LIMIT $%d", len(args)+1)
	args = append(args, page.Limit)
	if page.Mode == PageModeOffset {
		limit += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, page.Offset)
	}
	// The narrow page selection and full-row fetch both use scoped, never the
	// base table. Reverse selection is re-sorted into the normal display order.
	pageSQL := fmt.Sprintf(`WITH scoped AS NOT MATERIALIZED (%s),
page_keys AS MATERIALIZED (SELECT %s FROM scoped b %s %s %s)
SELECT b.* FROM scoped b JOIN page_keys k ON b.id = k.id %s`,
		set.SQL, listKeyProjection(keys), seek, listKeyOrder(keys, "b", reverse), limit, listKeyOrder(keys, "k", false))
	return ScopedPageQueries{
		PageSQL: pageSQL, PageArgs: args,
		CountSQL: "SELECT COUNT(*) FROM (" + set.SQL + ") b", CountArgs: append([]any{}, set.Args...),
		Page: page,
	}
}

// EncodePageCursor emits the shared k1 token for an adapter's first/last row.
// It returns empty for rows without a UUID-shaped id or invalid metadata.
func EncodePageCursor(page int32, direction, id string) string {
	if page < 1 || (direction != "next" && direction != "prev") || !uuidShaped(id) {
		return ""
	}
	return encodeKeysetToken(page, direction, id)
}
