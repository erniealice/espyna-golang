package rbac

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// Review wave-2 #1 (plan 20260924-approval-role-workflow): a PARTIAL binding
// (kind without principal id, or the reverse) must fail closed to the empty set
// and never reach the legacy union query; the exact zero pair keeps the
// documented non-session backstop; a complete binding scopes the lookup.

type recordingQuery struct {
	calls []struct {
		kind int32
		id   string
	}
}

func (q *recordingQuery) GetUserPermissionCodes(_ context.Context, _, _ string, kind int32, id, _, _ string) ([]string, error) {
	q.calls = append(q.calls, struct {
		kind int32
		id   string
	}{kind, id})
	return []string{"job_phase:verify", "approval_scope:workspace"}, nil
}

func bindingCtx(kind int32, id string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: "u1", WorkspaceID: "ws1", PrincipalType: kind, PrincipalID: id,
	})
}

func TestAuthorizer_PartialBindingFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind int32
		id   string
	}{{"staff kind without id", 7, ""}, {"operator kind without id", 2, ""}, {"id without kind", 0, "staff-1"}} {
		t.Run(tc.name, func(t *testing.T) {
			q := &recordingQuery{}
			a := &PermissionAuthorizer{query: q, cache: newPermCache(), enforce: true}
			ctx := bindingCtx(tc.kind, tc.id)
			if ok, err := a.HasPermissionStrictFresh(ctx, "u1", "approval_scope:workspace"); err != nil || ok {
				t.Fatalf("fresh verdict = (%v, %v), want (false, nil)", ok, err)
			}
			if ok, err := a.HasPermissionStrict(ctx, "u1", "approval_scope:workspace"); err != nil || ok {
				t.Fatalf("cached verdict = (%v, %v), want (false, nil)", ok, err)
			}
			if len(q.calls) != 0 {
				t.Fatalf("partial binding reached the query %d time(s) — must never widen to the union", len(q.calls))
			}
		})
	}
}

func TestAuthorizer_ZeroAndCompleteBindings(t *testing.T) {
	q := &recordingQuery{}
	a := &PermissionAuthorizer{query: q, cache: newPermCache(), enforce: true}
	if ok, _ := a.HasPermissionStrictFresh(bindingCtx(0, ""), "u1", "job_phase:verify"); !ok {
		t.Fatal("exact zero pair keeps the legacy backstop")
	}
	if ok, _ := a.HasPermissionStrictFresh(bindingCtx(7, "staff-1"), "u1", "job_phase:verify"); !ok {
		t.Fatal("complete binding resolves")
	}
	if len(q.calls) != 2 || q.calls[0].kind != 0 || q.calls[1].kind != 7 || q.calls[1].id != "staff-1" {
		t.Fatalf("unexpected query calls: %+v", q.calls)
	}
}
