//go:build http

package middleware

import "testing"

// The D-CLIENT canonical-form contract: the action_workspace_guard verifies the
// HMAC over the query-less r.URL.Path (handle() step 5), so a FormAction must be
// signed over the BARE path. rowActionTokens and every other action form already
// sign the bare path; the client edit drawer was the lone outlier that signed
// "...edit/{id}?mode=" and therefore 409'd on every save.

func TestWorkspaceFormSigner_BarePathRoundTrip(t *testing.T) {
	s := NewWorkspaceFormSigner("unit-test-key")
	const ws, path = "ws-1", "/action/clients/cl-1/edit"

	sig, err := s.SignFields(ws, path)
	if err != nil {
		t.Fatalf("SignFields: %v", err)
	}
	if err := s.Verify(ws, path, sig); err != nil {
		t.Fatalf("bare-path signature must verify against the bare r.URL.Path: %v", err)
	}
}

func TestWorkspaceFormSigner_QueryBearingPathFailsBareVerify(t *testing.T) {
	s := NewWorkspaceFormSigner("unit-test-key")
	const ws = "ws-1"
	const signedPath = "/action/clients/cl-1/edit?mode=" // pre-fix FormAction
	const verifyPath = "/action/clients/cl-1/edit"       // guard uses r.URL.Path

	sig, err := s.SignFields(ws, signedPath)
	if err != nil {
		t.Fatalf("SignFields: %v", err)
	}
	// This is the D-CLIENT 409: a signature bound to a query-bearing path never
	// matches the bare path the guard verifies.
	if err := s.Verify(ws, verifyPath, sig); err == nil {
		t.Fatal("query-bearing signed path must NOT verify against the bare r.URL.Path")
	}

	// The canonical (bare) signature verifies — the fix.
	sigOK, err := s.SignFields(ws, verifyPath)
	if err != nil {
		t.Fatalf("SignFields: %v", err)
	}
	if err := s.Verify(ws, verifyPath, sigOK); err != nil {
		t.Fatalf("canonical bare-path signature must verify: %v", err)
	}
}
