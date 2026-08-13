//go:build postgresql

package principalscope

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// staffCtx builds a session context whose active binding is a STAFF principal
// (kind 7) with the given staff.id and workspace.id — the same shape the session
// middleware stamps and the ONLY source these helpers read (never a request param).
func staffCtx(staffID, wsID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		WorkspaceID:   wsID,
		PrincipalType: PrincipalTypeStaff,
		PrincipalID:   staffID,
	})
}

// nonStaffCtx builds a session context whose active binding is a non-staff
// principal (operator kind 1) — the helpers must leave such a caller's query
// unchanged.
func nonStaffCtx() context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		WorkspaceID:   "ws-1",
		PrincipalType: 1,
		PrincipalID:   "op-1",
	})
}

var placeholderRE = regexp.MustCompile(`\$(\d+)`)

// distinctPlaceholders returns the sorted set of positional placeholder indices
// ($N) referenced in a SQL fragment.
func distinctPlaceholders(sql string) []int {
	seen := map[int]bool{}
	for _, m := range placeholderRE.FindAllStringSubmatch(sql, -1) {
		n, _ := strconv.Atoi(m[1])
		seen[n] = true
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// clauseFn is an exported *Clause helper: (ctx, alias/col, nextParam) → (sql, args).
type clauseFn func(ctx context.Context, arg string, nextParam int) (string, []any)

// TestClauseThreeStateContract asserts the 3-state contract for every exported
// *Clause helper: a non-staff principal is a no-op; a staff principal with an
// empty staff.id fails closed to a zero-row predicate; a staff principal with a
// real staff.id emits a scoping predicate whose positional placeholders line up
// exactly with the returned args (starting at nextParam).
func TestClauseThreeStateContract(t *testing.T) {
	cases := []struct {
		name       string
		fn         clauseFn
		arg        string // col (StaffScopeClause) or alias (reachable clauses)
		wantWSbind bool   // reachable clauses weave in a workspace_id bound
	}{
		{"StaffScopeClause", func(ctx context.Context, col string, n int) (string, []any) {
			return StaffScopeClause(ctx, col, n)
		}, "al.staff_id", false},
		{"StaffScopeClauseAny", func(ctx context.Context, _ string, n int) (string, []any) {
			return StaffScopeClauseAny(ctx, []string{"to_.recorded_by", "to_.reviewed_by"}, n)
		}, "", false},
		{"StaffReachableClientClause", func(ctx context.Context, alias string, n int) (string, []any) {
			return StaffReachableClientClause(ctx, alias, n)
		}, "c", true},
		{"StaffReachableJobClause", func(ctx context.Context, alias string, n int) (string, []any) {
			return StaffReachableJobClause(ctx, alias, n)
		}, "j", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// (1) non-staff → unchanged query (empty clause, no args).
			if clause, args := c.fn(nonStaffCtx(), c.arg, 4); clause != "" || args != nil {
				t.Errorf("non-staff: want (\"\", nil), got (%q, %v)", clause, args)
			}

			// (2) staff + empty staff.id → fail-closed zero-row predicate, no args.
			if clause, args := c.fn(staffCtx("", "ws-1"), c.arg, 4); clause != " AND 1=0" || args != nil {
				t.Errorf("staff+empty-id: want (\" AND 1=0\", nil), got (%q, %v)", clause, args)
			}

			// (3) staff + real staff.id, at several nextParam offsets → predicate whose
			// distinct placeholders are exactly nextParam..nextParam+len(args)-1.
			for _, nextParam := range []int{1, 2, 4, 6} {
				clause, args := c.fn(staffCtx("staff-1", "ws-1"), c.arg, nextParam)
				if clause == "" {
					t.Fatalf("staff+real-id nextParam=%d: empty clause", nextParam)
				}
				if len(args) == 0 {
					t.Fatalf("staff+real-id nextParam=%d: no args", nextParam)
				}
				// args[0] is always the session staff.id.
				if args[0] != "staff-1" {
					t.Errorf("nextParam=%d: args[0] want staff-1, got %v", nextParam, args[0])
				}
				got := distinctPlaceholders(clause)
				want := make([]int, len(args))
				for i := range args {
					want[i] = nextParam + i
				}
				if fmt.Sprint(got) != fmt.Sprint(want) {
					t.Errorf("nextParam=%d: placeholder set %v != expected %v (args len %d)", nextParam, got, want, len(args))
				}

				// Workspace bound: reachable clauses must weave workspace_id in and
				// bind the session workspace as their second arg.
				if c.wantWSbind {
					if !strings.Contains(clause, "workspace_id") {
						t.Errorf("nextParam=%d: reachable clause missing workspace_id bound: %s", nextParam, clause)
					}
					if len(args) != 2 || args[1] != "ws-1" {
						t.Errorf("nextParam=%d: want args [staff-1 ws-1], got %v", nextParam, args)
					}
				}
			}
		})
	}
}

// TestReachableSQLShape asserts the standalone SQL-returning seams carry the
// workspace bound, reference their tables via registry/entityid constants (no
// bare literals were reintroduced), and expose the documented placeholder arity.
func TestReachableSQLShape(t *testing.T) {
	cases := []struct {
		name     string
		sql      string
		wantPH   []int    // exact placeholder set
		wantRefs []string // entityid table constants that must appear
	}{
		{"StaffReachableClientExistsSQL", StaffReachableClientExistsSQL(), []int{1, 2, 3},
			// The class-edge tables (sgpps + member + the v2 pps eligibility link) are
			// REQUIRED here too (CF-2): the by-id EXISTS check must carry the same 4th
			// branch as the List seam.
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan, entityid.SubscriptionGroupProductPlanStaff, entityid.SubscriptionGroupMember, entityid.ProductPlanStaff}},
		{"StaffReachableJobExistsSQL", StaffReachableJobExistsSQL(), []int{1, 2, 3},
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan, entityid.SubscriptionGroupProductPlanStaff, entityid.SubscriptionGroupMember, entityid.ProductPlanStaff}},
		{"StaffReachableClientIDsSQL", StaffReachableClientIDsSQL(), []int{1, 2},
			// v2 cutover: the class-edge tier's staff resolution now joins the
			// eligibility table too (StaffReachableClientIDsSQL delegates to
			// reachableClientUnion, which carries the 4th class-edge branch).
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan, entityid.SubscriptionGroupProductPlanStaff, entityid.ProductPlanStaff}},
		{"StaffReachableJobIDsSQL", StaffReachableJobIDsSQL(), []int{1, 2},
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan, entityid.SubscriptionGroupProductPlanStaff, entityid.ProductPlanStaff}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(c.sql, "workspace_id") {
				t.Errorf("%s: missing workspace_id bound: %s", c.name, c.sql)
			}
			if got := distinctPlaceholders(c.sql); fmt.Sprint(got) != fmt.Sprint(c.wantPH) {
				t.Errorf("%s: placeholder set %v != expected %v", c.name, got, c.wantPH)
			}
			for _, ref := range c.wantRefs {
				if !strings.Contains(c.sql, ref) {
					t.Errorf("%s: missing table reference %q", c.name, ref)
				}
			}
			// The seat tier must stay narrowed to ACTIVE seats on
			// subscription-originated jobs, matched to the job's deliverable
			// (the seat's plan product must equal the job template's output
			// product) — a widened token here widens the visibility union for
			// every staff principal.
			if !strings.Contains(c.sql, "ss.status = 'active'") || !strings.Contains(c.sql, originTypeSubscription) {
				t.Errorf("%s: seat tier missing active-seat / origin_type restriction: %s", c.name, c.sql)
			}
			if !strings.Contains(c.sql, "pl.product_id = tpl.output_product_id") {
				t.Errorf("%s: seat tier missing the plan-product/template-output match: %s", c.name, c.sql)
			}
		})
	}
}

// TestClassEdgeReachabilityBranch asserts the THIRD reachability tier — the
// class edge (subscription_group_product_plan_staff, "sgpps") — is woven into
// BOTH graph unions with the exact 4-table shape (+ the v2 eligibility LEFT
// JOIN) and is fail-closed. It runs with NO live DB: it inspects the generated
// SQL string only.
//
//   - the class-edge JOIN chain is present (sgpps → subscription_group_member →
//     product_plan → job), matched to the job's deliverable on
//     jce.output_product_id = pp.product_id (the subject match) and to the member's
//     enrollment on jce.origin_id = m.subscription_id (no job_template /
//     subscription_group / subscription hops);
//   - staff resolution is v2-native (docs/plan/20260724-section-assignment-merged
//     espyna.md §1b/M5): a LEFT JOIN to product_plan_staff (pps) resolves the
//     edge's linked eligibility row (f13), and COALESCE(pps.staff_id,
//     e.staff_id) = $1 falls back to the edge's own legacy staff_id (f10) for
//     rows that predate the M3 link-up — the fallback retires at M7;
//   - that fallback is gated by classEdgeEligibilityLive (audit M5-G5): it must
//     sit IMMEDIATELY AFTER the staff match, in the WHERE and not in the LEFT
//     JOIN condition, so a linked-but-REVOKED eligibility is denied instead of
//     falling through to the still-dual-written legacy f10 column. The needles
//     below splice the production constant in at exactly its production position,
//     so moving it breaks this test. Behavioural coverage (the truth table, the
//     naive-fix counter-example and the live no-op proof) lives in
//     class_edge_eligibility_live_test.go — these are shape assertions only;
//   - BOTH filters land on the sgpps edge — the COALESCE'd staff match = $1 AND
//     e.workspace_id = $2 (plus e.active) — so an empty staff or workspace bind
//     matches no edge and the tier yields zero rows (fail-closed); neither
//     filter is a request-param seam.
func TestClassEdgeReachabilityBranch(t *testing.T) {
	// Reference the edge/member table constants (no bare literals) so a registry
	// rename keeps this assertion honest.
	needles := []string{
		entityid.SubscriptionGroupProductPlanStaff + " e",
		entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.workspace_id = $2 AND m.active",
		entityid.ProductPlan + " pp ON pp.id = e.product_plan_id",
		"jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id AND jce.workspace_id = $2",
		entityid.ProductPlanStaff + " pps ON pps.id = e.product_plan_staff_id AND pps.workspace_id = $2", // v2 eligibility link (f13), LEFT JOIN
		// v2-linked preferred, legacy f10 fallback (empty ⇒ zero rows), with the
		// eligibility-liveness gate spliced in at its production position.
		"COALESCE(pps.staff_id, e.staff_id) = $1" + classEdgeEligibilityLive,
		"e.active",            // only active class edges reach
		"e.workspace_id = $2", // workspace filter on the edge (empty ⇒ zero rows)
	}
	unions := map[string]string{
		"reachableJobUnion":    reachableJobUnion(1, 2),
		"reachableClientUnion": reachableClientUnion(1, 2),
	}

	// Tenant defense is per edge, not merely on the outer job. Every row in the
	// reachability walk that owns workspace_id must repeat the trusted bind. A
	// malformed foreign child must never establish reachability to a local job.
	workspaceNeedles := map[string][]string{
		"reachableJobUnion": {
			"jp.workspace_id = $2", "jt.workspace_id = $2",
			"jp2.workspace_id = $2", "jt2.workspace_id = $2", "t.workspace_id = $2",
			"jw.workspace_id = $2", "jw2.workspace_id = $2", "jw3.workspace_id = $2",
			"ss.workspace_id = $2", "tpl.workspace_id = $2",
			"m.workspace_id = $2", "jce.workspace_id = $2",
			"pps.workspace_id = $2", "e.workspace_id = $2",
		},
		"reachableClientUnion": {
			"jp.workspace_id = $2", "jt.workspace_id = $2",
			"jp2.workspace_id = $2", "jt2.workspace_id = $2", "t.workspace_id = $2",
			"j.workspace_id = $2", "j2.workspace_id = $2", "j3.workspace_id = $2",
			"ss.workspace_id = $2", "tpl.workspace_id = $2",
			"m.workspace_id = $2", "jce.workspace_id = $2",
			"pps.workspace_id = $2", "e.workspace_id = $2",
		},
	}
	for name, sql := range unions {
		t.Run(name+"_workspace_edges", func(t *testing.T) {
			for _, needle := range workspaceNeedles[name] {
				if !strings.Contains(sql, needle) {
					t.Errorf("%s: tenant-scoped reachability missing %q\nSQL: %s", name, needle, sql)
				}
			}
		})
	}
	for name, sql := range unions {
		t.Run(name, func(t *testing.T) {
			for _, n := range needles {
				if !strings.Contains(sql, n) {
					t.Errorf("%s: class-edge tier missing %q\nSQL: %s", name, n, sql)
				}
			}
			// The liveness gate must NOT be pushed into the LEFT JOIN condition:
			// there it merely nulls the pps row out, and COALESCE re-grants the
			// revoked staff through the legacy f10 fallback (proven in
			// TestClassEdgeEligibilityLive_NaiveJoinFixIsInsufficient).
			if strings.Contains(sql, "pps ON pps.id = e.product_plan_staff_id AND pps.active") {
				t.Errorf("%s: eligibility-liveness moved into the LEFT JOIN condition — that form is a no-op against the legacy f10 fallback; keep it in the WHERE\nSQL: %s", name, sql)
			}
			// Adding the tier must NOT introduce a new positional placeholder: it
			// reuses the existing $1 (staff) / $2 (workspace) binds, so the union's
			// arity stays exactly {1,2}.
			if got := distinctPlaceholders(sql); fmt.Sprint(got) != fmt.Sprint([]int{1, 2}) {
				t.Errorf("%s: placeholder set %v != [1 2] after adding class edge", name, got)
			}
		})
	}
	// The tier must also ride the exported List seams (the generic dbOps.List path
	// used by ListJobs / ListClients), which delegate to the unions.
	for name, sql := range map[string]string{
		"StaffReachableJobIDsSQL":    StaffReachableJobIDsSQL(),
		"StaffReachableClientIDsSQL": StaffReachableClientIDsSQL(),
	} {
		if !strings.Contains(sql, entityid.SubscriptionGroupProductPlanStaff) {
			t.Errorf("%s: exported List seam missing the class-edge tier: %s", name, sql)
		}
	}

	// CF-2: the class-edge tier must ALSO ride the by-id EXISTS seams (ReadClient /
	// ReadJob), else a class-edge-only teacher lists a row but gets a false
	// not-found opening it. The EXISTS shape carries the target id on $2 and the
	// workspace bind on $3 (vs $1/$2 in the union), so assert the join chain + both
	// edge binds ($1 staff, $3 workspace) plus the branch's own target predicate.
	// Staff resolution mirrors the union tier: COALESCE(pps.staff_id, e.staff_id).
	existsChain := []string{
		entityid.SubscriptionGroupProductPlanStaff + " e",
		entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active",
		entityid.ProductPlan + " pp ON pp.id = e.product_plan_id",
		"jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id",
		entityid.ProductPlanStaff + " pps ON pps.id = e.product_plan_staff_id",
		"COALESCE(pps.staff_id, e.staff_id) = $1" + classEdgeEligibilityLive + " AND e.active AND e.workspace_id = $3",
	}
	for name, spec := range map[string]struct {
		sql        string
		targetPred string
	}{
		"StaffReachableClientExistsSQL": {StaffReachableClientExistsSQL(), "jce.client_id = $2"},
		"StaffReachableJobExistsSQL":    {StaffReachableJobExistsSQL(), "jce.id = $2"},
	} {
		for _, n := range append(append([]string{}, existsChain...), spec.targetPred) {
			if !strings.Contains(spec.sql, n) {
				t.Errorf("%s: class-edge EXISTS branch missing %q\nSQL: %s", name, n, spec.sql)
			}
		}
	}
}

// TestExistsSQLWorkspaceArgOrder documents the (staffID, targetID, workspaceID)
// arg order the adapter callers rely on: the workspace bound is the LAST ($3)
// placeholder, so appending workspaceID after the target id keeps the indexes
// aligned.
func TestExistsSQLWorkspaceArgOrder(t *testing.T) {
	for _, sql := range []string{StaffReachableClientExistsSQL(), StaffReachableJobExistsSQL()} {
		if !strings.Contains(sql, "$3") {
			t.Errorf("Exists SQL missing $3 workspace placeholder: %s", sql)
		}
		if strings.Contains(sql, "$4") {
			t.Errorf("Exists SQL has an unexpected $4 placeholder (arity drift): %s", sql)
		}
	}
}
