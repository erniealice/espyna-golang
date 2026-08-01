//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

// Gate-rollup read — the ROW-SET half of the contract (plan 20260729 Phase 2),
// gated on TEST_DATABASE_URL per repo convention. Fixtures are committed rows
// namespaced with the pagrtest- prefix and removed by pagrtestCleanup (the
// ocitest- pattern): the read under test runs on the adapter's own *sql.DB
// connections, so in-tx fixtures would be invisible to it.
//
// The scenarios pin the locked design's row-set guarantees:
//
//	1. cross-group independence (one template phase, two groups, opposite states)
//	2. same-group mixed sheet (the entered member and the data-bearing member differ)
//	3. foreign-workspace group ⇒ phase ABSENT, never an error (no existence oracle)
//	4. inactive membership row ⇒ member excluded (active=true pin)
//	5. subscription_id = j.origin_id pin (the anti-cells-predicate case)
//	6. Go↔SQL any_workflow_entered parity (fayna phaseWorkflowEntered truth table)
//	7. >100 members in one group ⇒ exact target_count (no page cap — H-1 absence)

const (
	pagrtestWS  = "pagrtest-ws"
	pagrtestWS2 = "pagrtest-ws2"
)

func openGateRollupDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL unusable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	var ok sql.NullString
	if err := db.QueryRowContext(ctx,
		"SELECT to_regclass('public.subscription_group_member')::text").Scan(&ok); err != nil || !ok.Valid {
		db.Close()
		t.Skip("subscription_group_member not present (education grading wave migrations required)")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// pagrtestCleanup removes every committed pagrtest- row, children first.
func pagrtestCleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`DELETE FROM task_outcome WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM job_task WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM job_phase WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM job_template_phase WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM job WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM subscription_group_member WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM subscription_group WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM subscription WHERE id LIKE 'pagrtest-%'`,
		`DELETE FROM workspace WHERE id LIKE 'pagrtest-%'`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("pagrtest cleanup %q: %v", stmt, err)
		}
	}
}

// pagrtestSeed builds the whole fixture graph. Committed (see file comment) and
// removed by pagrtestCleanup, which also runs FIRST so a previously aborted run
// cannot poison this one.
//
// job.client_id / job.origin_id / sgm.client_id carry no FK in the schema, so
// clients are bare ids; subscription_group_member does FK subscription_group,
// subscription and workspace, so those parents are real rows — and
// job_phase.template_phase_id FKs job_template_phase on both integration DBs
// (job_phase_template_phase_id_fkey), so the tp parents are real rows too.
func pagrtestSeed(t *testing.T, db *sql.DB) {
	t.Helper()
	pagrtestCleanup(t, db)
	t.Cleanup(func() { pagrtestCleanup(t, db) })

	stmts := []string{
		`INSERT INTO workspace (id) VALUES ('pagrtest-ws'), ('pagrtest-ws2')`,

		`INSERT INTO subscription_group (id, name, kind, capacity_mode, active, workspace_id) VALUES
			('pagrtest-grp-a',   'pagrtest A',        'pagrtest', 'unlimited', true, 'pagrtest-ws'),
			('pagrtest-grp-b',   'pagrtest B',        'pagrtest', 'unlimited', true, 'pagrtest-ws'),
			('pagrtest-grp-f',   'pagrtest foreign',  'pagrtest', 'unlimited', true, 'pagrtest-ws2'),
			('pagrtest-grp-x',   'pagrtest crossay',  'pagrtest', 'unlimited', true, 'pagrtest-ws'),
			('pagrtest-grp-i',   'pagrtest inactive', 'pagrtest', 'unlimited', true, 'pagrtest-ws'),
			('pagrtest-grp-big', 'pagrtest big',      'pagrtest', 'unlimited', true, 'pagrtest-ws')`,

		`INSERT INTO subscription (id, active, workspace_id) VALUES
			('pagrtest-sub-a1', true, 'pagrtest-ws'),
			('pagrtest-sub-a2', true, 'pagrtest-ws'),
			('pagrtest-sub-b1', true, 'pagrtest-ws'),
			('pagrtest-sub-f1', true, 'pagrtest-ws2'),
			('pagrtest-sub-x1', true, 'pagrtest-ws')`,

		// Memberships. grp-x holds the SAME client as grp-a's first member but
		// through a DIFFERENT subscription (sub-x1 ≠ the job's origin sub-a1) —
		// the foreign-AY shape the origin pin must reject. grp-i's sole row is
		// INACTIVE for the same (client, subscription) pair the sheet uses.
		`INSERT INTO subscription_group_member (id, subscription_group_id, subscription_id, client_id, active, workspace_id) VALUES
			('pagrtest-sgm-a1', 'pagrtest-grp-a', 'pagrtest-sub-a1', 'pagrtest-c-a1', true,  'pagrtest-ws'),
			('pagrtest-sgm-a2', 'pagrtest-grp-a', 'pagrtest-sub-a2', 'pagrtest-c-a2', true,  'pagrtest-ws'),
			('pagrtest-sgm-b1', 'pagrtest-grp-b', 'pagrtest-sub-b1', 'pagrtest-c-b1', true,  'pagrtest-ws'),
			('pagrtest-sgm-f1', 'pagrtest-grp-f', 'pagrtest-sub-f1', 'pagrtest-c-a1', true,  'pagrtest-ws2'),
			('pagrtest-sgm-x1', 'pagrtest-grp-x', 'pagrtest-sub-x1', 'pagrtest-c-a1', true,  'pagrtest-ws'),
			('pagrtest-sgm-i1', 'pagrtest-grp-i', 'pagrtest-sub-a1', 'pagrtest-c-a1', false, 'pagrtest-ws')`,

		`INSERT INTO job (id, client_id, origin_id, workspace_id, active) VALUES
			('pagrtest-job-a1',  'pagrtest-c-a1', 'pagrtest-sub-a1', 'pagrtest-ws', true),
			('pagrtest-job-a1m', 'pagrtest-c-a1', 'pagrtest-sub-a1', 'pagrtest-ws', true),
			('pagrtest-job-a2',  'pagrtest-c-a2', 'pagrtest-sub-a2', 'pagrtest-ws', true),
			('pagrtest-job-b1',  'pagrtest-c-b1', 'pagrtest-sub-b1', 'pagrtest-ws', true)`,

		// Template-phase parents for the FK (id-only rows; every other column
		// is nullable/defaulted). Children-first cleanup removes job_phase
		// before these.
		`INSERT INTO job_template_phase (id) VALUES
			('pagrtest-tp-1'), ('pagrtest-tp-2'), ('pagrtest-tp-big')`,

		// tp-1 (cross-group): group A's member is FOR_REVIEW with data; group
		// B's is a pristine never-workflowed IN_PROGRESS.
		// tp-2 (same-group mixed, group A): one PUBLISHED member with data, one
		// pristine IN_PROGRESS member — the workflow-entered member and the
		// data-bearing member are the SAME here, so the outcome hangs on the
		// aggregate, and the pristine member drags all_published to false.
		// Audit stamp pairs are set actor+time together (schema CHECK).
		`INSERT INTO job_phase (id, job_id, active, template_phase_id, approval_status, submitted_by, submitted_at, verified_by, verified_at, published_by, published_at) VALUES
			('pagrtest-ph-a1',  'pagrtest-job-a1',  true, 'pagrtest-tp-1', 'PHASE_APPROVAL_STATUS_FOR_REVIEW',
			 'pagrtest-staff', 1, NULL, NULL, NULL, NULL),
			('pagrtest-ph-b1',  'pagrtest-job-b1',  true, 'pagrtest-tp-1', 'PHASE_APPROVAL_STATUS_IN_PROGRESS',
			 NULL, NULL, NULL, NULL, NULL, NULL),
			('pagrtest-ph-a1m', 'pagrtest-job-a1m', true, 'pagrtest-tp-2', 'PHASE_APPROVAL_STATUS_PUBLISHED',
			 'pagrtest-staff', 1, 'pagrtest-staff', 2, 'pagrtest-staff', 3),
			('pagrtest-ph-a2',  'pagrtest-job-a2',  true, 'pagrtest-tp-2', 'PHASE_APPROVAL_STATUS_IN_PROGRESS',
			 NULL, NULL, NULL, NULL, NULL, NULL)`,

		`INSERT INTO job_task (id, job_phase_id, active) VALUES
			('pagrtest-task-a1',  'pagrtest-ph-a1',  true),
			('pagrtest-task-a1m', 'pagrtest-ph-a1m', true)`,

		`INSERT INTO task_outcome (id, job_task_id, active) VALUES
			('pagrtest-out-a1',  'pagrtest-task-a1',  true),
			('pagrtest-out-a1m', 'pagrtest-task-a1m', true)`,

		// The >100 sheet: 120 members, 120 jobs, 120 pristine phases on tp-big.
		`INSERT INTO subscription (id, active, workspace_id)
			SELECT 'pagrtest-sub-big-'||g, true, 'pagrtest-ws' FROM generate_series(1,120) g`,
		`INSERT INTO subscription_group_member (id, subscription_group_id, subscription_id, client_id, active, workspace_id)
			SELECT 'pagrtest-sgm-big-'||g, 'pagrtest-grp-big', 'pagrtest-sub-big-'||g, 'pagrtest-c-big-'||g, true, 'pagrtest-ws'
			FROM generate_series(1,120) g`,
		`INSERT INTO job (id, client_id, origin_id, workspace_id, active)
			SELECT 'pagrtest-job-big-'||g, 'pagrtest-c-big-'||g, 'pagrtest-sub-big-'||g, 'pagrtest-ws', true
			FROM generate_series(1,120) g`,
		`INSERT INTO job_phase (id, job_id, active, template_phase_id, approval_status)
			SELECT 'pagrtest-ph-big-'||g, 'pagrtest-job-big-'||g, true, 'pagrtest-tp-big', 'PHASE_APPROVAL_STATUS_IN_PROGRESS'
			FROM generate_series(1,120) g`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("pagrtest seed:\n%s\n%v", stmt, err)
		}
	}
}

func gateRollupRead(t *testing.T, db *sql.DB, ws, groupID string, tpIDs ...string) *matrixpb.GetPhaseApprovalGateRollupResponse {
	t.Helper()
	a := &PostgresOutcomeMatrixQuery{db: db}
	resp, err := a.GetPhaseApprovalGateRollup(wsCtx(ws), &matrixpb.GetPhaseApprovalGateRollupRequest{
		SubscriptionGroupId: groupID,
		JobTemplatePhaseIds: tpIDs,
	})
	if err != nil {
		t.Fatalf("gate rollup read (%s / %v): %v", groupID, tpIDs, err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("gate rollup read (%s / %v): success=false", groupID, tpIDs)
	}
	return resp
}

func rollupByPhase(resp *matrixpb.GetPhaseApprovalGateRollupResponse) map[string]*matrixpb.PhaseApprovalGateRollup {
	out := map[string]*matrixpb.PhaseApprovalGateRollup{}
	for _, r := range resp.GetRollups() {
		out[r.GetJobTemplatePhaseId()] = r
	}
	return out
}

// TestIntegration_GateRollup_CrossGroupIndependence: the SAME template phase,
// read through group A then group B, must describe two independent sheets — A's
// review state and data never bleed into B's pristine one.
func TestIntegration_GateRollup_CrossGroupIndependence(t *testing.T) {
	db := openGateRollupDB(t)
	pagrtestSeed(t, db)

	a := rollupByPhase(gateRollupRead(t, db, pagrtestWS, "pagrtest-grp-a", "pagrtest-tp-1"))["pagrtest-tp-1"]
	if a == nil {
		t.Fatal("group A must produce a tp-1 rollup")
	}
	if a.GetAppliedSubscriptionGroupId() != "pagrtest-grp-a" {
		t.Errorf("A echo = %q", a.GetAppliedSubscriptionGroupId())
	}
	if a.GetTargetCount() != 1 || !a.GetAnyWorkflowEntered() || a.GetAllPublished() || !a.GetHasData() {
		t.Errorf("A rollup = {count:%d entered:%v allpub:%v data:%v}, want {1 true false true}",
			a.GetTargetCount(), a.GetAnyWorkflowEntered(), a.GetAllPublished(), a.GetHasData())
	}

	b := rollupByPhase(gateRollupRead(t, db, pagrtestWS, "pagrtest-grp-b", "pagrtest-tp-1"))["pagrtest-tp-1"]
	if b == nil {
		t.Fatal("group B must produce a tp-1 rollup")
	}
	if b.GetAppliedSubscriptionGroupId() != "pagrtest-grp-b" {
		t.Errorf("B echo = %q", b.GetAppliedSubscriptionGroupId())
	}
	if b.GetTargetCount() != 1 || b.GetAnyWorkflowEntered() || b.GetAllPublished() || b.GetHasData() {
		t.Errorf("B rollup = {count:%d entered:%v allpub:%v data:%v}, want {1 false false false}",
			b.GetTargetCount(), b.GetAnyWorkflowEntered(), b.GetAllPublished(), b.GetHasData())
	}
}

// TestIntegration_GateRollup_SameGroupMixedSheet: one group, two members on the
// same template phase — PUBLISHED-with-data plus pristine-IN_PROGRESS must roll
// up to {entered:true, all_published:false, has_data:true}: the sheet entered
// the workflow, is not uniformly published, and carries data.
func TestIntegration_GateRollup_SameGroupMixedSheet(t *testing.T) {
	db := openGateRollupDB(t)
	pagrtestSeed(t, db)

	r := rollupByPhase(gateRollupRead(t, db, pagrtestWS, "pagrtest-grp-a", "pagrtest-tp-2"))["pagrtest-tp-2"]
	if r == nil {
		t.Fatal("group A must produce a tp-2 rollup")
	}
	if r.GetTargetCount() != 2 || !r.GetAnyWorkflowEntered() || r.GetAllPublished() || !r.GetHasData() {
		t.Errorf("mixed rollup = {count:%d entered:%v allpub:%v data:%v}, want {2 true false true}",
			r.GetTargetCount(), r.GetAnyWorkflowEntered(), r.GetAllPublished(), r.GetHasData())
	}
}

// TestIntegration_GateRollup_ForeignWorkspaceGroupIsAbsentNotAnError: a group id
// owned by ANOTHER workspace yields zero rollups under success — the phase is
// simply absent (the consumer's coverage check owns that), and the response
// leaks neither an error nor the foreign group's existence.
func TestIntegration_GateRollup_ForeignWorkspaceGroupIsAbsentNotAnError(t *testing.T) {
	db := openGateRollupDB(t)
	pagrtestSeed(t, db)

	resp := gateRollupRead(t, db, pagrtestWS, "pagrtest-grp-f", "pagrtest-tp-1")
	if n := len(resp.GetRollups()); n != 0 {
		t.Fatalf("foreign-workspace group must produce NO rollups, got %d: %v", n, resp.GetRollups())
	}
}

// TestIntegration_GateRollup_InactiveMembershipExcluded: the sheet job's
// (client, subscription) pair IS a row of grp-i — but inactive. The active=true
// pin must exclude it, leaving the phase absent.
func TestIntegration_GateRollup_InactiveMembershipExcluded(t *testing.T) {
	db := openGateRollupDB(t)
	pagrtestSeed(t, db)

	resp := gateRollupRead(t, db, pagrtestWS, "pagrtest-grp-i", "pagrtest-tp-1")
	if n := len(resp.GetRollups()); n != 0 {
		t.Fatalf("inactive-membership group must produce NO rollups, got %d: %v", n, resp.GetRollups())
	}
}

// TestIntegration_GateRollup_OriginPinRejectsForeignSubscriptionMembership is
// the anti-cells-predicate case: grp-x holds the sheet job's CLIENT through a
// DIFFERENT subscription. The looser `client_id IN (...)` membership shape
// would admit the job; the shared predicate's subscription_id = j.origin_id pin
// must not.
func TestIntegration_GateRollup_OriginPinRejectsForeignSubscriptionMembership(t *testing.T) {
	db := openGateRollupDB(t)
	pagrtestSeed(t, db)

	resp := gateRollupRead(t, db, pagrtestWS, "pagrtest-grp-x", "pagrtest-tp-1")
	if n := len(resp.GetRollups()); n != 0 {
		t.Fatalf("foreign-subscription membership must NOT admit the job (origin pin), got %d rollups: %v", n, resp.GetRollups())
	}
}

// TestIntegration_GateRollup_Over100MembersExactCount: 120 members in one group
// must roll up to target_count 120 exactly — the aggregate reads the whole
// narrowed relation, so no 100-row page cap can shrink the sheet (H-1 absent by
// construction, pinned).
func TestIntegration_GateRollup_Over100MembersExactCount(t *testing.T) {
	db := openGateRollupDB(t)
	pagrtestSeed(t, db)

	r := rollupByPhase(gateRollupRead(t, db, pagrtestWS, "pagrtest-grp-big", "pagrtest-tp-big"))["pagrtest-tp-big"]
	if r == nil {
		t.Fatal("big group must produce a tp-big rollup")
	}
	if r.GetTargetCount() != 120 {
		t.Fatalf("target_count = %d, want the EXACT member count 120 (page-cap regression)", r.GetTargetCount())
	}
	if r.GetAnyWorkflowEntered() || r.GetHasData() {
		t.Errorf("pristine big sheet must be {entered:false data:false}, got {%v %v}",
			r.GetAnyWorkflowEntered(), r.GetHasData())
	}
}

// TestIntegration_GateRollup_AnyWorkflowEnteredSQLParity evaluates the EXACT
// shipped SQL expression (gateAnyWorkflowEnteredSQLExpr — the shape test pins
// that the status statement embeds it verbatim) against the consumer-side truth
// table (fayna phaseWorkflowEntered): status beyond IN_PROGRESS/UNSPECIFIED ⇒
// entered; otherwise entered iff ANY audit stamp is non-empty — including the
// returned-only shape (IN_PROGRESS with only returned_by set). NULL and ”
// stamps are equivalent (protojson zero-omission persists unset as NULL).
func TestIntegration_GateRollup_AnyWorkflowEnteredSQLParity(t *testing.T) {
	db := openGateRollupDB(t)

	const q = `SELECT BOOL_OR(
         ` + gateAnyWorkflowEnteredSQLExpr + `
       ) FROM (VALUES ($1::text, $2::text, $3::text, $4::text, $5::text))
         AS jp(approval_status, submitted_by, verified_by, published_by, returned_by)`

	// goEntered mirrors fayna's phaseWorkflowEntered over the persisted string
	// domain (the DB stores the enum NAME; absent ⇒ zero value UNSPECIFIED).
	goEntered := func(status string, stamps [4]*string) bool {
		switch status {
		case "PHASE_APPROVAL_STATUS_UNSPECIFIED", "PHASE_APPROVAL_STATUS_IN_PROGRESS":
			for _, s := range stamps {
				if s != nil && *s != "" {
					return true
				}
			}
			return false
		default:
			return true
		}
	}

	set := func(v string) *string { return &v }
	empty := set("")
	statuses := []string{
		"PHASE_APPROVAL_STATUS_UNSPECIFIED",
		"PHASE_APPROVAL_STATUS_IN_PROGRESS",
		"PHASE_APPROVAL_STATUS_FOR_REVIEW",
		"PHASE_APPROVAL_STATUS_VERIFIED",
		"PHASE_APPROVAL_STATUS_PUBLISHED",
	}
	stampCases := map[string][4]*string{
		"all NULL":         {nil, nil, nil, nil},
		"all empty":        {empty, empty, empty, empty},
		"submitted only":   {set("u1"), nil, nil, nil},
		"verified only":    {nil, set("u1"), nil, nil},
		"published only":   {nil, nil, set("u1"), nil},
		"returned only":    {nil, nil, nil, set("u1")}, // the carve-out's load-bearing row
		"all four stamped": {set("u1"), set("u2"), set("u3"), set("u4")},
	}

	for _, status := range statuses {
		for name, stamps := range stampCases {
			var got bool
			if err := db.QueryRow(q, status, stamps[0], stamps[1], stamps[2], stamps[3]).Scan(&got); err != nil {
				t.Fatalf("(%s / %s): %v", status, name, err)
			}
			if want := goEntered(status, stamps); got != want {
				t.Errorf("parity break at (%s / %s): SQL=%v Go=%v", status, name, got, want)
			}
		}
	}

	// The empty-string corner, pinned SEPARATELY because it is a KNOWN,
	// ACCEPTED divergence, not parity: SQL counts '' as workflow-entered
	// ('' NOT IN (IN_PROGRESS, UNSPECIFIED) is TRUE) while fayna's enum parse
	// maps '' → UNSPECIFIED → stamp-driven. Reachability is remote — the
	// pre-20260719 baseline column default was '' and the migration's backfill
	// eliminated those rows — and the divergence direction is over-BLOCK only
	// (409 where the enum table would render), never render. This pin keeps a
	// predicate rewrite from silently flipping the corner fail-open.
	for name, stamps := range stampCases {
		var got bool
		if err := db.QueryRow(q, "", stamps[0], stamps[1], stamps[2], stamps[3]).Scan(&got); err != nil {
			t.Fatalf("('' / %s): %v", name, err)
		}
		if !got {
			t.Errorf("empty-string status (%s): SQL must count '' as workflow-entered (fail-safe over-block), got false", name)
		}
	}
}
