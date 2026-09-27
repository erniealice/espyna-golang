//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/shared/identity"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

type matrixCaptureQueryer struct {
	db    *sql.DB
	query string
	args  []any
}

func (c *matrixCaptureQueryer) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	c.query, c.args = query, append([]any(nil), args...)
	return c.db.QueryContext(ctx, query, args...)
}

func matrixQueryBuffers(t *testing.T, ctx context.Context, c *matrixCaptureQueryer) int64 {
	t.Helper()
	var raw []byte
	if err := c.db.QueryRowContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+c.query, c.args...).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var plans []struct {
		Plan struct {
			SharedHitBlocks  int64 `json:"Shared Hit Blocks"`
			SharedReadBlocks int64 `json:"Shared Read Blocks"`
		} `json:"Plan"`
	}
	if err := json.Unmarshal(raw, &plans); err != nil || len(plans) != 1 {
		t.Fatalf("decode EXPLAIN plan: count=%d err=%v", len(plans), err)
	}
	return plans[0].Plan.SharedHitBlocks + plans[0].Plan.SharedReadBlocks
}

// Q6: loadRowsFromWithClassEdgeGuard(false) is the pre-P2 unguarded SQL oracle;
// true is the production query. Compare every returned cell, including Editable,
// after staff reachability has selected the same ordered roster.
func TestOutcomeMatrixStaffDB_Parity(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cohorts := []struct{ name, sample string }{
		{"assignee", `SELECT jt.assigned_to,jp.workspace_id FROM job_task jt JOIN job_phase jp ON jp.id=jt.job_phase_id AND jp.workspace_id=jt.workspace_id JOIN job j ON j.id=jp.job_id AND j.workspace_id=jp.workspace_id WHERE jt.assigned_to IS NOT NULL AND jt.assigned_to<>'' ORDER BY jt.assigned_to LIMIT 1`},
		{"recorder", `SELECT t.recorded_by,t.workspace_id FROM task_outcome t JOIN job_task jt ON jt.id=t.job_task_id AND jt.workspace_id=t.workspace_id JOIN job_phase jp ON jp.id=jt.job_phase_id AND jp.workspace_id=t.workspace_id JOIN job j ON j.id=jp.job_id AND j.workspace_id=t.workspace_id WHERE t.recorded_by IS NOT NULL AND t.recorded_by<>'' ORDER BY t.recorded_by LIMIT 1`},
		{"approver", `SELECT t.reviewed_by,t.workspace_id FROM task_outcome t JOIN job_task jt ON jt.id=t.job_task_id AND jt.workspace_id=t.workspace_id JOIN job_phase jp ON jp.id=jt.job_phase_id AND jp.workspace_id=t.workspace_id JOIN job j ON j.id=jp.job_id AND j.workspace_id=t.workspace_id WHERE t.reviewed_by IS NOT NULL AND t.reviewed_by<>'' ORDER BY t.reviewed_by LIMIT 1`},
		{"subscription_seat", `SELECT ss.staff_id,ss.workspace_id FROM subscription_seat ss JOIN job j ON j.origin_id=ss.subscription_id AND j.workspace_id=ss.workspace_id AND j.origin_type='ORIGIN_TYPE_SUBSCRIPTION' JOIN job_template tpl ON tpl.id=j.job_template_id AND tpl.workspace_id=ss.workspace_id JOIN product_plan pl ON pl.id=ss.product_plan_id AND pl.product_id=tpl.output_product_id WHERE ss.staff_id IS NOT NULL AND ss.status='active' AND ss.active ORDER BY ss.staff_id LIMIT 1`},
		{"class_edge", `SELECT COALESCE(pps.staff_id,e.staff_id),e.workspace_id FROM subscription_group_product_plan_staff e JOIN subscription_group_member m ON m.subscription_group_id=e.subscription_group_id AND m.workspace_id=e.workspace_id AND m.active JOIN product_plan pp ON pp.id=e.product_plan_id JOIN job jce ON jce.origin_id=m.subscription_id AND jce.output_product_id=pp.product_id AND jce.workspace_id=e.workspace_id LEFT JOIN product_plan_staff pps ON pps.id=e.product_plan_staff_id AND pps.workspace_id=e.workspace_id WHERE e.active AND (e.product_plan_staff_id IS NULL OR pps.active) AND COALESCE(pps.staff_id,e.staff_id) IS NOT NULL ORDER BY COALESCE(pps.staff_id,e.staff_id) LIMIT 1`},
		{"reviewer", `SELECT r.staff_id,r.workspace_id FROM product_plan_staff r JOIN subscription_group_product_plan rc ON rc.product_plan_id=r.product_plan_id AND rc.active AND rc.workspace_id=r.workspace_id JOIN subscription_group_member rm ON rm.subscription_group_id=rc.subscription_group_id AND rm.active AND rm.workspace_id=r.workspace_id JOIN product_plan rpp ON rpp.id=r.product_plan_id JOIN job jr ON jr.origin_id=rm.subscription_id AND jr.output_product_id=rpp.product_id AND jr.workspace_id=r.workspace_id WHERE r.staff_id IS NOT NULL AND r.role='reviewer' AND r.active ORDER BY r.staff_id LIMIT 1`},
	}
	a := &PostgresOutcomeMatrixQuery{db: db}
	compare := func(t *testing.T, ctx context.Context, template, workspace string, scope matrixpb.OutcomeMatrixScope) {
		t.Helper()
		req := &matrixpb.GetOutcomeMatrixRequest{JobTemplateId: template, Scope: scope}
		oldQuery := &matrixCaptureQueryer{db: db}
		newQuery := &matrixCaptureQueryer{db: db}
		oldRows, err := a.loadRowsFromWithClassEdgeGuard(ctx, oldQuery, req, workspace, false)
		if err != nil {
			t.Fatalf("legacy cells: %v", err)
		}
		newRows, err := a.loadRowsFromWithClassEdgeGuard(ctx, newQuery, req, workspace, true)
		if err != nil {
			t.Fatalf("candidate cells: %v", err)
		}
		if len(oldRows) != len(newRows) {
			t.Fatalf("client rows differ: old=%d new=%d", len(oldRows), len(newRows))
		}
		cells := 0
		for i := range oldRows {
			if !proto.Equal(oldRows[i], newRows[i]) {
				t.Fatalf("client row %d differs in cell values or editability", i)
			}
			cells += len(oldRows[i].GetCells())
		}
		if cells == 0 {
			t.Fatal("sampled matrix had no cells; parity would be vacuous")
		}
		t.Logf("rows=%d cells=%d", len(oldRows), cells)
		if os.Getenv("TEST_PERF_BUDGET") == "1" && (strings.HasSuffix(t.Name(), "/assignee") || strings.HasSuffix(t.Name(), "/class_edge")) {
			before := matrixQueryBuffers(t, ctx, oldQuery)
			after := matrixQueryBuffers(t, ctx, newQuery)
			t.Logf("production SQL EXPLAIN shared hit+read: old=%d new=%d", before, after)
			if after*2 > before {
				t.Fatalf("buffer budget exceeded: old=%d new=%d", before, after)
			}
		}
	}
	for _, cohort := range cohorts {
		t.Run(cohort.name, func(t *testing.T) {
			var staff, workspace, template string
			if err := db.QueryRow(cohort.sample).Scan(&staff, &workspace); err == sql.ErrNoRows {
				t.Skip("no sampled staff in this cohort")
			} else if err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(`SELECT j.job_template_id FROM job j WHERE j.workspace_id=$2 AND j.active AND j.id IN (`+principalscope.StaffReachableJobIDsSQL()+`) AND j.job_template_id IS NOT NULL ORDER BY j.job_template_id LIMIT 1`, staff, workspace).Scan(&template); err == sql.ErrNoRows {
				t.Skip("sampled staff has no active template")
			} else if err != nil {
				t.Fatal(err)
			}
			ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace, PrincipalType: principalscope.PrincipalTypeStaff, PrincipalID: staff})
			compare(t, ctx, template, workspace, matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_MINE)
		})
	}
	t.Run("non_staff_all", func(t *testing.T) {
		var template, workspace string
		if err := db.QueryRow(`SELECT j.job_template_id,j.workspace_id FROM job j JOIN job_phase jp ON jp.job_id=j.id AND jp.active JOIN job_task jt ON jt.job_phase_id=jp.id AND jt.active JOIN template_task_criteria ttc ON ttc.job_template_task_id=jt.template_task_id AND ttc.active WHERE j.active AND j.job_template_id IS NOT NULL ORDER BY j.job_template_id LIMIT 1`).Scan(&template, &workspace); err != nil {
			t.Fatal(err)
		}
		ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace, PrincipalType: 1, PrincipalID: "operator"})
		compare(t, ctx, template, workspace, matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_ALL)
		req := &matrixpb.GetOutcomeMatrixRequest{JobTemplateId: template, Scope: matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_MINE}
		for _, candidate := range []bool{false, true} {
			rows, err := a.loadRowsFromWithClassEdgeGuard(ctx, db, req, workspace, candidate)
			if err != nil || len(rows) != 0 {
				t.Fatalf("non-staff MINE must fail closed: candidate=%v rows=%d err=%v", candidate, len(rows), err)
			}
		}
	})
}
