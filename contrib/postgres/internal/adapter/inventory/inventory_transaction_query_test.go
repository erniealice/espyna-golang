//go:build postgresql

package inventory

import (
	"context"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
)

func TestInventoryTransactionListPageDataSQL_RequiresJoinedItemWorkspace(t *testing.T) {
	query := inventoryTransactionListPageDataSQL()

	for _, want := range []string{
		"ii.workspace_id = $2",
		"SELECT * FROM enriched",
		"($1::text IS NULL",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("inventory transaction page query missing %q:\n%s", want, query)
		}
	}
	q, err := postgresCore.ResolveScopedPage(context.Background(), nil, postgresCore.ScopedPageSet{
		SQL: query, Args: []any{"%search%", "workspace"},
		Sort: []postgresCore.AdapterSortKey{{Column: "date_created", Desc: true, NullsFirst: true}},
	}, postgresCore.Page{Mode: postgresCore.PageModeOffset, Limit: 7, Number: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{q.PageSQL, q.CountSQL} {
		if !strings.Contains(sql, "ii.workspace_id = $2") || !strings.Contains(sql, "ii.name ILIKE $1") {
			t.Fatalf("page/count lost full scope: %s", sql)
		}
	}
	if !strings.Contains(q.PageSQL, "ORDER BY") || !strings.Contains(q.PageSQL, "LIMIT $3 OFFSET $4") {
		t.Fatalf("page SQL lost sort/limit: %s", q.PageSQL)
	}
}

func TestInventoryTransactionItemPageDataSQL_RequiresJoinedItemWorkspace(t *testing.T) {
	query := inventoryTransactionItemPageDataSQL()

	for _, want := range []string{
		"WHERE it.id = $1",
		"AND ii.workspace_id = $2",
		"SELECT * FROM enriched LIMIT 1",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("inventory transaction item query missing %q:\n%s", want, query)
		}
	}
}

func TestRequireInventoryWorkspace_RejectsMissingCapabilityBeforeQuery(t *testing.T) {
	if _, err := requireInventoryWorkspace(context.Background(), nil); err == nil {
		t.Fatal("requireInventoryWorkspace() accepted a repository without direct workspace capability")
	}
}

func TestInventoryMovementsListPageDataSQL_UsesEscapedBoundSearchAndCompatibilityCap(t *testing.T) {
	query := inventoryMovementsListPageDataSQL()

	for _, want := range []string{
		"AND ii.workspace_id = $1",
		"p.name ILIKE $6 ESCAPE '\\'",
		"pv.sku ILIKE $6 ESCAPE '\\'",
		"ii.sku ILIKE $6 ESCAPE '\\'",
		"ii.name ILIKE $6 ESCAPE '\\'",
		"LIMIT $7 OFFSET $8",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("inventory movements query missing %q:\n%s", want, query)
		}
	}
	if strings.Contains(query, "ILIKE '%' || $6 || '%'") {
		t.Fatalf("inventory movements query must not concatenate an unescaped search term:\n%s", query)
	}
	if strings.Contains(query, "($1 = '' OR ii.workspace_id = $1)") {
		t.Fatalf("inventory movements query must not permit an empty-workspace bypass:\n%s", query)
	}
}

func TestInventoryMovementsSearchPattern_EscapesMetacharactersAndRejectsOversize(t *testing.T) {
	pattern, err := inventoryMovementsSearchPattern("a%_\\b")
	if err != nil {
		t.Fatalf("inventoryMovementsSearchPattern() error = %v", err)
	}
	if want := "%a\\%\\_\\\\b%"; pattern != want {
		t.Fatalf("inventoryMovementsSearchPattern() = %q, want %q", pattern, want)
	}

	if _, err := inventoryMovementsSearchPattern(strings.Repeat("x", 257)); err == nil {
		t.Fatal("inventoryMovementsSearchPattern() accepted an overlong search")
	}
}
