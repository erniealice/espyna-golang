//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"testing"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"github.com/lib/pq"
)

// This is an adapter-shaped copy of the job page's enriched CTE. The test
// deliberately does not call the operation adapter: both its offset oracle
// and the public builder receive the same workspace/search/id/principal set.
const adapterJobSetSQL = `WITH enriched AS NOT MATERIALIZED (
 SELECT j.id, j.date_created, j.name
 FROM job j
 WHERE j.active = true
   AND ($1 = '' OR j.workspace_id = $1)
   AND ($2::text IS NULL OR $2::text = '' OR j.name ILIKE $2)
   AND j.id = ANY($3)
   AND ($4::text IS NULL OR j.created_by = $4)
) SELECT id, date_created, name FROM enriched`

func adapterJobPageIDs(t *testing.T, db *sql.DB, q ScopedPageQueries) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), q.PageSQL, q.PageArgs...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id, name string
		var created time.Time
		if err := rows.Scan(&id, &created, &name); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

func adapterJobOracle(t *testing.T, db *sql.DB, set ScopedPageSet, page int32) []string {
	t.Helper()
	args := append(append([]any{}, set.Args...), int32(7), (page-1)*7)
	query := fmt.Sprintf(`SELECT id FROM (%s) b ORDER BY b.date_created DESC NULLS LAST, b.id ASC NULLS LAST LIMIT $5 OFFSET $6`, set.SQL)
	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

func adapterPageRequest(t *testing.T, token string, number int32) Page {
	t.Helper()
	p := &commonpb.PaginationRequest{Limit: 7}
	if token == "" {
		p.Method = &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: number}}
	} else {
		p.Method = &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}
	}
	page, err := BoundedPageRequest(p, 50)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func TestAdapterScopedPageCTE_KeysetParity(t *testing.T) {
	db := openParityDB(t)
	defer db.Close()
	ctx := context.Background()
	var workspace string
	if err := db.QueryRowContext(ctx, `SELECT workspace_id FROM job WHERE active GROUP BY workspace_id ORDER BY count(*) DESC LIMIT 1`).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	ids := keysetParityIDs(t, db, "job", workspace, false)
	set := ScopedPageSet{
		SQL:  adapterJobSetSQL,
		Args: []any{workspace, nil, pq.Array(ids), nil},
		Sort: []AdapterSortKey{{Column: "date_created", Desc: true, NullsFirst: false}},
	}
	var expected [][]string
	for n := int32(1); ; n++ {
		page := adapterJobOracle(t, db, set, n)
		if len(page) == 0 {
			break
		}
		expected = append(expected, page)
	}
	if len(expected) < 3 {
		t.Fatalf("need at least 3 pages, got %d", len(expected))
	}
	page := adapterPageRequest(t, "", 1)
	for i, want := range expected {
		q, err := ResolveScopedPage(ctx, db, set, page)
		if err != nil {
			t.Fatal(err)
		}
		got := adapterJobPageIDs(t, db, q)
		if !slices.Equal(got, want) || q.Page.Number != int32(i+1) {
			t.Fatalf("next page %d: got %v want %v (mode %s)", i+1, got, want, q.Page.Mode)
		}
		var count int
		if err := db.QueryRowContext(ctx, q.CountSQL, q.CountArgs...).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != len(ids) {
			t.Fatalf("count=%d, want %d", count, len(ids))
		}
		if i+1 < len(expected) {
			page = adapterPageRequest(t, EncodePageCursor(int32(i+2), "next", got[len(got)-1]), 0)
		}
	}
	for i := len(expected) - 1; i > 0; i-- {
		q, err := ResolveScopedPage(ctx, db, set, page)
		if err != nil {
			t.Fatal(err)
		}
		got := adapterJobPageIDs(t, db, q)
		if !slices.Equal(got, expected[i]) {
			t.Fatalf("reverse start %d: %v", i, got)
		}
		page = adapterPageRequest(t, EncodePageCursor(int32(i), "prev", got[0]), 0)
		q, err = ResolveScopedPage(ctx, db, set, page)
		if err != nil {
			t.Fatal(err)
		}
		if got = adapterJobPageIDs(t, db, q); !slices.Equal(got, expected[i-1]) {
			t.Fatalf("prev page %d: got %v want %v", i, got, expected[i-1])
		}
	}
	t.Logf("job CTE: sampled_ids=%d pages=%d; Next/Prev identical to offset", len(ids), len(expected))
}

func TestAdapterScopedPageCTE_ForeignBoundary(t *testing.T) {
	db := openParityDB(t)
	defer db.Close()
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, `SELECT workspace_id FROM product WHERE active GROUP BY workspace_id ORDER BY count(*) DESC LIMIT 2`)
	if err != nil {
		t.Fatal(err)
	}
	var workspaces []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		workspaces = append(workspaces, id)
	}
	rows.Close()
	if len(workspaces) < 2 {
		t.Skip("fewer than two populated product workspaces")
	}
	var foreignID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM product WHERE active AND workspace_id = $1 ORDER BY id LIMIT 1`, workspaces[1]).Scan(&foreignID); err != nil {
		t.Fatal(err)
	}
	var sample []string
	sampleRows, err := db.QueryContext(ctx, `SELECT id FROM product WHERE active AND workspace_id = $1 ORDER BY id LIMIT 40`, workspaces[0])
	if err != nil {
		t.Fatal(err)
	}
	for sampleRows.Next() {
		var id string
		if err := sampleRows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		sample = append(sample, id)
	}
	sampleRows.Close()
	productSet := `WITH enriched AS NOT MATERIALIZED (
 SELECT p.id, p.date_created, p.id AS name FROM product p
 WHERE p.active = true AND p.workspace_id = $1
   AND ($2::text IS NULL OR p.id = $2)
   AND p.id = ANY($3)
   AND ($4::text IS NULL OR p.id = $4)
) SELECT id, date_created, name FROM enriched`
	set := ScopedPageSet{SQL: productSet, Args: []any{workspaces[0], nil, pq.Array(sample), nil}, Sort: []AdapterSortKey{{Column: "date_created", Desc: true, NullsFirst: false}}}
	want := adapterJobOracle(t, db, set, 2)
	if len(want) == 0 {
		t.Fatal("caller workspace has no second page")
	}
	page := adapterPageRequest(t, EncodePageCursor(2, "next", foreignID), 0)
	q, err := ResolveScopedPage(ctx, db, set, page)
	if err != nil {
		t.Fatal(err)
	}
	got := adapterJobPageIDs(t, db, q)
	if q.Page.Mode != PageModeOffset || q.Page.Number != 2 || !slices.Equal(got, want) {
		t.Fatalf("foreign boundary: mode=%s page=%d got=%v want=%v", q.Page.Mode, q.Page.Number, got, want)
	}
	var foreignRows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM product WHERE id = ANY($1) AND workspace_id <> $2`, pq.Array(got), workspaces[0]).Scan(&foreignRows); err != nil {
		t.Fatal(err)
	}
	if foreignRows != 0 {
		t.Fatalf("returned %d foreign rows", foreignRows)
	}
	t.Logf("foreign product boundary fell back to offset page 2; rows=%d foreign_rows=%d", len(got), foreignRows)
}
