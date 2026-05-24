package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// stubIDGenerator returns a fixed ID so we can assert it was wired through.
type stubIDGenerator struct{ id string }

func (s stubIDGenerator) GenerateID() string                        { return s.id }
func (s stubIDGenerator) GenerateIDWithPrefix(prefix string) string { return prefix + s.id }
func (s stubIDGenerator) IsEnabled() bool                           { return true }
func (s stubIDGenerator) GetProviderInfo() string                   { return "stub" }

func TestIssueSession_Execute(t *testing.T) {
	const (
		userID   = "user-1"
		wsUserID = "wsu-1"
		wsID     = "ws-1"
		stubID   = "sess-stub-id"
	)

	t.Run("nil_session_repo_fails_closed_service_unavailable", func(t *testing.T) {
		uc := NewIssueSessionUseCase(
			IssueSessionRepositories{Session: nil},
			IssueSessionServices{Translator: newKeyEchoTranslator()},
		)
		_, err := uc.Execute(context.Background(), &authpb.IssueSessionRequest{UserId: userID})
		if err == nil || !strings.Contains(err.Error(), "auth.errors.service_unavailable") {
			t.Fatalf("want service_unavailable, got %v", err)
		}
	})

	t.Run("nil_request_returns_validation_request_required", func(t *testing.T) {
		uc := NewIssueSessionUseCase(
			IssueSessionRepositories{Session: &fakeSessionRepo{}},
			IssueSessionServices{Translator: newKeyEchoTranslator()},
		)
		_, err := uc.Execute(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "auth.validation.request_required") {
			t.Fatalf("want request_required, got %v", err)
		}
	})

	t.Run("empty_user_id_returns_validation_user_required", func(t *testing.T) {
		uc := NewIssueSessionUseCase(
			IssueSessionRepositories{Session: &fakeSessionRepo{}},
			IssueSessionServices{Translator: newKeyEchoTranslator()},
		)
		_, err := uc.Execute(context.Background(), &authpb.IssueSessionRequest{UserId: ""})
		if err == nil || !strings.Contains(err.Error(), "auth.validation.user_required") {
			t.Fatalf("want user_required, got %v", err)
		}
	})

	t.Run("happy_path_creates_session_and_returns_token", func(t *testing.T) {
		repo := &fakeSessionRepo{}
		before := time.Now().UnixMilli()
		uc := NewIssueSessionUseCase(
			IssueSessionRepositories{Session: repo},
			IssueSessionServices{
				Translator:  newKeyEchoTranslator(),
				IDGenerator: stubIDGenerator{id: stubID},
			},
		)
		resp, err := uc.Execute(context.Background(), &authpb.IssueSessionRequest{
			UserId:          userID,
			WorkspaceUserId: wsUserID,
			WorkspaceId:     wsID,
		})
		after := time.Now().UnixMilli()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.createN != 1 {
			t.Errorf("CreateSession call count: want 1, got %d", repo.createN)
		}
		if resp == nil {
			t.Fatal("nil response")
		}
		if resp.GetSessionId() != stubID {
			t.Errorf("SessionId: want %q, got %q", stubID, resp.GetSessionId())
		}
		if resp.GetToken() == "" {
			t.Error("Token: want non-empty hex string, got empty")
		}
		// Default expiry is 7 days (~604800000 ms) — verify it lands within the
		// pre/post window plus the TTL.
		const defaultTTLms = int64(7 * 24 * 60 * 60 * 1000)
		if resp.GetExpiresAtUnixMs() < before+defaultTTLms-1000 || resp.GetExpiresAtUnixMs() > after+defaultTTLms+1000 {
			t.Errorf("ExpiresAtUnixMs %d outside expected window [%d, %d]",
				resp.GetExpiresAtUnixMs(), before+defaultTTLms, after+defaultTTLms)
		}
		// Verify CreateSession was called with the expected shape.
		if repo.lastCreate == nil || repo.lastCreate.GetData() == nil {
			t.Fatal("CreateSession received nil request/data")
		}
		got := repo.lastCreate.GetData()
		if got.GetUserId() != userID {
			t.Errorf("CreateSession UserId: want %q, got %q", userID, got.GetUserId())
		}
		if !got.Active {
			t.Error("CreateSession Active: want true")
		}
		if got.WorkspaceUserId == nil || *got.WorkspaceUserId != wsUserID {
			t.Errorf("CreateSession WorkspaceUserId: want %q, got %v", wsUserID, got.WorkspaceUserId)
		}
		if got.WorkspaceId == nil || *got.WorkspaceId != wsID {
			t.Errorf("CreateSession WorkspaceId: want %q, got %v", wsID, got.WorkspaceId)
		}
		if resp.GetWorkspaceUserId() != wsUserID || resp.GetWorkspaceId() != wsID {
			t.Errorf("response workspace fields not echoed: %+v", resp)
		}
	})

	t.Run("custom_expiry_overrides_default_ttl", func(t *testing.T) {
		repo := &fakeSessionRepo{}
		customTTL := 30 * time.Minute
		before := time.Now().UnixMilli()
		uc := NewIssueSessionUseCase(
			IssueSessionRepositories{Session: repo},
			IssueSessionServices{
				Translator:  newKeyEchoTranslator(),
				IDGenerator: stubIDGenerator{id: stubID},
				Expiry:      SessionExpiryConfig{Duration: customTTL},
			},
		)
		resp, err := uc.Execute(context.Background(), &authpb.IssueSessionRequest{UserId: userID})
		after := time.Now().UnixMilli()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ttlMs := customTTL.Milliseconds()
		if resp.GetExpiresAtUnixMs() < before+ttlMs-1000 || resp.GetExpiresAtUnixMs() > after+ttlMs+1000 {
			t.Errorf("ExpiresAtUnixMs %d outside expected window for custom TTL %s",
				resp.GetExpiresAtUnixMs(), customTTL)
		}
	})
}
