//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/grade_compute"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// FIX-3 DB-gated ROLLBACK-ONLY integration proof (education1): the submit-time
// freshness barrier runs the REAL grade_compute use cases INSIDE the transition
// transaction — summary rewrites are visible pre-commit within the tx and absent
// after rollback — and a recompute failure fails the submit (nothing flips).
//
// Fixture strategy: education1's live sheets are ALL hard-frozen (authoritative
// imported finals / closed schedules), so the test neutralizes the freeze sources
// for ONE chosen sheet INSIDE the rolled-back transaction (in-tx UPDATEs that are
// never committed), then drives the REAL SubmitJobPhaseApproval with the REAL
// grade_compute.UseCases.SheetRecompute port — the exact closure production wires
// via SetSheetRecompute in operation/usecases.go. Admin override is minted through
// the internal approvalctx (this test lives inside the espyna module) so the D7
// staff-ownership leg — covered by its own suite — is not the subject here.

type fixp2bIDGen struct{ n int }

func (g *fixp2bIDGen) GenerateID() string {
	g.n++
	return fmt.Sprintf("fixp2b-%06d", g.n)
}
func (g *fixp2bIDGen) GenerateIDWithPrefix(prefix string) string {
	g.n++
	return fmt.Sprintf("%s-fixp2b-%06d", prefix, g.n)
}
func (g *fixp2bIDGen) IsEnabled() bool          { return true }
func (g *fixp2bIDGen) GetProviderInfo() string  { return "fixp2b-test" }

// pickSubmitFixtureSheet selects one uniform-IN_PROGRESS sheet that has computed
// phase summaries (the recompute will REWRITE them — a crisp in-tx observable) and
// no phase with numeric data but a missing summary (avoids a first-compute on
// unvetted config). Freeze state is NOT filtered — it is neutralized in-tx.
func pickSubmitFixtureSheet(t *testing.T, db *sql.DB) sheetFixture {
	t.Helper()
	const q = `
		WITH sheet AS (
			SELECT j.job_template_id AS tid, jp.template_phase_id AS pid, j.workspace_id AS ws,
			       count(*) AS n,
			       bool_and(jp.approval_status = 'PHASE_APPROVAL_STATUS_IN_PROGRESS') AS uniform_ip
			FROM job_phase jp JOIN job j ON j.id = jp.job_id
			WHERE jp.active = true AND jp.template_phase_id IS NOT NULL
			  AND j.job_template_id IS NOT NULL AND j.workspace_id IS NOT NULL
			GROUP BY 1,2,3 HAVING count(*) >= 2
		)
		SELECT s.tid, s.pid, s.ws
		FROM sheet s
		WHERE s.uniform_ip
		  AND EXISTS (
		    SELECT 1 FROM job_phase jp JOIN job j ON j.id = jp.job_id
		    JOIN phase_outcome_summary pos ON pos.job_phase_id = jp.id AND pos.active AND pos.summary_score IS NOT NULL
		    WHERE jp.template_phase_id = s.pid AND j.job_template_id = s.tid AND j.workspace_id = s.ws AND jp.active)
		  AND NOT EXISTS (
		    SELECT 1 FROM job_phase jp JOIN job j ON j.id = jp.job_id
		    WHERE jp.template_phase_id = s.pid AND j.job_template_id = s.tid AND j.workspace_id = s.ws AND jp.active
		      AND EXISTS (SELECT 1 FROM job_task jt JOIN task_outcome o ON o.job_task_id = jt.id AND o.active AND o.numeric_value IS NOT NULL
		                  WHERE jt.job_phase_id = jp.id AND jt.active)
		      AND NOT EXISTS (SELECT 1 FROM phase_outcome_summary pos WHERE pos.job_phase_id = jp.id AND pos.active AND pos.summary_score IS NOT NULL))
		ORDER BY s.n ASC LIMIT 1`
	var fx sheetFixture
	if err := db.QueryRow(q).Scan(&fx.templateID, &fx.phaseID, &fx.workspace); err != nil {
		t.Skipf("no submit-recompute fixture sheet on this DB: %v", err)
	}
	return fx
}

// neutralizeFreezeInTx relaxes the sheet's hard-freeze sources INSIDE the ambient
// transaction (rolled back later — never committed): authoritative job summaries
// and closed enclosing schedules for the sheet's jobs.
func neutralizeFreezeInTx(ctx context.Context, t *testing.T, exec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, fx sheetFixture) {
	t.Helper()
	const jobsPred = `
		SELECT j.id FROM job_phase jp JOIN job j ON j.id = jp.job_id
		WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3 AND jp.active = true`
	if _, err := exec.ExecContext(ctx,
		`UPDATE job_outcome_summary SET is_authoritative = false
		 WHERE active = true AND is_authoritative = true AND job_id IN (`+jobsPred+`)`,
		fx.templateID, fx.phaseID, fx.workspace); err != nil {
		t.Fatalf("in-tx relax authoritative summaries: %v", err)
	}
	if _, err := exec.ExecContext(ctx,
		`UPDATE price_schedule ps SET closed = false
		 WHERE ps.closed = true AND ps.id IN (
		   SELECT sg.price_schedule_id FROM subscription_group sg
		   JOIN subscription_group_member sgm ON sgm.subscription_group_id = sg.id
		   JOIN job j ON j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION' AND j.origin_id = sgm.subscription_id
		   WHERE sgm.workspace_id = j.workspace_id AND sg.workspace_id = j.workspace_id
		     AND j.id IN (`+jobsPred+`))`,
		fx.templateID, fx.phaseID, fx.workspace); err != nil {
		t.Fatalf("in-tx relax closed schedules: %v", err)
	}
}

func fixp2bCtx(base context.Context, ws string) context.Context {
	ctx := identity.WithRequestIdentity(base, &identity.RequestIdentity{UserID: "fixp2b-user", WorkspaceID: ws})
	// D4 admin override, minted via the internal approvalctx (only espyna-internal
	// code can) — the D7 ownership leg has its own coverage.
	return approvalctx.WithSubmitDecision(ctx, approvalctx.SubmitDecision{AdminOverride: true})
}

// TestIntegration_SubmitRecompute_RunsInsideTransitionTx is the FIX-3 in-tx proof.
func TestIntegration_SubmitRecompute_RunsInsideTransitionTx(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	fx := pickSubmitFixtureSheet(t, db)

	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repoIface := NewPostgresJobPhaseRepository(wsOps, "job_phase")
	repo := repoIface.(*PostgresJobPhaseRepository)

	// The REAL grade_compute wiring on REAL postgres repositories — the exact
	// aggregate production injects (operation/usecases.go SetSheetRecompute).
	gcUC := grade_compute.NewUseCases(grade_compute.Repositories{
		JobPhase:                 repoIface,
		JobTemplatePhase:         NewPostgresJobTemplatePhaseRepository(wsOps, "job_template_phase"),
		ScoringScheme:            NewPostgresScoringSchemeRepository(wsOps, "scoring_scheme"),
		ScoringComponent:         NewPostgresScoringComponentRepository(wsOps, "scoring_component"),
		ScoringComponentCriteria: NewPostgresScoringComponentCriteriaRepository(wsOps, "scoring_component_criteria"),
		ScoreScale:               NewPostgresScoreScaleRepository(wsOps, "score_scale"),
		ScoreScaleBand:           NewPostgresScoreScaleBandRepository(wsOps, "score_scale_band"),
		TaskOutcome:              NewPostgresTaskOutcomeRepository(wsOps, "task_outcome"),
		PhaseOutcomeSummary:      NewPostgresPhaseOutcomeSummaryRepository(wsOps, "phase_outcome_summary"),
		Job:                      NewPostgresJobRepository(wsOps, "job"),
		JobOutcomeSummary:        NewPostgresJobOutcomeSummaryRepository(wsOps, "job_outcome_summary"),
		JobOutcomeLine:           NewPostgresJobOutcomeLineRepository(wsOps, "job_outcome_line"),
	}, grade_compute.Services{IDGenerator: &fixp2bIDGen{}})

	// Wrap the real port to record its invocation (proves post-lock, full-sheet).
	var recomputeCalls int
	var recomputedPhases []string
	repo.SetSheetRecompute(func(ctx context.Context, phaseIDs, jobIDs []string) error {
		recomputeCalls++
		recomputedPhases = append([]string(nil), phaseIDs...)
		return gcUC.SheetRecompute(ctx, phaseIDs, jobIDs)
	})

	type preSummary struct{ id, dateModified string }
	preByPhase := map[string]preSummary{}
	var sheetPhaseIDs []string

	rollback := fmt.Errorf("fixp2b: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec, terr := repo.txExecutor(txCtx)
		if terr != nil {
			return terr
		}
		neutralizeFreezeInTx(txCtx, t, exec, fx)

		// Capture the sheet's phase ids + current summary stamps (in-tx, pre-submit).
		rows, qerr := exec.QueryContext(txCtx, `
			SELECT jp.id, COALESCE(pos.id, ''), COALESCE(pos.date_modified::text, '')
			FROM job_phase jp JOIN job j ON j.id = jp.job_id
			LEFT JOIN phase_outcome_summary pos ON pos.job_phase_id = jp.id AND pos.active
			WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3 AND jp.active = true
			ORDER BY jp.id`, fx.templateID, fx.phaseID, fx.workspace)
		if qerr != nil {
			return qerr
		}
		for rows.Next() {
			var pid, sid, dm string
			if serr := rows.Scan(&pid, &sid, &dm); serr != nil {
				rows.Close()
				return serr
			}
			sheetPhaseIDs = append(sheetPhaseIDs, pid)
			if sid != "" {
				preByPhase[pid] = preSummary{id: sid, dateModified: dm}
			}
		}
		rows.Close()
		if rerr := rows.Err(); rerr != nil {
			return rerr
		}
		if len(preByPhase) == 0 {
			t.Skip("fixture sheet has no active summaries — cannot observe a rewrite")
		}

		// Capture the sheet jobs' JOB outcome summary stamps (in-tx, pre-submit). The
		// JOB recompute (RecomputeJobInAmbientTx → ListByJob) MUST read on the AMBIENT
		// tx so it sees the phase summaries this pass writes in the SAME tx; if that
		// read fell back to the pool it would see stale/absent in-tx phase state and
		// could skip the job roll-up (codex P3 §A3 in-tx visibility).
		preJobSum := map[string]string{}
		jrows, jqerr := exec.QueryContext(txCtx, `
			SELECT jos.id, COALESCE(jos.date_modified::text, '')
			FROM job_phase jp JOIN job j ON j.id = jp.job_id
			JOIN job_outcome_summary jos ON jos.job_id = j.id AND jos.active = true
			WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3 AND jp.active = true`,
			fx.templateID, fx.phaseID, fx.workspace)
		if jqerr != nil {
			return jqerr
		}
		for jrows.Next() {
			var id, dm string
			if serr := jrows.Scan(&id, &dm); serr != nil {
				jrows.Close()
				return serr
			}
			preJobSum[id] = dm
		}
		jrows.Close()
		if jerr := jrows.Err(); jerr != nil {
			return jerr
		}

		// Drive the REAL submit.
		idCtx := fixp2bCtx(txCtx, fx.workspace)
		resp, serr := repo.SubmitJobPhaseApproval(idCtx, &pb.SubmitJobPhaseApprovalRequest{
			JobTemplateId: fx.templateID, JobTemplatePhaseId: fx.phaseID,
		})
		if serr != nil {
			return fmt.Errorf("submit failed: %w", serr)
		}
		if int(resp.GetAffectedCount()) != len(sheetPhaseIDs) {
			return fmt.Errorf("affected=%d want full sheet %d", resp.GetAffectedCount(), len(sheetPhaseIDs))
		}

		// Barrier contract: called exactly once with the FULL locked phase set.
		if recomputeCalls != 1 {
			return fmt.Errorf("recompute port called %d times, want 1", recomputeCalls)
		}
		gotP := append([]string(nil), recomputedPhases...)
		sort.Strings(gotP)
		wantP := append([]string(nil), sheetPhaseIDs...)
		sort.Strings(wantP)
		if strings.Join(gotP, ",") != strings.Join(wantP, ",") {
			return fmt.Errorf("recompute phase set %v != locked sheet %v", gotP, wantP)
		}

		// IN-TX VISIBILITY: the flip AND at least one summary rewrite are visible on
		// the ambient executor before commit.
		var advanced int
		if aerr := exec.QueryRowContext(txCtx, `
			SELECT count(*) FROM job_phase jp JOIN job j ON j.id = jp.job_id
			WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
			  AND jp.active = true AND jp.approval_status = 'PHASE_APPROVAL_STATUS_FOR_REVIEW'`,
			fx.templateID, fx.phaseID, fx.workspace).Scan(&advanced); aerr != nil {
			return aerr
		}
		if advanced != len(sheetPhaseIDs) {
			return fmt.Errorf("in-tx: %d/%d phases FOR_REVIEW", advanced, len(sheetPhaseIDs))
		}
		rewrites := 0
		for pid, pre := range preByPhase {
			var dm string
			if derr := exec.QueryRowContext(txCtx,
				`SELECT COALESCE(date_modified::text,'') FROM phase_outcome_summary WHERE id = $1`,
				pre.id).Scan(&dm); derr != nil {
				return fmt.Errorf("in-tx summary reread (phase %s): %w", pid, derr)
			}
			if dm != pre.dateModified {
				rewrites++
			}
		}
		if rewrites == 0 {
			return fmt.Errorf("in-tx: no summary rewrite visible — the recompute did not run in this transaction")
		}

		// IN-TX JOB VISIBILITY (codex P3 §A3): the JOB roll-up read the phase summaries
		// written above ON THE AMBIENT TX and (re)wrote the job_outcome_summary in THIS
		// tx. Assert at least one pre-existing job summary was rewritten in-tx (if the
		// sheet's jobs had no job summaries there is nothing to observe → tolerate).
		jobRewrites := 0
		for jid, pre := range preJobSum {
			var dm string
			if derr := exec.QueryRowContext(txCtx,
				`SELECT COALESCE(date_modified::text,'') FROM job_outcome_summary WHERE id = $1`,
				jid).Scan(&dm); derr != nil {
				return fmt.Errorf("in-tx job summary reread (%s): %w", jid, derr)
			}
			if dm != pre {
				jobRewrites++
			}
		}
		if len(preJobSum) > 0 && jobRewrites == 0 {
			return fmt.Errorf("in-tx: %d job summaries existed but none rewritten — the JOB recompute could not see the in-tx phase summaries (pool-read visibility miss)", len(preJobSum))
		}
		t.Logf("IN-TX: submit advanced %d phases; recompute rewrote %d/%d phase summaries + %d/%d job summaries inside the SAME tx (visible pre-commit)",
			advanced, rewrites, len(preByPhase), jobRewrites, len(preJobSum))
		return rollback // roll everything back — no committed mutation
	})
	if err != rollback {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}

	// POST-ROLLBACK: every stamp and status is back to its committed value (pool).
	var advanced int
	if err := db.QueryRow(`
		SELECT count(*) FROM job_phase jp JOIN job j ON j.id = jp.job_id
		WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
		  AND jp.active = true AND jp.approval_status <> 'PHASE_APPROVAL_STATUS_IN_PROGRESS'`,
		fx.templateID, fx.phaseID, fx.workspace).Scan(&advanced); err != nil {
		t.Fatalf("post-rollback status probe: %v", err)
	}
	if advanced != 0 {
		t.Fatalf("post-rollback: %d phases left advanced — rollback leaked", advanced)
	}
	for pid, pre := range preByPhase {
		var dm string
		if err := db.QueryRow(
			`SELECT COALESCE(date_modified::text,'') FROM phase_outcome_summary WHERE id = $1`,
			pre.id).Scan(&dm); err != nil {
			t.Fatalf("post-rollback summary probe (phase %s): %v", pid, err)
		}
		if dm != pre.dateModified {
			t.Fatalf("post-rollback: summary %s date_modified changed (%q -> %q) — recompute write leaked past rollback", pre.id, pre.dateModified, dm)
		}
	}
	t.Logf("POST-ROLLBACK: all %d summaries and %d statuses restored — the recompute wrote ONLY inside the rolled-back transition tx", len(preByPhase), len(sheetPhaseIDs))
}

// TestIntegration_SubmitRecompute_FailureRollsBackSubmit proves a failing recompute
// fails the submit BEFORE the status flip (nothing advances even in-tx), and the
// nil (unwired) barrier fails closed.
func TestIntegration_SubmitRecompute_FailureRollsBackSubmit(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	fx := pickSubmitFixtureSheet(t, db)

	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresJobPhaseRepository(wsOps, "job_phase").(*PostgresJobPhaseRepository)

	boom := fmt.Errorf("fixp2b: injected recompute failure")
	repo.SetSheetRecompute(func(_ context.Context, _, _ []string) error { return boom })

	rollback := fmt.Errorf("fixp2b: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec, terr := repo.txExecutor(txCtx)
		if terr != nil {
			return terr
		}
		neutralizeFreezeInTx(txCtx, t, exec, fx)

		idCtx := fixp2bCtx(txCtx, fx.workspace)
		_, serr := repo.SubmitJobPhaseApproval(idCtx, &pb.SubmitJobPhaseApprovalRequest{
			JobTemplateId: fx.templateID, JobTemplatePhaseId: fx.phaseID,
		})
		if serr == nil {
			return fmt.Errorf("submit MUST fail when the recompute fails")
		}
		if !strings.Contains(serr.Error(), "freshness-barrier recompute failed") {
			return fmt.Errorf("unexpected submit error: %v", serr)
		}
		// The barrier precedes the flip: even IN-TX nothing advanced.
		var advanced int
		if aerr := exec.QueryRowContext(txCtx, `
			SELECT count(*) FROM job_phase jp JOIN job j ON j.id = jp.job_id
			WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
			  AND jp.active = true AND jp.approval_status <> 'PHASE_APPROVAL_STATUS_IN_PROGRESS'`,
			fx.templateID, fx.phaseID, fx.workspace).Scan(&advanced); aerr != nil {
			return aerr
		}
		if advanced != 0 {
			return fmt.Errorf("in-tx: %d phases advanced despite recompute failure", advanced)
		}
		t.Logf("FAILURE PATH: submit refused (%v); 0 phases advanced in-tx", serr)

		// Unwired barrier fails closed too.
		repo.SetSheetRecompute(nil)
		_, nerr := repo.SubmitJobPhaseApproval(idCtx, &pb.SubmitJobPhaseApprovalRequest{
			JobTemplateId: fx.templateID, JobTemplatePhaseId: fx.phaseID,
		})
		if nerr == nil || !strings.Contains(nerr.Error(), "not wired") {
			return fmt.Errorf("nil barrier must fail closed, got %v", nerr)
		}
		return rollback
	})
	if err != rollback {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}
}
