package outcome_completion

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/shared/identity"
	ocpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/outcome_completion"
)

// fakeAuthorizer is a minimal RBAC port. IsEnabled() is true so Check actually
// consults HasPermission (a disabled authorizer would allow everything and
// defeat the point). allowed is the set of permission codes the principal holds.
type fakeAuthorizer struct{ allowed map[string]bool }

func (f *fakeAuthorizer) IsEnabled() bool { return true }
func (f *fakeAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.allowed[permission], nil
}

func gate(allowed ...string) *actiongate.ActionGatekeeper {
	set := map[string]bool{}
	for _, p := range allowed {
		set[p] = true
	}
	return actiongate.NewActionGatekeeper(&fakeAuthorizer{allowed: set}, nil)
}

// theCode is the Q7 locked permission code the gate must consult.
const theCode = "outcome_completion:read"

// fakeRepo records the workspace the use case forwards and returns canned rows.
type fakeRepo struct {
	called bool
	gotWS  string
	rows   []CategoryPeriodCount
	err    error
}

func (r *fakeRepo) CompletionSummary(_ context.Context, workspaceID string) ([]CategoryPeriodCount, error) {
	r.called = true
	r.gotWS = workspaceID
	if r.err != nil {
		return nil, r.err
	}
	return r.rows, nil
}

func newUC(repo OutcomeCompletionSummaryRepository, g *actiongate.ActionGatekeeper) *GetOutcomeCompletionSummaryUseCase {
	return NewUseCases(&Deps{OutcomeCompletion: repo, ActionGatekeeper: g}).GetOutcomeCompletionSummary
}

// ctxWithKind builds a ctx carrying the actiongate user id AND a session
// RequestIdentity of the given principal kind.
func ctxWithKind(kind int32, principalID, workspaceID string) context.Context {
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	return identity.WithRequestIdentity(ctx, &identity.RequestIdentity{
		UserID:        "user-1",
		WorkspaceID:   workspaceID,
		PrincipalType: kind,
		PrincipalID:   principalID,
	})
}

func req(ws string) *ocpb.GetOutcomeCompletionSummaryRequest {
	return &ocpb.GetOutcomeCompletionSummaryRequest{WorkspaceId: ws}
}

// TestGateDenied — the principal lacks outcome_completion:read → error, and the
// repository is NEVER touched (Gate 1 runs before any repo call).
func TestGateDenied(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUC(repo, gate()) // grants nothing
	if _, err := uc.Execute(ctxWithKind(principalKindStaff, "staff-1", "ws-1"), req("ws-1")); err == nil {
		t.Fatalf("missing %s must deny", theCode)
	}
	if repo.called {
		t.Errorf("denied principal must not reach the repository")
	}
}

// TestNilGatekeeperDenies — a nil gatekeeper fails closed (Check's nil-receiver
// DENY guard).
func TestNilGatekeeperDenies(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUC(repo, nil)
	if _, err := uc.Execute(ctxWithKind(principalKindStaff, "staff-1", "ws-1"), req("ws-1")); err == nil {
		t.Fatalf("nil gatekeeper must deny")
	}
	if repo.called {
		t.Errorf("nil gatekeeper must not reach the repository")
	}
}

// TestNoIdentityDenies — gate passes but the ctx carries no RequestIdentity →
// fail-closed deny (identity comes from ctx, never the wire).
func TestNoIdentityDenies(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUC(repo, gate(theCode))
	ctx := contextutil.WithUserID(context.Background(), "user-1") // no identity
	if _, err := uc.Execute(ctx, req("ws-1")); err == nil {
		t.Fatalf("absent identity must deny")
	}
	if repo.called {
		t.Errorf("identity-less caller must not reach the repository")
	}
}

// TestKindZeroDenies — the unresolved kind-0 sentinel is denied fail-closed
// (plan.md Phase 2: "kind 0 → deny"), even with the permission granted.
func TestKindZeroDenies(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUC(repo, gate(theCode))
	if _, err := uc.Execute(ctxWithKind(0, "", "ws-1"), req("ws-1")); err == nil {
		t.Fatalf("kind 0 must deny")
	}
	if repo.called {
		t.Errorf("kind 0 must not reach the repository")
	}
}

// TestUndefinedKindDenies — a kind this read does not define (e.g. a client
// principal) is denied, not silently given workspace grain.
func TestUndefinedKindDenies(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUC(repo, gate(theCode))
	if _, err := uc.Execute(ctxWithKind(3, "client-1", "ws-1"), req("ws-1")); err == nil {
		t.Fatalf("undefined principal kind must deny")
	}
	if repo.called {
		t.Errorf("undefined kind must not reach the repository")
	}
}

// TestSessionWorkspaceWins — the repository receives the SESSION workspace,
// not a (potentially spoofed) wire workspace_id.
func TestSessionWorkspaceWins(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUC(repo, gate(theCode))
	if _, err := uc.Execute(ctxWithKind(principalKindOperatorStaff, "op-1", "ws-session"), req("ws-spoofed")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotWS != "ws-session" {
		t.Errorf("session workspace must win: got %q", repo.gotWS)
	}
}

// TestEmptySessionWorkspaceDenies — §A-1.4 hard-deny (skeptic S2-F2): a
// session that resolved NO workspace is denied fail-closed; the wire
// workspace_id must NEVER be read as a fallback (a kind-1/2 session with an
// empty workspace would otherwise become a caller-chosen cross-tenant read
// at workspace grain).
func TestEmptySessionWorkspaceDenies(t *testing.T) {
	repo := &fakeRepo{}
	uc := newUC(repo, gate(theCode))
	if _, err := uc.Execute(ctxWithKind(principalKindOperatorOwner, "own-1", ""), req("ws-attacker")); err == nil {
		t.Fatalf("empty session workspace must deny — wire workspace_id must not be honoured")
	}
	if repo.called {
		t.Errorf("empty-workspace session must not reach the repository")
	}
}

// TestDeniesAreTypedAuthorizationErrors — every fail-closed refusal carries a
// *security.AuthorizationError in its chain (errors.As) so the consuming view
// can render the designed DENIED card instead of a benign empty state
// (skeptic F2 / T-9 observability).
func TestDeniesAreTypedAuthorizationErrors(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		g    *actiongate.ActionGatekeeper
	}{
		{"gate deny", ctxWithKind(principalKindStaff, "staff-1", "ws-1"), gate()},
		{"no identity", contextutil.WithUserID(context.Background(), "user-1"), gate(theCode)},
		{"kind 0", ctxWithKind(0, "", "ws-1"), gate(theCode)},
		{"undefined kind", ctxWithKind(3, "client-1", "ws-1"), gate(theCode)},
		{"empty workspace", ctxWithKind(principalKindStaff, "staff-1", ""), gate(theCode)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := newUC(&fakeRepo{}, tc.g)
			_, err := uc.Execute(tc.ctx, req("ws-1"))
			if err == nil {
				t.Fatalf("%s must deny", tc.name)
			}
			var ae *security.AuthorizationError
			if !errors.As(err, &ae) {
				t.Errorf("%s: deny must be a typed *security.AuthorizationError, got %v", tc.name, err)
			}
		})
	}
}

// strictFakeAuthorizer models the rbac adapter in SHADOW mode: HasPermission
// allows everything (shadow allow-on-deny) while HasPermissionStrict returns
// the REAL verdict. The strict gate must consult the strict path, so a
// missing grant denies even though the shadow path would allow (skeptic F1).
type strictFakeAuthorizer struct{ strictAllowed map[string]bool }

func (f *strictFakeAuthorizer) IsEnabled() bool { return true }
func (f *strictFakeAuthorizer) HasPermission(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil // shadow mode: would-be deny is allowed
}
func (f *strictFakeAuthorizer) HasPermissionStrict(_ context.Context, _ string, permission string) (bool, error) {
	return f.strictAllowed[permission], nil
}

// TestShadowModeDeniesViaStrictPath — the Q7 gate is NOT shadow-bypassable:
// with a shadow-allowing authorizer that strictly lacks the code, Execute
// denies and never reaches the repository.
func TestShadowModeDeniesViaStrictPath(t *testing.T) {
	repo := &fakeRepo{}
	g := actiongate.NewActionGatekeeper(&strictFakeAuthorizer{strictAllowed: map[string]bool{}}, nil)
	uc := newUC(repo, g)
	if _, err := uc.Execute(ctxWithKind(principalKindStaff, "staff-1", "ws-1"), req("ws-1")); err == nil {
		t.Fatalf("shadow-mode allow-on-deny must not bypass the strict gate")
	}
	if repo.called {
		t.Errorf("strictly denied principal must not reach the repository")
	}
	// And the strictly GRANTED principal still passes.
	g2 := actiongate.NewActionGatekeeper(&strictFakeAuthorizer{strictAllowed: map[string]bool{theCode: true}}, nil)
	uc2 := newUC(&fakeRepo{}, g2)
	if _, err := uc2.Execute(ctxWithKind(principalKindStaff, "staff-1", "ws-1"), req("ws-1")); err != nil {
		t.Fatalf("strictly granted principal must pass: %v", err)
	}
}

// TestNilRepoDegrades — a permitted read with no registered adapter (mock /
// non-postgres builds) returns a zero-valued successful response, no panic.
func TestNilRepoDegrades(t *testing.T) {
	uc := newUC(nil, gate(theCode))
	resp, err := uc.Execute(ctxWithKind(principalKindOperatorOwner, "own-1", "ws-1"), req("ws-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() || len(resp.GetCategoryRows()) != 0 || resp.GetAllRollup() == nil {
		t.Errorf("nil repo must yield a zero-valued success response, got %+v", resp)
	}
}

// TestRepoErrorPropagates — a DB/query failure is NOT swallowed (T-9): it
// propagates so the view renders a designed error state.
func TestRepoErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	uc := newUC(&fakeRepo{err: sentinel}, gate(theCode))
	if _, err := uc.Execute(ctxWithKind(principalKindStaff, "staff-1", "ws-1"), req("ws-1")); !errors.Is(err, sentinel) {
		t.Fatalf("repo error must propagate, got %v", err)
	}
}

// richRows is the zia.barles-shaped oracle (_discovery.md §3): a rich staff
// persona with three categories, two period slices each (the A2 dimension).
// Counts are trimmed but keep the discovery shape: a large academic category,
// smaller deportment categories, recorded=0 on the current window (legitimate
// new-period zero).
func richRows() []CategoryPeriodCount {
	return []CategoryPeriodCount{
		{CategoryID: "cat-a", CategoryName: "Category A", HasSlice: true, PhaseOrder: 1,
			ExpectedCells: 1400, RecordedCells: 0, NotStarted: 12, InProgress: 0},
		{CategoryID: "cat-a", CategoryName: "Category A", HasSlice: true, PhaseOrder: 2,
			ExpectedCells: 1376, RecordedCells: 0, NotStarted: 12},
		{CategoryID: "cat-b", CategoryName: "Category B", HasSlice: true, PhaseOrder: 1,
			ExpectedCells: 350, RecordedCells: 0, NotStarted: 6},
		{CategoryID: "cat-b", CategoryName: "Category B", HasSlice: true, PhaseOrder: 2,
			ExpectedCells: 344, RecordedCells: 0, NotStarted: 6},
		{CategoryID: "cat-c", CategoryName: "Category C", HasSlice: true, PhaseOrder: 1,
			ExpectedCells: 390, RecordedCells: 0, NotStarted: 4},
		{CategoryID: "cat-c", CategoryName: "Category C", HasSlice: true, PhaseOrder: 2,
			ExpectedCells: 378, RecordedCells: 0, NotStarted: 4},
	}
}

// TestRichPersonaAssembly — zia-shaped rows fold into ordered category rows
// whose totals equal the sum of their slices, plus an All rollup that sums the
// categories and windows by phase_order.
func TestRichPersonaAssembly(t *testing.T) {
	uc := newUC(&fakeRepo{rows: richRows()}, gate(theCode))
	resp, err := uc.Execute(ctxWithKind(principalKindStaff, "staff-1", "ws-1"), req("ws-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rowsOut := resp.GetCategoryRows()
	if len(rowsOut) != 3 {
		t.Fatalf("want 3 category rows, got %d", len(rowsOut))
	}
	// Adapter order preserved (sort_order asc — the adapter's fixed ORDER BY).
	for i, want := range []string{"cat-a", "cat-b", "cat-c"} {
		if rowsOut[i].GetCategoryId() != want {
			t.Errorf("row %d: want %s got %s", i, want, rowsOut[i].GetCategoryId())
		}
	}
	// Row totals = sum of slices (proto contract).
	a := rowsOut[0]
	if a.GetExpectedCells() != 2776 || a.GetRecordedCells() != 0 {
		t.Errorf("cat-a totals want 2776/0, got %d/%d", a.GetExpectedCells(), a.GetRecordedCells())
	}
	if len(a.GetPeriodSlices()) != 2 || a.GetPeriodSlices()[0].GetPhaseOrder() != 1 || a.GetPeriodSlices()[1].GetPhaseOrder() != 2 {
		t.Errorf("cat-a slices malformed: %+v", a.GetPeriodSlices())
	}
	if a.GetApprovalCounts().GetNotStarted() != 24 {
		t.Errorf("cat-a not_started want 24, got %d", a.GetApprovalCounts().GetNotStarted())
	}
	// All rollup: sums across categories; slices windowed by phase_order asc.
	all := resp.GetAllRollup()
	if all.GetExpectedCells() != 4238 { // the zia current-window "0/4,238" fixture
		t.Errorf("all rollup expected want 4238, got %d", all.GetExpectedCells())
	}
	if all.GetRecordedCells() != 0 {
		t.Errorf("all rollup recorded want 0 (legitimate new-period zero), got %d", all.GetRecordedCells())
	}
	if all.GetCategoryId() != "" || all.GetCategoryName() != "" {
		t.Errorf("all rollup must carry no category identity (label comes from lyngua)")
	}
	if len(all.GetPeriodSlices()) != 2 {
		t.Fatalf("all rollup want 2 phase slices, got %d", len(all.GetPeriodSlices()))
	}
	s1, s2 := all.GetPeriodSlices()[0], all.GetPeriodSlices()[1]
	if s1.GetPhaseOrder() != 1 || s2.GetPhaseOrder() != 2 {
		t.Errorf("all slices must be ordered by phase_order asc")
	}
	if s1.GetExpectedCells() != 2140 || s2.GetExpectedCells() != 2098 {
		t.Errorf("all slice sums want 2140/2098, got %d/%d", s1.GetExpectedCells(), s2.GetExpectedCells())
	}
	if got := s1.GetApprovalCounts().GetNotStarted() + s2.GetApprovalCounts().GetNotStarted(); got != 44 {
		t.Errorf("all approval sum want 44, got %d", got)
	}
}

// TestEmptyPersonaAssembly — the aila.cagang-shaped empty persona (_discovery
// §3: 1 edge, 0 outcomes, 0 assigned tasks): the adapter still emits every
// active category as a zero-cell row (HasSlice=false) so the view renders
// designed empty-states — NOT errors, NOT missing tabs.
func TestEmptyPersonaAssembly(t *testing.T) {
	rows := []CategoryPeriodCount{
		{CategoryID: "cat-a", CategoryName: "Category A"},
		{CategoryID: "cat-b", CategoryName: "Category B"},
		{CategoryID: "cat-c", CategoryName: "Category C"},
	}
	uc := newUC(&fakeRepo{rows: rows}, gate(theCode))
	resp, err := uc.Execute(ctxWithKind(principalKindStaff, "staff-empty", "ws-1"), req("ws-1"))
	if err != nil {
		t.Fatalf("empty persona must not error: %v", err)
	}
	if len(resp.GetCategoryRows()) != 3 {
		t.Fatalf("want 3 zero-value category rows, got %d", len(resp.GetCategoryRows()))
	}
	for _, r := range resp.GetCategoryRows() {
		if r.GetExpectedCells() != 0 || r.GetRecordedCells() != 0 || len(r.GetPeriodSlices()) != 0 {
			t.Errorf("zero-cell category %s must be all-zero with no slices, got %+v", r.GetCategoryId(), r)
		}
	}
	all := resp.GetAllRollup()
	if all.GetExpectedCells() != 0 || len(all.GetPeriodSlices()) != 0 {
		t.Errorf("empty persona all rollup must be zero-valued, got %+v", all)
	}
}

// TestRotationStaffAssembly — the A1 worked example (storybook-findings
// F2, Florie-shaped): a phase-scoped Semester-2 rotation teacher. The adapter
// (most-specific-wins SQL) returns ONLY her phase-2 cells for the rotation
// category — the use case must NOT re-synthesize the sibling semester: the
// category total equals the phase-true denominator (3,806-shape), never the
// job-grain 4,868, and no phase-1 slice appears.
func TestRotationStaffAssembly(t *testing.T) {
	rows := []CategoryPeriodCount{
		// Her non-rotation classes span both semesters…
		{CategoryID: "cat-a", CategoryName: "Category A", HasSlice: true, PhaseOrder: 1,
			ExpectedCells: 1500, NotStarted: 10},
		{CategoryID: "cat-a", CategoryName: "Category A", HasSlice: true, PhaseOrder: 2,
			ExpectedCells: 1244, NotStarted: 10},
		// …the rotation category carries ONLY the phase-2 slice (A1: her
		// phase-scoped sgpps edge restricts to job_phase.template_phase_id).
		{CategoryID: "cat-rot", CategoryName: "Rotation Category", HasSlice: true, PhaseOrder: 2,
			ExpectedCells: 1062, NotStarted: 8},
	}
	uc := newUC(&fakeRepo{rows: rows}, gate(theCode))
	resp, err := uc.Execute(ctxWithKind(principalKindStaff, "staff-rot", "ws-1"), req("ws-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var rot *ocpb.OutcomeCompletionCategoryRow
	for _, r := range resp.GetCategoryRows() {
		if r.GetCategoryId() == "cat-rot" {
			rot = r
		}
	}
	if rot == nil {
		t.Fatalf("rotation category row missing")
	}
	if len(rot.GetPeriodSlices()) != 1 || rot.GetPeriodSlices()[0].GetPhaseOrder() != 2 {
		t.Fatalf("rotation category must carry ONLY the phase-2 slice, got %+v", rot.GetPeriodSlices())
	}
	if rot.GetExpectedCells() != 1062 {
		t.Errorf("rotation denominator must be phase-true (1062), got %d", rot.GetExpectedCells())
	}
	// The All rollup's phase-1 slice contains only the non-rotation cells —
	// the sibling-semester strand cells never re-enter through the rollup.
	all := resp.GetAllRollup()
	if all.GetExpectedCells() != 3806 {
		t.Errorf("all-rollup denominator must be the phase-true 3806, got %d", all.GetExpectedCells())
	}
	for _, s := range all.GetPeriodSlices() {
		if s.GetPhaseOrder() == 1 && s.GetExpectedCells() != 1500 {
			t.Errorf("phase-1 rollup must exclude rotation strand cells, got %d", s.GetExpectedCells())
		}
	}
}
