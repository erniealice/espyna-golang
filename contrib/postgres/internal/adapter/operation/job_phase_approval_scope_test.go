//go:build postgresql

package operation

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// Plan 20260924-approval-role-workflow D3/D4 — adapter scope axis. The DB-backed
// behaviour is proven live on an education2 clone (plan progress.md); these pin
// the SQL shape and the database-free branches.

func TestReviewerEdgeOwnedSQL_Shape(t *testing.T) {
	q := reviewerEdgeOwnedSQL(4, 3)
	for _, want := range []string{
		"rr.role = 'reviewer'",
		"rr.staff_id = $4",
		"rr.workspace_id = $3",
		"rr.active",
		"rpp.product_id = j.output_product_id",
		"rsg.status = 'current'",
		"rc.workspace_id = $3",
		"rm.workspace_id = $3",
		"rm.client_id = j.client_id",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("reviewerEdgeOwnedSQL missing %q:\n%s", want, q)
		}
	}
}

func TestSheetOutsideReviewerScopeSQL_ProbesEveryActivePhase(t *testing.T) {
	q := sheetOutsideReviewerScopeSQL("")
	for _, want := range []string{"jp.template_phase_id = $2", "j.job_template_id = $1", "j.workspace_id = $3", "jp.active = true", "AND NOT EXISTS"} {
		if !strings.Contains(q, want) {
			t.Errorf("sheetOutsideReviewerScopeSQL missing %q:\n%s", want, q)
		}
	}
}

func TestTaskUnownedProbe_ReviewerEdgeSatisfiesOwnership(t *testing.T) {
	q := taskUnownedProbeSQL("")
	if !strings.Contains(q, "rr.role = 'reviewer'") {
		t.Fatalf("submit ownership probe does not accept a reviewer edge:\n%s", q)
	}
	// The class-edge and explicit-assignment legs are still present.
	for _, want := range []string{"jt.assigned_to = $4", "e.staff_id = $4"} {
		if !strings.Contains(q, want) {
			t.Errorf("ownership probe lost %q", want)
		}
	}
}

func staffCtx() context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: "u1", WorkspaceID: "ws1", PrincipalType: 7, PrincipalID: "staff-1",
	})
}

func TestAssertApprovalScope_OperatorPassesWithoutDB(t *testing.T) {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: 2, PrincipalID: "wu-1"})
	// nil exec: an operator session must not touch the database here.
	if err := assertApprovalScope(ctx, nil, "verify", "t1", "p1", "ws1", ""); err != nil {
		t.Fatalf("operator session should pass the staff scope axis unchanged: %v", err)
	}
}

func TestAssertApprovalScope_NonOperatorNonStaffFailsClosed(t *testing.T) {
	for _, kind := range []int32{0, 3, 4, 5, 6} {
		ctx := approvalctx.WithScopeDecision(identity.WithRequestIdentity(context.Background(),
			&identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: kind, PrincipalID: "p"}),
			approvalctx.ScopeDecision{WorkspaceScope: true})
		if err := assertApprovalScope(ctx, nil, "verify", "t1", "p1", "ws1", ""); err == nil {
			t.Errorf("kind %d must fail closed even with a workspace-scope decision", kind)
		}
	}
}

func TestAssertApprovalScope_StaffWorkspaceScopePassesWithoutDB(t *testing.T) {
	ctx := approvalctx.WithScopeDecision(staffCtx(), approvalctx.ScopeDecision{WorkspaceScope: true})
	if err := assertApprovalScope(ctx, nil, "publish", "t1", "p1", "ws1", ""); err != nil {
		t.Fatalf("staff with approval_scope:workspace should pass: %v", err)
	}
}

func TestAssertApprovalScope_StaffWithoutScopeNeedsFacet(t *testing.T) {
	// No workspace scope → the facet must be re-proven; with no executor that
	// path cannot run, so the call must not silently pass. resolveStaffFacet
	// queries the executor, so a nil exec would panic — use a malformed staff
	// session (empty principal id) that fails closed before any query.
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: 7})
	if err := assertApprovalScope(ctx, nil, "return", "t1", "p1", "ws1", ""); err == nil {
		t.Fatal("malformed staff session without workspace scope must fail closed")
	}
}

func TestPublishSourceState(t *testing.T) {
	verified := []lockedPhase{{id: "a", approvalStatus: apVerified}, {id: "b", approvalStatus: apVerified}}
	forReview := []lockedPhase{{id: "a", approvalStatus: apForReview}, {id: "b", approvalStatus: apForReview}}
	mixed := []lockedPhase{{id: "a", approvalStatus: apForReview}, {id: "b", approvalStatus: apVerified}}
	skip := approvalctx.WithScopeDecision(context.Background(), approvalctx.ScopeDecision{PublishUnverified: true})

	if got, err := publishSourceState(context.Background(), verified); err != nil || got != apVerified {
		t.Fatalf("verified sheet: got %q, %v", got, err)
	}
	if _, err := publishSourceState(context.Background(), forReview); err == nil {
		t.Fatal("FOR_REVIEW sheet must not publish without job_phase:publish_unverified")
	}
	if got, err := publishSourceState(skip, forReview); err != nil || got != apForReview {
		t.Fatalf("FOR_REVIEW with publish_unverified: got %q, %v", got, err)
	}
	if _, err := publishSourceState(skip, mixed); err == nil {
		t.Fatal("a mixed sheet must never publish, even with publish_unverified")
	}
}

func TestAssertNotSelfVerify_VerifyOwnSkipsProbe(t *testing.T) {
	ctx := approvalctx.WithScopeDecision(context.Background(), approvalctx.ScopeDecision{VerifyOwn: true})
	if err := assertNotSelfVerify(ctx, nil, []lockedPhase{{id: "a"}}, "u1"); err != nil {
		t.Fatalf("verify_own holder must skip the separation-of-duties probe: %v", err)
	}
	if err := assertNotSelfVerify(context.Background(), nil, []lockedPhase{{id: "a"}}, ""); err == nil {
		t.Fatal("missing actor must fail closed")
	}
}

func TestAssertApprovalScope_MalformedStaffIgnoresWorkspaceScope(t *testing.T) {
	ctx := approvalctx.WithScopeDecision(identity.WithRequestIdentity(context.Background(),
		&identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: 7}),
		approvalctx.ScopeDecision{WorkspaceScope: true})
	if err := assertApprovalScope(ctx, nil, "publish", "t1", "p1", "ws1", ""); err == nil {
		t.Fatal("a malformed staff session must fail closed even with a workspace-scope decision")
	}
}
