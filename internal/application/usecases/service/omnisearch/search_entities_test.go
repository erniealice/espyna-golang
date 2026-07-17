package omnisearch

import (
	"context"
	"testing"

	omnisearchpb "github.com/erniealice/esqyma/pkg/schema/v1/service/omni_search"

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

// fakePort records the (gated) request the use case forwards, and returns an
// empty successful response. Embedding Unimplemented satisfies the generated
// interface's unexported marker.
type fakePort struct {
	omnisearchpb.UnimplementedOmniSearchServiceServer
	called    bool
	gotReq    *omnisearchpb.OmniSearchRequest
	respCats  []*omnisearchpb.OmniSearchCategoryResults
}

func (p *fakePort) SearchEntities(_ context.Context, req *omnisearchpb.OmniSearchRequest) (*omnisearchpb.OmniSearchResponse, error) {
	p.called = true
	p.gotReq = req
	return &omnisearchpb.OmniSearchResponse{Success: true, Categories: p.respCats}, nil
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

// TestNilGatekeeperDeniesAll — a nil gatekeeper fails closed: zero categories,
// empty successful response, and the adapter port is never called.
func TestNilGatekeeperDeniesAll(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{ActionGatekeeper: nil})
	resp, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{Query: "nick"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Errorf("expected success")
	}
	if len(resp.GetCategories()) != 0 {
		t.Errorf("nil gatekeeper must yield zero categories, got %d", len(resp.GetCategories()))
	}
	if port.called {
		t.Errorf("adapter port must not be called when all categories are denied")
	}
}

// TestEmptyPermsYieldsEmptyPanel — an authorizer that grants nothing produces an
// empty panel (never an error that would enumerate categories).
func TestEmptyPermsYieldsEmptyPanel(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{ActionGatekeeper: gate()})
	resp, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{Query: "nick"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetCategories()) != 0 {
		t.Errorf("empty perms must yield zero categories, got %d", len(resp.GetCategories()))
	}
	if port.called {
		t.Errorf("adapter port must not be called with no permitted category")
	}
}

// TestPerCategoryGateSubset — only the permitted categories reach the adapter,
// in registry order; denied categories are absent.
func TestPerCategoryGateSubset(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{
		ActionGatekeeper: gate(perm(entityid.Client), perm(entityid.Product)),
	})
	_, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{Query: "nick"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !port.called {
		t.Fatalf("adapter port should be called when >=1 category is permitted")
	}
	got := port.gotReq.GetCategories()
	want := []string{"client", "product"}
	if len(got) != len(want) {
		t.Fatalf("gated categories = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("gated categories = %v, want %v (registry order)", got, want)
		}
	}
}

// TestMinCharsShortCircuits — a query below the 2-char minimum returns an empty
// successful response and never touches the gate or the port.
func TestMinCharsShortCircuits(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{ActionGatekeeper: gate(perm(entityid.Client))})
	resp, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{Query: "n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetCategories()) != 0 {
		t.Errorf("short query must yield zero categories")
	}
	if port.called {
		t.Errorf("short query must not call the adapter port")
	}
}

// TestUnknownCategoryIsRejected — an explicit unknown category key is an
// INVALID_ARGUMENT-style error (the use case never queries an unregistered key).
func TestUnknownCategoryIsRejected(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{ActionGatekeeper: gate(perm(entityid.Client))})
	_, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{
		Query:      "nick",
		Categories: []string{"bogus"},
	})
	if err == nil {
		t.Fatalf("expected an error for an unknown category")
	}
	if port.called {
		t.Errorf("adapter port must not be called on an unknown-category request")
	}
}

// TestExplicitSubsetIsGated — an explicit category subset is still gated: a
// permitted key survives, a denied key in the same request is dropped.
func TestExplicitSubsetIsGated(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{
		ActionGatekeeper: gate(perm(entityid.Plan)),
	})
	_, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{
		Query:      "nick",
		Categories: []string{"plan", "client"}, // client denied
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := port.gotReq.GetCategories()
	if len(got) != 1 || got[0] != "plan" {
		t.Errorf("explicit subset gate = %v, want [plan]", got)
	}
}

// TestNilPortDegradesGracefully — a permitted search with no registered adapter
// port (mock / non-postgres) returns an empty successful response, no panic.
func TestNilPortDegradesGracefully(t *testing.T) {
	uc := NewUseCases(Repositories{Query: nil}, Services{ActionGatekeeper: gate(perm(entityid.Client))})
	resp, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{Query: "nick"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() || len(resp.GetCategories()) != 0 {
		t.Errorf("nil port should yield an empty successful response")
	}
}

// TestLimitClampedToMax — a request over the server cap is clamped to
// maxLimitPerCategory before the adapter sees it.
func TestLimitClampedToMax(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{ActionGatekeeper: gate(perm(entityid.Client))})
	big := int32(999)
	_, err := uc.SearchEntities.Execute(userCtx(), &omnisearchpb.OmniSearchRequest{
		Query:            "nick",
		LimitPerCategory: &big,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port.gotReq.GetLimitPerCategory() != maxLimitPerCategory {
		t.Errorf("limit not clamped: got %d, want %d", port.gotReq.GetLimitPerCategory(), maxLimitPerCategory)
	}
}

// TestNilRequestIsEmptySuccess — a nil request degrades to empty success.
func TestNilRequestIsEmptySuccess(t *testing.T) {
	port := &fakePort{}
	uc := NewUseCases(Repositories{Query: port}, Services{ActionGatekeeper: gate(perm(entityid.Client))})
	resp, err := uc.SearchEntities.Execute(userCtx(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() || len(resp.GetCategories()) != 0 {
		t.Errorf("nil request should yield an empty successful response")
	}
	if port.called {
		t.Errorf("nil request must not call the adapter port")
	}
}
