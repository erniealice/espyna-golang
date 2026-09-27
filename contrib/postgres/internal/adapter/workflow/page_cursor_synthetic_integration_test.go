//go:build postgresql

package workflow

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	_ "github.com/lib/pq"
)

// TestScopedPageSyntheticCursorParity proves the package's cursor metadata and
// sort adapter against a two-workspace SQL relation without mutating a table.
// It is synthetic evidence; it does not claim live adapter-row parity.
func TestScopedPageSyntheticCursorParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var readOnly string
	if err := db.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database session is not read-only")
	}
	keys, err := scopedPageSort("ORDER BY ordinal ASC, id ASC")
	if err != nil {
		t.Fatal(err)
	}
	set := postgresCore.ScopedPageSet{SQL: `SELECT id, workspace_id, ordinal FROM (VALUES
      ('00000000-0000-0000-0000-000000000001', 'caller', 1),
      ('00000000-0000-0000-0000-000000000002', 'caller', 2),
      ('00000000-0000-0000-0000-000000000003', 'caller', 3),
      ('00000000-0000-0000-0000-000000000004', 'foreign', 2)
    ) v(id, workspace_id, ordinal) WHERE workspace_id = $1`, Args: []any{"caller"}, Sort: keys}
	read := func(q postgresCore.ScopedPageQueries) []string {
		t.Helper()
		rows, err := db.QueryContext(ctx, q.PageSQL, q.PageArgs...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id, ws string
			var ordinal int
			if err := rows.Scan(&id, &ws, &ordinal); err != nil {
				t.Fatal(err)
			}
			if ws != "caller" {
				t.Fatalf("foreign row %q", id)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return ids
	}
	first, err := postgresCore.ResolveScopedPage(ctx, db, set, postgresCore.Page{Mode: postgresCore.PageModeOffset, Limit: 2, Number: 1})
	if err != nil {
		t.Fatal(err)
	}
	firstIDs := read(first)
	if !slices.Equal(firstIDs, []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"}) {
		t.Fatalf("first: %v", firstIDs)
	}
	next := scopedPageMetadata(first.Page, 3, firstIDs[0], firstIDs[len(firstIDs)-1]).GetNextCursor()
	p2, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: next}}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	second, err := postgresCore.ResolveScopedPage(ctx, db, set, p2)
	if err != nil {
		t.Fatal(err)
	}
	secondIDs := read(second)
	if !slices.Equal(secondIDs, []string{"00000000-0000-0000-0000-000000000003"}) {
		t.Fatalf("second: %v", secondIDs)
	}
	prev := scopedPageMetadata(second.Page, 3, secondIDs[0], secondIDs[0]).GetPrevCursor()
	p1, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: prev}}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	back, err := postgresCore.ResolveScopedPage(ctx, db, set, p1)
	if err != nil {
		t.Fatal(err)
	}
	if got := read(back); !slices.Equal(got, firstIDs) {
		t.Fatalf("prev: %v want %v", got, firstIDs)
	}
	forged, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: "k1:2:next:00000000-0000-0000-0000-000000000004"}}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	fallback, err := postgresCore.ResolveScopedPage(ctx, db, set, forged)
	if err != nil {
		t.Fatal(err)
	}
	if fallback.Page.Mode != postgresCore.PageModeOffset {
		t.Fatalf("foreign boundary mode=%s", fallback.Page.Mode)
	}
	if got := read(fallback); !slices.Equal(got, secondIDs) {
		t.Fatalf("foreign boundary page: %v want %v", got, secondIDs)
	}
	t.Log("3 caller rows, 2 pages: Next/Prev match offset; forged foreign token returned 0 foreign rows")
}
