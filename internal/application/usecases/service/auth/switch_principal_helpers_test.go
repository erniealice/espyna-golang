package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

func TestHomeRouteForProtoPrincipal(t *testing.T) {
	const (
		clientA   = "client-a"
		clientB   = "client-b"
		supplierA = "supplier-a"
		supplierB = "supplier-b"
	)
	cases := []struct {
		name string
		p    *authpb.Principal
		want string
	}{
		{
			name: "nil_principal_falls_back_to_no_access",
			p:    nil,
			want: "/auth/no-access",
		},
		{
			name: "unspecified_falls_back_to_no_access",
			p:    &authpb.Principal{Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_UNSPECIFIED},
			want: "/auth/no-access",
		},
		{
			name: "operator_owner_lands_on_me_inbox",
			p:    &authpb.Principal{Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER},
			want: "/me/inbox",
		},
		{
			name: "operator_staff_lands_on_me_inbox",
			p:    &authpb.Principal{Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF},
			want: "/me/inbox",
		},
		{
			name: "client_lands_on_me_inbox",
			p:    &authpb.Principal{Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT},
			want: "/me/inbox",
		},
		{
			name: "supplier_lands_on_me_inbox",
			p:    &authpb.Principal{Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER},
			want: "/me/inbox",
		},
		{
			name: "client_delegate_with_zero_targets_routes_to_select",
			p: &authpb.Principal{
				Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
			},
			want: "/me/inboxselect",
		},
		{
			name: "client_delegate_with_one_target_pre_selects",
			p: &authpb.Principal{
				Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
				ActingAsTargets: []*authpb.ActingAsTarget{
					{Id: clientA},
				},
			},
			want: "/me/inbox?acting_as_client_id=" + clientA,
		},
		{
			name: "client_delegate_with_multiple_targets_routes_to_select",
			p: &authpb.Principal{
				Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
				ActingAsTargets: []*authpb.ActingAsTarget{
					{Id: clientA},
					{Id: clientB},
				},
			},
			want: "/me/inboxselect",
		},
		{
			name: "supplier_delegate_with_zero_targets_routes_to_select",
			p: &authpb.Principal{
				Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
			},
			want: "/me/inboxselect",
		},
		{
			name: "supplier_delegate_with_one_target_pre_selects",
			p: &authpb.Principal{
				Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
				ActingAsTargets: []*authpb.ActingAsTarget{
					{Id: supplierA},
				},
			},
			want: "/me/inbox?acting_as_supplier_id=" + supplierA,
		},
		{
			name: "supplier_delegate_with_multiple_targets_routes_to_select",
			p: &authpb.Principal{
				Type: principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
				ActingAsTargets: []*authpb.ActingAsTarget{
					{Id: supplierA},
					{Id: supplierB},
				},
			},
			want: "/me/inboxselect",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := homeRouteForProtoPrincipal(tc.p)
			if got != tc.want {
				t.Errorf("homeRouteForProtoPrincipal: want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestTranslateSwitchPrincipalError(t *testing.T) {
	ctx := context.Background()
	tr := newKeyEchoTranslator()

	cases := []struct {
		name       string
		raw        error
		wantNil    bool
		wantSubstr string
	}{
		{
			name:    "nil_raw_returns_nil",
			raw:     nil,
			wantNil: true,
		},
		{
			name:       "binding_revoked_maps_to_binding_revoked_key",
			raw:        errors.New("session adapter: SwitchPrincipal: binding revoked or not in workspace (type=X principal_id=Y workspace_id=Z)"),
			wantSubstr: "auth.switch_principal.binding_revoked",
		},
		{
			name:       "read_current_session_maps_to_session_invalid_key",
			raw:        errors.New("session adapter: SwitchPrincipal: read current session: pq: connection refused"),
			wantSubstr: "auth.switch_principal.session_invalid",
		},
		{
			name:       "invalidate_old_session_maps_to_session_invalid_key",
			raw:        errors.New("session adapter: SwitchPrincipal: invalidate old session: pq: deadlock"),
			wantSubstr: "auth.switch_principal.session_invalid",
		},
		{
			name:       "in_place_update_maps_to_session_invalid_key",
			raw:        errors.New("session adapter: SwitchPrincipal: in-place update: pq: serialization failure"),
			wantSubstr: "auth.switch_principal.session_invalid",
		},
		{
			name:       "insert_new_session_maps_to_session_invalid_key",
			raw:        errors.New("session adapter: SwitchPrincipal: insert new session: pq: unique violation"),
			wantSubstr: "auth.switch_principal.session_invalid",
		},
		{
			name:       "acting_as_mismatch_maps_to_acting_as_mismatch_key",
			raw:        errors.New(`session adapter: SwitchPrincipal: requested acting_as_client_id "x" is not in the resolved binding's targets`),
			wantSubstr: "auth.switch_principal.acting_as_mismatch",
		},
		{
			name:       "unsupported_principal_type_maps_to_unsupported_key",
			raw:        errors.New("session adapter: SwitchPrincipal: unsupported principal type for binding lock: PRINCIPAL_TYPE_UNSPECIFIED"),
			wantSubstr: "auth.switch_principal.unsupported_principal_type",
		},
		{
			name:       "unknown_error_falls_through_to_failed_key",
			raw:        errors.New("something totally unexpected"),
			wantSubstr: "auth.switch_principal.failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := translateSwitchPrincipalError(ctx, tr, tc.raw)
			if tc.wantNil {
				if got != nil {
					t.Errorf("translateSwitchPrincipalError: want nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("translateSwitchPrincipalError: want non-nil error containing %q, got nil", tc.wantSubstr)
			}
			if !strings.Contains(got.Error(), tc.wantSubstr) {
				t.Errorf("translateSwitchPrincipalError: %q does not contain %q", got.Error(), tc.wantSubstr)
			}
		})
	}
}
