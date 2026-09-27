//go:build postgresql

package revenue

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	"github.com/lib/pq"
)

// This read-only probe calls the actual page-data methods. It walks offset
// pages as an oracle, then follows Next to the end and Prev to the start.
func TestRevenuePageDataKeysetLiveParity(t *testing.T) {
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
		t.Fatalf("read-only local database required: setting=%q err=%v", readOnly, err)
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "default-workspace"})
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	cases := []struct {
		name, table string
		fetch       func(*commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error)
	}{
		{"Revenue", "revenue", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresRevenueRepository(ops, entityid.Revenue)
			resp, err := repo.GetRevenueListPageData(ctx, &revenuepb.GetRevenueListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.RevenueList))
			for _, row := range resp.RevenueList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"RevenueAttribute", "revenue_attribute", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresRevenueAttributeRepository(ops, entityid.RevenueAttribute)
			resp, err := repo.GetRevenueAttributeListPageData(ctx, &revenueattributepb.GetRevenueAttributeListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.RevenueAttributeList))
			for _, row := range resp.RevenueAttributeList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
		{"RevenueLineItem", "revenue_line_item", func(p *commonpb.PaginationRequest) ([]string, *commonpb.PaginationResponse, error) {
			repo := NewPostgresRevenueLineItemRepository(ops, entityid.RevenueLineItem)
			resp, err := repo.GetRevenueLineItemListPageData(ctx, &revenuelineitempb.GetRevenueLineItemListPageDataRequest{Pagination: p})
			if err != nil {
				return nil, nil, err
			}
			ids := make([]string, 0, len(resp.RevenueLineItemList))
			for _, row := range resp.RevenueLineItemList {
				ids = append(ids, row.GetId())
			}
			return ids, resp.Pagination, nil
		}},
	}
	offset := func(page int32) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}}}
	}
	cursor := func(token string) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var present sql.NullString
			if err := db.QueryRow("SELECT to_regclass($1)", tc.table).Scan(&present); err != nil {
				t.Fatal(err)
			}
			if !present.Valid {
				t.Skipf("%s absent on lane DB", tc.table)
			}
			first, meta, err := tc.fetch(offset(1))
			if err != nil {
				t.Fatal(err)
			}
			if meta == nil {
				t.Fatal("missing pagination metadata")
			}
			if meta.GetTotalPages() < 2 {
				t.Logf("LIVE_UNPROVEN: %s has %d rows / %d pages", tc.table, meta.GetTotalItems(), meta.GetTotalPages())
				return
			}
			if meta.GetTotalPages() > 50 {
				t.Fatalf("unexpected walk size %d", meta.GetTotalPages())
			}
			oracle := [][]string{first}
			for n := int32(2); n <= meta.GetTotalPages(); n++ {
				ids, _, err := tc.fetch(offset(n))
				if err != nil {
					t.Fatal(err)
				}
				oracle = append(oracle, ids)
			}
			got := [][]string{first}
			k1Hops, offsetHops := 0, 0
			for n := int32(2); n <= meta.GetTotalPages(); n++ {
				if meta.GetNextCursor() == "" {
					t.Fatalf("page %d missing next cursor", n-1)
				}
				if strings.HasPrefix(meta.GetNextCursor(), "k1:") {
					k1Hops++
				} else if strings.HasPrefix(meta.GetNextCursor(), "offset:") {
					offsetHops++
				}
				ids, next, err := tc.fetch(cursor(meta.GetNextCursor()))
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, ids)
				meta = next
			}
			if !slices.EqualFunc(got, oracle, func(a, b []string) bool { return slices.Equal(a, b) }) {
				t.Fatalf("next walk mismatch: got=%v offset=%v", got, oracle)
			}
			for n := len(oracle) - 2; n >= 0; n-- {
				if meta.GetPrevCursor() == "" {
					t.Fatalf("page %d missing prev cursor", n+2)
				}
				if strings.HasPrefix(meta.GetPrevCursor(), "k1:") {
					k1Hops++
				} else if strings.HasPrefix(meta.GetPrevCursor(), "offset:") {
					offsetHops++
				}
				ids, prev, err := tc.fetch(cursor(meta.GetPrevCursor()))
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(ids, oracle[n]) {
					t.Fatalf("prev page %d mismatch: got=%v offset=%v", n+1, ids, oracle[n])
				}
				meta = prev
			}
			if tc.name == "Revenue" {
				var foreignID string
				if err := db.QueryRow("SELECT id FROM revenue WHERE workspace_id <> $1 LIMIT 1", "default-workspace").Scan(&foreignID); err != nil {
					t.Fatal(err)
				}
				token := postgresCore.EncodePageCursor(2, "next", foreignID)
				if token == "" {
					t.Fatalf("foreign revenue id %q is not k1 encodable", foreignID)
				}
				forged, _, err := tc.fetch(cursor(token))
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(forged, oracle[1]) || slices.Contains(forged, foreignID) {
					t.Fatalf("foreign boundary exposed another workspace: forged=%v offset=%v", forged, oracle[1])
				}
				t.Logf("forged foreign revenue boundary: 0 foreign rows; page-2 fallback exact")
			}
			t.Logf("LIVE parity: %s %d rows %d pages next/prev exact; k1 hops=%d offset hops=%d", tc.table, meta.GetTotalItems(), len(oracle), k1Hops, offsetHops)
		})
	}
}
