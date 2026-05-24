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

// TestBuildPermissionQuerySQL_GrantRowPredicates is the load-bearing test
// for the codex A2-P0-1 / A2-P1-1 / A2-P1-3 fixes. It exercises the
// pure SQL-builder helper with no live DB and asserts:
//
//  1. Each binding kind's full assembled SQL contains the per-binding CTE
//     name AND the expected grant-row predicate (e.g. `cpg.id = $3`,
//     `dc.client_id = $4`).
//  2. The SQL does NOT leak other chains' table names — a CLIENT lookup
//     must not touch workspace_user_role, etc.
//  3. The arg vector is exactly what the predicate references (3 args
//     for operator/client/supplier, 4 args for the two delegate kinds).
//  4. ok=true for every well-formed shape.
func TestBuildPermissionQuerySQL_GrantRowPredicates(t *testing.T) {
	const (
		userID     = "user-1"
		wsID       = "ws-1"
		bindingID  = "bind-1"
		clientID   = "client-1"
		supplierID = "supplier-1"
	)
	cases := []struct {
		name            string
		kind            int32
		bindingID       string
		actingAsClient  string
		actingAsSup     string
		wantPredicate   string   // substring the assembled SQL MUST contain
		wantArgs        []any    // exact arg vector
		mustNotLeak     []string // tables/predicates from OTHER chains
		extraPredicates []string // additional load-bearing predicates beyond the row id
	}{
		{
			name:          "operator_owner_scopes_to_workspace_user_row",
			kind:          principalTypeOperatorOwner,
			bindingID:     bindingID,
			wantPredicate: "wu.id = $3",
			wantArgs:      []any{userID, wsID, bindingID},
			mustNotLeak: []string{
				"client_portal_grant", "supplier_portal_grant",
				"delegate_client", "delegate_supplier",
			},
			extraPredicates: []string{"wu.user_id = $1", "wu.workspace_id = $2"},
		},
		{
			name:          "operator_staff_scopes_to_workspace_user_row",
			kind:          principalTypeOperatorStaff,
			bindingID:     bindingID,
			wantPredicate: "wu.id = $3",
			wantArgs:      []any{userID, wsID, bindingID},
			mustNotLeak: []string{
				"client_portal_grant", "supplier_portal_grant",
				"delegate_client", "delegate_supplier",
			},
		},
		{
			name:          "client_scopes_to_client_portal_grant_row",
			kind:          principalTypeClient,
			bindingID:     bindingID,
			wantPredicate: "cpg.id = $3",
			wantArgs:      []any{userID, wsID, bindingID},
			mustNotLeak: []string{
				"workspace_user_role", "supplier_portal_grant",
				"delegate_client", "delegate_supplier",
			},
		},
		{
			name:          "supplier_scopes_to_supplier_portal_grant_row",
			kind:          principalTypeSupplier,
			bindingID:     bindingID,
			wantPredicate: "spg.id = $3",
			wantArgs:      []any{userID, wsID, bindingID},
			mustNotLeak: []string{
				"workspace_user_role", "client_portal_grant",
				"delegate_client", "delegate_supplier",
			},
		},
		{
			name:           "client_delegate_scopes_to_delegate_client_row_with_acting_as",
			kind:           principalTypeClientDelegate,
			bindingID:      bindingID,
			actingAsClient: clientID,
			wantPredicate:  "dc.client_id = $4",
			wantArgs:       []any{userID, wsID, bindingID, clientID},
			mustNotLeak: []string{
				"workspace_user_role", "client_portal_grant",
				"supplier_portal_grant", "delegate_supplier",
			},
			extraPredicates: []string{"d.id = $3", "d.user_id = $1"},
		},
		{
			name:          "supplier_delegate_scopes_to_delegate_supplier_row_with_acting_as",
			kind:          principalTypeSupplierDelegate,
			bindingID:     bindingID,
			actingAsSup:   supplierID,
			wantPredicate: "ds.supplier_id = $4",
			wantArgs:      []any{userID, wsID, bindingID, supplierID},
			mustNotLeak: []string{
				"workspace_user_role", "client_portal_grant",
				"supplier_portal_grant", "delegate_client",
			},
			extraPredicates: []string{"d.id = $3", "d.user_id = $1"},
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
			if !strings.Contains(stmt, tc.wantPredicate) {
				t.Errorf("missing grant-row predicate %q\nstmt:\n%s", tc.wantPredicate, stmt)
			}
			for _, extra := range tc.extraPredicates {
				if !strings.Contains(stmt, extra) {
					t.Errorf("missing required predicate %q\nstmt:\n%s", extra, stmt)
				}
			}
			for _, banned := range tc.mustNotLeak {
				if strings.Contains(stmt, banned) {
					t.Errorf("SQL leaked other-chain substring %q (would re-introduce elevation bug)\nstmt:\n%s",
						banned, stmt)
				}
			}
			if len(args) != len(tc.wantArgs) {
				t.Fatalf("args length: want %d, got %d (args=%v)", len(tc.wantArgs), len(args), args)
			}
			for i := range args {
				if args[i] != tc.wantArgs[i] {
					t.Errorf("args[%d]: want %v, got %v", i, tc.wantArgs[i], args[i])
				}
			}
			// Every assembled stmt should also include the DENY-wins
			// permissionSelect tail, otherwise the use case would
			// return roles instead of permission codes.
			if !strings.Contains(stmt, "permission_code") {
				t.Errorf("assembled SQL missing permissionSelect tail (no permission_code reference)\nstmt:\n%s", stmt)
			}
			if !strings.Contains(stmt, "PERMISSION_TYPE_ALLOW") || !strings.Contains(stmt, "PERMISSION_TYPE_DENY") {
				t.Errorf("assembled SQL missing ALLOW/DENY predicate\nstmt:\n%s", stmt)
			}
		})
	}
}

// TestBuildPermissionQuerySQL_FailClosedPaths is the codex A2-P1-1 +
// A2-P0-1 regression test. Every malformed / partial / under-specified
// hint shape MUST return ok=false (caller treats as empty permission
// set). The exact legacy zero pair is the ONLY shape that may union.
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
		wantOK         bool // true => SQL assembled; false => fail closed (empty)
		wantUnion      bool // when ok=true, must the SQL be the legacy union?
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
				if len(args) != 2 {
					t.Errorf("legacy union should have 2 args, got %d (%v)", len(args), args)
				}
			} else {
				// Per-binding scoped path: must NOT carry every chain's table.
				// Spot-check: there should NOT be a UNION between user_roles members.
				if strings.Count(stmt, "UNION") > 0 {
					t.Errorf("per-binding scoped path should have zero UNIONs; got %d\nstmt:\n%s",
						strings.Count(stmt, "UNION"), stmt)
				}
			}
		})
	}
}

// TestBuildPermissionQuerySQL_DelegateLeakageRegression is the explicit
// regression test for codex A2-P0-1: the parent delegate row only
// anchors user_id/active; the role grant lives on the per-target
// delegate_client / delegate_supplier row. Two distinct acting-as
// targets MUST produce two distinct SQL arg vectors (NOT a union over
// every target the delegate owns).
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

	// Same statement template (same CTE) but different $4 args.
	if stmtA != stmtB {
		t.Errorf("delegate SQL template should be identical between targets; differs")
	}
	if argsA[3] == argsB[3] {
		t.Errorf("delegate $4 should differ between targets; both = %v", argsA[3])
	}
	if argsA[3] != clientA {
		t.Errorf("clientA arg[3]: want %q, got %v", clientA, argsA[3])
	}
	if argsB[3] != clientB {
		t.Errorf("clientB arg[3]: want %q, got %v", clientB, argsB[3])
	}

	// Crucial leakage guard: the SQL MUST filter by dc.client_id = $4 —
	// without that predicate, the parent-only filter would union both
	// target rows.
	if !strings.Contains(stmtA, "dc.client_id = $4") {
		t.Errorf("CLIENT_DELEGATE SQL missing dc.client_id = $4 predicate (would union all targets)\nstmt:\n%s", stmtA)
	}
}
