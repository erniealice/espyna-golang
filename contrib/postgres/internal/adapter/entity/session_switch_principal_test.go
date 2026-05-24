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
