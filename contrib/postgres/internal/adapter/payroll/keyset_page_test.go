//go:build postgresql

package payroll

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"github.com/lib/pq"
)

const syntheticForeignID = "99999999-9999-4999-8999-999999999999"

func TestPayrollPageMetadataTokenModes(t *testing.T) {
	page := postgresCore.Page{Mode: postgresCore.PageModeOffset, Limit: 2, Number: 1}
	uuid := scopedPageMetadata(4, page, "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222")
	if !strings.HasPrefix(uuid.GetNextCursor(), "k1:2:next:") {
		t.Fatalf("UUID boundary token = %q", uuid.GetNextCursor())
	}
	legacy := scopedPageMetadata(4, page, "acct-1", "acct-2")
	if legacy.GetNextCursor() != "offset:2" {
		t.Fatalf("legacy ID cursor = %q", legacy.GetNextCursor())
	}
	for _, token := range []string{uuid.GetNextCursor(), legacy.GetNextCursor(), "malformed"} {
		decoded, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}, 50)
		if err != nil {
			t.Fatal(err)
		}
		if token == "malformed" && (decoded.Number != 1 || decoded.Mode != postgresCore.PageModeOffset) {
			t.Fatalf("malformed token = %+v", decoded)
		}
		if token == legacy.GetNextCursor() && (decoded.Number != 2 || decoded.Mode != postgresCore.PageModeOffset) {
			t.Fatalf("legacy token = %+v", decoded)
		}
		if token == uuid.GetNextCursor() && (decoded.Number != 2 || decoded.Mode != postgresCore.PageModeKeyset) {
			t.Fatalf("k1 token = %+v", decoded)
		}
	}
}

// The source tables have fewer than two pages for some methods. Each case
// exercises that method's projected default sort with the shared SQL builder
// against a SELECT-only relation with duplicate sort values and a foreign row.
func TestPayrollPageSyntheticKeysetParity(t *testing.T) {
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
	var readOnly string
	if err := db.QueryRow("SHOW default_transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("read-only database required: %q %v", readOnly, err)
	}
	const seed = `WITH seed(id, workspace_id, name, date_created, code, entry_date, ordinal, effective_from) AS (
 VALUES
 ('11111111-1111-4111-8111-111111111111','ws-a','needle','2026-09-03'::timestamptz,'B','2026-09-03'::date,2,'2026-09-03'),
 ('22222222-2222-4222-8222-222222222222','ws-a','needle','2026-09-02'::timestamptz,'A','2026-09-02'::date,1,'2026-09-02'),
 ('33333333-3333-4333-8333-333333333333','ws-a','needle','2026-09-02'::timestamptz,'A','2026-09-02'::date,1,'2026-09-02'),
 ('44444444-4444-4444-8444-444444444444','ws-a','needle','2026-09-01'::timestamptz,'C','2026-09-01'::date,3,'2026-09-01'),
 ('99999999-9999-4999-8999-999999999999','ws-b','needle','2026-09-04'::timestamptz,'Z','2026-09-04'::date,4,'2026-09-04')
 ) SELECT id, %s FROM seed WHERE workspace_id = $1 AND name ILIKE $2`
	cases := []struct {
		name, column string
		desc         bool
	}{
		{"LeaveBalance", "date_created", true},
		{"LeaveRequest", "date_created", true},
		{"PayCycle", "date_created", true},
		{"PayrollRemittance", "date_created", true},
		{"PayrollRun", "date_created", true},
		{"RateBand", "ordinal", false},
		{"RateTable", "effective_from", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			direction := "ASC"
			if tc.desc {
				direction = "DESC"
			}
			sort, err := pageSortFromOrderBy(fmt.Sprintf("ORDER BY %s %s, id ASC", tc.column, direction))
			if err != nil || len(sort) != 1 || sort[0].Column != tc.column || sort[0].Desc != tc.desc {
				t.Fatalf("sort=%+v err=%v", sort, err)
			}
			set := postgresCore.ScopedPageSet{SQL: fmt.Sprintf(seed, tc.column), Args: []any{"ws-a", "needle"}, Sort: sort}
			read := func(page postgresCore.Page) ([]string, postgresCore.Page) {
				t.Helper()
				q, err := postgresCore.ResolveScopedPage(context.Background(), db, set, page)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(q.PageSQL, "workspace_id = $1") || !strings.Contains(q.CountSQL, "workspace_id = $1") {
					t.Fatal("scope missing from page/count")
				}
				if page.Mode == postgresCore.PageModeKeyset && q.Page.Mode == postgresCore.PageModeKeyset && strings.Contains(q.PageSQL, " OFFSET ") {
					t.Fatal("keyset page retained offset")
				}
				rows, err := db.Query(q.PageSQL, q.PageArgs...)
				if err != nil {
					t.Fatal(err)
				}
				defer rows.Close()
				ids := []string{}
				for rows.Next() {
					var id string
					var value any
					if err := rows.Scan(&id, &value); err != nil {
						t.Fatal(err)
					}
					ids = append(ids, id)
				}
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				return ids, q.Page
			}
			first, _ := read(postgresCore.Page{Mode: postgresCore.PageModeOffset, Limit: 2, Number: 1})
			second, _ := read(postgresCore.Page{Mode: postgresCore.PageModeOffset, Limit: 2, Offset: 2, Number: 2})
			decode := func(number int32, direction, boundaryID string) postgresCore.Page {
				t.Helper()
				token := postgresCore.EncodePageCursor(number, direction, boundaryID)
				page, err := postgresCore.BoundedPageRequest(&commonpb.PaginationRequest{
					Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}},
				}, 50)
				if err != nil || page.Mode != postgresCore.PageModeKeyset {
					t.Fatalf("decode %s token: page=%+v err=%v", direction, page, err)
				}
				return page
			}
			next, mode := read(decode(2, "next", first[len(first)-1]))
			prev, _ := read(decode(1, "prev", second[0]))
			forged, forgedMode := read(decode(2, "next", syntheticForeignID))
			if !slices.Equal(next, second) || !slices.Equal(prev, first) || !slices.Equal(forged, second) || slices.Contains(forged, syntheticForeignID) || mode.Mode != postgresCore.PageModeKeyset || forgedMode.Mode != postgresCore.PageModeOffset {
				t.Fatalf("parity first=%v second=%v next=%v prev=%v forged=%v modes=%s/%s", first, second, next, prev, forged, mode.Mode, forgedMode.Mode)
			}
			t.Logf("synthetic parity: 4 caller ids, 2 pages, k1 next/prev exact; forged foreign boundary returned 0 foreign rows")
		})
	}
}
