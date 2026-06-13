package identity

import (
	"context"
	"errors"
	"testing"
)

func TestWithRequestIdentity_and_Must(t *testing.T) {
	t.Parallel()

	id := &RequestIdentity{
		UserID:             "user-123",
		WorkspaceID:        "ws-456",
		WorkspaceUserID:    "wsu-789",
		Email:              "test@example.com",
		SessionToken:       "tok-abc",
		ActingAsClientID:   "client-1",
		ActingAsSupplierID: "supplier-2",
	}

	ctx := WithRequestIdentity(context.Background(), id)
	got := Must(ctx)

	if got != id {
		t.Fatal("Must returned a different pointer than what was stored")
	}
	if got.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user-123")
	}
	if got.WorkspaceID != "ws-456" {
		t.Errorf("WorkspaceID = %q, want %q", got.WorkspaceID, "ws-456")
	}
	if got.WorkspaceUserID != "wsu-789" {
		t.Errorf("WorkspaceUserID = %q, want %q", got.WorkspaceUserID, "wsu-789")
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}
	if got.SessionToken != "tok-abc" {
		t.Errorf("SessionToken = %q, want %q", got.SessionToken, "tok-abc")
	}
	if got.ActingAsClientID != "client-1" {
		t.Errorf("ActingAsClientID = %q, want %q", got.ActingAsClientID, "client-1")
	}
	if got.ActingAsSupplierID != "supplier-2" {
		t.Errorf("ActingAsSupplierID = %q, want %q", got.ActingAsSupplierID, "supplier-2")
	}
}

func TestMust_panics_on_empty_context(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Must did not panic on empty context")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value is not a string: %v", r)
		}
		if msg == "" {
			t.Fatal("panic message is empty")
		}
	}()

	Must(context.Background())
}

func TestMust_panics_on_nil_identity(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), contextKey{}, (*RequestIdentity)(nil))
	defer func() {
		if recover() == nil {
			t.Fatal("Must did not panic on nil identity")
		}
	}()

	Must(ctx)
}

func TestRequire_returns_identity(t *testing.T) {
	t.Parallel()

	id := &RequestIdentity{UserID: "user-1", WorkspaceID: "ws-1"}
	ctx := WithRequestIdentity(context.Background(), id)

	got, err := Require(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id {
		t.Fatal("Require returned a different pointer")
	}
}

func TestRequire_returns_error_on_empty_context(t *testing.T) {
	t.Parallel()

	got, err := Require(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrIdentityNotInContext) {
		t.Errorf("expected ErrIdentityNotInContext, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil identity, got %+v", got)
	}
}

func TestFromContext_present(t *testing.T) {
	t.Parallel()

	id := &RequestIdentity{UserID: "user-1"}
	ctx := WithRequestIdentity(context.Background(), id)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext returned ok=false for a present identity")
	}
	if got != id {
		t.Fatal("FromContext returned a different pointer")
	}
}

func TestFromContext_absent(t *testing.T) {
	t.Parallel()

	got, ok := FromContext(context.Background())
	if ok {
		t.Fatal("FromContext returned ok=true for an absent identity")
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestZeroValueFields(t *testing.T) {
	t.Parallel()

	// A RequestIdentity with only UserID set — all other fields are
	// zero-value strings, which is the expected state for pre-workspace-selection.
	id := &RequestIdentity{UserID: "user-1"}
	ctx := WithRequestIdentity(context.Background(), id)
	got := Must(ctx)

	if got.WorkspaceID != "" {
		t.Errorf("WorkspaceID = %q, want empty", got.WorkspaceID)
	}
	if got.ActingAsClientID != "" {
		t.Errorf("ActingAsClientID = %q, want empty", got.ActingAsClientID)
	}
	if got.ActingAsSupplierID != "" {
		t.Errorf("ActingAsSupplierID = %q, want empty", got.ActingAsSupplierID)
	}
}

func TestDefaultSessionCookieName(t *testing.T) {
	t.Parallel()
	if DefaultSessionCookieName != "ichizen_session" {
		t.Errorf("DefaultSessionCookieName = %q, want %q", DefaultSessionCookieName, "ichizen_session")
	}
}
