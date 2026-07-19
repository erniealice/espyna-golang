package job_list_tab_support

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// fakeAuthorizer is a minimal RBAC port for the gate tests. IsEnabled() is true
// so Check actually consults HasPermission (a disabled authorizer would allow
// everything and defeat the point). allowed is the set of permission codes the
// principal holds.
type fakeAuthorizer struct{ allowed map[string]bool }

func (f *fakeAuthorizer) IsEnabled() bool { return true }
func (f *fakeAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.allowed[permission], nil
}

// fakePort records the (gated) request the use case forwards and returns an empty
// response. It implements the hand-written ports.JobListTabSupportQueryService
// (no proto Unimplemented marker — it is a plain Go interface).
type fakePort struct {
	called bool
	gotReq *ports.JobListTabSupportRequest
	err    error
}

func (p *fakePort) ListJobListTabSupport(_ context.Context, req *ports.JobListTabSupportRequest) (*ports.JobListTabSupportResponse, error) {
	p.called = true
	p.gotReq = req
	if p.err != nil {
		return nil, p.err
	}
	return &ports.JobListTabSupportResponse{}, nil
}

func gate(allowed ...string) *actiongate.ActionGatekeeper {
	set := map[string]bool{}
	for _, p := range allowed {
		set[p] = true
	}
	return actiongate.NewActionGatekeeper(&fakeAuthorizer{allowed: set}, nil)
}

func perm(entity string) string { return entityid.EntityPermission(entity, entityid.ActionList) }

func userCtx() context.Context {
	return contextutil.WithUserID(context.Background(), "user-1")
}

func newUC(port query, g *actiongate.ActionGatekeeper) *ListJobListTabSupportUseCase {
	return NewUseCases(Repositories{Query: port}, Services{ActionGatekeeper: g}).ListJobListTabSupport
}

// TestBothKindsPermitted — both gates pass → the port is called with BOTH include
// flags set (the steady 1-statement case: one UNION-ALL branch per kind).
func TestBothKindsPermitted(t *testing.T) {
	port := &fakePort{}
	uc := newUC(port, gate(perm(entityid.JobCategory), perm(entityid.JobTemplate)))
	if _, err := uc.Execute(userCtx()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !port.called {
		t.Fatalf("port must be called when >=1 kind is permitted")
	}
	if !port.gotReq.IncludeCategories || !port.gotReq.IncludeTemplates {
		t.Errorf("both kinds permitted must set both flags, got %+v", port.gotReq)
	}
}

// TestCategoryDenied — job_category:list denied, job_template:list granted → the
// category branch is omitted (IncludeCategories=false), the template branch runs.
func TestCategoryDenied(t *testing.T) {
	port := &fakePort{}
	uc := newUC(port, gate(perm(entityid.JobTemplate))) // no job_category:list
	if _, err := uc.Execute(userCtx()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !port.called {
		t.Fatalf("port must be called when the template kind is permitted")
	}
	if port.gotReq.IncludeCategories {
		t.Errorf("category kind is denied — IncludeCategories must be false")
	}
	if !port.gotReq.IncludeTemplates {
		t.Errorf("template kind is granted — IncludeTemplates must be true")
	}
}

// TestTemplateDenied — job_template:list denied, job_category:list granted → the
// template branch is omitted (IncludeTemplates=false), the category branch runs.
func TestTemplateDenied(t *testing.T) {
	port := &fakePort{}
	uc := newUC(port, gate(perm(entityid.JobCategory))) // no job_template:list
	if _, err := uc.Execute(userCtx()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !port.called {
		t.Fatalf("port must be called when the category kind is permitted")
	}
	if !port.gotReq.IncludeCategories {
		t.Errorf("category kind is granted — IncludeCategories must be true")
	}
	if port.gotReq.IncludeTemplates {
		t.Errorf("template kind is denied — IncludeTemplates must be false")
	}
}

// TestBothKindsDenied — neither gate passes → empty response and the port is
// NEVER called (no SQL). Fail-closed: the same degradation as the former
// component-level log-and-empty for both reads.
func TestBothKindsDenied(t *testing.T) {
	port := &fakePort{}
	uc := newUC(port, gate()) // grants nothing
	resp, err := uc.Execute(userCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.called {
		t.Errorf("both kinds denied — the port must not be called")
	}
	if resp == nil || len(resp.Categories) != 0 || len(resp.Templates) != 0 {
		t.Errorf("both kinds denied must yield an empty response, got %+v", resp)
	}
}

// TestNilGatekeeperDeniesAll — a nil gatekeeper fails closed: both kinds omitted,
// the port is never called (Check has a nil-receiver DENY guard).
func TestNilGatekeeperDeniesAll(t *testing.T) {
	port := &fakePort{}
	uc := newUC(port, nil)
	resp, err := uc.Execute(userCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.called {
		t.Errorf("nil gatekeeper must not call the port")
	}
	if resp == nil || len(resp.Categories) != 0 || len(resp.Templates) != 0 {
		t.Errorf("nil gatekeeper must yield an empty response, got %+v", resp)
	}
}

// TestNilPortDegradesGracefully — a permitted read with no registered adapter
// port (mock / non-postgres) returns an empty response, no panic.
func TestNilPortDegradesGracefully(t *testing.T) {
	uc := newUC(nil, gate(perm(entityid.JobCategory), perm(entityid.JobTemplate)))
	resp, err := uc.Execute(userCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Categories) != 0 || len(resp.Templates) != 0 {
		t.Errorf("nil port should yield an empty response, got %+v", resp)
	}
}

// TestQueryErrorPropagates — a DB/query failure is NOT swallowed: it propagates
// so the view renders a real error state (no silent partial tabs).
func TestQueryErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	port := &fakePort{err: sentinel}
	uc := newUC(port, gate(perm(entityid.JobCategory), perm(entityid.JobTemplate)))
	if _, err := uc.Execute(userCtx()); !errors.Is(err, sentinel) {
		t.Fatalf("query error must propagate, got %v", err)
	}
}
