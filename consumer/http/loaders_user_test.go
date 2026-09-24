package http

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/consumer"
)

// fakeUserReader is a UserReader test double whose ReadUserDisplay result and
// error are set per test case.
type fakeUserReader struct {
	display UserDisplay
	err     error
}

func (f *fakeUserReader) ReadUserDisplay(_ context.Context, _ string) (UserDisplay, error) {
	return f.display, f.err
}

// fakeSelfDisplayReader is a SelfDisplayReader test double.
type fakeSelfDisplayReader struct {
	display UserDisplay
	err     error
	calls   int
}

func (f *fakeSelfDisplayReader) ReadSelfDisplay(_ context.Context) (UserDisplay, error) {
	f.calls++
	return f.display, f.err
}

// TestLoadCurrentUser_PermissionDeniedFallsBackToSelfDisplay proves that when
// the permission-gated UserReader is denied (the "Permission denied" case
// observed for staff/teacher principals under enforced RBAC), the loader uses
// the gate-free SelfDisplayReader instead of falling all the way through to
// the generic "Signed"/"In" placeholder derived from an empty session email.
func TestLoadCurrentUser_PermissionDeniedFallsBackToSelfDisplay(t *testing.T) {
	reader := &fakeUserReader{err: errors.New("permission denied")}
	self := &fakeSelfDisplayReader{display: UserDisplay{
		FirstName: "Junrey", LastName: "Tejas", Email: "junrey.tejas@mmis.edu.ph",
	}}
	loader := NewDBUserLoader(reader, self, nil, ProfileURLs{})

	ctx := consumer.WithUserID(context.Background(), "u1")
	got := loader.LoadCurrentUser(ctx)

	if got.FirstName != "Junrey" || got.LastName != "Tejas" {
		t.Fatalf("got name (%q, %q), want (Junrey, Tejas) — the sidebar should never show the generic placeholder when self-display succeeds", got.FirstName, got.LastName)
	}
	if got.Email != "junrey.tejas@mmis.edu.ph" {
		t.Fatalf("got email %q, want junrey.tejas@mmis.edu.ph", got.Email)
	}
	if self.calls != 1 {
		t.Fatalf("self-display called %d times, want 1", self.calls)
	}
}

// TestLoadCurrentUser_NamelessReadFallsBackToSelfDisplay covers the second
// call site: the permission-gated read SUCCEEDS but comes back with no name
// (e.g. a redacted/partial row), which must also try self-display before the
// session-derived placeholder.
func TestLoadCurrentUser_NamelessReadFallsBackToSelfDisplay(t *testing.T) {
	reader := &fakeUserReader{display: UserDisplay{Active: true, Email: "from-primary-read@example.com"}}
	self := &fakeSelfDisplayReader{display: UserDisplay{FirstName: "Ana", LastName: "Cruz"}}
	loader := NewDBUserLoader(reader, self, nil, ProfileURLs{})

	ctx := consumer.WithUserID(context.Background(), "u2")
	got := loader.LoadCurrentUser(ctx)

	if got.FirstName != "Ana" || got.LastName != "Cruz" {
		t.Fatalf("got name (%q, %q), want (Ana, Cruz)", got.FirstName, got.LastName)
	}
	// Self-display returned no email; the primary (permission-gated) read's
	// email is still usable and must not be discarded.
	if got.Email != "from-primary-read@example.com" {
		t.Fatalf("got email %q, want the primary read's email to survive as a fallback", got.Email)
	}
}

// TestLoadCurrentUser_BothReadsFailFallsBackToSessionIdentity proves the
// pre-existing last-resort behavior survives unchanged: when neither the
// primary nor the self-display read produce a name, the loader still renders
// a non-empty block derived from whatever the session carries.
func TestLoadCurrentUser_BothReadsFailFallsBackToSessionIdentity(t *testing.T) {
	reader := &fakeUserReader{err: errors.New("permission denied")}
	self := &fakeSelfDisplayReader{err: errors.New("self display: user not found")}
	loader := NewDBUserLoader(reader, self, nil, ProfileURLs{})

	ctx := consumer.WithUserID(context.Background(), "u3")
	got := loader.LoadCurrentUser(ctx)

	if got.UserID != "u3" {
		t.Fatalf("got UserID %q, want u3 (block must still render)", got.UserID)
	}
	if got.FirstName == "" || got.LastName == "" {
		t.Fatalf("got empty name — the template slices FirstName/LastName for initials and must never receive an empty string")
	}
}

// fakePrincipalCountReader is a PrincipalCountReader test double that also
// counts calls, so tests can assert on cache behavior.
type fakePrincipalCountReader struct {
	count int
	err   error
	calls int
}

func (f *fakePrincipalCountReader) CountPrincipals(_ context.Context, _ string) (int, error) {
	f.calls++
	return f.count, f.err
}

// TestLoadCurrentUser_SwitchRole_MultiPrincipalShows proves the "Switch Role"
// menu item (ShowSwitchPrincipal) is shown, with the configured URL, when the
// user holds more than one selectable principal binding.
func TestLoadCurrentUser_SwitchRole_MultiPrincipalShows(t *testing.T) {
	reader := &fakeUserReader{display: UserDisplay{Active: true, FirstName: "Ana", LastName: "Cruz"}}
	principals := &fakePrincipalCountReader{count: 2}
	loader := NewDBUserLoader(reader, nil, principals, ProfileURLs{SwitchPrincipal: "/auth/select-workspace-role"})

	ctx := consumer.WithUserID(context.Background(), "u1")
	got := loader.LoadCurrentUser(ctx)

	if !got.ShowSwitchPrincipal {
		t.Fatalf("ShowSwitchPrincipal = false, want true for a 2-principal user")
	}
	if got.SwitchPrincipalURL != "/auth/select-workspace-role" {
		t.Fatalf("SwitchPrincipalURL = %q, want /auth/select-workspace-role", got.SwitchPrincipalURL)
	}
}

// TestLoadCurrentUser_SwitchRole_SinglePrincipalHides proves the item stays
// hidden for a single-principal user — the common case — and that a resolver
// error also hides it (fail closed on the UI affordance).
func TestLoadCurrentUser_SwitchRole_SinglePrincipalHides(t *testing.T) {
	cases := map[string]*fakePrincipalCountReader{
		"single principal": {count: 1},
		"zero principals":  {count: 0},
		"resolver error":   {err: errors.New("resolve failed")},
	}
	for name, principals := range cases {
		t.Run(name, func(t *testing.T) {
			reader := &fakeUserReader{display: UserDisplay{Active: true, FirstName: "Ana", LastName: "Cruz"}}
			loader := NewDBUserLoader(reader, nil, principals, ProfileURLs{SwitchPrincipal: "/auth/select-workspace-role"})

			ctx := consumer.WithUserID(context.Background(), "u1")
			got := loader.LoadCurrentUser(ctx)

			if got.ShowSwitchPrincipal {
				t.Fatalf("ShowSwitchPrincipal = true, want false (%s)", name)
			}
			if got.SwitchPrincipalURL != "" {
				t.Fatalf("SwitchPrincipalURL = %q, want empty (%s)", got.SwitchPrincipalURL, name)
			}
		})
	}
}

// TestLoadCurrentUser_SwitchRole_EmptyRouteHides proves that even a qualifying
// multi-principal user never gets a menu item with an empty href: no
// registered "personal.switch_principal" route means no item, matching the
// "validate both route keys" rule for a visible link and its mounted page.
func TestLoadCurrentUser_SwitchRole_EmptyRouteHides(t *testing.T) {
	reader := &fakeUserReader{display: UserDisplay{Active: true, FirstName: "Ana", LastName: "Cruz"}}
	principals := &fakePrincipalCountReader{count: 2}
	loader := NewDBUserLoader(reader, nil, principals, ProfileURLs{}) // SwitchPrincipal left empty

	ctx := consumer.WithUserID(context.Background(), "u1")
	got := loader.LoadCurrentUser(ctx)

	if got.ShowSwitchPrincipal || got.SwitchPrincipalURL != "" {
		t.Fatalf("got ShowSwitchPrincipal=%v URL=%q, want hidden when no route is registered", got.ShowSwitchPrincipal, got.SwitchPrincipalURL)
	}
}

// TestLoadCurrentUser_SwitchRole_NilReaderHides proves a nil PrincipalCountReader
// (older wiring, or the resolver use case unavailable) degrades to "hidden",
// never a panic.
func TestLoadCurrentUser_SwitchRole_NilReaderHides(t *testing.T) {
	reader := &fakeUserReader{display: UserDisplay{Active: true, FirstName: "Ana", LastName: "Cruz"}}
	loader := NewDBUserLoader(reader, nil, nil, ProfileURLs{SwitchPrincipal: "/auth/select-workspace-role"})

	ctx := consumer.WithUserID(context.Background(), "u1")
	got := loader.LoadCurrentUser(ctx)

	if got.ShowSwitchPrincipal || got.SwitchPrincipalURL != "" {
		t.Fatalf("got ShowSwitchPrincipal=%v URL=%q, want hidden with a nil PrincipalCountReader", got.ShowSwitchPrincipal, got.SwitchPrincipalURL)
	}
}

// TestLoadCurrentUser_SwitchRole_CountIsCached proves the principal count is
// cached per user — the sidebar renders on every page, and ResolvePrincipals
// runs several queries, so a fresh call on every render would be a
// regression. A second LoadCurrentUser call for the same user must not hit
// the resolver again within the TTL.
func TestLoadCurrentUser_SwitchRole_CountIsCached(t *testing.T) {
	reader := &fakeUserReader{display: UserDisplay{Active: true, FirstName: "Ana", LastName: "Cruz"}}
	principals := &fakePrincipalCountReader{count: 2}
	loader := NewDBUserLoader(reader, nil, principals, ProfileURLs{SwitchPrincipal: "/auth/select-workspace-role"})

	ctx := consumer.WithUserID(context.Background(), "u1")
	loader.LoadCurrentUser(ctx)
	loader.LoadCurrentUser(ctx)
	loader.LoadCurrentUser(ctx)

	if principals.calls != 1 {
		t.Fatalf("resolver called %d times across 3 renders, want 1 (cached)", principals.calls)
	}
}

// TestLoadCurrentUser_NilSelfDisplayPreservesPriorBehavior proves a nil
// SelfDisplayReader (e.g. an older wiring, or ReadSelfDisplay unavailable)
// degrades to exactly the pre-fix session-identity fallback, not a panic.
func TestLoadCurrentUser_NilSelfDisplayPreservesPriorBehavior(t *testing.T) {
	reader := &fakeUserReader{err: errors.New("permission denied")}
	loader := NewDBUserLoader(reader, nil, nil, ProfileURLs{})

	ctx := consumer.WithUserID(context.Background(), "u4")
	got := loader.LoadCurrentUser(ctx)

	if got.UserID != "u4" {
		t.Fatalf("got UserID %q, want u4", got.UserID)
	}
	if got.FirstName == "" || got.LastName == "" {
		t.Fatalf("got empty name with a nil SelfDisplayReader — must still fail closed to a non-empty placeholder")
	}
}
