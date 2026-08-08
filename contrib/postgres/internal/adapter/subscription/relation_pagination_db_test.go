//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	planSchwpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule_workspace_user"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	sgpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	memberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
	"github.com/lib/pq"
)

func openRelationPaginationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("DATABASE_POSTGRES_HOST")
		user := os.Getenv("DATABASE_POSTGRES_USER")
		database := os.Getenv("DATABASE_POSTGRES_DBNAME")
		if host == "" || user == "" || database == "" {
			t.Skip("TEST_DATABASE_URL or DATABASE_POSTGRES_* is required for the read-only relation pagination test")
		}
		port := os.Getenv("DATABASE_POSTGRES_PORT")
		if port == "" {
			port = "5432"
		}
		u := &url.URL{
			Scheme: "postgres",
			Host:   net.JoinHostPort(host, port),
			Path:   "/" + database,
			User:   url.UserPassword(user, os.Getenv("DATABASE_POSTGRES_PASSWORD")),
		}
		q := u.Query()
		q.Set("sslmode", os.Getenv("DATABASE_POSTGRES_SSLMODE"))
		u.RawQuery = q.Encode()
		dsn = u.String()
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open relation pagination database: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("relation pagination database is unreachable: %v", err)
	}
	for _, setting := range []string{
		"SET default_transaction_read_only = on",
		"SET statement_timeout = '10s'",
		"SET lock_timeout = '1s'",
	} {
		if _, err := db.Exec(setting); err != nil {
			db.Close()
			t.Fatalf("establish read-only relation test session: %v", err)
		}
	}
	return db
}

// TestCountActiveByClientIdsRestrictsToRequestedPage is the read-only parity
// oracle for PERF-02's Subscription-owned display count. It proves the
// repository returns no client outside the supplied page IDs and matches an
// independent scoped aggregate for every supplied ID.
func TestCountActiveByClientIdsRestrictsToRequestedPage(t *testing.T) {
	db := openRelationPaginationDB(t)
	defer db.Close()

	ctx := context.Background()
	var workspaceID string
	if err := db.QueryRowContext(ctx, `SELECT workspace_id FROM `+entityid.Client+` WHERE active = true AND workspace_id IS NOT NULL GROUP BY workspace_id ORDER BY count(*) DESC LIMIT 1`).Scan(&workspaceID); err != nil {
		t.Fatalf("resolve client-count workspace: %v", err)
	}
	rows, err := db.QueryContext(ctx, `SELECT id FROM `+entityid.Client+` WHERE active = true AND workspace_id = $1 ORDER BY id LIMIT 5`, workspaceID)
	if err != nil {
		t.Fatalf("select page client IDs: %v", err)
	}
	var clientIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("scan page client ID: %v", err)
		}
		clientIDs = append(clientIDs, id)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close page client rows: %v", err)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate page client IDs: %v", err)
	}
	if len(clientIDs) == 0 {
		t.Fatal("client-count fixture returned no active clients")
	}

	ctx = identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspaceID})
	repo := NewPostgresSubscriptionRepository(postgresCore.NewWorkspaceAwareOperations(db), entityid.Subscription)
	resp, err := repo.CountActiveByClientIds(ctx, &subscriptionpb.CountActiveByClientIdsRequest{ClientIds: clientIDs})
	if err != nil {
		t.Fatalf("CountActiveByClientIds: %v", err)
	}

	expected := make(map[string]int32, len(clientIDs))
	countRows, err := db.QueryContext(ctx, `
		SELECT requested.id, count(s.id)::int
		  FROM unnest($1::text[]) AS requested(id)
		  LEFT JOIN `+entityid.Subscription+` s
		    ON s.client_id = requested.id
		   AND s.workspace_id = $2
		   AND s.active = true
		 GROUP BY requested.id`, pq.Array(clientIDs), workspaceID)
	if err != nil {
		t.Fatalf("independent page count: %v", err)
	}
	defer countRows.Close()
	for countRows.Next() {
		var id string
		var count int32
		if err := countRows.Scan(&id, &count); err != nil {
			t.Fatalf("scan independent page count: %v", err)
		}
		expected[id] = count
	}
	if err := countRows.Err(); err != nil {
		t.Fatalf("iterate independent page counts: %v", err)
	}

	requested := make(map[string]struct{}, len(clientIDs))
	for _, id := range clientIDs {
		requested[id] = struct{}{}
		if got := resp.GetCounts()[id]; got != expected[id] {
			t.Errorf("client %s count = %d, independent count = %d", id, got, expected[id])
		}
	}
	for id := range resp.GetCounts() {
		if _, ok := requested[id]; !ok {
			t.Errorf("response included non-page client %s", id)
		}
	}
}

// TestRelationPageDataPaginatesInDatabase is a read-only live-schema oracle for
// COR-01. It compares each adapter's metadata and returned page length with an
// independent scoped count. Any relation above 100 rows is read from page 3,
// proving the adapter no longer paginates an already-truncated first-100 slice.
func TestRelationPageDataPaginatesInDatabase(t *testing.T) {
	db := openRelationPaginationDB(t)
	defer db.Close()

	ctx := context.Background()
	var workspaceID string
	if err := db.QueryRowContext(ctx, `
		SELECT workspace_id
		  FROM `+entityid.SubscriptionGroup+`
		 WHERE workspace_id IS NOT NULL
		 GROUP BY workspace_id
		 ORDER BY count(*) DESC
		 LIMIT 1`,
	).Scan(&workspaceID); err != nil {
		t.Fatalf("resolve relation-test workspace: %v", err)
	}
	ctx = identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspaceID})
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)

	type result struct {
		items      int
		pagination *commonpb.PaginationResponse
	}
	tests := []struct {
		name  string
		table string
		call  func(*commonpb.PaginationRequest) (result, error)
	}{
		{
			name: "subscription_group", table: entityid.SubscriptionGroup,
			call: func(p *commonpb.PaginationRequest) (result, error) {
				resp, err := NewPostgresSubscriptionGroupRepository(dbOps, entityid.SubscriptionGroup).
					GetSubscriptionGroupListPageData(ctx, &sgpb.GetSubscriptionGroupListPageDataRequest{Pagination: p})
				if resp == nil {
					return result{}, err
				}
				return result{len(resp.GetSubscriptionGroupList()), resp.GetPagination()}, err
			},
		},
		{
			name: "subscription_group_product_plan", table: entityid.SubscriptionGroupProductPlan,
			call: func(p *commonpb.PaginationRequest) (result, error) {
				resp, err := NewPostgresSubscriptionGroupProductPlanRepository(dbOps, entityid.SubscriptionGroupProductPlan).
					GetSubscriptionGroupProductPlanListPageData(ctx, &planpb.GetSubscriptionGroupProductPlanListPageDataRequest{Pagination: p})
				if resp == nil {
					return result{}, err
				}
				return result{len(resp.GetSubscriptionGroupProductPlanList()), resp.GetPagination()}, err
			},
		},
		{
			name: "subscription_group_member", table: entityid.SubscriptionGroupMember,
			call: func(p *commonpb.PaginationRequest) (result, error) {
				resp, err := NewPostgresSubscriptionGroupMemberRepository(dbOps, entityid.SubscriptionGroupMember).
					GetSubscriptionGroupMemberListPageData(ctx, &memberpb.GetSubscriptionGroupMemberListPageDataRequest{Pagination: p})
				if resp == nil {
					return result{}, err
				}
				return result{len(resp.GetSubscriptionGroupMemberList()), resp.GetPagination()}, err
			},
		},
		{
			name: "subscription_group_workspace_user", table: entityid.SubscriptionGroupWorkspaceUser,
			call: func(p *commonpb.PaginationRequest) (result, error) {
				resp, err := NewPostgresSubscriptionGroupWorkspaceUserRepository(dbOps, entityid.SubscriptionGroupWorkspaceUser).
					GetSubscriptionGroupWorkspaceUserListPageData(ctx, &workspaceuserpb.GetSubscriptionGroupWorkspaceUserListPageDataRequest{Pagination: p})
				if resp == nil {
					return result{}, err
				}
				return result{len(resp.GetSubscriptionGroupWorkspaceUserList()), resp.GetPagination()}, err
			},
		},
		{
			name: "subscription_group_product_plan_staff", table: entityid.SubscriptionGroupProductPlanStaff,
			call: func(p *commonpb.PaginationRequest) (result, error) {
				resp, err := NewPostgresSubscriptionGroupProductPlanStaffRepository(dbOps, entityid.SubscriptionGroupProductPlanStaff).
					GetSubscriptionGroupProductPlanStaffListPageData(ctx, &staffpb.GetSubscriptionGroupProductPlanStaffListPageDataRequest{Pagination: p})
				if resp == nil {
					return result{}, err
				}
				return result{len(resp.GetSubscriptionGroupProductPlanStaffList()), resp.GetPagination()}, err
			},
		},
		{
			name: "price_schedule_workspace_user", table: entityid.PriceScheduleWorkspaceUser,
			call: func(p *commonpb.PaginationRequest) (result, error) {
				resp, err := NewPostgresPriceScheduleWorkspaceUserRepository(dbOps, entityid.PriceScheduleWorkspaceUser).
					GetPriceScheduleWorkspaceUserListPageData(ctx, &planSchwpb.GetPriceScheduleWorkspaceUserListPageDataRequest{Pagination: p})
				if resp == nil {
					return result{}, err
				}
				return result{len(resp.GetPriceScheduleWorkspaceUserList()), resp.GetPagination()}, err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := postgresCore.ValidateSQLIdent(tc.table); err != nil {
				t.Fatalf("invalid author-owned table identifier: %v", err)
			}
			var total int32
			countSQL := fmt.Sprintf(`SELECT count(*) FROM %s WHERE active = true AND workspace_id = $1`, tc.table)
			if err := db.QueryRowContext(ctx, countSQL, workspaceID).Scan(&total); err != nil {
				t.Fatalf("independent scoped count: %v", err)
			}

			page := int32(1)
			if total > 100 {
				page = 3
			}
			got, err := tc.call(&commonpb.PaginationRequest{
				Limit: 50,
				Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{
					Page: page,
				}},
			})
			if err != nil {
				t.Fatalf("adapter page: %v", err)
			}
			if got.pagination == nil {
				t.Fatal("adapter returned nil pagination")
			}
			if got.pagination.GetTotalItems() != total {
				t.Fatalf("total_items = %d, independent count = %d", got.pagination.GetTotalItems(), total)
			}
			wantItems := total - (page-1)*50
			if wantItems < 0 {
				wantItems = 0
			}
			if wantItems > 50 {
				wantItems = 50
			}
			if got.items != int(wantItems) {
				t.Fatalf("page %d items = %d, want %d from total %d", page, got.items, wantItems, total)
			}
			if got.pagination.GetCurrentPage() != page {
				t.Fatalf("current_page = %d, want %d", got.pagination.GetCurrentPage(), page)
			}
		})
	}
}
