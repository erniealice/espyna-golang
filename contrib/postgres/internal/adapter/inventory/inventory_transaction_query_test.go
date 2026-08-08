//go:build postgresql

package inventory

import (
	"context"
	"strings"
	"testing"
)

func TestInventoryTransactionListPageDataSQL_RequiresJoinedItemWorkspace(t *testing.T) {
	query := inventoryTransactionListPageDataSQL("ORDER BY date_created DESC")

	for _, want := range []string{
		"ii.workspace_id = $4",
		"LIMIT $2 OFFSET $3",
		"($1::text IS NULL",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("inventory transaction page query missing %q:\n%s", want, query)
		}
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
