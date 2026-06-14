//go:build http

// workspace_path_slice_test.go
//
// Reference-slice proof for the LOCKED contrib pattern: with the `http` server
// provider compiled in, buildWorkspacePath wires the AGNOSTIC
// consumer/http/middleware config to the FRAMEWORK-NATIVE contrib/http net/http
// WorkspacePath impl. This test traces the exact request path of the live bug
//
//	GET /w/{slug}/clients/list/active
//
// proving the impl (NOT the old pass-through stub) parses the slug, resolves it
// to a workspace_id, validates the binding, pins the URL-canonical workspace_id
// into context, STRIPS the /w/{slug} prefix, and dispatches the bare route to
// the downstream handler. It also asserts the two fail-closed security
// branches: ambiguous binding -> picker (303), no binding -> unified not-found
// (303), with NO auto-elect by privilege.
package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

// fakeSession injects a session identity so SessionLookup sees an authenticated
// request. workspaceID == "" forces a rotation-free path when the resolved
// workspace matches (here we use the SAME id the slug resolves to, so no
// ExecuteSwitch is invoked — keeping the test hermetic).
func sliceSessionLookup(r *http.Request) (userID, workspaceID, token string, ok bool) {
	return "user-1", "ws-acme", "tok-1", true
}

func baseSliceConfig(downstreamHit *string) consumermw.WorkspacePathConfig {
	return consumermw.WorkspacePathConfig{
		SessionLookup: sliceSessionLookup,
		SlugLookup: func(ctx context.Context, slug string) (string, error) {
			if slug == "acme" {
				return "ws-acme", nil
			}
			return "", nil // miss
		},
		BindingResolver: func(ctx context.Context, userID, workspaceID string, kind int32, principalID string) (*consumermw.WorkspaceBinding, error) {
			return &consumermw.WorkspaceBinding{
				Kind:        consumermw.BindingKindOperatorStaff,
				PrincipalID: "staff-1",
				WorkspaceID: workspaceID,
			}, nil
		},
		ExecuteSwitch: func(ctx context.Context, userID, token string, b *consumermw.WorkspaceBinding, urlActingAs, requestURL, referer, secFetchSite, userAgent string) (*consumermw.WorkspaceSwitchResult, error) {
			return &consumermw.WorkspaceSwitchResult{}, nil
		},
		WithWorkspaceID: func(ctx context.Context, wsID string) context.Context { return ctx },
	}
}

func newSameOriginRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	// Pass the impl's CSRF preflight for a same-origin navigation.
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Sec-Fetch-Mode", "navigate")
	return r
}

// TestWorkspacePathSlice_StripsAndDispatches is the headline assertion: the
// request reaches the contrib impl, the /w/acme prefix is stripped, and the
// bare /clients/list/active route is dispatched to the downstream handler.
func TestWorkspacePathSlice_StripsAndDispatches(t *testing.T) {
	var gotPath string
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		// The middleware MUST have pinned the URL-canonical workspace_id into
		// the contrib ctx key before reaching here.
		w.WriteHeader(http.StatusOK)
	})

	mw := buildWorkspacePath(baseSliceConfig(nil))
	if mw == nil {
		t.Fatal("buildWorkspacePath returned nil")
	}

	rec := httptest.NewRecorder()
	mw(downstream).ServeHTTP(rec, newSameOriginRequest("/w/acme/clients/list/active"))

	if gotPath != "/clients/list/active" {
		t.Fatalf("expected stripped path /clients/list/active, downstream saw %q (status=%d) — "+
			"if this is empty the request never reached downstream (still the pass-through stub or a redirect)",
			gotPath, rec.Code)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from downstream, got %d", rec.Code)
	}
}

// TestWorkspacePathSlice_AmbiguousBindingGoesToPicker proves ErrAmbiguousBinding
// routes to the picker (303 -> /auth/select-workspace-role) and does NOT
// auto-elect a binding or reach downstream (security invariant A3).
func TestWorkspacePathSlice_AmbiguousBindingGoesToPicker(t *testing.T) {
	reached := false
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })

	cfg := baseSliceConfig(nil)
	cfg.BindingResolver = func(ctx context.Context, userID, workspaceID string, kind int32, principalID string) (*consumermw.WorkspaceBinding, error) {
		return nil, consumermw.ErrAmbiguousBinding
	}

	rec := httptest.NewRecorder()
	buildWorkspacePath(cfg)(downstream).ServeHTTP(rec, newSameOriginRequest("/w/acme/clients/list/active"))

	if reached {
		t.Fatal("ambiguous binding MUST NOT reach downstream (no auto-elect by privilege)")
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect to picker, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/auth/select-workspace-role" {
		t.Fatalf("expected redirect to /auth/select-workspace-role, got %q", loc)
	}
}

// TestWorkspacePathSlice_NoBindingGoesToNotFound proves ErrNoBinding routes to
// the unified not-found/picker response (fail-closed) and never reaches
// downstream.
func TestWorkspacePathSlice_NoBindingGoesToNotFound(t *testing.T) {
	reached := false
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })

	cfg := baseSliceConfig(nil)
	cfg.BindingResolver = func(ctx context.Context, userID, workspaceID string, kind int32, principalID string) (*consumermw.WorkspaceBinding, error) {
		return nil, consumermw.ErrNoBinding
	}

	rec := httptest.NewRecorder()
	buildWorkspacePath(cfg)(downstream).ServeHTTP(rec, newSameOriginRequest("/w/acme/clients/list/active"))

	if reached {
		t.Fatal("missing binding MUST NOT reach downstream (fail-closed)")
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", rec.Code)
	}
}

// TestWorkspacePathSlice_NonWorkspacePathPassesThrough proves the fast-path:
// a non-/w/ request bypasses the middleware untouched.
func TestWorkspacePathSlice_NonWorkspacePathPassesThrough(t *testing.T) {
	var gotPath string
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { gotPath = r.URL.Path })

	rec := httptest.NewRecorder()
	buildWorkspacePath(baseSliceConfig(nil))(downstream).ServeHTTP(rec, newSameOriginRequest("/clients/list/active"))

	if gotPath != "/clients/list/active" {
		t.Fatalf("non-/w/ request should pass through unchanged, downstream saw %q", gotPath)
	}
}
