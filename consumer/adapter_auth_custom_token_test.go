package consumer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubCustomTokenMinter struct {
	identifier string
	token      string
	err        error
	calls      int
}

func (s *stubCustomTokenMinter) CreateCustomToken(_ context.Context, identifier string) (string, error) {
	s.calls++
	s.identifier = identifier
	return s.token, s.err
}

func TestAuthAdapterCreateCustomToken(t *testing.T) {
	t.Run("delegates trimmed identifier", func(t *testing.T) {
		minter := &stubCustomTokenMinter{token: "signed-token"}
		adapter := &AuthAdapter{customTokenMinter: minter}

		got, err := adapter.CreateCustomToken(context.Background(), "  dev@example.com  ")
		if err != nil {
			t.Fatalf("CreateCustomToken returned error: %v", err)
		}
		if got != "signed-token" {
			t.Fatalf("token = %q, want signed-token", got)
		}
		if minter.identifier != "dev@example.com" || minter.calls != 1 {
			t.Fatalf("minter received identifier=%q calls=%d", minter.identifier, minter.calls)
		}
	})

	t.Run("rejects empty identifier before provider", func(t *testing.T) {
		minter := &stubCustomTokenMinter{token: "should-not-be-used"}
		adapter := &AuthAdapter{customTokenMinter: minter}

		if _, err := adapter.CreateCustomToken(context.Background(), " \t "); err == nil {
			t.Fatal("CreateCustomToken should reject an empty identifier")
		}
		if minter.calls != 0 {
			t.Fatalf("minter calls = %d, want 0", minter.calls)
		}
	})

	t.Run("fails closed when capability is absent", func(t *testing.T) {
		if _, err := (&AuthAdapter{}).CreateCustomToken(context.Background(), "dev@example.com"); err == nil {
			t.Fatal("CreateCustomToken should fail when the active provider has no capability")
		}
	})

	t.Run("wraps provider failure without a token", func(t *testing.T) {
		minter := &stubCustomTokenMinter{err: errors.New("signing unavailable")}
		adapter := &AuthAdapter{customTokenMinter: minter}

		token, err := adapter.CreateCustomToken(context.Background(), "dev@example.com")
		if token != "" {
			t.Fatalf("token = %q, want empty on provider failure", token)
		}
		if err == nil || !strings.Contains(err.Error(), "signing unavailable") {
			t.Fatalf("error = %v, want wrapped provider failure", err)
		}
	})
}

func TestResolveCustomTokenMinter(t *testing.T) {
	want := &stubCustomTokenMinter{}
	if got := resolveCustomTokenMinter(struct{}{}, want); got != want {
		t.Fatalf("resolved minter = %#v, want %#v", got, want)
	}
	if got := resolveCustomTokenMinter(struct{}{}); got != nil {
		t.Fatalf("resolved minter = %#v, want nil", got)
	}
}
