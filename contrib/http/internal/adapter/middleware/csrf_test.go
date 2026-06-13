//go:build http

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Tests for the v1 workspace-claim CSRF token logic.
//
// The four cases from plan §3.C2:
//  1. Token issued with (S1, W1) then verified against (S1, W1) → pass
//  2. Token issued with (S1, W1) then verified against (S1, W2) → fail (workspace mismatch)
//  3. Token issued with (S1, W1) then verified against (S2, W1) → fail (session mismatch)
//  4. Token with tampered HMAC → fail

var testCSRFSecret = []byte("test-secret-for-csrf-unit-tests")

func TestCSRFToken_IssueAndVerify_SameSessionAndWorkspace(t *testing.T) {
	tok := issueCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	if err := verifyCSRFToken(testCSRFSecret, tok, "session-S1", "workspace-W1"); err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}
}

func TestCSRFToken_WorkspaceMismatch(t *testing.T) {
	tok := issueCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	err := verifyCSRFToken(testCSRFSecret, tok, "session-S1", "workspace-W2")
	if err == nil {
		t.Error("expected workspace claim mismatch error, got nil")
	}
}

func TestCSRFToken_SessionMismatch(t *testing.T) {
	tok := issueCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	err := verifyCSRFToken(testCSRFSecret, tok, "session-S2", "workspace-W1")
	if err == nil {
		t.Error("expected session claim mismatch error, got nil")
	}
}

func TestCSRFToken_TamperedHMAC(t *testing.T) {
	tok := issueCSRFToken(testCSRFSecret, "session-S1", "workspace-W1")
	// Replace the HMAC segment (3rd dot-delimited part) with an all-A signature.
	parts := splitDotParts(tok)
	if len(parts) != 3 {
		t.Fatalf("expected 3 dot-parts, got %d", len(parts))
	}
	// Craft a clearly wrong signature: 32 zero bytes base64url-encoded.
	wrongSig := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	tampered := parts[0] + "." + parts[1] + "." + wrongSig
	err := verifyCSRFToken(testCSRFSecret, tampered, "session-S1", "workspace-W1")
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
	// With no secret, issueCSRFToken returns a random opaque string, and
	// verifyCSRFToken is a no-op (returns nil regardless).
	tok := issueCSRFToken(nil, "session-S1", "workspace-W1")
	if tok == "" {
		t.Error("expected non-empty token in legacy mode")
	}
	if err := verifyCSRFToken(nil, tok, "session-S1", "workspace-W1"); err != nil {
		t.Errorf("legacy mode should be a no-op, got error: %v", err)
	}
}

// TestNewCSRFMiddleware_WorkspaceClaimVerification exercises the full middleware
// HTTP path: a POST with a cookie and header token that carry stale workspace
// claims is rejected with 403.
func TestNewCSRFMiddleware_WorkspaceClaimVerification(t *testing.T) {
	const currentSession = "current-session-tok"
	const currentWorkspace = "workspace-W1"
	const oldWorkspace = "workspace-W_old"

	// Issue a token tied to the OLD workspace.
	staleTok := issueCSRFToken(testCSRFSecret, currentSession, oldWorkspace)

	mw := NewCSRFMiddleware(CSRFConfig{
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
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: staleTok})
	req.Header.Set(csrfHeaderName, staleTok)

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for stale workspace CSRF token, got %d", rr.Code)
	}
}

// TestNewCSRFMiddleware_ValidClaimsPass verifies that a POST with a fresh token
// (matching session and workspace) is allowed through.
func TestNewCSRFMiddleware_ValidClaimsPass(t *testing.T) {
	const currentSession = "current-session-tok"
	const currentWorkspace = "workspace-W1"

	freshTok := issueCSRFToken(testCSRFSecret, currentSession, currentWorkspace)

	mw := NewCSRFMiddleware(CSRFConfig{
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
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: freshTok})
	req.Header.Set(csrfHeaderName, freshTok)

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid CSRF token, got %d", rr.Code)
	}
}
