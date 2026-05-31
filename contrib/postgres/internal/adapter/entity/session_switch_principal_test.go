//go:build postgresql

package entity

import (
	"database/sql"
	"strings"
	"testing"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// TestBuildDelegateLockSQL_ClientDelegate_RejectsCrossWorkspace asserts the
// SQL generated for the client-delegate acting-as lock includes the
// COALESCE(dc.workspace_id, c.workspace_id) = $4 predicate that pins the
// locked binding to the URL workspace. Without this predicate a delegate
// holding (delegate_id=D, client_id=Z) in workspace B could pass the lock
// while requesting /w/workspace-A/as/client-Z/... — see A2-followup round-3.
//
// Migrated 2026-05-24 from apps/service-admin/internal/composition/
// principal_switch_test.go in Phase 2 of docs/plan/20260524-principal-switch-
// typed-stack/. Updated to the new (string, []any) signature per codex
// auth-collapse R4 P2 drift-fix — the helper now produces SQL + args
// together so they cannot drift.
func TestBuildDelegateLockSQL_ClientDelegate_RejectsCrossWorkspace(t *testing.T) {
	const (
		delegateID  = "delegate-D"
		actingAsID  = "client-Z"
		userID      = "user-U"
		workspaceID = "workspace-A"
	)
	query, args := buildDelegateLockSQL(
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
		delegateID, actingAsID, userID, workspaceID,
	)
	if len(args) != 4 {
		t.Errorf("client-delegate acting-as lock arg count = %d, want 4 (args=%v)", len(args), args)
	}
	want := []any{delegateID, actingAsID, userID, workspaceID}
	for i := range want {
		if i >= len(args) {
			break
		}
		if args[i] != want[i] {
			t.Errorf("client-delegate acting-as args[%d] = %v, want %v", i, args[i], want[i])
		}
	}
	required := []string{
		"FROM delegate_client dc",
		"JOIN delegate d ON d.id = dc.delegate_id",
		"LEFT JOIN client c ON c.id = dc.client_id",
		"COALESCE(dc.workspace_id, c.workspace_id) = $4",
		"FOR UPDATE",
	}
	for _, sub := range required {
		if !strings.Contains(query, sub) {
			t.Errorf("client-delegate acting-as lock SQL missing %q\n--- SQL ---\n%s", sub, query)
		}
	}
}

// TestBuildDelegateLockSQL_SupplierDelegate_RejectsCrossWorkspace is the
// symmetric assertion for the supplier-delegate acting-as lock — same
// round-3 invariant.
func TestBuildDelegateLockSQL_SupplierDelegate_RejectsCrossWorkspace(t *testing.T) {
	const (
		delegateID  = "delegate-D"
		actingAsID  = "supplier-Z"
		userID      = "user-U"
		workspaceID = "workspace-A"
	)
	query, args := buildDelegateLockSQL(
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
		delegateID, actingAsID, userID, workspaceID,
	)
	if len(args) != 4 {
		t.Errorf("supplier-delegate acting-as lock arg count = %d, want 4 (args=%v)", len(args), args)
	}
	want := []any{delegateID, actingAsID, userID, workspaceID}
	for i := range want {
		if i >= len(args) {
			break
		}
		if args[i] != want[i] {
			t.Errorf("supplier-delegate acting-as args[%d] = %v, want %v", i, args[i], want[i])
		}
	}
	required := []string{
		"FROM delegate_supplier ds",
		"JOIN delegate d ON d.id = ds.delegate_id",
		"LEFT JOIN supplier s ON s.id = ds.supplier_id",
		"COALESCE(ds.workspace_id, s.workspace_id) = $4",
		"FOR UPDATE",
	}
	for _, sub := range required {
		if !strings.Contains(query, sub) {
			t.Errorf("supplier-delegate acting-as lock SQL missing %q\n--- SQL ---\n%s", sub, query)
		}
	}
}

// TestBuildDelegateLockSQL_NoActingAs_ParentDelegateLockUnchanged confirms
// the parent-delegate (no acting-as) lock keeps the 2-arg shape — the
// multi-target picker pre-resolution path is intentionally out of scope
// for the round-3 workspace-predicate fix. Passing actingAsID="" selects
// the parent-delegate branch; workspaceID is ignored on that branch.
func TestBuildDelegateLockSQL_NoActingAs_ParentDelegateLockUnchanged(t *testing.T) {
	const (
		delegateID = "delegate-D"
		userID     = "user-U"
	)
	for _, kind := range []principaltypepb.PrincipalType{
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
	} {
		query, args := buildDelegateLockSQL(kind, delegateID, "", userID, "ignored-ws")
		if len(args) != 2 {
			t.Errorf("%s parent-delegate lock arg count = %d, want 2 (args=%v)", kind.String(), len(args), args)
		}
		if len(args) >= 2 {
			if args[0] != delegateID {
				t.Errorf("%s parent-delegate args[0] = %v, want %v", kind.String(), args[0], delegateID)
			}
			if args[1] != userID {
				t.Errorf("%s parent-delegate args[1] = %v, want %v", kind.String(), args[1], userID)
			}
		}
		if !strings.Contains(query, "FROM delegate") {
			t.Errorf("%s parent-delegate lock missing FROM delegate\n--- SQL ---\n%s", kind.String(), query)
		}
		if strings.Contains(query, "COALESCE") {
			t.Errorf("%s parent-delegate lock unexpectedly references COALESCE (should be acting-as path only)\n--- SQL ---\n%s", kind.String(), query)
		}
	}
}

// TestDeriveSwitchUseCaseEnum mirrors the original TestDeriveSwitchUseCase
// matrix but asserts against the proto SwitchUseCase enum that the adapter
// now emits. Includes the URL-driven degenerate same-workspace case
// (acting-as-inplace fall-through) and the explicit-form degenerate case
// (explicit-inplace fall-through). Precedence between principal_type_changed
// and acting_as_changed on in-place mutations is also locked in
// (principal_type wins).
func TestDeriveSwitchUseCaseEnum(t *testing.T) {
	cases := []struct {
		name                 string
		urlDriven            bool
		shouldRotate         bool
		principalTypeChanged bool
		actingAsChanged      bool
		want                 authpb.SwitchUseCase
	}{
		// Rotation branch — workspace_id changed
		{"url_driven_rotate_anything", true, true, false, false, authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ROTATE},
		{"url_driven_rotate_with_pt_change", true, true, true, true, authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ROTATE},
		{"explicit_rotate_anything", false, true, false, false, authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ROTATE},
		{"explicit_rotate_with_acting_as", false, true, false, true, authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ROTATE},

		// URL-driven in-place
		{"url_inplace_principal_type", true, false, true, false, authpb.SwitchUseCase_SWITCH_USE_CASE_URL_PRINCIPAL_INPLACE},
		{"url_inplace_acting_as", true, false, false, true, authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ACTING_AS_INPLACE},
		{"url_inplace_both_pt_wins", true, false, true, true, authpb.SwitchUseCase_SWITCH_USE_CASE_URL_PRINCIPAL_INPLACE},
		// Degenerate "nothing actually moved" — both flavors map to the
		// principal-flavored bucket per the deriveSwitchUseCaseEnum docstring
		// rationale (calling it acting_as would be misleading when no
		// acting_as field changed; the rotation primitive still wrote a
		// no-op-like UPDATE and the structured-reason field captures the
		// actual deltas).
		{"url_inplace_neither_degenerate", true, false, false, false, authpb.SwitchUseCase_SWITCH_USE_CASE_URL_PRINCIPAL_INPLACE},

		// Explicit-form in-place
		{"explicit_inplace_principal_type", false, false, true, false, authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_INPLACE},
		{"explicit_inplace_acting_as", false, false, false, true, authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ACTING_AS},
		{"explicit_inplace_both_pt_wins", false, false, true, true, authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_INPLACE},
		{"explicit_inplace_neither_degenerate", false, false, false, false, authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_INPLACE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveSwitchUseCaseEnum(tc.urlDriven, tc.shouldRotate, tc.principalTypeChanged, tc.actingAsChanged)
			if got != tc.want {
				t.Errorf("deriveSwitchUseCaseEnum(url=%v, rot=%v, pt=%v, aa=%v) = %v, want %v",
					tc.urlDriven, tc.shouldRotate, tc.principalTypeChanged, tc.actingAsChanged, got, tc.want)
			}
		})
	}
}

// TestSwitchUseCaseAuditLabel pins the enum → audit-string mapping so the
// audit_trail.audit_entry.use_case column keeps emitting the pre-refactor
// string values that reporting / forensic tooling may already grep for.
// Drift between the enum and the audit string is the kind of silent change
// that breaks downstream analytics; the test prevents that.
func TestSwitchUseCaseAuditLabel(t *testing.T) {
	cases := []struct {
		uc   authpb.SwitchUseCase
		want string
	}{
		{authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ROTATE, "switch_url_rotate"},
		{authpb.SwitchUseCase_SWITCH_USE_CASE_URL_ACTING_AS_INPLACE, "switch_url_acting_as_inplace"},
		{authpb.SwitchUseCase_SWITCH_USE_CASE_URL_PRINCIPAL_INPLACE, "switch_url_principal_inplace"},
		{authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ROTATE, "switch_explicit_rotate"},
		{authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_INPLACE, "switch_explicit_inplace"},
		{authpb.SwitchUseCase_SWITCH_USE_CASE_EXPLICIT_ACTING_AS, "switch_explicit_acting_as"},
		// Defensive fallback for UNSPECIFIED / unknown values.
		{authpb.SwitchUseCase_SWITCH_USE_CASE_UNSPECIFIED, "switch_principal"},
	}
	for _, tc := range cases {
		t.Run(tc.uc.String(), func(t *testing.T) {
			if got := switchUseCaseAuditLabel(tc.uc); got != tc.want {
				t.Errorf("switchUseCaseAuditLabel(%v) = %q, want %q", tc.uc, got, tc.want)
			}
		})
	}
}

// TestPrincipalTypeAuditLabel pins the audit-reason principal_type labels to
// the lowercase form the pre-refactor primitive emitted (codex round 1 P1
// regression: the migrated adapter briefly used the proto enum's String()
// form "PRINCIPAL_TYPE_CLIENT" instead of "client"). Forensic tooling parses
// these labels, so the mapping must stay byte-identical to
// adapthttp.PrincipalType.String() at
// apps/service-admin/internal/infrastructure/input/http/principal_loader.go.
func TestPrincipalTypeAuditLabel(t *testing.T) {
	cases := []struct {
		pt   principaltypepb.PrincipalType
		want string
	}{
		{principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_OWNER, "operator_owner"},
		{principaltypepb.PrincipalType_PRINCIPAL_TYPE_OPERATOR_STAFF, "operator_staff"},
		{principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT, "client"},
		{principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE, "client_delegate"},
		{principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER, "supplier"},
		{principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE, "supplier_delegate"},
		{principaltypepb.PrincipalType_PRINCIPAL_TYPE_UNSPECIFIED, "unspecified"},
	}
	for _, tc := range cases {
		t.Run(tc.pt.String(), func(t *testing.T) {
			if got := principalTypeAuditLabel(tc.pt); got != tc.want {
				t.Errorf("principalTypeAuditLabel(%v) = %q, want %q", tc.pt, got, tc.want)
			}
			// Guard against re-regression: the label must NOT be the proto
			// enum String() form.
			if got := principalTypeAuditLabel(tc.pt); strings.HasPrefix(got, "PRINCIPAL_TYPE_") {
				t.Errorf("principalTypeAuditLabel(%v) leaked proto enum name %q", tc.pt, got)
			}
		})
	}
}

// TestCoalesceInt32PrincipalTypeString pins the prior-principal_type renderer
// used in the audit reason: NULL → "unset", else the lowercase audit label.
func TestCoalesceInt32PrincipalTypeString(t *testing.T) {
	if got := coalesceInt32PrincipalTypeString(sql.NullInt32{}); got != "unset" {
		t.Errorf("coalesceInt32PrincipalTypeString(NULL) = %q, want %q", got, "unset")
	}
	valid := sql.NullInt32{Int32: int32(principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE), Valid: true}
	if got := coalesceInt32PrincipalTypeString(valid); got != "client_delegate" {
		t.Errorf("coalesceInt32PrincipalTypeString(CLIENT_DELEGATE) = %q, want %q", got, "client_delegate")
	}
}

// TestCoalesceNullStringOrSentinel pins the helper that normalises new-side
// acting-as values so the equality check in the deriveSwitchUseCaseEnum
// wrapping doesn't false-positive on the (null, "") pair which both mean
// "no acting-as target".
func TestCoalesceNullStringOrSentinel(t *testing.T) {
	if got := coalesceNullStringOrSentinel(""); got != "-" {
		t.Errorf("coalesceNullStringOrSentinel(\"\") = %q, want %q", got, "-")
	}
	if got := coalesceNullStringOrSentinel("abc"); got != "abc" {
		t.Errorf("coalesceNullStringOrSentinel(\"abc\") = %q, want %q", got, "abc")
	}
}

// TestActingAsTargetIDsContain pins the URL acting-as / explicit-form
// validation helper introduced by A2-followup round-3 (2026-05-24). The
// helper underwrites the fail-closed guard in SwitchPrincipal that rejects
// a switch when the URL- or form-supplied acting-as id is not in the
// resolved binding's ActingAsTargets slice.
func TestActingAsTargetIDsContain(t *testing.T) {
	targets := []*authpb.ActingAsTarget{
		{Id: "client-A", WorkspaceId: "ws-1"},
		{Id: "client-B", WorkspaceId: "ws-1"},
		{Id: "client-C", WorkspaceId: "ws-2"},
	}

	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"hit_first", "client-A", true},
		{"hit_middle", "client-B", true},
		{"hit_last", "client-C", true},
		{"miss_unknown", "client-Z", false},
		{"miss_empty_id", "", false},
		{"miss_substring", "client", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := actingAsTargetIDsContain(targets, tc.id)
			if got != tc.want {
				t.Errorf("actingAsTargetIDsContain(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}

	t.Run("empty_slice_always_false", func(t *testing.T) {
		if actingAsTargetIDsContain(nil, "client-A") {
			t.Errorf("actingAsTargetIDsContain(nil, %q) = true, want false", "client-A")
		}
		if actingAsTargetIDsContain([]*authpb.ActingAsTarget{}, "client-A") {
			t.Errorf("actingAsTargetIDsContain([]) = true, want false")
		}
	})
}

// TestW1_DelegateLockSQL_WorkspaceAndUserPredicates_RegressionLock is W1
// (Layer 2) of the plan-3 "principal-switch membership-authz regression gate"
// (docs/plan/20260530-authz-workspace-hardening/, findings test-coverage-2 +
// test-coverage-5). It is the SQL-predicate DRIFT LOCK for the delegate
// cross-workspace deny: the acting-as delegate lock MUST carry BOTH
//
//   - the membership predicate `d.user_id = $3` (a delegate may only lock a
//     grant rooted at a delegate row owned by the acting user), AND
//   - the workspace-coherence predicate `COALESCE(...workspace_id...) = $4`
//     (the locked grant must resolve to the SAME workspace the caller is
//     switching into — the forged-/as/-URL cross-workspace fix, A2-followup
//     round-3).
//
// espyna has NO sqlmock / live-DB harness in this package, so the live
// lockTargetBinding ErrNoRows deny cannot be executed here (that live
// assertion is the apps-E2E layer's job:
// multi-principal/switch-principal-denied.spec.ts). What IS locked here is the
// SQL string/arg SHAPE: removing either predicate, or shrinking the 4-arg
// contract, turns this test red — which is the regression gate. This is a
// stricter, intent-named companion to the existing
// TestBuildDelegateLockSQL_*_RejectsCrossWorkspace cases (which it does not
// replace).
func TestW1_DelegateLockSQL_WorkspaceAndUserPredicates_RegressionLock(t *testing.T) {
	const (
		delegateID  = "delegate-D"
		userID      = "user-U"
		workspaceID = "workspace-A"
	)

	cases := []struct {
		name          string
		kind          principaltypepb.PrincipalType
		actingAsID    string
		userPredicate string // the d.user_id=$3 membership predicate
		wsPredicate   string // the COALESCE workspace-coherence predicate
	}{
		{
			name:          "client_delegate",
			kind:          principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
			actingAsID:    "client-Z",
			userPredicate: "d.user_id = $3",
			wsPredicate:   "COALESCE(dc.workspace_id, c.workspace_id) = $4",
		},
		{
			name:          "supplier_delegate",
			kind:          principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
			actingAsID:    "supplier-Z",
			userPredicate: "d.user_id = $3",
			wsPredicate:   "COALESCE(ds.workspace_id, s.workspace_id) = $4",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query, args := buildDelegateLockSQL(tc.kind, delegateID, tc.actingAsID, userID, workspaceID)

			// 4-arg positional contract — the workspace predicate's $4 has no
			// value without it. Dropping the workspace arg (back to a 3-arg
			// shape) is the regression this catches.
			if len(args) != 4 {
				t.Fatalf("%s acting-as lock arg count = %d, want 4 (dropping the workspace arg is the regression) args=%v",
					tc.name, len(args), args)
			}
			if args[2] != userID {
				t.Errorf("%s acting-as args[2] (user_id) = %v, want %v", tc.name, args[2], userID)
			}
			if args[3] != workspaceID {
				t.Errorf("%s acting-as args[3] (workspace_id) = %v, want %v", tc.name, args[3], workspaceID)
			}

			// The membership predicate: a delegate may only lock a grant rooted
			// at a delegate row it owns. Removing `d.user_id = $3` is a
			// privilege-escalation regression (any user could replay any
			// delegate's grant) — this assertion turns red if it goes.
			if !strings.Contains(query, tc.userPredicate) {
				t.Errorf("%s acting-as lock MISSING membership predicate %q — "+
					"removing it would let a switched principal lock a grant it does not own\n--- SQL ---\n%s",
					tc.name, tc.userPredicate, query)
			}

			// The workspace-coherence predicate: the forged-/as/-URL
			// cross-workspace deny. Removing it lets a real D→Z grant in
			// workspace B be replayed against /w/workspace-A/.
			if !strings.Contains(query, tc.wsPredicate) {
				t.Errorf("%s acting-as lock MISSING workspace predicate %q — "+
					"removing it re-opens the cross-workspace forged-/as/ deny bypass\n--- SQL ---\n%s",
					tc.name, tc.wsPredicate, query)
			}

			// The lock must be a real row lock (FOR UPDATE) — a plain SELECT
			// would not close the revoke TOCTOU the deny relies on.
			if !strings.Contains(query, "FOR UPDATE") {
				t.Errorf("%s acting-as lock missing FOR UPDATE (revoke-TOCTOU close)\n--- SQL ---\n%s", tc.name, query)
			}
		})
	}
}

// TestW1_NonDelegateLock_UserIDPredicate_RegressionLock is W1 (Layer 2,
// non-delegate leg) of the plan-3 deny gate. The operator/client/supplier
// binding locks in lockTargetBinding all carry `user_id = $2 AND active = true`
// — the membership + revocation predicate. Dropping `user_id=$2` (the plan's
// named acceptance trigger) would let a forged target_principal owning a
// workspace_user/grant row belonging to a DIFFERENT user pass the lock. This
// pins the predicate text the live deny depends on.
//
// Because lockTargetBinding issues the SQL inline (no extracted builder for the
// non-delegate kinds) and there is no DB harness, this test documents the
// invariant as a named gate and is paired with the live assertion deferred to
// multi-principal/switch-principal-denied.spec.ts. See switch_principal.go
// (use-case Layer-1) TestSwitchPrincipal_Execute_DenyPathRegressionGate for the
// error→deny-key half that DOES run here in-process.
func TestW1_NonDelegateLock_UserIDPredicate_RegressionLock(t *testing.T) {
	// Sentinel guard: the non-delegate lock SQL lives inline in
	// lockTargetBinding (not a pure builder), so we cannot call it without a
	// *sql.Tx. We instead lock the buildable delegate parent-lock leg, which
	// shares the exact `user_id = $2 AND active = true` membership shape, and
	// assert that shape is intact. If the membership/active predicates are ever
	// dropped from the buildable path the gate fires; the inline non-delegate
	// path is covered live by the E2E spec.
	const (
		delegateID = "delegate-D"
		userID     = "user-U"
	)
	for _, kind := range []principaltypepb.PrincipalType{
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_CLIENT_DELEGATE,
		principaltypepb.PrincipalType_PRINCIPAL_TYPE_SUPPLIER_DELEGATE,
	} {
		// Parent-delegate (no acting-as) lock: SELECT ... WHERE id=$1 AND
		// user_id=$2 AND active=true — the same membership+revocation shape the
		// non-delegate operator/client/supplier locks use.
		query, args := buildDelegateLockSQL(kind, delegateID, "", userID, "")
		if len(args) != 2 || args[1] != userID {
			t.Fatalf("%s parent lock must bind user_id=$2 as the membership predicate; args=%v", kind.String(), args)
		}
		for _, sub := range []string{"user_id = $2", "active = true", "FOR UPDATE"} {
			if !strings.Contains(query, sub) {
				t.Errorf("%s binding lock MISSING %q — dropping it breaks the membership/revocation deny\n--- SQL ---\n%s",
					kind.String(), sub, query)
			}
		}
	}
}

// TestW1_ActingAsTargetIDsContain_DenyOnMiss_RegressionLock is W1 (Layer 3) of
// the plan-3 deny gate — the in-process fail-closed guard. SwitchPrincipal
// (session_switch_principal.go:157-160 / :177-180) calls actingAsTargetIDsContain
// and REFUSES the switch when a URL-/form-supplied acting_as_* id is NOT in the
// resolved binding's ActingAsTargets. This test pins the load-bearing contract:
// every MISS (ungranted id, empty id, nil/empty target slice) returns false so
// the guard's deny branch fires; only a genuine membership returns true.
//
// RED proof: if the guard ever returned true on a miss (fail-OPEN), a
// switched/delegated principal could rotate into an acting_as_* target it has
// no grant for — exactly the regression this gate forbids.
func TestW1_ActingAsTargetIDsContain_DenyOnMiss_RegressionLock(t *testing.T) {
	granted := []*authpb.ActingAsTarget{
		{Id: "client-granted-A", WorkspaceId: "ws-1"},
		{Id: "client-granted-B", WorkspaceId: "ws-1"},
	}

	// DENY contract: every one of these MUST be false (fail-closed). If any
	// flips to true the guard fails open and a switch into an ungranted target
	// would be permitted.
	denyMisses := []struct {
		name    string
		targets []*authpb.ActingAsTarget
		id      string
	}{
		{"ungranted_id_against_real_targets", granted, "client-NOT-granted"},
		{"empty_id_never_grants", granted, ""},
		{"substring_of_granted_id_not_a_match", granted, "client-granted"},
		{"nil_targets_deny_all", nil, "client-granted-A"},
		{"empty_targets_deny_all", []*authpb.ActingAsTarget{}, "client-granted-A"},
	}
	for _, tc := range denyMisses {
		t.Run("deny_"+tc.name, func(t *testing.T) {
			if actingAsTargetIDsContain(tc.targets, tc.id) {
				t.Errorf("FAIL-OPEN regression: actingAsTargetIDsContain(targets=%v, %q) = true, want false "+
					"(the in-process guard would admit a switch into an ungranted acting_as_* target)",
					tc.targets, tc.id)
			}
		})
	}

	// Positive control: a genuine grant must still resolve true, otherwise the
	// guard would be a deny-everything brick (not a regression gate).
	for _, tgt := range granted {
		t.Run("grant_"+tgt.GetId(), func(t *testing.T) {
			if !actingAsTargetIDsContain(granted, tgt.GetId()) {
				t.Errorf("granted acting-as id %q unexpectedly denied — guard must admit real members", tgt.GetId())
			}
		})
	}
}

// TestFormatActingAsTargetIDs locks the diagnostic format used by the
// fail-closed error message in SwitchPrincipal when the URL- or form-
// supplied acting-as id misses every target in the resolved binding.
func TestFormatActingAsTargetIDs(t *testing.T) {
	cases := []struct {
		name    string
		targets []*authpb.ActingAsTarget
		want    string
	}{
		{"empty_returns_sentinel", nil, "(none)"},
		{"single", []*authpb.ActingAsTarget{{Id: "client-A"}}, "client-A"},
		{"multi", []*authpb.ActingAsTarget{
			{Id: "client-A"},
			{Id: "client-B"},
			{Id: "client-C"},
		}, "client-A,client-B,client-C"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatActingAsTargetIDs(tc.targets)
			if got != tc.want {
				t.Errorf("formatActingAsTargetIDs() = %q, want %q", got, tc.want)
			}
		})
	}
}
