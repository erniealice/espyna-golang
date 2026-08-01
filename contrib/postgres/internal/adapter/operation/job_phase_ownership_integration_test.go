//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// D7 submit-ownership matrix under the 2026-07-26 COALESCE model — DB-gated,
// ROLLBACK-ONLY (education1 convention: every mutation happens inside a
// transaction that is always rolled back; nothing is ever committed). Gated on
// TEST_DATABASE_URL like the other integration suites (reuses openGroupNarrowDB).
//
// Fixture strategy: pick ONE live sheet whose active jobs all carry an output
// product, whose active tasks are ALL unassigned (the post-20260726 shape), and
// ONE staff facet whose class-wide (f14 NULL) active PRIMARY sgpps edges cover
// every job of the sheet. Each matrix cell then flips exactly one factor in-tx
// and asserts the ownership verdict flips with it — the deny flips are the
// behavioral content (they cannot be satisfied by the fixture selection).

type ownershipFixture struct {
	templateID  string
	phaseID     string // sheet parent (job_template_phase), phase_order NOT NULL
	workspace   string
	edgeFacet   string // staff whose class-wide primary edges cover the whole sheet
	siblingID   string // sibling template phase, different non-null phase_order
	noEdgeFacet string // active staff in ws holding NO sgpps edge at all
}

func pickOwnershipFixture(t *testing.T, db *sql.DB) ownershipFixture {
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
			HAVING count(DISTINCT jp.id) BETWEEN 2 AND 120
			   AND bool_and(j.output_product_id IS NOT NULL)
			   AND bool_and(EXISTS (SELECT 1 FROM job_task jt
			                        WHERE jt.job_phase_id = jp.id AND jt.active = true))
			   AND bool_and(NOT EXISTS (SELECT 1 FROM job_task jt
			                            WHERE jt.job_phase_id = jp.id AND jt.active = true
			                              AND COALESCE(jt.assigned_to, '') <> ''))
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
		AND EXISTS (
			SELECT 1 FROM job_template_phase me
			JOIN job_template_phase sib ON sib.job_template_id = me.job_template_id
			WHERE me.id = s.pid AND me.phase_order IS NOT NULL
			  AND sib.id <> me.id AND sib.phase_order IS NOT NULL
			  AND sib.phase_order <> me.phase_order
		)
		ORDER BY s.members ASC
		LIMIT 1`
	var fx ownershipFixture
	if err := db.QueryRow(q).Scan(&fx.templateID, &fx.phaseID, &fx.workspace, &fx.edgeFacet); err != nil {
		t.Skipf("no COALESCE-ownership fixture sheet on this DB: %v", err)
	}
	if err := db.QueryRow(`
		SELECT sib.id FROM job_template_phase me
		JOIN job_template_phase sib ON sib.job_template_id = me.job_template_id
		WHERE me.id = $1 AND me.phase_order IS NOT NULL
		  AND sib.id <> me.id AND sib.phase_order IS NOT NULL
		  AND sib.phase_order <> me.phase_order
		LIMIT 1`, fx.phaseID).Scan(&fx.siblingID); err != nil {
		t.Skipf("no sibling phase with a different phase_order: %v", err)
	}
	if err := db.QueryRow(`
		SELECT st.id FROM staff st
		WHERE st.workspace_id = $1 AND st.active = true
		  AND NOT EXISTS (SELECT 1 FROM subscription_group_product_plan_staff e WHERE e.staff_id = st.id)
		LIMIT 1`, fx.workspace).Scan(&fx.noEdgeFacet); err != nil {
		t.Skipf("no edge-less staff facet in the fixture workspace: %v", err)
	}
	return fx
}

// inTx runs fn inside a transaction that is ALWAYS rolled back.
func inTx(t *testing.T, db *sql.DB, fn func(ctx context.Context, tx *sql.Tx)) {
	t.Helper()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck — rollback-only harness
	fn(ctx, tx)
}

// assertOwned asserts the D7 verdict. A denial must be the ownership error —
// any other error (SQL failure, bad bind) fails the test loudly instead of
// masquerading as a deny.
func assertOwned(t *testing.T, err error, wantOwned bool, scenario string) {
	t.Helper()
	if wantOwned {
		if err != nil {
			t.Fatalf("%s: expected OWNED, got: %v", scenario, err)
		}
		return
	}
	if err == nil {
		t.Fatalf("%s: expected DENY, got owned", scenario)
	}
	if !strings.Contains(err.Error(), "D7 ownership") {
		t.Fatalf("%s: expected the D7 ownership denial, got: %v", scenario, err)
	}
}

// assignSheetTasks sets assigned_to on every active task of the fixture sheet
// (in-tx only). Pass sql NULL via nil.
func assignSheetTasks(ctx context.Context, t *testing.T, tx *sql.Tx, fx ownershipFixture, assignee any) {
	t.Helper()
	if _, err := tx.ExecContext(ctx, `
		UPDATE job_task jt SET assigned_to = $4
		FROM job_phase jp, job j
		WHERE jt.job_phase_id = jp.id AND jp.job_id = j.id
		  AND jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
		  AND jp.active = true AND jt.active = true`,
		fx.templateID, fx.phaseID, fx.workspace, assignee); err != nil {
		t.Fatalf("in-tx assign sheet tasks: %v", err)
	}
}

func TestIntegration_D7OwnershipMatrix_CoalesceModel(t *testing.T) {
	db := openGroupNarrowDB(t)
	fx := pickOwnershipFixture(t, db)
	t.Logf("fixture: template=%s phase=%s ws=%s edgeFacet=%s sibling=%s noEdgeFacet=%s",
		fx.templateID, fx.phaseID, fx.workspace, fx.edgeFacet, fx.siblingID, fx.noEdgeFacet)

	probe := func(ctx context.Context, tx *sql.Tx, facet string) error {
		return assertAllTasksOwned(ctx, tx, fx.templateID, fx.phaseID, fx.workspace, facet, "")
	}

	t.Run("edge_only_class_wide_owns", func(t *testing.T) {
		// All tasks unassigned; the facet's class-wide primary edges govern.
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), true, "class-wide edge, no overrides")
		})
	})

	t.Run("override_binds_alone", func(t *testing.T) {
		// Explicit assigned_to to an edge-less facet: that facet owns, and the
		// edge holder is DENIED — an edge never trumps someone else's assignment.
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			assignSheetTasks(ctx, t, tx, fx, fx.noEdgeFacet)
			assertOwned(t, probe(ctx, tx, fx.noEdgeFacet), true, "override for edge-less facet")
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), false, "edge holder vs foreign override")
		})
	})

	t.Run("no_override_no_edge_denies", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			assertOwned(t, probe(ctx, tx, fx.noEdgeFacet), false, "no override, no edge")
		})
	})

	t.Run("legacy_null_output_product", func(t *testing.T) {
		// A legacy job (output_product_id NULL) is out of the class model: the
		// edge can never own it — only an explicit assigned_to can.
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE job j SET output_product_id = NULL
				WHERE j.job_template_id = $1 AND j.workspace_id = $3
				  AND EXISTS (SELECT 1 FROM job_phase jp
				              WHERE jp.job_id = j.id AND jp.template_phase_id = $2 AND jp.active = true)`,
				fx.templateID, fx.phaseID, fx.workspace); err != nil {
				t.Fatalf("in-tx null output_product_id: %v", err)
			}
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), false, "legacy job, NULL assigned_to")
			assignSheetTasks(ctx, t, tx, fx, fx.edgeFacet)
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), true, "legacy job, explicit assignment")
		})
	})

	t.Run("secondary_role_denies", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE subscription_group_product_plan_staff SET role = 'secondary'
				WHERE staff_id = $1 AND active = true`, fx.edgeFacet); err != nil {
				t.Fatalf("in-tx demote edges: %v", err)
			}
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), false, "secondary-role edge")
		})
	})

	t.Run("inactive_edge_denies", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE subscription_group_product_plan_staff SET active = false
				WHERE staff_id = $1`, fx.edgeFacet); err != nil {
				t.Fatalf("in-tx deactivate edges: %v", err)
			}
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), false, "inactive edge")
		})
	})

	t.Run("group_not_current_denies", func(t *testing.T) {
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
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), false, "group not current")
		})
	})

	// Phase-scoped (f14) edges own by PHASE ORDER — and ONLY their order.
	scopeEdges := func(ctx context.Context, t *testing.T, tx *sql.Tx, toPhase string) {
		t.Helper()
		// Neutralize any pre-existing phase-scoped edges of the facet so the
		// verdict is attributable to the freshly scoped ones alone.
		if _, err := tx.ExecContext(ctx, `
			UPDATE subscription_group_product_plan_staff SET active = false
			WHERE staff_id = $1 AND active = true AND job_template_phase_id IS NOT NULL`,
			fx.edgeFacet); err != nil {
			t.Fatalf("in-tx neutralize scoped edges: %v", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE subscription_group_product_plan_staff SET job_template_phase_id = $2
			WHERE staff_id = $1 AND active = true AND job_template_phase_id IS NULL`,
			fx.edgeFacet, toPhase); err != nil {
			t.Fatalf("in-tx scope edges: %v", err)
		}
	}

	t.Run("phase_scoped_edge_owns_matching_order", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			scopeEdges(ctx, t, tx, fx.phaseID)
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), true, "f14 edge, matching phase_order")
		})
	})

	t.Run("phase_scoped_edge_denies_other_order", func(t *testing.T) {
		inTx(t, db, func(ctx context.Context, tx *sql.Tx) {
			scopeEdges(ctx, t, tx, fx.siblingID)
			assertOwned(t, probe(ctx, tx, fx.edgeFacet), false, "f14 edge, foreign phase_order")
		})
	})
}
