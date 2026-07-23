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
			// The class-edge tables (sgpps + member) are REQUIRED here too (CF-2): the
			// by-id EXISTS check must carry the same 4th branch as the List seam.
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan, entityid.SubscriptionGroupProductPlanStaff, entityid.SubscriptionGroupMember}},
		{"StaffReachableJobExistsSQL", StaffReachableJobExistsSQL(), []int{1, 2, 3},
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan, entityid.SubscriptionGroupProductPlanStaff, entityid.SubscriptionGroupMember}},
		{"StaffReachableClientIDsSQL", StaffReachableClientIDsSQL(), []int{1, 2},
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan}},
		{"StaffReachableJobIDsSQL", StaffReachableJobIDsSQL(), []int{1, 2},
			[]string{entityid.Job, entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome, entityid.SubscriptionSeat, entityid.JobTemplate, entityid.ProductPlan}},
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
// BOTH graph unions with the exact 4-table shape and is fail-closed. It runs with
// NO live DB: it inspects the generated SQL string only.
//
//   - the class-edge JOIN chain is present (sgpps → subscription_group_member →
//     product_plan → job), matched to the job's deliverable on
//     jce.output_product_id = pp.product_id (the subject match) and to the member's
//     enrollment on jce.origin_id = m.subscription_id (no job_template /
//     subscription_group / subscription hops);
//   - BOTH filters land on the sgpps edge — e.staff_id = $1 AND e.workspace_id = $2
//     (plus e.active) — so an empty staff or workspace bind matches no edge and the
//     tier yields zero rows (fail-closed); neither filter is a request-param seam.
func TestClassEdgeReachabilityBranch(t *testing.T) {
	// Reference the edge/member table constants (no bare literals) so a registry
	// rename keeps this assertion honest.
	needles := []string{
		entityid.SubscriptionGroupProductPlanStaff + " e",
		entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active",
		entityid.ProductPlan + " pp ON pp.id = e.product_plan_id",
		"jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id",
		"e.staff_id = $1",     // staff filter on the edge (empty ⇒ zero rows)
		"e.active",            // only active class edges reach
		"e.workspace_id = $2", // workspace filter on the edge (empty ⇒ zero rows)
	}
	unions := map[string]string{
		"reachableJobUnion":    reachableJobUnion(1, 2),
		"reachableClientUnion": reachableClientUnion(1, 2),
	}
	for name, sql := range unions {
		t.Run(name, func(t *testing.T) {
			for _, n := range needles {
				if !strings.Contains(sql, n) {
					t.Errorf("%s: class-edge tier missing %q\nSQL: %s", name, n, sql)
				}
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
	existsChain := []string{
		entityid.SubscriptionGroupProductPlanStaff + " e",
		entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active",
		entityid.ProductPlan + " pp ON pp.id = e.product_plan_id",
		"jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id",
		"e.staff_id = $1 AND e.active AND e.workspace_id = $3",
	}
	for name, spec := range map[string]struct {
		sql       string
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
