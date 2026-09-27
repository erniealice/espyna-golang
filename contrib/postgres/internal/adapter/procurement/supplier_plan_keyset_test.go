//go:build postgresql

package procurement

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"github.com/lib/pq"
)

const supplierPlanBoundaryID = "11111111-1111-4111-8111-111111111111"

type supplierPlanBoundaryConnector struct {
	query string
	args  []driver.NamedValue
}

func (c *supplierPlanBoundaryConnector) Connect(context.Context) (driver.Conn, error) {
	return &supplierPlanBoundaryConn{c}, nil
}
func (c *supplierPlanBoundaryConnector) Driver() driver.Driver { return supplierPlanBoundaryDriver{c} }

type supplierPlanBoundaryDriver struct {
	connector *supplierPlanBoundaryConnector
}

func (d supplierPlanBoundaryDriver) Open(string) (driver.Conn, error) {
	return &supplierPlanBoundaryConn{d.connector}, nil
}

type supplierPlanBoundaryConn struct {
	connector *supplierPlanBoundaryConnector
}

func (*supplierPlanBoundaryConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (*supplierPlanBoundaryConn) Close() error { return nil }
func (*supplierPlanBoundaryConn) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected begin")
}
func (c *supplierPlanBoundaryConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.connector.query = query
	c.connector.args = append([]driver.NamedValue(nil), args...)
	return &supplierPlanBoundaryRows{}, nil
}

type supplierPlanBoundaryRows struct{ done bool }

func (*supplierPlanBoundaryRows) Columns() []string { return []string{"date_created", "id"} }
func (*supplierPlanBoundaryRows) Close() error      { return nil }
func (r *supplierPlanBoundaryRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0], dest[1] = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), supplierPlanBoundaryID
	return nil
}

func TestSupplierPlanKeysetQueryKeepsWorkspaceAndSearch(t *testing.T) {
	ctx := context.Background()
	connector := &supplierPlanBoundaryConnector{}
	db := sql.OpenDB(connector)
	defer db.Close()
	set := supplierPlanScopedPageSet("%needle%", "workspace-1", nil)
	for _, direction := range []string{"next", "prev"} {
		t.Run(direction, func(t *testing.T) {
			token := postgresCore.EncodePageCursor(2, direction, supplierPlanBoundaryID)
			page, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{
				Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}},
			}, 50)
			if err != nil || page.Mode != postgresCore.PageModeKeyset {
				t.Fatalf("decode keyset: page=%+v err=%v", page, err)
			}
			queries, err := postgresCore.ResolveScopedPage(ctx, db, set, page)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(connector.query, "workspace_id = $2") || !strings.Contains(connector.query, "name ILIKE $1") || len(connector.args) != 3 {
				t.Fatalf("boundary lost workspace/search: SQL=%s args=%v", connector.query, connector.args)
			}
			if !strings.Contains(queries.PageSQL, "workspace_id = $2") || !strings.Contains(queries.PageSQL, "name ILIKE $1") ||
				!strings.Contains(queries.PageSQL, `"date_created"`) || strings.Contains(queries.PageSQL, " OFFSET ") {
				t.Fatalf("seek lost scope/order or retained offset: %s", queries.PageSQL)
			}
			if queries.Page.Number != 2 || queries.Page.Mode != postgresCore.PageModeKeyset {
				t.Fatalf("resolved page = %+v", queries.Page)
			}
		})
	}
	malformed, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{
		Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: "k1:bad"}},
	}, 50)
	if err != nil || malformed.Mode != postgresCore.PageModeOffset || malformed.Number != 1 {
		t.Fatalf("malformed cursor did not restart page one: %+v, %v", malformed, err)
	}
	legacy, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{
		Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: "offset:2"}},
	}, 50)
	if err != nil || legacy.Mode != postgresCore.PageModeOffset || legacy.Offset != 2 || legacy.Number != 2 {
		t.Fatalf("legacy offset token changed route: %+v, %v", legacy, err)
	}
	numbered, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{
		Limit: 2, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 3}},
	}, 50)
	if err != nil || numbered.Mode != postgresCore.PageModeOffset || numbered.Offset != 4 || numbered.Number != 3 {
		t.Fatalf("numbered page changed route: %+v, %v", numbered, err)
	}
}

// Inline rows exercise the adapter SQL builder when no local supplier_plan
// table has two pages. They are SELECT-only and leave the database unchanged.
func TestSupplierPlanKeysetSyntheticParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is unset")
	}
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	ctx := context.Background()
	var readOnly string
	if err := db.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("read-only database required: setting=%q err=%v", readOnly, err)
	}
	const foreignID = "99999999-9999-4999-8999-999999999999"
	set := postgresCore.ScopedPageSet{
		SQL: `WITH seed(id, workspace_id, name, date_created) AS (
			VALUES
			('11111111-1111-4111-8111-111111111111', 'ws-a', 'needle', '2026-09-03'::timestamptz),
			('22222222-2222-4222-8222-222222222222', 'ws-a', 'needle', '2026-09-02'::timestamptz),
			('33333333-3333-4333-8333-333333333333', 'ws-a', 'needle', '2026-09-02'::timestamptz),
			('44444444-4444-4444-8444-444444444444', 'ws-a', 'needle', '2026-09-01'::timestamptz),
			('99999999-9999-4999-8999-999999999999', 'ws-b', 'needle', '2026-09-04'::timestamptz)
		) SELECT id, name, date_created FROM seed WHERE workspace_id = $1 AND name ILIKE $2`,
		Args: []any{"ws-a", "needle"}, Sort: []postgresCore.AdapterSortKey{{Column: "date_created", Desc: true, NullsFirst: true}},
	}
	readPage := func(page postgresCore.Page) []string {
		t.Helper()
		queries, err := postgresCore.ResolveScopedPage(ctx, db, set, page)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := db.QueryContext(ctx, queries.PageSQL, queries.PageArgs...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var ids []string
		for rows.Next() {
			var id, name string
			var created time.Time
			if err := rows.Scan(&id, &name, &created); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return ids
	}
	first := readPage(postgresCore.Page{Mode: postgresCore.PageModeOffset, Limit: 2, Number: 1})
	second := readPage(postgresCore.Page{Mode: postgresCore.PageModeOffset, Limit: 2, Offset: 2, Number: 2})
	next := readPage(postgresCore.Page{Mode: postgresCore.PageModeKeyset, Limit: 2, Offset: 2, Number: 2, Direction: "next", BoundaryID: first[len(first)-1]})
	prev := readPage(postgresCore.Page{Mode: postgresCore.PageModeKeyset, Limit: 2, Number: 1, Direction: "prev", BoundaryID: second[0]})
	if len(first) != 2 || len(second) != 2 || !slices.Equal(next, second) || !slices.Equal(prev, first) {
		t.Fatalf("offset/keyset mismatch first=%v second=%v next=%v prev=%v", first, second, next, prev)
	}
	forged := readPage(postgresCore.Page{Mode: postgresCore.PageModeKeyset, Limit: 2, Offset: 2, Number: 2, Direction: "next", BoundaryID: foreignID})
	if !slices.Equal(forged, second) || slices.Contains(forged, foreignID) {
		t.Fatalf("foreign boundary did not fall back within scope: %v", forged)
	}
	t.Logf("synthetic parity: 4 caller ids, 2 pages, Next/Prev exact; foreign boundary returns 0 foreign ids")
}
