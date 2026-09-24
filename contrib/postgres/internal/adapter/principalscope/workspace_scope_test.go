//go:build postgresql

package principalscope

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// Plan 20260924-approval-role-workflow D3: approval_scope:workspace lifts STAFF
// row scoping (via the request marker); the reviewer tier widens reachability.

func staffIdentityCtx(staffID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: "u1", WorkspaceID: "ws1", PrincipalType: PrincipalTypeStaff, PrincipalID: staffID,
	})
}

func TestStaffRowScope_WorkspaceMarkerLiftsRowScope(t *testing.T) {
	ctx := staffIdentityCtx("staff-1")
	if id, applies := StaffRowScope(ctx); !applies || id != "staff-1" {
		t.Fatalf("unmarked staff session must be row-scoped: (%q, %v)", id, applies)
	}
	marked := identity.WithWorkspaceRowScope(ctx)
	if _, applies := StaffRowScope(marked); applies {
		t.Fatal("marked staff session must read the whole workspace (applies=false)")
	}
	// The clause helpers follow StaffRowScope: unchanged query for the marked session.
	if clause, args := StaffReachableJobClause(marked, "j", 1); clause != "" || args != nil {
		t.Fatalf("marked session got a job clause %q %v", clause, args)
	}
	// ActingStaff still reports who is acting.
	if id, isStaff := ActingStaff(marked); !isStaff || id != "staff-1" {
		t.Fatalf("ActingStaff lost the acting staff: (%q, %v)", id, isStaff)
	}
}

func TestStaffRowScope_MarkerNeverLiftsMalformedStaff(t *testing.T) {
	// A marked STAFF session with an empty principal id stays fail-closed: the
	// marker never lifts scope for an unresolved staff identity.
	ctx := identity.WithWorkspaceRowScope(staffIdentityCtx(""))
	if id, applies := StaffRowScope(ctx); !applies || id != "" {
		t.Fatalf("StaffRowScope = (%q, %v), want (\"\", true) — zero rows", id, applies)
	}
	if clause, _ := StaffReachableJobClause(ctx, "j", 1); clause != " AND 1=0" {
		t.Fatalf("malformed marked staff must get the always-false clause, got %q", clause)
	}
	if WorkspaceWideStaff(ctx) {
		t.Fatal("malformed staff is never workspace-wide")
	}
}

func TestIsOperatorSession(t *testing.T) {
	for kind, want := range map[int32]bool{0: false, 1: true, 2: true, 3: false, 4: false, 5: false, 6: false, 7: false} {
		ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: kind, PrincipalID: "p"})
		if got := IsOperatorSession(ctx); got != want {
			t.Errorf("kind %d: IsOperatorSession = %v, want %v", kind, got, want)
		}
	}
}

func TestReachability_IncludesReviewerTier(t *testing.T) {
	for name, q := range map[string]string{
		"reachableJobUnion":             reachableJobUnion(1, 2),
		"reachableClientUnion":          reachableClientUnion(1, 2),
		"StaffReachableJobExistsSQL":    StaffReachableJobExistsSQL(),
		"StaffReachableClientExistsSQL": StaffReachableClientExistsSQL(),
	} {
		if !strings.Contains(q, "r.role = '"+ProductPlanStaffRoleReviewer+"'") || !strings.Contains(q, "r.active") {
			t.Errorf("%s lacks the reviewer tier:\n%s", name, q)
		}
	}
	job := reachableJobUnion(1, 2)
	for _, want := range []string{"r.staff_id = $1", "r.workspace_id = $2", "rc.workspace_id = $2", "rm.workspace_id = $2", "jr.workspace_id = $2", "jr.output_product_id = rpp.product_id"} {
		if !strings.Contains(job, want) {
			t.Errorf("reviewer tier missing bind %q", want)
		}
	}
}

func TestWorkspaceWideStaff(t *testing.T) {
	staff := staffIdentityCtx("staff-1")
	if WorkspaceWideStaff(staff) {
		t.Fatal("unmarked staff is not workspace-wide")
	}
	if !WorkspaceWideStaff(identity.WithWorkspaceRowScope(staff)) {
		t.Fatal("marked staff is workspace-wide")
	}
	operator := identity.WithWorkspaceRowScope(identity.WithRequestIdentity(context.Background(),
		&identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: 2, PrincipalID: "wu-1"}))
	if WorkspaceWideStaff(operator) {
		t.Fatal("operators never take the staff workspace-wide path (they use the authorized ALL widen)")
	}
}
