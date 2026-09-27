//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// Plan 20260927-db-query-performance AC-08 (audit DB-01). The Courses-list SQL
// must return exactly what the frozen legacy builder returned — same rows, same
// order, same metadata — for every request shape. SELECT-only; skips without
// TEST_DATABASE_URL. With TEST_PERF_BUDGET=1 the new statement must also read
// fewer shared buffers than the legacy one on the admin first page.

type summaryParityCase struct {
	name         string
	status       string
	groupID      string
	limit        int32
	offset       int32
	options      jobTemplateSummaryQueryOptions
	staff        bool
	budgetMaxPct int // TEST_PERF_BUDGET=1: new buffers must be <= this % of legacy (0 = not asserted)
}

func TestJobTemplateSummaryDB_Parity(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()

	var ws string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC, workspace_id LIMIT 1`,
	).Scan(&ws); err != nil {
		t.Skipf("no jobs: %v", err)
	}
	var groupID string
	_ = db.QueryRowContext(ctx,
		`SELECT subscription_group_id FROM `+entityid.SubscriptionGroupMember+`
		 WHERE workspace_id = $1 AND active GROUP BY subscription_group_id
		 ORDER BY COUNT(*) DESC, subscription_group_id LIMIT 1`, ws).Scan(&groupID)
	var staffID string
	_ = db.QueryRowContext(ctx,
		`SELECT staff_id FROM `+entityid.SubscriptionSeat+`
		 WHERE status = 'active' AND active AND workspace_id = $1
		 GROUP BY staff_id ORDER BY COUNT(*) DESC, staff_id LIMIT 1`, ws).Scan(&staffID)

	search := func(q string) *commonpb.SearchRequest { return &commonpb.SearchRequest{Query: q} }
	cases := []summaryParityCase{
		{name: "admin_first_page", status: "JOB_STATUS_ACTIVE", limit: 50, budgetMaxPct: 50},
		{name: "admin_deep_page", status: "JOB_STATUS_ACTIVE", limit: 50, offset: 250},
		{name: "admin_unpaged_all_status"},
		{name: "search_broad", status: "JOB_STATUS_ACTIVE", limit: 50, options: jobTemplateSummaryQueryOptions{search: search("a")}},
		{name: "search_empty", status: "JOB_STATUS_ACTIVE", limit: 50, options: jobTemplateSummaryQueryOptions{search: search("zzqqxx-no-match")}},
		{name: "group_filter", status: "JOB_STATUS_ACTIVE", groupID: groupID, limit: 50},
		{name: "template_fallback", status: "JOB_STATUS_ACTIVE", limit: 50, options: jobTemplateSummaryQueryOptions{includeTemplateFallback: true}},
		{name: "price_schedule_active", status: "JOB_STATUS_ACTIVE", limit: 50, options: jobTemplateSummaryQueryOptions{priceScheduleActive: true}},
		{name: "staff_first_page", status: "JOB_STATUS_ACTIVE", limit: 50, staff: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.groupID == "" && tc.name == "group_filter" {
				t.Skip("no subscription group sample")
			}
			var scopeFn func(int) (string, []any)
			if tc.staff {
				if staffID == "" {
					t.Skip("no seat staff sample")
				}
				staffCtx := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{
					WorkspaceID: ws, PrincipalType: 7, PrincipalID: staffID,
				})
				scopeFn = func(startParam int) (string, []any) {
					return principalscope.StaffReachableJobClause(staffCtx, "j", startParam)
				}
			}

			legacyStmt, legacyArgs, err := legacyBuildListJobTemplateSummariesRequestSQL(
				ws, tc.status, tc.groupID, tc.limit, tc.offset, tc.options, scopeFn)
			if err != nil {
				t.Fatalf("legacy builder: %v", err)
			}
			stmt, args, err := buildListJobTemplateSummariesRequestSQL(
				ws, tc.status, tc.groupID, tc.limit, tc.offset, tc.options, scopeFn)
			if err != nil {
				t.Fatalf("builder: %v", err)
			}

			want := summaryRowsAsText(t, db, legacyStmt, legacyArgs)
			got := summaryRowsAsText(t, db, stmt, args)
			if len(got) != len(want) {
				t.Fatalf("row count: new=%d legacy=%d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("row %d differs:\n new:    %s\n legacy: %s", i, got[i], want[i])
				}
			}

			legacyBuffers := summaryStatementBuffers(t, db, legacyStmt, legacyArgs)
			newBuffers := summaryStatementBuffers(t, db, stmt, args)
			t.Logf("rows=%d buffers legacy=%d new=%d", len(got), legacyBuffers, newBuffers)
			if os.Getenv("TEST_PERF_BUDGET") == "1" && tc.budgetMaxPct > 0 &&
				newBuffers*100 > legacyBuffers*int64(tc.budgetMaxPct) {
				t.Errorf("buffers new=%d > %d%% of legacy=%d", newBuffers, tc.budgetMaxPct, legacyBuffers)
			}
		})
	}
}

// summaryRowsAsText renders every result row as one string (NULL-aware), in
// result order, so two statements can be compared exactly.
func summaryRowsAsText(t *testing.T, db *sql.DB, stmt string, args []any) []string {
	t.Helper()
	rows, err := db.Query(stmt, args...)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("columns: %v", err)
	}
	var out []string
	for rows.Next() {
		values := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan: %v", err)
		}
		parts := make([]string, len(cols))
		for i, v := range values {
			if v.Valid {
				parts[i] = cols[i] + "=" + v.String
			} else {
				parts[i] = cols[i] + "=<NULL>"
			}
		}
		out = append(out, strings.Join(parts, " | "))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// summaryStatementBuffers returns shared hit+read buffers of the top plan node.
func summaryStatementBuffers(t *testing.T, db *sql.DB, stmt string, args []any) int64 {
	t.Helper()
	var raw []byte
	if err := db.QueryRow("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+stmt, args...).Scan(&raw); err != nil {
		t.Fatalf("explain: %v", err)
	}
	var plans []struct {
		Plan struct {
			Hit  int64 `json:"Shared Hit Blocks"`
			Read int64 `json:"Shared Read Blocks"`
		} `json:"Plan"`
	}
	if err := json.Unmarshal(raw, &plans); err != nil || len(plans) == 0 {
		t.Fatalf("explain json: %v", fmt.Sprint(err))
	}
	return plans[0].Plan.Hit + plans[0].Plan.Read
}
