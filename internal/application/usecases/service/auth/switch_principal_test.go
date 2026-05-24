package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

func TestSwitchPrincipal_Execute(t *testing.T) {
	const (
		userID   = "user-1"
		clientID = "client-1"
		wsID     = "ws-1"
	)
	goodPrincipal := &authpb.Principal{
		Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER,
		PrincipalId: "wsu-1",
		WorkspaceId: wsID,
	}

	cases := []struct {
		name           string
		adapter        SessionSwitchAdapter
		req            *authpb.SwitchPrincipalRequest
		wantErrKey     string // substring assertion on the error
		assertCallsZ   bool   // when true, the adapter must not be called
		wantNewToken   string // empty means do not assert
		wantRedirectIn string // substring assertion on RedirectUrl when no error
	}{
		{
			name:         "nil_adapter_fails_closed_service_unavailable",
			adapter:      nil,
			req:          &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal},
			wantErrKey:   "auth.errors.service_unavailable",
			assertCallsZ: true,
		},
		{
			name:         "nil_request_returns_request_required",
			adapter:      &fakeSessionSwitchAdapter{},
			req:          nil,
			wantErrKey:   "auth.validation.request_required",
			assertCallsZ: true,
		},
		{
			name:         "missing_user_id_returns_user_id_required",
			adapter:      &fakeSessionSwitchAdapter{},
			req:          &authpb.SwitchPrincipalRequest{TargetPrincipal: goodPrincipal},
			wantErrKey:   "auth.validation.user_id_required",
			assertCallsZ: true,
		},
		{
			name:         "nil_target_principal_returns_target_principal_required",
			adapter:      &fakeSessionSwitchAdapter{},
			req:          &authpb.SwitchPrincipalRequest{UserId: userID},
			wantErrKey:   "auth.validation.target_principal_required",
			assertCallsZ: true,
		},
		{
			name: "adapter_binding_revoked_translated_to_binding_revoked_key",
			adapter: &fakeSessionSwitchAdapter{
				err: errors.New("session adapter: SwitchPrincipal: binding revoked or not in workspace (type=PRINCIPAL_TYPE_CLIENT principal_id=x workspace_id=y)"),
			},
			req:        &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal},
			wantErrKey: "auth.switch_principal.binding_revoked",
		},
		{
			name: "adapter_session_invalid_translated_to_session_invalid_key",
			adapter: &fakeSessionSwitchAdapter{
				err: errors.New("session adapter: SwitchPrincipal: read current session: sql error"),
			},
			req:        &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal},
			wantErrKey: "auth.switch_principal.session_invalid",
		},
		{
			name: "adapter_acting_as_mismatch_translated_to_acting_as_mismatch_key",
			adapter: &fakeSessionSwitchAdapter{
				err: errors.New(`session adapter: SwitchPrincipal: requested acting_as_client_id "foo" is not in the resolved binding's targets (delegate=d available=bar)`),
			},
			req:        &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal, ActingAsClientId: "foo"},
			wantErrKey: "auth.switch_principal.acting_as_mismatch",
		},
		{
			name: "adapter_unknown_error_translated_to_failed_key",
			adapter: &fakeSessionSwitchAdapter{
				err: errors.New("totally unexpected error"),
			},
			req:        &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal},
			wantErrKey: "auth.switch_principal.failed",
		},
		{
			name: "adapter_nil_response_returns_failed",
			adapter: &fakeSessionSwitchAdapter{
				resp: nil, err: nil,
			},
			req:        &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal},
			wantErrKey: "auth.switch_principal.failed",
		},
		{
			name: "happy_path_returns_response_and_computes_redirect_url",
			adapter: &fakeSessionSwitchAdapter{
				resp: &authpb.SwitchPrincipalResponse{NewToken: "new-token-xyz"},
			},
			req: &authpb.SwitchPrincipalRequest{
				UserId:          userID,
				TargetPrincipal: goodPrincipal,
			},
			wantNewToken:   "new-token-xyz",
			wantRedirectIn: "/me/inbox",
		},
		{
			name: "happy_path_does_not_override_adapter_provided_redirect_url",
			adapter: &fakeSessionSwitchAdapter{
				resp: &authpb.SwitchPrincipalResponse{
					NewToken:    "",
					RedirectUrl: "/custom/path",
				},
			},
			req: &authpb.SwitchPrincipalRequest{
				UserId:          userID,
				TargetPrincipal: goodPrincipal,
			},
			wantRedirectIn: "/custom/path",
		},
		{
			name: "delegate_with_single_target_renders_acting_as_query_param",
			adapter: &fakeSessionSwitchAdapter{
				resp: &authpb.SwitchPrincipalResponse{NewToken: "tok"},
			},
			req: &authpb.SwitchPrincipalRequest{
				UserId: userID,
				TargetPrincipal: &authpb.Principal{
					Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
					PrincipalId: "delegate-1",
					WorkspaceId: wsID,
					ActingAsTargets: []*authpb.ActingAsTarget{
						{Id: clientID, WorkspaceId: wsID},
					},
				},
			},
			wantRedirectIn: "acting_as_client_id=" + clientID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := NewSwitchPrincipalUseCase(
				SwitchPrincipalRepositories{SessionSwitch: tc.adapter},
				SwitchPrincipalServices{Translator: newKeyEchoTranslator()},
			)
			resp, err := uc.Execute(context.Background(), tc.req)

			if tc.wantErrKey != "" {
				if err == nil {
					t.Fatalf("expected error with key %q, got nil (resp=%+v)", tc.wantErrKey, resp)
				}
				if !strings.Contains(err.Error(), tc.wantErrKey) {
					t.Errorf("error %q does not contain expected key %q", err.Error(), tc.wantErrKey)
				}
				if tc.assertCallsZ {
					if fake, ok := tc.adapter.(*fakeSessionSwitchAdapter); ok && fake.calls != 0 {
						t.Errorf("expected adapter to NOT be called (validation must short-circuit); got calls=%d", fake.calls)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatalf("expected non-nil response, got nil")
			}
			if tc.wantNewToken != "" && resp.GetNewToken() != tc.wantNewToken {
				t.Errorf("NewToken: want %q, got %q", tc.wantNewToken, resp.GetNewToken())
			}
			if tc.wantRedirectIn != "" && !strings.Contains(resp.GetRedirectUrl(), tc.wantRedirectIn) {
				t.Errorf("RedirectUrl %q does not contain %q", resp.GetRedirectUrl(), tc.wantRedirectIn)
			}
			if fake, ok := tc.adapter.(*fakeSessionSwitchAdapter); ok {
				if fake.calls != 1 {
					t.Errorf("expected adapter to be called exactly once, got calls=%d", fake.calls)
				}
				if fake.lastReq == nil {
					t.Errorf("expected adapter to record lastReq, got nil")
				}
			}
		})
	}
}
