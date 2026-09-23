//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"testing"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// D7 record/edit-ownership matrix for the outcome_matrix grid's class-edge
// fallback (classEdgeExpr, outcome_matrix_query.go) — DB-gated, ROLLBACK-ONLY,
// mirroring job_phase_ownership_integration_test.go's harness. This proves the
// 2026-09-24 DEC-6 owner decision at the actual API surface fayna reads
// (OutcomeCell.Editable), not just the shared SQL fragment: a class-edge
// teacher whose ONLY edge is role='secondary' must see Editable=true on an
// unassigned, not-yet-recorded cell — the same widening already proven for the
// submit gate in job_phase_ownership_integration_test.go, and now structurally
// guaranteed in lockstep because outcome_matrix_query.go's classEdgeExpr calls
// classEdgeOwnedSQL directly (no second hand-maintained copy of the fragment).
//
// Fixture strategy: pick_ownershipFixture's sheet guarantee (all active tasks
// unassigned, output-product-bearing, class-wide PRIMARY edge covering it) is
// necessary but not sufficient here — computeCellEditable also branches on
// hasOutcome (a RECORDED cell is recorder-only regardless of the class edge),
// so this picker additionally requires NO active task_outcome yet on any task
// in the sheet, so every observed cell takes the class-edge fallback branch.

type editableFixture struct {
	templateID string
	phaseID    string
	workspace  string
	edgeFacet  string // staff whose class-wide PRIMARY edge covers the sheet, no outcomes recorded yet
	jobIDs     map[string]bool
}

func pickEditableFixture(t *testing.T, db *sql.DB) editableFixture {
	t.Helper()
	const q = `
		WITH sheet AS (
			SELECT j.job_template_id AS tid, jp.template_phase_id AS pid, j.workspace_id AS ws,
			       count(DISTINCT jp.id) AS members
			FROM job_phase jp
			JOIN job j ON j.id = jp.job_id
			WHERE jp.active = true AND jp.template_phase_id IS NOT NULL
			  AND j.job_template_id IS NOT NULL AND j.workspace_id IS NOT NULL
			GROUP BY 1, 2, 3
			HAVING count(DISTINCT jp.id) BETWEEN 1 AND 60
			   AND bool_and(j.output_product_id IS NOT NULL)
			   AND bool_and(EXISTS (SELECT 1 FROM job_task jt
			                        WHERE jt.job_phase_id = jp.id AND jt.active = true))
			   AND bool_and(NOT EXISTS (SELECT 1 FROM job_task jt
			                            WHERE jt.job_phase_id = jp.id AND jt.active = true
			                              AND COALESCE(jt.assigned_to, '') <> ''))
			   AND bool_and(NOT EXISTS (SELECT 1 FROM job_task jt
			                            JOIN task_outcome to_ ON to_.job_task_id = jt.id AND to_.active = true
			                            WHERE jt.job_phase_id = jp.id AND jt.active = true))
		)
		SELECT s.tid, s.pid, s.ws, cand.staff_id
		FROM sheet s
		JOIN LATERAL (
			SELECT DISTINCT e.staff_id
			FROM subscription_group_product_plan_staff e
			WHERE e.active AND e.role = 'primary' AND e.job_template_phase_id IS NULL
			  AND e.workspace_id = s.ws
		) cand ON true
		WHERE NOT EXISTS (
			SELECT 1
			FROM job_phase jp JOIN job j ON j.id = jp.job_id
			WHERE jp.template_phase_id = s.pid AND j.job_template_id = s.tid
			  AND j.workspace_id = s.ws AND jp.active = true
			  AND NOT EXISTS (
			    SELECT 1
			    FROM subscription_group_member m
			    JOIN subscription_group sg ON sg.id = m.subscription_group_id AND sg.status = 'current'
			    JOIN subscription_group_product_plan c
			           ON c.subscription_group_id = m.subscription_group_id AND c.active AND c.workspace_id = s.ws
			    JOIN product_plan pp ON pp.id = c.product_plan_id AND pp.product_id = j.output_product_id
			    JOIN subscription_group_product_plan_staff e
			           ON e.subscription_group_product_plan_id = c.id AND e.active AND e.role = 'primary'
			          AND e.job_template_phase_id IS NULL
			          AND e.staff_id = cand.staff_id AND e.workspace_id = s.ws
			    WHERE m.client_id = j.client_id AND m.active AND m.workspace_id = s.ws
			  )
			)
		ORDER BY s.members ASC
		LIMIT 1`
	var fx editableFixture
	if err := db.QueryRow(q).Scan(&fx.templateID, &fx.phaseID, &fx.workspace, &fx.edgeFacet); err != nil {
		t.Skipf("no editable-fixture sheet (unassigned, unrecorded, class-wide-primary-covered) on this DB: %v", err)
	}
	fx.jobIDs = map[string]bool{}
	rows, err := db.Query(`
		SELECT DISTINCT j.id FROM job j JOIN job_phase jp ON jp.job_id = j.id
		WHERE jp.template_phase_id = $1 AND j.job_template_id = $2 AND j.workspace_id = $3 AND jp.active = true`,
		fx.phaseID, fx.templateID, fx.workspace)
	if err != nil {
		t.Fatalf("load fixture job ids: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan fixture job id: %v", err)
		}
		fx.jobIDs[id] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate fixture job ids: %v", err)
	}
	if len(fx.jobIDs) == 0 {
		t.Skip("editable-fixture sheet resolved zero jobs")
	}
	return fx
}

// sheetCellsEditable runs the grid's cell query (loadRowsFrom, the injectable
// seam) inside the caller's tx as facet, and reports whether EVERY cell
// belonging to the fixture's sheet (matched by OutcomeCell.JobId) came back
// Editable=true. Any cell outside the sheet (other phases of the same
// template) is ignored — only the fixture's tasks are asserted.
func sheetCellsEditable(ctx context.Context, t *testing.T, tx *sql.Tx, fx editableFixture, facet string) (allEditable bool, sheetCells int) {
	t.Helper()
	idctx := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{
		UserID:        "test-user-" + facet,
		WorkspaceID:   fx.workspace,
		PrincipalType: principalscope.PrincipalTypeStaff,
		PrincipalID:   facet,
	})
	q := &PostgresOutcomeMatrixQuery{}
	rows, err := q.loadRowsFrom(idctx, tx, &matrixpb.GetOutcomeMatrixRequest{JobTemplateId: fx.templateID}, fx.workspace)
	if err != nil {
		t.Fatalf("loadRowsFrom: %v", err)
	}
	allEditable = true
	for _, row := range rows {
		for _, cell := range row.GetCells() {
			if !fx.jobIDs[cell.GetJobId()] {
				continue
			}
			sheetCells++
			if !cell.GetEditable() {
				allEditable = false
			}
		}
	}
	return allEditable, sheetCells
}

func TestIntegration_OutcomeMatrixClassEdgeEditable(t *testing.T) {
	db := openGroupNarrowDB(t)
	fx := pickEditableFixture(t, db)
	t.Logf("editable fixture: template=%s phase=%s ws=%s edgeFacet=%s jobs=%d",
		fx.templateID, fx.phaseID, fx.workspace, fx.edgeFacet, len(fx.jobIDs))

	t.Run("primary_role_editable", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			editable, n := sheetCellsEditable(ctx, t, tx, fx, fx.edgeFacet)
			if n == 0 {
				t.Fatal("no sheet cells observed — fixture/query mismatch")
			}
			if !editable {
				t.Error("expected every unassigned, unrecorded sheet cell editable via the class-wide PRIMARY edge")
			}
		})
	})

	t.Run("secondary_role_editable", func(t *testing.T) {
		// 2026-09-24 DEC-6 owner decision: a class-edge teacher may edit/record
		// regardless of edge role — 'secondary' must NOT be excluded.
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE subscription_group_product_plan_staff SET role = 'secondary'
				WHERE staff_id = $1 AND active = true`, fx.edgeFacet); err != nil {
				t.Fatalf("in-tx demote edges: %v", err)
			}
			editable, n := sheetCellsEditable(ctx, t, tx, fx, fx.edgeFacet)
			if n == 0 {
				t.Fatal("no sheet cells observed — fixture/query mismatch")
			}
			if !editable {
				t.Error("expected every unassigned, unrecorded sheet cell editable via a SECONDARY class edge")
			}
		})
	})

	t.Run("inactive_edge_not_editable", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE subscription_group_product_plan_staff SET active = false
				WHERE staff_id = $1`, fx.edgeFacet); err != nil {
				t.Fatalf("in-tx deactivate edges: %v", err)
			}
			editable, n := sheetCellsEditable(ctx, t, tx, fx, fx.edgeFacet)
			if n == 0 {
				t.Fatal("no sheet cells observed — fixture/query mismatch")
			}
			if editable {
				t.Error("expected sheet cells NOT editable once the class edge is inactive")
			}
		})
	})

	t.Run("no_edge_not_editable", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			var noEdgeFacet string
			if err := tx.QueryRowContext(ctx, `
				SELECT st.id FROM staff st
				WHERE st.workspace_id = $1 AND st.active = true
				  AND NOT EXISTS (SELECT 1 FROM subscription_group_product_plan_staff e WHERE e.staff_id = st.id)
				LIMIT 1`, fx.workspace).Scan(&noEdgeFacet); err != nil {
				t.Skipf("no edge-less staff facet in the fixture workspace: %v", err)
			}
			editable, n := sheetCellsEditable(ctx, t, tx, fx, noEdgeFacet)
			if n == 0 {
				t.Fatal("no sheet cells observed — fixture/query mismatch")
			}
			if editable {
				t.Error("expected sheet cells NOT editable for a staff facet with no class edge at all")
			}
		})
	})

	t.Run("other_group_not_editable", func(t *testing.T) {
		// Completing the fixture's section removes it from the reachable-set:
		// sg.status <> 'current' denies the class-edge fallback (the
		// completed-AY leak gate), the same "other group" shape the submit
		// matrix proves via group_not_current_denies.
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE subscription_group sg SET status = 'completed'
				WHERE sg.status = 'current' AND EXISTS (
				  SELECT 1 FROM subscription_group_product_plan c
				  JOIN subscription_group_product_plan_staff e ON e.subscription_group_product_plan_id = c.id
				  WHERE c.subscription_group_id = sg.id AND e.staff_id = $1 AND e.active = true)`,
				fx.edgeFacet); err != nil {
				t.Fatalf("in-tx complete groups: %v", err)
			}
			editable, n := sheetCellsEditable(ctx, t, tx, fx, fx.edgeFacet)
			if n == 0 {
				t.Fatal("no sheet cells observed — fixture/query mismatch")
			}
			if editable {
				t.Error("expected sheet cells NOT editable once the member's section is no longer current")
			}
		})
	})
}
