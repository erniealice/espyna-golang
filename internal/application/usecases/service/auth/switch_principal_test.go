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

// TestSwitchPrincipal_Execute_DenyPathRegressionGate is W1 (Layer 1) of the
// plan-3 "principal-switch membership-authz regression gate"
// (docs/plan/20260530-authz-workspace-hardening/, findings test-coverage-2 +
// test-coverage-5). It LOCKS the use-case half of the DENY path: when the
// adapter refuses a switch — because a switched/delegated principal (acting_as_*)
// tried to rotate into a binding it lacks — Execute must surface a DENY
// translator key AND must NOT leak a rotated session (no NewToken, no
// RedirectUrl, nil response). It is read-only over the deny code; it only
// pins behaviour so a future regression cannot ship green.
//
// The three negatives mirror the three real adapter deny signatures emitted by
// contrib/postgres/internal/adapter/entity/session_switch_principal.go:
//
//  1. forged/foreign target_principal OR revoked binding (active=false) — the
//     lockTargetBinding sql.ErrNoRows branch (:611-613) returns
//     "binding revoked or not in workspace" → auth.switch_principal.binding_revoked.
//  2. delegate acting_as_* id the delegate has no grant for — the in-process
//     actingAsTargetIDsContain guard (:157-160 / :177-180) returns
//     "...is not in the resolved binding's targets" → auth.switch_principal.acting_as_mismatch.
//  3. an explicitly cross-workspace / foreign-target binding lock miss — also
//     surfaces via the binding_revoked signature (the COALESCE workspace
//     predicate + user_id=$2 leg that turns a forged /as/ URL into ErrNoRows).
//
// RED proof: if translateSwitchPrincipalError dropped a case (regressed to the
// default "failed" allow-through bucket), the deny-key assertion below fails.
// The "no rotation leaked" assertions fail if Execute ever returned a non-nil
// response (a minted token / redirect) on an adapter deny.
func TestSwitchPrincipal_Execute_DenyPathRegressionGate(t *testing.T) {
	const (
		userID    = "user-1"
		wsID      = "ws-1"
		foreignWS = "ws-foreign"
	)
	goodPrincipal := &authpb.Principal{
		Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER,
		PrincipalId: "wsu-1",
		WorkspaceId: wsID,
	}
	delegatePrincipal := &authpb.Principal{
		Type:        principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
		PrincipalId: "delegate-1",
		WorkspaceId: wsID,
		ActingAsTargets: []*authpb.ActingAsTarget{
			{Id: "client-granted", WorkspaceId: wsID},
		},
	}

	denials := []struct {
		name        string
		req         *authpb.SwitchPrincipalRequest
		adapterErr  error
		wantDenyKey string
	}{
		{
			// Negative 1: forged target_principal owned by a DIFFERENT user, or
			// a binding the acting user is not a member of — the adapter's
			// user_id=$2/active=true FOR UPDATE lock returns ErrNoRows. Dropping
			// the user_id=$2 predicate in the adapter is exactly what would let
			// this through; here the adapter has already denied and we assert
			// the use case translates it to the binding_revoked DENY key.
			name: "forged_or_foreign_target_principal_denied_binding_revoked",
			req:  &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal},
			adapterErr: errors.New(
				"session adapter: SwitchPrincipal: binding revoked or not in workspace " +
					"(type=operator_owner principal_id=wsu-foreign workspace_id=" + foreignWS + ")"),
			wantDenyKey: "auth.switch_principal.binding_revoked",
		},
		{
			// Negative 2: CLIENT_DELEGATE supplies an acting_as_client_id that
			// is NOT in the resolved binding's ActingAsTargets — the in-process
			// fail-closed guard (actingAsTargetIDsContain == false) fires before
			// any tx write.
			name: "delegate_ungranted_acting_as_id_denied_acting_as_mismatch",
			req: &authpb.SwitchPrincipalRequest{
				UserId:           userID,
				TargetPrincipal:  delegatePrincipal,
				ActingAsClientId: "client-NOT-granted",
			},
			adapterErr: errors.New(
				`session adapter: SwitchPrincipal: requested acting_as_client_id "client-NOT-granted" ` +
					`is not in the resolved binding's targets (delegate=delegate-1, available=client-granted)`),
			wantDenyKey: "auth.switch_principal.acting_as_mismatch",
		},
		{
			// Negative 3: revoked binding (active=false) — same adapter
			// signature as negative 1 but driven by a since-revoked grant
			// rather than a foreign one; the COALESCE(...workspace_id...)=$4 +
			// active=true predicates collapse both into the same deny.
			name: "revoked_binding_denied_binding_revoked",
			req:  &authpb.SwitchPrincipalRequest{UserId: userID, TargetPrincipal: goodPrincipal},
			adapterErr: errors.New(
				"session adapter: SwitchPrincipal: binding revoked or not in workspace " +
					"(type=operator_owner principal_id=wsu-1 workspace_id=" + wsID + ")"),
			wantDenyKey: "auth.switch_principal.binding_revoked",
		},
	}

	for _, tc := range denials {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeSessionSwitchAdapter{err: tc.adapterErr}
			uc := NewSwitchPrincipalUseCase(
				SwitchPrincipalRepositories{SessionSwitch: fake},
				SwitchPrincipalServices{Translator: newKeyEchoTranslator()},
			)

			resp, err := uc.Execute(context.Background(), tc.req)

			// (1) The switch must be REFUSED with the expected DENY key.
			if err == nil {
				t.Fatalf("DENY regression: expected deny error %q, got nil (resp=%+v) — "+
					"a switched/delegated principal was allowed into a binding it lacks",
					tc.wantDenyKey, resp)
			}
			if !strings.Contains(err.Error(), tc.wantDenyKey) {
				t.Errorf("DENY regression: error %q does not carry deny key %q "+
					"(did translateSwitchPrincipalError drop a case to the generic 'failed' bucket?)",
					err.Error(), tc.wantDenyKey)
			}
			// Must NOT be the generic catch-all — a deny that decays to
			// auth.switch_principal.failed would still look like an error but
			// loses the specific deny semantics the gate guards.
			if tc.wantDenyKey != "auth.switch_principal.failed" &&
				strings.Contains(err.Error(), "auth.switch_principal.failed") {
				t.Errorf("DENY regression: deny decayed to the generic 'failed' bucket: %q", err.Error())
			}

			// (2) No rotation may leak through on deny: nil response, hence no
			//     NewToken and no RedirectUrl. This is the load-bearing half of
			//     the gate — a deny that still returned a minted token/redirect
			//     would be a switch-into-unauthorized-binding bypass.
			if resp != nil {
				t.Fatalf("DENY regression: deny must return a nil response (no token rotation), got %+v", resp)
			}

			// (3) The adapter was actually consulted exactly once — the deny is
			//     the adapter's decision, not a pre-adapter validation
			//     short-circuit (which would make this a vacuous gate).
			if fake.calls != 1 {
				t.Errorf("expected adapter to be consulted exactly once for the deny, got calls=%d", fake.calls)
			}
		})
	}
}
