//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"

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

func TestAssertApprovalScope_OperatorWithWorkspaceScopePassesWithoutDB(t *testing.T) {
	for _, kind := range []int32{1, 2} {
		ctx := approvalctx.WithScopeDecision(identity.WithRequestIdentity(context.Background(),
			&identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: kind, PrincipalID: "wu-1"}),
			approvalctx.ScopeDecision{WorkspaceScope: true})
		// nil exec: an operator session must not touch the database here.
		if err := assertApprovalScope(ctx, nil, "verify", "t1", "p1", "ws1", ""); err != nil {
			t.Fatalf("kind %d operator with approval_scope:workspace should pass: %v", kind, err)
		}
	}
}

// A reviewer's operator login has no staff facet to scope by, so it must not
// approve at all (the staff login carries the reviewer edge).
func TestAssertApprovalScope_OperatorWithoutWorkspaceScopeFailsClosed(t *testing.T) {
	for _, kind := range []int32{1, 2} {
		ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: kind, PrincipalID: "wu-1"})
		for _, verb := range []string{"verify", "return", "publish"} {
			err := assertApprovalScope(ctx, nil, verb, "t1", "p1", "ws1", "")
			if err == nil {
				t.Errorf("kind %d %s without approval_scope:workspace must fail closed", kind, verb)
			} else if !errors.Is(err, portsdomain.ErrJobPhaseApprovalOperatorLogin) {
				t.Errorf("kind %d %s refusal must be ErrJobPhaseApprovalOperatorLogin, got %v", kind, verb, err)
			}
		}
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

// boolRowDriver is a one-column bool database/sql driver so the DB-backed
// assertNotSelfVerify probe can be exercised without a live database.
type boolRowDriver struct{ val bool }

func (d boolRowDriver) Open(string) (driver.Conn, error) { return boolRowConn(d), nil }

type boolRowConn struct{ val bool }

func (c boolRowConn) Prepare(string) (driver.Stmt, error) { return boolRowStmt(c), nil }
func (c boolRowConn) Close() error                        { return nil }
func (c boolRowConn) Begin() (driver.Tx, error)           { return nil, errors.New("unsupported") }

type boolRowStmt struct{ val bool }

func (s boolRowStmt) Close() error  { return nil }
func (s boolRowStmt) NumInput() int { return -1 }
func (s boolRowStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unsupported")
}
func (s boolRowStmt) Query([]driver.Value) (driver.Rows, error) { return &boolRows{val: s.val}, nil }

type boolRows struct {
	val  bool
	done bool
}

func (r *boolRows) Columns() []string { return []string{"exists"} }
func (r *boolRows) Close() error      { return nil }
func (r *boolRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.val
	return nil
}

func TestAssertNotSelfVerify_IsSelfVerify(t *testing.T) {
	sql.Register("bool_row_true_selfverify", boolRowDriver{val: true})
	db, err := sql.Open("bool_row_true_selfverify", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = assertNotSelfVerify(context.Background(), db, []lockedPhase{{id: "a"}}, "u1")
	if !errors.Is(err, portsdomain.ErrJobPhaseApprovalSelfVerify) {
		t.Fatalf("own submission must be ErrJobPhaseApprovalSelfVerify, got %v", err)
	}
}

func TestAssertApprovalScope_UnresolvedKindStaysUntyped(t *testing.T) {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: 3, PrincipalID: "p"})
	err := assertApprovalScope(ctx, nil, "verify", "t1", "p1", "ws1", "")
	if err == nil || errors.Is(err, portsdomain.ErrJobPhaseApprovalNotAllowed) {
		t.Fatalf("portal kind stays a generic fail-closed refusal, got %v", err)
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
	_, err := publishSourceState(context.Background(), forReview)
	if err == nil {
		t.Fatal("FOR_REVIEW sheet must not publish without job_phase:publish_unverified")
	}
	var se *portsdomain.JobPhaseApprovalStateError
	if !errors.As(err, &se) {
		t.Fatalf("publish refusal must carry JobPhaseApprovalStateError, got %v", err)
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

// idRowsDriver returns n single-column id rows so resolveStaffFacet's
// missing/ambiguous branches run without a live database.
type idRowsDriver struct{ n int }

func (d idRowsDriver) Open(string) (driver.Conn, error) { return idRowsConn(d), nil }

type idRowsConn struct{ n int }

func (c idRowsConn) Prepare(string) (driver.Stmt, error) { return idRowsStmt(c), nil }
func (c idRowsConn) Close() error                        { return nil }
func (c idRowsConn) Begin() (driver.Tx, error)           { return nil, errors.New("unsupported") }

type idRowsStmt struct{ n int }

func (s idRowsStmt) Close() error  { return nil }
func (s idRowsStmt) NumInput() int { return -1 }
func (s idRowsStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unsupported")
}
func (s idRowsStmt) Query([]driver.Value) (driver.Rows, error) { return &idRows{left: s.n}, nil }

type idRows struct{ left int }

func (r *idRows) Columns() []string { return []string{"id"} }
func (r *idRows) Close() error      { return nil }
func (r *idRows) Next(dest []driver.Value) error {
	if r.left == 0 {
		return io.EOF
	}
	r.left--
	dest[0] = "staff-1"
	return nil
}

// An operator login (kinds 1/2) without the Submit override that cannot be
// mapped to exactly one staff facet is refused as ErrJobPhaseApprovalOperatorLogin;
// a staff session or an unresolved/portal kind keeps the untyped refusal, and
// one matching row still resolves.
func TestApprovalResolveStaffFacet_OperatorRefusalIsTyped(t *testing.T) {
	for _, n := range []int{0, 2} {
		name := fmt.Sprintf("id_rows_%d_facet", n)
		sql.Register(name, idRowsDriver{n: n})
		db, err := sql.Open(name, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			kind     int32
			pid      string
			operator bool
		}{{1, "wu-1", true}, {2, "wu-1", true}, {7, "staff-1", false}, {0, "", false}, {3, "p", false}, {6, "p", false}} {
			ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: tc.kind, PrincipalID: tc.pid})
			_, err := resolveStaffFacet(ctx, db, "ws1")
			if err == nil {
				t.Fatalf("rows=%d kind %d: must fail closed", n, tc.kind)
			}
			if got := errors.Is(err, portsdomain.ErrJobPhaseApprovalOperatorLogin); got != tc.operator {
				t.Errorf("rows=%d kind %d: errors.Is(OperatorLogin)=%v, want %v (%v)", n, tc.kind, got, tc.operator, err)
			}
		}
		db.Close()
	}

	sql.Register("id_rows_1_facet", idRowsDriver{n: 1})
	db, err := sql.Open("id_rows_1_facet", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1", PrincipalType: 2, PrincipalID: "wu-1"})
	if facet, err := resolveStaffFacet(ctx, db, "ws1"); err != nil || facet != "staff-1" {
		t.Fatalf("one matching row must resolve: %q, %v", facet, err)
	}
}
