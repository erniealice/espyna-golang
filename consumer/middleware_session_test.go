package consumer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
)

// fakeSessionRow models the security-relevant columns of a single session row.
type fakeSessionRow struct {
	userID   string
	wsUserID string
	wsID     string
	active   bool
}

// fakeAuthProvider is an in-memory stand-in for the password (db_auth) provider.
// It satisfies both ports.AuthProvider (so it can be assigned to the unexported
// AuthAdapter.provider field) and databaseAuthOperations (so the adapter's
// ValidateSession / GetSessionWorkspaceContext type assertions route to it).
//
// The store is keyed by token, exactly like the real session table, so the test
// can model a principal switch as either a token rotation or an in-place row
// mutation and assert the middleware always resolves the CURRENT binding.
type fakeAuthProvider struct {
	mu    sync.Mutex
	store map[string]fakeSessionRow
}

func newFakeAuthProvider() *fakeAuthProvider {
	return &fakeAuthProvider{store: map[string]fakeSessionRow{}}
}

func (f *fakeAuthProvider) put(token string, row fakeSessionRow) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[token] = row
}

// --- ports.AuthProvider surface ---

func (f *fakeAuthProvider) Name() string                              { return "fake-db-auth" }
func (f *fakeAuthProvider) Initialize(_ *authpb.ProviderConfig) error { return nil }
func (f *fakeAuthProvider) GetAuthService() authServiceOperations     { return nil }
func (f *fakeAuthProvider) IsHealthy(_ context.Context) error         { return nil }
func (f *fakeAuthProvider) Close() error                              { return nil }
func (f *fakeAuthProvider) IsEnabled() bool                           { return true }

// --- databaseAuthOperations surface (only the two used by the middleware
//     do real work; the rest are inert stubs to satisfy the interface) ---

func (f *fakeAuthProvider) Register(context.Context, string, string, string, string, string) (string, error) {
	return "", nil
}
func (f *fakeAuthProvider) Login(context.Context, string, string) (string, *authpb.Identity, error) {
	return "", nil, nil
}
func (f *fakeAuthProvider) RequestPasswordReset(context.Context, string) (string, error) {
	return "", nil
}
func (f *fakeAuthProvider) ExecutePasswordReset(context.Context, string, string) error { return nil }
func (f *fakeAuthProvider) ChangePassword(context.Context, string, string, string) error {
	return nil
}
func (f *fakeAuthProvider) CreateSession(context.Context, string) (string, error) { return "", nil }
func (f *fakeAuthProvider) InvalidateSession(context.Context, string) error       { return nil }

func (f *fakeAuthProvider) ValidateSession(_ context.Context, token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.store[token]
	if !ok || !row.active || row.userID == "" {
		return "", http.ErrNoCookie // any non-nil error → middleware treats as invalid
	}
	return row.userID, nil
}

func (f *fakeAuthProvider) GetSessionWorkspaceContext(_ context.Context, token string) (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.store[token]
	if !ok || !row.active {
		return "", ""
	}
	return row.wsUserID, row.wsID
}

// capturedIdentity records what the middleware injected into the request ctx.
type capturedIdentity struct {
	userID   string
	wsID     string
	wsUserID string
	reached  bool
}

func newCapturingNext(cap *capturedIdentity) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cap.reached = true
		cap.userID = ExtractUserIDFromContext(r.Context())
		cap.wsID = GetWorkspaceIDFromContext(r.Context())
		cap.wsUserID = GetWorkspaceUserIDFromContext(r.Context())
	})
}

func newMiddlewareWithProvider(f *fakeAuthProvider) *SessionMiddleware {
	adapter := &AuthAdapter{provider: f}
	mw := NewSessionMiddleware(adapter)
	mw.CookieName = DefaultSessionCookieName
	return mw
}

func doRequest(mw *SessionMiddleware, next http.Handler, token string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: DefaultSessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mw.Handler(next).ServeHTTP(rec, req)
	return rec.Result()
}

// TestSessionMiddleware_NoCarryOverOnRotateSwitch is the core P3-W3 regression:
// a principal switch that ROTATES the cookie (workspace change A→B) must leave
// the request resolving B's binding, and the OLD token must no longer resolve a
// live binding (it 401s → redirect), so there is no path where B's request can
// read A's workspace context.
func TestSessionMiddleware_NoCarryOverOnRotateSwitch(t *testing.T) {
	f := newFakeAuthProvider()
	// Principal A: workspace W1, staff binding.
	f.put("tokA", fakeSessionRow{userID: "user-1", wsUserID: "wsu-A", wsID: "ws-1", active: true})
	mw := newMiddlewareWithProvider(f)

	// First request on tokA resolves A's binding.
	var capA capturedIdentity
	respA := doRequest(mw, newCapturingNext(&capA), "tokA")
	if !capA.reached {
		t.Fatalf("rotate: request on tokA should reach next handler, got status %d", respA.StatusCode)
	}
	if capA.wsID != "ws-1" || capA.wsUserID != "wsu-A" {
		t.Fatalf("rotate: expected A's binding (ws-1/wsu-A), got (%s/%s)", capA.wsID, capA.wsUserID)
	}

	// Switch A→B: rotate to a NEW token in workspace W2, mark old token inactive.
	f.put("tokB", fakeSessionRow{userID: "user-1", wsUserID: "wsu-B", wsID: "ws-2", active: true})
	f.put("tokA", fakeSessionRow{userID: "user-1", wsUserID: "wsu-A", wsID: "ws-1", active: false})

	// Request on the NEW cookie must resolve B's binding — never A's.
	var capB capturedIdentity
	if resp := doRequest(mw, newCapturingNext(&capB), "tokB"); !capB.reached {
		t.Fatalf("rotate: request on tokB should reach next, got status %d", resp.StatusCode)
	}
	if capB.wsID != "ws-2" || capB.wsUserID != "wsu-B" {
		t.Fatalf("rotate: post-switch request must use B's binding (ws-2/wsu-B), got (%s/%s) — STALE CARRY-OVER", capB.wsID, capB.wsUserID)
	}

	// The old token must no longer resolve a live binding: middleware redirects.
	var capStale capturedIdentity
	respStale := doRequest(mw, newCapturingNext(&capStale), "tokA")
	if capStale.reached {
		t.Fatalf("rotate: stale tokA must NOT reach next (it was rotated out); leaked binding (%s/%s)", capStale.wsID, capStale.wsUserID)
	}
	if respStale.StatusCode != http.StatusFound {
		t.Fatalf("rotate: stale tokA should redirect (302 Found), got %d", respStale.StatusCode)
	}
}

// TestSessionMiddleware_NoCarryOverOnInPlaceSwitch covers the same-workspace
// in-place switch (e.g. operator→client in W1): the token does not change, the
// session row is mutated. The next request on the SAME cookie must observe the
// post-switch binding (workspace_user_id NULL'd for a client principal), never
// the pre-switch operator binding.
func TestSessionMiddleware_NoCarryOverOnInPlaceSwitch(t *testing.T) {
	f := newFakeAuthProvider()
	// Before: operator binding in W1 with a workspace_user_id.
	f.put("tok1", fakeSessionRow{userID: "user-1", wsUserID: "wsu-operator", wsID: "ws-1", active: true})
	mw := newMiddlewareWithProvider(f)

	var before capturedIdentity
	doRequest(mw, newCapturingNext(&before), "tok1")
	if before.wsUserID != "wsu-operator" {
		t.Fatalf("in-place: pre-switch should see operator wsUserID, got %q", before.wsUserID)
	}

	// In-place switch operator→client in the SAME workspace: same token, the row
	// is mutated. A client principal carries no workspace_user_id (NULL → empty).
	f.put("tok1", fakeSessionRow{userID: "user-1", wsUserID: "", wsID: "ws-1", active: true})

	var after capturedIdentity
	if resp := doRequest(mw, newCapturingNext(&after), "tok1"); !after.reached {
		t.Fatalf("in-place: request should reach next, got status %d", resp.StatusCode)
	}
	if after.wsUserID != "" {
		t.Fatalf("in-place: post-switch must drop operator wsUserID (got %q) — STALE CARRY-OVER", after.wsUserID)
	}
	if after.wsID != "ws-1" {
		t.Fatalf("in-place: workspace should remain ws-1, got %q", after.wsID)
	}
}

// TestSessionMiddleware_InvalidSessionRedirects confirms the deny path is intact:
// a token that names no live session never reaches the next handler and is sent
// to login (no over-correction in the opposite direction either — a valid token
// still reaches next, asserted by the tests above).
func TestSessionMiddleware_InvalidSessionRedirects(t *testing.T) {
	f := newFakeAuthProvider()
	mw := newMiddlewareWithProvider(f)

	var cap capturedIdentity
	resp := doRequest(mw, newCapturingNext(&cap), "ghost-token")
	if cap.reached {
		t.Fatalf("invalid session must not reach next handler")
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("invalid session should redirect (302 Found), got %d", resp.StatusCode)
	}
}

// TestSessionMiddleware_ValidSessionNoWorkspaceStillAuthenticates guards against
// over-correction: a valid session that has not yet selected a workspace
// (empty wsID/wsUserID) must still authenticate and reach the handler — empty
// workspace context is a legitimate pre-selection state, not a denial.
func TestSessionMiddleware_ValidSessionNoWorkspaceStillAuthenticates(t *testing.T) {
	f := newFakeAuthProvider()
	f.put("tok-nows", fakeSessionRow{userID: "user-1", wsUserID: "", wsID: "", active: true})
	mw := newMiddlewareWithProvider(f)

	var cap capturedIdentity
	if resp := doRequest(mw, newCapturingNext(&cap), "tok-nows"); !cap.reached {
		t.Fatalf("valid session with no workspace must still reach next, got status %d", resp.StatusCode)
	}
	if cap.userID != "user-1" {
		t.Fatalf("expected authenticated user-1, got %q", cap.userID)
	}
	if cap.wsID != "" || cap.wsUserID != "" {
		t.Fatalf("expected empty workspace context, got (%s/%s)", cap.wsID, cap.wsUserID)
	}
}
