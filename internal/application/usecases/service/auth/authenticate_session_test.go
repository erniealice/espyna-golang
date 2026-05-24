package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

func TestAuthenticateSession_Execute(t *testing.T) {
	const (
		tokenOK    = "token-valid"
		userID     = "user-1"
		userEmail  = "alice@example.com"
		wsUserID   = "wsu-1"
		wsID       = "ws-1"
		futureSkew = 24 * time.Hour
	)
	future := time.Now().Add(futureSkew).UnixMilli()
	past := time.Now().Add(-futureSkew).UnixMilli()

	cases := []struct {
		name             string
		sessionRepo      sessionpb.SessionDomainServiceServer
		userRepo         userpb.UserDomainServiceServer
		req              *authpb.AuthenticateSessionRequest
		wantErrKey       string // expected translator key substring of the error
		wantIdentityUser string // empty means do not assert identity
		assertReadNZero  bool   // when true, assert sessionRepo.readN == 0 post-Execute (codex round 1 P1-1 regression pin)
	}{
		{
			name:        "nil_session_repo_fails_closed_service_unavailable",
			sessionRepo: nil,
			userRepo:    &fakeUserRepo{},
			req:         &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantErrKey:  "auth.errors.service_unavailable",
		},
		{
			name:        "nil_request_returns_request_required",
			sessionRepo: &fakeSessionRepo{},
			userRepo:    &fakeUserRepo{},
			req:         nil,
			wantErrKey:  "auth.validation.request_required",
		},
		{
			name:        "empty_token_returns_missing_token",
			sessionRepo: &fakeSessionRepo{},
			userRepo:    &fakeUserRepo{},
			req:         &authpb.AuthenticateSessionRequest{Token: ""},
			wantErrKey:  "auth.errors.missing_token",
		},
		{
			name:        "session_lookup_zero_rows_returns_session_invalid",
			sessionRepo: &fakeSessionRepo{readResp: &sessionpb.ReadSessionResponse{}},
			userRepo:    &fakeUserRepo{},
			req:         &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantErrKey:  "auth.errors.session_invalid",
		},
		{
			name: "session_lookup_error_returns_session_invalid",
			sessionRepo: &fakeSessionRepo{
				readErr: errors.New("boom"),
			},
			userRepo:   &fakeUserRepo{},
			req:        &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantErrKey: "auth.errors.session_invalid",
		},
		{
			name: "inactive_session_returns_session_inactive",
			sessionRepo: &fakeSessionRepo{
				readResp: &sessionpb.ReadSessionResponse{Data: []*sessionpb.Session{{
					Id: "s-1", UserId: userID, Token: tokenOK, ExpiresAt: future, Active: false,
				}}},
			},
			userRepo:   &fakeUserRepo{},
			req:        &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantErrKey: "auth.errors.session_inactive",
		},
		{
			name: "expired_session_returns_session_expired",
			sessionRepo: &fakeSessionRepo{
				readResp: &sessionpb.ReadSessionResponse{Data: []*sessionpb.Session{{
					Id: "s-1", UserId: userID, Token: tokenOK, ExpiresAt: past, Active: true,
				}}},
			},
			userRepo:   &fakeUserRepo{},
			req:        &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantErrKey: "auth.errors.session_expired",
		},
		{
			// Codex round 1 P1-1 regression: with User nil, the use case must
			// fail closed at body entry BEFORE calling Session.ReadSession.
			// readN==0 post-Execute proves the short-circuit.
			name:            "nil_user_repo_fails_closed_at_entry_no_session_read",
			sessionRepo:     &fakeSessionRepo{},
			userRepo:        nil,
			req:             &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantErrKey:      "auth.errors.service_unavailable",
			assertReadNZero: true,
		},
		{
			name: "missing_user_row_returns_session_user_missing",
			sessionRepo: &fakeSessionRepo{
				readResp: &sessionpb.ReadSessionResponse{Data: []*sessionpb.Session{{
					Id: "s-1", UserId: userID, Token: tokenOK, ExpiresAt: future, Active: true,
				}}},
			},
			userRepo:   &fakeUserRepo{readResp: &userpb.ReadUserResponse{}},
			req:        &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantErrKey: "auth.errors.session_user_missing",
		},
		{
			name: "happy_path_returns_identity_with_workspace_fields",
			sessionRepo: &fakeSessionRepo{
				readResp: &sessionpb.ReadSessionResponse{Data: []*sessionpb.Session{{
					Id: "s-1", UserId: userID, Token: tokenOK, ExpiresAt: future, Active: true,
					WorkspaceUserId: stringPtr(wsUserID),
					WorkspaceId:     stringPtr(wsID),
				}}},
			},
			userRepo: &fakeUserRepo{readResp: &userpb.ReadUserResponse{
				Data: []*userpb.User{{Id: userID, EmailAddress: userEmail}},
			}},
			req:              &authpb.AuthenticateSessionRequest{Token: tokenOK},
			wantIdentityUser: userID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := NewAuthenticateSessionUseCase(
				AuthenticateSessionRepositories{Session: tc.sessionRepo, User: tc.userRepo},
				AuthenticateSessionServices{Translator: newKeyEchoTranslator()},
			)
			resp, err := uc.Execute(context.Background(), tc.req)

			if tc.wantErrKey != "" {
				if err == nil {
					t.Fatalf("expected error with key %q, got nil (resp=%+v)", tc.wantErrKey, resp)
				}
				if !strings.Contains(err.Error(), tc.wantErrKey) {
					t.Errorf("error %q does not contain expected key %q", err.Error(), tc.wantErrKey)
				}
				if tc.assertReadNZero {
					fake, ok := tc.sessionRepo.(*fakeSessionRepo)
					if !ok {
						t.Fatalf("assertReadNZero requires sessionRepo to be *fakeSessionRepo; got %T", tc.sessionRepo)
					}
					if fake.readN != 0 {
						t.Errorf("expected Session.ReadSession to NOT be called (fail-closed at body entry); got readN=%d", fake.readN)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil || resp.GetIdentity() == nil {
				t.Fatalf("expected non-nil response + identity, got resp=%+v", resp)
			}
			id := resp.GetIdentity()
			if id.GetUserId() != tc.wantIdentityUser {
				t.Errorf("UserId: want %q, got %q", tc.wantIdentityUser, id.GetUserId())
			}
			if id.GetEmail() != userEmail {
				t.Errorf("Email: want %q, got %q", userEmail, id.GetEmail())
			}
			if id.GetWorkspaceUserId() != wsUserID {
				t.Errorf("WorkspaceUserId: want %q, got %q", wsUserID, id.GetWorkspaceUserId())
			}
			if id.GetWorkspaceId() != wsID {
				t.Errorf("WorkspaceId: want %q, got %q", wsID, id.GetWorkspaceId())
			}
			if id.GetToken() != tokenOK {
				t.Errorf("Token: want %q, got %q", tokenOK, id.GetToken())
			}
			if id.GetExpiresAtUnixMs() != future {
				t.Errorf("ExpiresAtUnixMs: want %d, got %d", future, id.GetExpiresAtUnixMs())
			}
		})
	}
}
