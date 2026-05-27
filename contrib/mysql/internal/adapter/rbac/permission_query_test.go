//go:build mysql

package rbac

import (
	"strings"
	"testing"
)

// TestBindingCTEForKind locks the per-binding CTE selection that closes the A2
// / WKR-P0-2 silent-elevation hole. Each PrincipalType integer MUST map to a
// single grant-chain CTE; UNSPECIFIED and out-of-range values MUST fail closed.
//
// MySQL dialect note: predicates use ? instead of $N, and boolean literals use
// 1 instead of true. The structural invariants (table containment / exclusion)
// are identical to the postgres gold standard.
func TestBindingCTEForKind(t *testing.T) {
	cases := []struct {
		name           string
		kind           int32
		wantOK         bool
		mustContain    []string
		mustNotContain []string
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

// TestUserRolesUnionCTE_BackwardsCompatPath documents that the legacy union CTE
// — preserved for the zero-hint fall-back — still touches all five grant chains.
func TestUserRolesUnionCTE_BackwardsCompatPath(t *testing.T) {
	want := []string{
		"workspace_user_role",
		"client_portal_grant",
		"supplier_portal_grant",
		"delegate_client",
		"delegate_supplier",
	}
	for _, w := range want {
		if !strings.Contains(userRolesUnionCTE, w) {
			t.Errorf("legacy union CTE missing %q — fall-back path changed shape", w)
		}
	}
}

// TestBuildPermissionQuerySQL_GrantRowPredicates is the load-bearing test for
// the A2-P0-1 / A2-P1-1 / A2-P1-3 fixes. It exercises the pure SQL-builder
// helper with no live DB.
//
// MySQL dialect: predicates use ? (no $N numbering), so we assert table-level
// containment rather than specific positional markers. We also check the arg
// vector length and values, which encode the same security guarantee.
func TestBuildPermissionQuerySQL_GrantRowPredicates(t *testing.T) {
	const (
		userID     = "user-1"
		wsID       = "ws-1"
		bindingID  = "bind-1"
		clientID   = "client-1"
		supplierID = "supplier-1"
	)
	cases := []struct {
		name           string
		kind           int32
		bindingID      string
		actingAsClient string
		actingAsSup    string
		// mustContain: table/predicate substrings the assembled SQL MUST contain
		mustContain []string
		// mustNotLeak: table names from OTHER grant chains
		mustNotLeak []string
		// wantArgLen: expected number of bound args
		wantArgLen int
		// wantArgBindingID: position in args slice where bindingID should appear
		// (for per-binding CTEs it is always args[0])
		wantArgs []any
	}{
		{
			name:        "operator_owner_scopes_to_workspace_user_chain",
			kind:        principalTypeOperatorOwner,
			bindingID:   bindingID,
			mustContain: []string{"workspace_user", "workspace_user_role"},
			mustNotLeak: []string{
				"client_portal_grant", "supplier_portal_grant",
				"delegate_client", "delegate_supplier",
			},
			wantArgLen: 3,
			wantArgs:   []any{bindingID, userID, wsID},
		},
		{
			name:        "operator_staff_scopes_to_workspace_user_chain",
			kind:        principalTypeOperatorStaff,
			bindingID:   bindingID,
			mustContain: []string{"workspace_user", "workspace_user_role"},
			mustNotLeak: []string{
				"client_portal_grant", "supplier_portal_grant",
				"delegate_client", "delegate_supplier",
			},
			wantArgLen: 3,
			wantArgs:   []any{bindingID, userID, wsID},
		},
		{
			name:        "client_scopes_to_client_portal_grant_chain",
			kind:        principalTypeClient,
			bindingID:   bindingID,
			mustContain: []string{"client_portal_grant"},
			mustNotLeak: []string{
				"workspace_user_role", "supplier_portal_grant",
				"delegate_client", "delegate_supplier",
			},
			wantArgLen: 3,
			wantArgs:   []any{bindingID, userID, wsID},
		},
		{
			name:        "supplier_scopes_to_supplier_portal_grant_chain",
			kind:        principalTypeSupplier,
			bindingID:   bindingID,
			mustContain: []string{"supplier_portal_grant"},
			mustNotLeak: []string{
				"workspace_user_role", "client_portal_grant",
				"delegate_client", "delegate_supplier",
			},
			wantArgLen: 3,
			wantArgs:   []any{bindingID, userID, wsID},
		},
		{
			name:           "client_delegate_scopes_to_delegate_client_chain_with_acting_as",
			kind:           principalTypeClientDelegate,
			bindingID:      bindingID,
			actingAsClient: clientID,
			mustContain:    []string{"delegate", "delegate_client", "dc.client_id"},
			mustNotLeak: []string{
				"workspace_user_role", "client_portal_grant",
				"supplier_portal_grant", "delegate_supplier",
			},
			wantArgLen: 4,
			wantArgs:   []any{bindingID, clientID, userID, wsID},
		},
		{
			name:        "supplier_delegate_scopes_to_delegate_supplier_chain_with_acting_as",
			kind:        principalTypeSupplierDelegate,
			bindingID:   bindingID,
			actingAsSup: supplierID,
			mustContain: []string{"delegate", "delegate_supplier", "ds.supplier_id"},
			mustNotLeak: []string{
				"workspace_user_role", "client_portal_grant",
				"supplier_portal_grant", "delegate_client",
			},
			wantArgLen: 4,
			wantArgs:   []any{bindingID, supplierID, userID, wsID},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt, args, ok := buildPermissionQuerySQL(
				userID, wsID, tc.kind, tc.bindingID,
				tc.actingAsClient, tc.actingAsSup,
			)
			if !ok {
				t.Fatalf("ok=false for well-formed hint; expected SQL to be assembled")
			}
			for _, want := range tc.mustContain {
				if !strings.Contains(stmt, want) {
					t.Errorf("missing required substring %q\nstmt:\n%s", want, stmt)
				}
			}
			for _, banned := range tc.mustNotLeak {
				if strings.Contains(stmt, banned) {
					t.Errorf("SQL leaked other-chain substring %q (would re-introduce elevation bug)\nstmt:\n%s",
						banned, stmt)
				}
			}
			if len(args) != tc.wantArgLen {
				t.Fatalf("args length: want %d, got %d (args=%v)", tc.wantArgLen, len(args), args)
			}
			for i, want := range tc.wantArgs {
				if args[i] != want {
					t.Errorf("args[%d]: want %v, got %v", i, want, args[i])
				}
			}
			// Every assembled stmt should include the DENY-wins permissionSelect tail.
			if !strings.Contains(stmt, "permission_code") {
				t.Errorf("assembled SQL missing permissionSelect tail\nstmt:\n%s", stmt)
			}
			if !strings.Contains(stmt, "PERMISSION_TYPE_ALLOW") || !strings.Contains(stmt, "PERMISSION_TYPE_DENY") {
				t.Errorf("assembled SQL missing ALLOW/DENY predicate\nstmt:\n%s", stmt)
			}
		})
	}
}

// TestBuildPermissionQuerySQL_FailClosedPaths is the codex A2-P1-1 + A2-P0-1
// regression test. Every malformed / partial / under-specified hint shape MUST
// return ok=false.
func TestBuildPermissionQuerySQL_FailClosedPaths(t *testing.T) {
	const (
		userID = "user-1"
		wsID   = "ws-1"
	)
	cases := []struct {
		name           string
		kind           int32
		bindingID      string
		actingAsClient string
		actingAsSup    string
		wantOK         bool
		wantUnion      bool
	}{
		{
			name:      "exact_legacy_zero_pair_unions",
			kind:      principalTypeUnspecified,
			bindingID: "",
			wantOK:    true,
			wantUnion: true,
		},
		{
			name:      "partial_client_no_id_fails_closed",
			kind:      principalTypeClient,
			bindingID: "",
			wantOK:    false,
		},
		{
			name:      "partial_operator_no_id_fails_closed",
			kind:      principalTypeOperatorStaff,
			bindingID: "",
			wantOK:    false,
		},
		{
			name:      "unspecified_with_id_fails_closed",
			kind:      principalTypeUnspecified,
			bindingID: "cpg-1",
			wantOK:    false,
		},
		{
			name:      "out_of_range_high_fails_closed",
			kind:      99,
			bindingID: "x",
			wantOK:    false,
		},
		{
			name:      "out_of_range_high_no_id_fails_closed",
			kind:      99,
			bindingID: "",
			wantOK:    false,
		},
		{
			name:      "out_of_range_negative_fails_closed",
			kind:      -1,
			bindingID: "x",
			wantOK:    false,
		},
		{
			name:           "client_delegate_without_acting_as_fails_closed",
			kind:           principalTypeClientDelegate,
			bindingID:      "delegate-1",
			actingAsClient: "",
			wantOK:         false,
		},
		{
			name:        "supplier_delegate_without_acting_as_fails_closed",
			kind:        principalTypeSupplierDelegate,
			bindingID:   "delegate-1",
			actingAsSup: "",
			wantOK:      false,
		},
		{
			// Codex A2-P1-1 round 2: zero kind/id with non-empty acting_as is NOT
			// the legacy zero quadruple — must fail closed.
			name:           "zero_kind_id_with_acting_as_client_fails_closed",
			kind:           principalTypeUnspecified,
			bindingID:      "",
			actingAsClient: "client-1",
			wantOK:         false,
		},
		{
			name:        "zero_kind_id_with_acting_as_supplier_fails_closed",
			kind:        principalTypeUnspecified,
			bindingID:   "",
			actingAsSup: "supplier-1",
			wantOK:      false,
		},
		{
			name:           "client_delegate_with_acting_as_ok",
			kind:           principalTypeClientDelegate,
			bindingID:      "delegate-1",
			actingAsClient: "client-1",
			wantOK:         true,
			wantUnion:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt, args, ok := buildPermissionQuerySQL(
				userID, wsID, tc.kind, tc.bindingID,
				tc.actingAsClient, tc.actingAsSup,
			)
			if ok != tc.wantOK {
				t.Fatalf("ok: want %v, got %v (stmt=%q args=%v)", tc.wantOK, ok, stmt, args)
			}
			if !ok {
				if stmt != "" || args != nil {
					t.Errorf("fail-closed path should produce empty stmt/nil args; got stmt=%q args=%v", stmt, args)
				}
				return
			}
			if tc.wantUnion {
				// Legacy union CTE must touch ALL five grant chains.
				for _, mustHave := range []string{
					"workspace_user_role",
					"client_portal_grant",
					"supplier_portal_grant",
					"delegate_client",
					"delegate_supplier",
				} {
					if !strings.Contains(stmt, mustHave) {
						t.Errorf("legacy union path missing %q (shape changed)\nstmt:\n%s", mustHave, stmt)
					}
				}
				// Union CTE uses 10 args (5 chains × 2 per chain).
				if len(args) != 10 {
					t.Errorf("legacy union should have 10 args, got %d (%v)", len(args), args)
				}
			} else {
				// Per-binding scoped path: must NOT carry every chain's table.
				if strings.Count(stmt, "UNION") > 0 {
					t.Errorf("per-binding scoped path should have zero UNIONs; got %d\nstmt:\n%s",
						strings.Count(stmt, "UNION"), stmt)
				}
			}
		})
	}
}

// TestBuildPermissionQuerySQL_DelegateLeakageRegression is the explicit
// regression test for A2-P0-1: two distinct acting-as targets MUST produce
// two distinct arg vectors for the same SQL template.
func TestBuildPermissionQuerySQL_DelegateLeakageRegression(t *testing.T) {
	const (
		userID     = "user-1"
		wsID       = "ws-1"
		delegateID = "delegate-1"
		clientA    = "client-A"
		clientB    = "client-B"
	)

	stmtA, argsA, okA := buildPermissionQuerySQL(
		userID, wsID, principalTypeClientDelegate, delegateID, clientA, "")
	if !okA {
		t.Fatal("clientA lookup should be ok=true")
	}
	stmtB, argsB, okB := buildPermissionQuerySQL(
		userID, wsID, principalTypeClientDelegate, delegateID, clientB, "")
	if !okB {
		t.Fatal("clientB lookup should be ok=true")
	}

	// Same statement template but different actingAs arg (args[1] = actingAsClientID).
	if stmtA != stmtB {
		t.Errorf("delegate SQL template should be identical between targets; differs")
	}
	if argsA[1] == argsB[1] {
		t.Errorf("delegate acting-as arg should differ between targets; both = %v", argsA[1])
	}
	if argsA[1] != clientA {
		t.Errorf("clientA args[1]: want %q, got %v", clientA, argsA[1])
	}
	if argsB[1] != clientB {
		t.Errorf("clientB args[1]: want %q, got %v", clientB, argsB[1])
	}

	// Crucial leakage guard: SQL must filter by dc.client_id (the per-target
	// predicate). Without it the parent-only filter would union both target rows.
	if !strings.Contains(stmtA, "dc.client_id") {
		t.Errorf("CLIENT_DELEGATE SQL missing dc.client_id predicate (would union all targets)\nstmt:\n%s", stmtA)
	}
}
