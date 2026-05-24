//go:build postgresql

package rbac

import (
	"strings"
	"testing"
)

// TestBindingCTEForKind locks the per-binding CTE selection that closes
// the A2 / WKR-P0-2 silent-elevation hole. Each PrincipalType integer
// MUST map to a single grant-chain CTE; UNSPECIFIED and out-of-range
// values MUST fail closed (false).
//
// The behavioural invariant this defends: the SQL the adapter executes
// must reference ONLY the table that owns the binding's grant chain.
// An OPERATOR_STAFF lookup must touch workspace_user / workspace_user_role
// only — never client_portal_grant, supplier_portal_grant, or delegate.
// And vice-versa: a CLIENT lookup must touch client_portal_grant only.
func TestBindingCTEForKind(t *testing.T) {
	cases := []struct {
		name           string
		kind           int32
		wantOK         bool
		mustContain    []string // substrings the CTE MUST contain
		mustNotContain []string // substrings the CTE MUST NOT contain (other grant chains)
	}{
		{
			name:        "operator_owner_picks_workspace_user_chain_only",
			kind:        principalTypeOperatorOwner,
			wantOK:      true,
			mustContain: []string{"workspace_user", "workspace_user_role"},
			mustNotContain: []string{
				"client_portal_grant",
				"supplier_portal_grant",
				"delegate_client",
				"delegate_supplier",
			},
		},
		{
			name:        "operator_staff_picks_workspace_user_chain_only",
			kind:        principalTypeOperatorStaff,
			wantOK:      true,
			mustContain: []string{"workspace_user", "workspace_user_role"},
			mustNotContain: []string{
				"client_portal_grant",
				"supplier_portal_grant",
				"delegate_client",
				"delegate_supplier",
			},
		},
		{
			name:        "client_picks_client_portal_grant_only",
			kind:        principalTypeClient,
			wantOK:      true,
			mustContain: []string{"client_portal_grant"},
			mustNotContain: []string{
				"workspace_user_role",
				"supplier_portal_grant",
				"delegate_client",
				"delegate_supplier",
			},
		},
		{
			name:        "supplier_picks_supplier_portal_grant_only",
			kind:        principalTypeSupplier,
			wantOK:      true,
			mustContain: []string{"supplier_portal_grant"},
			mustNotContain: []string{
				"workspace_user_role",
				"client_portal_grant",
				"delegate_client",
				"delegate_supplier",
			},
		},
		{
			name:        "client_delegate_picks_delegate_client_chain_only",
			kind:        principalTypeClientDelegate,
			wantOK:      true,
			mustContain: []string{"delegate", "delegate_client"},
			mustNotContain: []string{
				"workspace_user_role",
				"client_portal_grant",
				"supplier_portal_grant",
				"delegate_supplier",
			},
		},
		{
			name:        "supplier_delegate_picks_delegate_supplier_chain_only",
			kind:        principalTypeSupplierDelegate,
			wantOK:      true,
			mustContain: []string{"delegate", "delegate_supplier"},
			mustNotContain: []string{
				"workspace_user_role",
				"client_portal_grant",
				"supplier_portal_grant",
				"delegate_client",
			},
		},
		{
			name:   "unspecified_fails_closed",
			kind:   principalTypeUnspecified,
			wantOK: false,
		},
		{
			name:   "out_of_range_high_fails_closed",
			kind:   99,
			wantOK: false,
		},
		{
			name:   "out_of_range_negative_fails_closed",
			kind:   -1,
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cte, ok := bindingCTEForKind(tc.kind)
			if ok != tc.wantOK {
				t.Fatalf("ok: want %v, got %v", tc.wantOK, ok)
			}
			if !tc.wantOK {
				if cte != "" {
					t.Errorf("expected empty CTE for fail-closed kind, got %q", cte)
				}
				return
			}
			for _, want := range tc.mustContain {
				if !strings.Contains(cte, want) {
					t.Errorf("CTE missing required substring %q\nCTE:\n%s", want, cte)
				}
			}
			for _, banned := range tc.mustNotContain {
				if strings.Contains(cte, banned) {
					t.Errorf("CTE leaked other-binding substring %q (would re-introduce the elevation bug)\nCTE:\n%s",
						banned, cte)
				}
			}
		})
	}
}

// TestUserRolesUnionCTE_BackwardsCompatPath documents that the legacy
// union CTE — preserved for the zero-hint fall-back — still touches all
// five grant chains. Production callers post-A2 must NOT reach this
// path (the use case's request_required guard ensures a non-nil
// request, and production wiring always passes the session binding
// hint). This test guards against accidentally rewriting the fall-back
// path to a different shape that would silently re-introduce the
// elevation bug for unhinted callers.
func TestUserRolesUnionCTE_BackwardsCompatPath(t *testing.T) {
	want := []string{
		"workspace_user_role",
		"client_portal_grant",
		"supplier_portal_grant",
		"delegate_client",
		"delegate_supplier",
	}
	for _, want := range want {
		if !strings.Contains(userRolesUnionCTE, want) {
			t.Errorf("legacy union CTE missing %q — fall-back path changed shape", want)
		}
	}
}
