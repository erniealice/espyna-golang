//go:build http

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Tests for the v1 workspace-claim CSRF token logic.
//
// The four cases from plan 3.C2:
//  1. Token issued with (S1, W1) then verified against (S1, W1) -> pass
//  2. Token issued with (S1, W1) then verified against (S1, W2) -> fail (workspace mismatch)
//  3. Token issued with (S1, W1) then verified against (S2, W1) -> fail (session mismatch)
//  4. Token with tampered HMAC -> fail

var testCSRFSecret = []byte("test-secret-for-csrf-unit-tests")

func TestCSRFToken_IssueAndVerify_SameSessionAndWorkspace(t *testing.T) {
	tok := issueWorkspaceCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	if err := verifyWorkspaceCSRFToken(testCSRFSecret, tok, "session-S1", "workspace-W1"); err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}
}

func TestCSRFToken_WorkspaceMismatch(t *testing.T) {
	tok := issueWorkspaceCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	err := verifyWorkspaceCSRFToken(testCSRFSecret, tok, "session-S1", "workspace-W2")
	if err == nil {
		t.Error("expected workspace claim mismatch error, got nil")
	}
}

func TestCSRFToken_SessionMismatch(t *testing.T) {
	tok := issueWorkspaceCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	err := verifyWorkspaceCSRFToken(testCSRFSecret, tok, "session-S2", "workspace-W1")
	if err == nil {
		t.Error("expected session claim mismatch error, got nil")
	}
}

func TestCSRFToken_TamperedHMAC(t *testing.T) {
	tok := issueWorkspaceCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	// Replace the HMAC segment (3rd dot-delimited part) with an all-A signature.
	parts := splitDotParts(tok)
	if len(parts) != 3 {
		t.Fatalf("expected 3 dot-parts, got %d", len(parts))
	}
	// Craft a clearly wrong signature: 32 zero bytes base64url-encoded.
	wrongSig := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	tampered := parts[0] + "." + parts[1] + "." + wrongSig
	err := verifyWorkspaceCSRFToken(testCSRFSecret, tampered, "session-S1", "workspace-W1")
	if err == nil {
		t.Error("expected HMAC verification failure, got nil")
	}
}

// splitDotParts splits s into at most 3 parts on ".".
func splitDotParts(s string) []string {
	var parts []string
	for i := 0; i < 2; i++ {
		idx := -1
		for j, c := range s {
			if c == '.' {
				idx = j
				break
			}
		}
		if idx < 0 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	parts = append(parts, s)
	return parts
}

func TestCSRFToken_EmptySecret_LegacyMode(t *testing.T) {
	// With no secret, issueWorkspaceCSRFToken returns a random opaque string, and
	// verifyWorkspaceCSRFToken is a no-op (returns nil regardless).
	tok := issueWorkspaceCSRFToken(nil, "session-S1", "workspace-W1")
	if tok == "" {
		t.Error("expected non-empty token in legacy mode")
	}
	if err := verifyWorkspaceCSRFToken(nil, tok, "session-S1", "workspace-W1"); err != nil {
		t.Errorf("legacy mode should be a no-op, got error: %v", err)
	}
}

// TestNewWorkspaceCSRFMiddleware_WorkspaceClaimVerification exercises the full
// middleware HTTP path: a POST with a cookie and header token that carry stale
// workspace claims is rejected with 403.
func TestNewWorkspaceCSRFMiddleware_WorkspaceClaimVerification(t *testing.T) {
	const currentSession = "current-session-tok"
	const currentWorkspace = "workspace-W1"
	const oldWorkspace = "workspace-W_old"

	// Issue a token tied to the OLD workspace.
	staleTok := issueWorkspaceCSRFToken(testCSRFSecret, currentSession, oldWorkspace)

	mw := NewWorkspaceCSRFMiddleware(WorkspaceCSRFConfig{
		Secret: testCSRFSecret,
		SessionToken: func(r *http.Request) string {
			return currentSession
		},
		WorkspaceID: func(r *http.Request) string {
			return currentWorkspace
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/action/clients/add", nil)
	req.AddCookie(&http.Cookie{Name: WorkspaceCSRFCookieName, Value: staleTok})
	req.Header.Set(WorkspaceCSRFHeaderName, staleTok)

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for stale workspace CSRF token, got %d", rr.Code)
	}
}

// TestNewWorkspaceCSRFMiddleware_ValidClaimsPass verifies that a POST with a
// fresh token (matching session and workspace) is allowed through.
func TestNewWorkspaceCSRFMiddleware_ValidClaimsPass(t *testing.T) {
	const currentSession = "current-session-tok"
	const currentWorkspace = "workspace-W1"

	freshTok := issueWorkspaceCSRFToken(testCSRFSecret, currentSession, currentWorkspace)

	mw := NewWorkspaceCSRFMiddleware(WorkspaceCSRFConfig{
		Secret: testCSRFSecret,
		SessionToken: func(r *http.Request) string {
			return currentSession
		},
		WorkspaceID: func(r *http.Request) string {
			return currentWorkspace
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/action/clients/add", nil)
	req.AddCookie(&http.Cookie{Name: WorkspaceCSRFCookieName, Value: freshTok})
	req.Header.Set(WorkspaceCSRFHeaderName, freshTok)

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid CSRF token, got %d", rr.Code)
	}
}

// --- 20261104 session-expired signal (S1) + empty-session GET evidence (S2) ---

// TestWorkspaceCSRFRejection_CarriesSessionExpiredCode verifies every CSRF
// failure mode answers 403 with X-Session-Expired: 1 and a JSON body whose
// "code" is "session_expired", so the browser client can stop retrying and
// offer a fresh sign-in.
func TestWorkspaceCSRFRejection_CarriesSessionExpiredCode(t *testing.T) {
	const liveSession = "live-session"
	const workspace = "workspace-W1"
	valid := issueWorkspaceCSRFToken(testCSRFSecret, liveSession, workspace)
	otherSession := issueWorkspaceCSRFToken(testCSRFSecret, "other-session", workspace)
	emptyClaim := issueWorkspaceCSRFToken(testCSRFSecret, "", "")

	cases := []struct {
		name   string
		cookie string
		header string
	}{
		{name: "missing cookie", cookie: "", header: valid},
		{name: "missing header", cookie: valid, header: ""},
		{name: "cookie header mismatch", cookie: valid, header: otherSession},
		{name: "session claim mismatch", cookie: otherSession, header: otherSession},
		{name: "empty claim minted on an excluded GET", cookie: emptyClaim, header: emptyClaim},
	}
	mw := NewWorkspaceCSRFMiddleware(WorkspaceCSRFConfig{
		Secret:       testCSRFSecret,
		SessionToken: func(r *http.Request) string { return liveSession },
		WorkspaceID:  func(r *http.Request) string { return workspace },
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not run on a CSRF rejection")
	})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/action/grade-sheet/jt-1/record", nil)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: WorkspaceCSRFCookieName, Value: tc.cookie})
			}
			if tc.header != "" {
				req.Header.Set(WorkspaceCSRFHeaderName, tc.header)
			}
			rr := httptest.NewRecorder()
			mw(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", rr.Code)
			}
			if got := rr.Header().Get("X-Session-Expired"); got != "1" {
				t.Errorf("X-Session-Expired = %q, want \"1\"", got)
			}
			var body struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not JSON: %v (%q)", err, rr.Body.String())
			}
			if body.Code != "session_expired" {
				t.Errorf("code = %q, want session_expired", body.Code)
			}
			if body.Error == "" {
				t.Error("error message must stay populated")
			}
		})
	}
}

// TestWorkspaceCSRF_EmptySessionGET_KeepsValidCookie is the evidence for the
// 20261104 plan's S2 question. A GET that carries no session token (the /auth/*,
// /assets/*, /health paths the session middleware excludes) does NOT overwrite a
// still-valid ws_csrf cookie: the claim check skips the session/workspace
// comparison when the wanted value is empty, so needsWorkspaceCSRFRefresh is
// false and no Set-Cookie is written. Opening the sign-in page in another tab
// therefore cannot poison a live grade-sheet tab.
func TestWorkspaceCSRF_EmptySessionGET_KeepsValidCookie(t *testing.T) {
	live := issueWorkspaceCSRFToken(testCSRFSecret, "live-session", "workspace-W1")
	mw := NewWorkspaceCSRFMiddleware(WorkspaceCSRFConfig{
		Secret:       testCSRFSecret,
		SessionToken: func(r *http.Request) string { return "" },
		WorkspaceID:  func(r *http.Request) string { return "" },
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	for _, path := range []string{"/auth/login", "/assets/js/pyeza/cell-grid.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: WorkspaceCSRFCookieName, Value: live})
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)
		if sc := rr.Header().Values("Set-Cookie"); len(sc) != 0 {
			t.Errorf("GET %s with a valid cookie and no session wrote Set-Cookie %v; want none", path, sc)
		}
	}
}

// TestWorkspaceCSRF_EmptySessionGET_MintsOnlyWhenAbsent documents the residual
// case: once the 1-hour ws_csrf cookie has EXPIRED, an empty-session GET mints a
// replacement whose session claim is empty. A later /action/* POST then fails
// with "session claim mismatch" instead of "missing from cookie" — the same
// rejection either way, and the next session-bearing /w/ GET repairs both.
func TestWorkspaceCSRF_EmptySessionGET_MintsOnlyWhenAbsent(t *testing.T) {
	mw := NewWorkspaceCSRFMiddleware(WorkspaceCSRFConfig{
		Secret:       testCSRFSecret,
		SessionToken: func(r *http.Request) string { return "" },
		WorkspaceID:  func(r *http.Request) string { return "" },
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)
	var minted string
	for _, c := range rr.Result().Cookies() {
		if c.Name == WorkspaceCSRFCookieName {
			minted = c.Value
		}
	}
	if minted == "" {
		t.Fatal("expected a ws_csrf cookie to be minted when none was sent")
	}
	if err := verifyWorkspaceCSRFToken(testCSRFSecret, minted, "live-session", "workspace-W1"); err == nil {
		t.Error("an empty-claim cookie must not validate against a live session")
	}
}
