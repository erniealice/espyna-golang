package auth

import (
	"context"
	"strings"
	"testing"

	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

func TestInvalidateSession_Execute(t *testing.T) {
	const (
		tokenOK   = "token-valid"
		sessionID = "sess-1"
	)

	t.Run("nil_session_repo_fails_closed_service_unavailable", func(t *testing.T) {
		uc := NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: nil},
			InvalidateSessionServices{Translator: newKeyEchoTranslator()},
		)
		_, err := uc.Execute(context.Background(), &authpb.InvalidateSessionRequest{Token: tokenOK})
		if err == nil || !strings.Contains(err.Error(), "auth.errors.service_unavailable") {
			t.Fatalf("want service_unavailable, got %v", err)
		}
	})

	t.Run("nil_request_returns_validation_request_required", func(t *testing.T) {
		uc := NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: &fakeSessionRepo{}},
			InvalidateSessionServices{Translator: newKeyEchoTranslator()},
		)
		_, err := uc.Execute(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "auth.validation.request_required") {
			t.Fatalf("want request_required, got %v", err)
		}
	})

	t.Run("both_identifiers_empty_returns_session_identifier_required", func(t *testing.T) {
		uc := NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: &fakeSessionRepo{}},
			InvalidateSessionServices{Translator: newKeyEchoTranslator()},
		)
		_, err := uc.Execute(context.Background(), &authpb.InvalidateSessionRequest{})
		if err == nil || !strings.Contains(err.Error(), "auth.validation.session_identifier_required") {
			t.Fatalf("want session_identifier_required, got %v", err)
		}
	})

	t.Run("unknown_token_is_noop_returns_invalidated_false", func(t *testing.T) {
		repo := &fakeSessionRepo{readResp: &sessionpb.ReadSessionResponse{}}
		uc := NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: repo},
			InvalidateSessionServices{Translator: newKeyEchoTranslator()},
		)
		resp, err := uc.Execute(context.Background(), &authpb.InvalidateSessionRequest{Token: "unknown"})
		if err != nil {
			t.Fatalf("expected nil error for unknown token (idempotent semantics), got %v", err)
		}
		if resp == nil || resp.GetInvalidated() {
			t.Errorf("expected Invalidated=false for unknown token, got %+v", resp)
		}
		if repo.updateN != 0 {
			t.Errorf("UpdateSession should not be called for unknown token; called %d times", repo.updateN)
		}
	})

	t.Run("happy_path_by_session_id_marks_inactive", func(t *testing.T) {
		repo := &fakeSessionRepo{}
		uc := NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: repo},
			InvalidateSessionServices{Translator: newKeyEchoTranslator()},
		)
		resp, err := uc.Execute(context.Background(), &authpb.InvalidateSessionRequest{SessionId: sessionID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || !resp.GetInvalidated() {
			t.Errorf("expected Invalidated=true, got %+v", resp)
		}
		if repo.updateN != 1 {
			t.Errorf("UpdateSession call count: want 1, got %d", repo.updateN)
		}
		if repo.lastUpdate == nil || repo.lastUpdate.GetData() == nil {
			t.Fatal("UpdateSession received nil request/data")
		}
		got := repo.lastUpdate.GetData()
		if got.GetId() != sessionID {
			t.Errorf("UpdateSession Id: want %q, got %q", sessionID, got.GetId())
		}
		if got.Active {
			t.Error("UpdateSession Active: want false")
		}
	})

	t.Run("happy_path_by_token_resolves_then_marks_inactive", func(t *testing.T) {
		repo := &fakeSessionRepo{
			readResp: &sessionpb.ReadSessionResponse{Data: []*sessionpb.Session{{
				Id: sessionID, Token: tokenOK, Active: true,
			}}},
		}
		uc := NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: repo},
			InvalidateSessionServices{Translator: newKeyEchoTranslator()},
		)
		resp, err := uc.Execute(context.Background(), &authpb.InvalidateSessionRequest{Token: tokenOK})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || !resp.GetInvalidated() {
			t.Errorf("expected Invalidated=true, got %+v", resp)
		}
		if repo.updateN != 1 {
			t.Errorf("UpdateSession call count: want 1, got %d", repo.updateN)
		}
		if repo.lastUpdate.GetData().GetId() != sessionID {
			t.Errorf("UpdateSession should target resolved session id %q, got %q",
				sessionID, repo.lastUpdate.GetData().GetId())
		}
	})

	t.Run("idempotent_invalidating_already_inactive_session_still_ok", func(t *testing.T) {
		// UpdateSession on an already-inactive row succeeds (the use case does
		// not gate on prior Active state — that's a DB-layer concern).
		repo := &fakeSessionRepo{
			readResp: &sessionpb.ReadSessionResponse{Data: []*sessionpb.Session{{
				Id: sessionID, Token: tokenOK, Active: false,
			}}},
		}
		uc := NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: repo},
			InvalidateSessionServices{Translator: newKeyEchoTranslator()},
		)
		resp, err := uc.Execute(context.Background(), &authpb.InvalidateSessionRequest{Token: tokenOK})
		if err != nil {
			t.Fatalf("unexpected error on already-inactive session: %v", err)
		}
		if resp == nil || !resp.GetInvalidated() {
			t.Errorf("expected Invalidated=true on idempotent retry, got %+v", resp)
		}
	})
}
