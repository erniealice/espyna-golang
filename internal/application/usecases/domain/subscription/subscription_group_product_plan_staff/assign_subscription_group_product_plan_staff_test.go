package subscription_group_product_plan_staff

// Upsert tests for the thin assign use case (§6.3). Reuses the in-package mocks
// (mockPPSRepo/mockPPRepo/mockSGRepo, eligibleRow/group/stubIDSGPPS) from
// create_subscription_group_product_plan_staff_test.go and adds a full-CRUD
// sgpps mock so the update/create/clear branches can be observed.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// ----- full-CRUD sgpps mock ------------------------------------------------

type mockAssignSGPPSRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	rows                                  []*pb.SubscriptionGroupProductPlanStaff
	createCalls, updateCalls, deleteCalls int
}

func (m *mockAssignSGPPSRepo) ListSubscriptionGroupProductPlanStaffs(_ context.Context, _ *pb.ListSubscriptionGroupProductPlanStaffsRequest) (*pb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	return &pb.ListSubscriptionGroupProductPlanStaffsResponse{Success: true, Data: m.rows}, nil
}

func (m *mockAssignSGPPSRepo) ReadSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.ReadSubscriptionGroupProductPlanStaffRequest) (*pb.ReadSubscriptionGroupProductPlanStaffResponse, error) {
	id := req.GetData().GetId()
	for _, row := range m.rows {
		if row.GetId() == id {
			return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{row}}, nil
		}
	}
	return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Success: true}, nil
}

func (m *mockAssignSGPPSRepo) CreateSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	m.createCalls++
	m.rows = append(m.rows, req.GetData())
	return &pb.CreateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{req.GetData()}}, nil
}

func (m *mockAssignSGPPSRepo) UpdateSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.UpdateSubscriptionGroupProductPlanStaffRequest) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
	m.updateCalls++
	in := req.GetData()
	for _, row := range m.rows {
		if row.GetId() == in.GetId() {
			if in.GetStaffId() != "" {
				row.StaffId = in.GetStaffId()
			}
			row.Role = in.GetRole()
			// Reactivation parity with the postgres adapter: protojson emits
			// active only when true, so a partial UPDATE flips a soft-deleted
			// row back to active=true (never clobbers it to false — clearing
			// goes through Delete).
			if in.GetActive() {
				row.Active = true
			}
			return &pb.UpdateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{row}}, nil
		}
	}
	return &pb.UpdateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{in}}, nil
}

func (m *mockAssignSGPPSRepo) DeleteSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.DeleteSubscriptionGroupProductPlanStaffRequest) (*pb.DeleteSubscriptionGroupProductPlanStaffResponse, error) {
	m.deleteCalls++
	id := req.GetData().GetId()
	for _, row := range m.rows {
		if row.GetId() == id {
			row.Active = false // soft-delete parity with the postgres adapter
		}
	}
	return &pb.DeleteSubscriptionGroupProductPlanStaffResponse{Success: true}, nil
}

// ----- helpers -------------------------------------------------------------

func activeEdge(id, staffID, role, ws string) *pb.SubscriptionGroupProductPlanStaff {
	return &pb.SubscriptionGroupProductPlanStaff{
		Id:                  id,
		SubscriptionGroupId: "sg-1",
		ProductPlanId:       "pp-1",
		StaffId:             staffID,
		Role:                role,
		WorkspaceId:         ws,
		Active:              true,
	}
}

func inactiveEdge(id, staffID, role, ws string) *pb.SubscriptionGroupProductPlanStaff {
	e := activeEdge(id, staffID, role, ws)
	e.Active = false // post-clear (soft-deleted) state
	return e
}

func newAssignSGPPSUC(sgpps *mockAssignSGPPSRepo, pps *mockPPSRepo, pp *mockPPRepo, sg *mockSGRepo) *AssignSubscriptionGroupProductPlanStaffUseCase {
	return NewAssignSubscriptionGroupProductPlanStaffUseCase(
		AssignSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: sgpps,
			ProductPlanStaff:                  pps,
			ProductPlan:                       pp,
			SubscriptionGroup:                 sg,
		},
		AssignSubscriptionGroupProductPlanStaffServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDSGPPS{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func assignReq(staffID, role string) *AssignSubscriptionGroupProductPlanStaffRequest {
	return &AssignSubscriptionGroupProductPlanStaffRequest{
		WorkspaceID:         "ws-1",
		SubscriptionGroupID: "sg-1",
		ProductPlanID:       "pp-1",
		StaffID:             staffID,
		Role:                role,
	}
}

func eligiblePPS() *mockPPSRepo {
	return &mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{eligibleRow("ws-1")}}
}

// ----- tests ---------------------------------------------------------------

// Update-existing: an active edge exists → the SAME edge is updated, never a new one.
func TestAssign_UpdatesExistingEdge(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{rows: []*pb.SubscriptionGroupProductPlanStaff{activeEdge("sgpps-1", "staff-old", "primary", "ws-1")}}
	uc := newAssignSGPPSUC(sgpps, eligiblePPS(), &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	resp, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("staff-1", "primary"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Outcome != AssignOutcomeUpdated {
		t.Fatalf("expected updated outcome, got %q", resp.Outcome)
	}
	if sgpps.updateCalls != 1 || sgpps.createCalls != 0 {
		t.Fatalf("expected update-once/create-none, got update=%d create=%d", sgpps.updateCalls, sgpps.createCalls)
	}
	if sgpps.rows[0].GetStaffId() != "staff-1" {
		t.Fatalf("expected staff reassigned to staff-1, got %q", sgpps.rows[0].GetStaffId())
	}
}

// Create-new: no active edge → a fresh edge is created.
func TestAssign_CreatesWhenNoActiveEdge(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{}
	uc := newAssignSGPPSUC(sgpps, eligiblePPS(), &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	resp, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("staff-1", "primary"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Outcome != AssignOutcomeCreated {
		t.Fatalf("expected created outcome, got %q", resp.Outcome)
	}
	if sgpps.createCalls != 1 || sgpps.updateCalls != 0 {
		t.Fatalf("expected create-once/update-none, got create=%d update=%d", sgpps.createCalls, sgpps.updateCalls)
	}
	if resp.Edge.GetStaffId() != "staff-1" || resp.Edge.GetWorkspaceId() != "ws-1" {
		t.Fatalf("created edge mismatch: staff=%q ws=%q", resp.Edge.GetStaffId(), resp.Edge.GetWorkspaceId())
	}
}

// Clear: empty staff → the active edge is soft-deactivated (active=false), no create/update.
func TestAssign_ClearsActiveEdge(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{rows: []*pb.SubscriptionGroupProductPlanStaff{activeEdge("sgpps-1", "staff-1", "primary", "ws-1")}}
	uc := newAssignSGPPSUC(sgpps, eligiblePPS(), &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	resp, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Outcome != AssignOutcomeCleared {
		t.Fatalf("expected cleared outcome, got %q", resp.Outcome)
	}
	if sgpps.deleteCalls != 1 || sgpps.createCalls != 0 || sgpps.updateCalls != 0 {
		t.Fatalf("expected delete-once only, got delete=%d create=%d update=%d", sgpps.deleteCalls, sgpps.createCalls, sgpps.updateCalls)
	}
	if sgpps.rows[0].GetActive() {
		t.Fatalf("expected edge soft-deactivated")
	}
}

// Clear with no active edge is a no-op.
func TestAssign_ClearNoActiveEdge_Noop(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{}
	uc := newAssignSGPPSUC(sgpps, eligiblePPS(), &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	resp, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Outcome != AssignOutcomeNoop || sgpps.deleteCalls != 0 {
		t.Fatalf("expected no-op clear, got outcome=%q delete=%d", resp.Outcome, sgpps.deleteCalls)
	}
}

// Ineligible on the create path is rejected fail-loud; no write occurs.
func TestAssign_IneligibleRejected(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{}
	uc := newAssignSGPPSUC(sgpps, &mockPPSRepo{rows: nil}, &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("staff-1", "primary"))
	if err == nil {
		t.Fatalf("expected ineligibility rejection")
	}
	if !strings.Contains(err.Error(), "eligible") {
		t.Errorf("expected eligibility error, got %q", err.Error())
	}
	if sgpps.createCalls != 0 || sgpps.updateCalls != 0 {
		t.Errorf("no write may run on rejection, got create=%d update=%d", sgpps.createCalls, sgpps.updateCalls)
	}
}

// Re-saving the same staff+role on an active edge is a no-op (idempotent).
func TestAssign_SameValueIsNoop(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{rows: []*pb.SubscriptionGroupProductPlanStaff{activeEdge("sgpps-1", "staff-1", "primary", "ws-1")}}
	uc := newAssignSGPPSUC(sgpps, eligiblePPS(), &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	resp, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("staff-1", "primary"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Outcome != AssignOutcomeNoop || sgpps.updateCalls != 0 {
		t.Fatalf("expected idempotent no-op, got outcome=%q update=%d", resp.Outcome, sgpps.updateCalls)
	}
}

// C10: clear → re-assign the SAME staff must reactivate the soft-deleted row,
// never INSERT a duplicate (the plain INSERT collides with
// uq_subscription_group_product_plan_staff_1, which has no active filter). Full
// roundtrip through the use case: assign → clear → assign-same. The SAME row id
// comes back active, its role updated, with no new row and no create call.
func TestAssign_ReactivatesClearedEdge_Roundtrip(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{rows: []*pb.SubscriptionGroupProductPlanStaff{activeEdge("sgpps-1", "staff-1", "primary", "ws-1")}}
	uc := newAssignSGPPSUC(sgpps, eligiblePPS(), &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")

	// Clear the active edge → soft-deactivated (still occupies the triple).
	if _, err := uc.Execute(ctx, assignReq("", "")); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if sgpps.deleteCalls != 1 || sgpps.rows[0].GetActive() {
		t.Fatalf("expected the edge soft-deactivated after clear (delete=%d active=%v)", sgpps.deleteCalls, sgpps.rows[0].GetActive())
	}

	// Re-assign the SAME staff (with a different role) → reactivate the SAME row.
	resp, err := uc.Execute(ctx, assignReq("staff-1", "access"))
	if err != nil {
		t.Fatalf("reassign-after-clear failed (C10 regression): %v", err)
	}
	if sgpps.createCalls != 0 {
		t.Fatalf("reactivation must not INSERT a duplicate, got create=%d", sgpps.createCalls)
	}
	if sgpps.updateCalls != 1 {
		t.Fatalf("expected one reactivating update, got update=%d", sgpps.updateCalls)
	}
	if len(sgpps.rows) != 1 {
		t.Fatalf("expected the SAME single row, got %d rows", len(sgpps.rows))
	}
	if sgpps.rows[0].GetId() != "sgpps-1" {
		t.Fatalf("expected same row id sgpps-1 reactivated, got %q", sgpps.rows[0].GetId())
	}
	if !sgpps.rows[0].GetActive() {
		t.Fatalf("expected the row reactivated (active=true)")
	}
	if sgpps.rows[0].GetRole() != "access" {
		t.Fatalf("expected role updated to access on reactivation, got %q", sgpps.rows[0].GetRole())
	}
	if resp.Outcome != AssignOutcomeCreated {
		t.Fatalf("expected created outcome for a re-populated offering, got %q", resp.Outcome)
	}
	if resp.Edge.GetId() != "sgpps-1" {
		t.Fatalf("response edge should be the reactivated row, got %q", resp.Edge.GetId())
	}
}

// The eligibility guard rides the reactivate path too (defense-in-depth): a
// staff no longer eligible for the product plan cannot be reactivated, and no
// write runs — the soft-deleted row stays inactive.
func TestAssign_ReactivateIneligible_Rejected(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{rows: []*pb.SubscriptionGroupProductPlanStaff{inactiveEdge("sgpps-1", "staff-1", "primary", "ws-1")}}
	// Empty eligibility pool → staff-1 is not eligible for pp-1.
	uc := newAssignSGPPSUC(sgpps, &mockPPSRepo{rows: nil}, &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("staff-1", "primary"))
	if err == nil {
		t.Fatalf("expected ineligible reactivation to be rejected")
	}
	if !strings.Contains(err.Error(), "eligible") {
		t.Errorf("expected eligibility error, got %q", err.Error())
	}
	if sgpps.updateCalls != 0 || sgpps.createCalls != 0 {
		t.Errorf("no write may run on rejection, got update=%d create=%d", sgpps.updateCalls, sgpps.createCalls)
	}
	if sgpps.rows[0].GetActive() {
		t.Errorf("the soft-deleted row must stay inactive on rejection")
	}
}

// C16 (sibling of C10, UPDATE branch): reassigning an offering to staffA whose
// soft-deleted (group, plan, staffA) row survives must REACTIVATE that corpse and
// deactivate the current staffB edge — never mutate staffB's row into the
// (group, plan, staffA) triple (a plain UPDATE there collides with
// uq_subscription_group_product_plan_staff_1, which has no active filter).
// Direct setup: staffB active + staffA corpse.
func TestAssign_ChangeServicer_ReactivatesTargetCorpse(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{rows: []*pb.SubscriptionGroupProductPlanStaff{
		activeEdge("sgpps-B", "staff-2", "primary", "ws-1"),   // current servicer
		inactiveEdge("sgpps-A", "staff-1", "primary", "ws-1"), // soft-deleted corpse for the target
	}}
	bothEligible := &mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{
		{Id: "pps-1", ProductPlanId: "pp-1", StaffId: "staff-1", WorkspaceId: "ws-1", Active: true},
		{Id: "pps-2", ProductPlanId: "pp-1", StaffId: "staff-2", WorkspaceId: "ws-1", Active: true},
	}}
	uc := newAssignSGPPSUC(sgpps, bothEligible, &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	resp, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), assignReq("staff-1", "access"))
	if err != nil {
		t.Fatalf("reassign onto a target-triple corpse failed (C16 regression): %v", err)
	}
	if resp.Outcome != AssignOutcomeUpdated {
		t.Fatalf("expected updated outcome, got %q", resp.Outcome)
	}
	if sgpps.createCalls != 0 {
		t.Fatalf("reassign must NOT INSERT (collision) — reactivate the corpse, got create=%d", sgpps.createCalls)
	}
	if len(sgpps.rows) != 2 {
		t.Fatalf("expected the two existing rows (no new row), got %d", len(sgpps.rows))
	}
	// The corpse (sgpps-A) is now the active servicer with the requested role...
	corpse := findRowByID(sgpps.rows, "sgpps-A")
	if corpse == nil || !corpse.GetActive() || corpse.GetStaffId() != "staff-1" || corpse.GetRole() != "access" {
		t.Fatalf("expected sgpps-A reactivated to (staff-1, access, active), got %+v", corpse)
	}
	// ...and the prior active edge (sgpps-B) is soft-deactivated.
	old := findRowByID(sgpps.rows, "sgpps-B")
	if old == nil || old.GetActive() {
		t.Fatalf("expected the prior servicer sgpps-B soft-deactivated, got %+v", old)
	}
	if resp.Edge.GetId() != "sgpps-A" {
		t.Fatalf("response edge should be the reactivated corpse sgpps-A, got %q", resp.Edge.GetId())
	}
}

// C16 full ROUNDTRIP through the use case (staffB→staffA-with-corpse): the corpse
// is produced ORGANICALLY (assign staff-1 → clear → assign staff-2 leaves a
// staff-1 corpse), then reassigning back to staff-1 must reactivate it, never
// collide. Proves the UPDATE-branch path end-to-end, not just a hand-built corpse.
func TestAssign_ChangeServicer_CorpseRoundtrip(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{rows: []*pb.SubscriptionGroupProductPlanStaff{activeEdge("sgpps-1", "staff-1", "primary", "ws-1")}}
	bothEligible := &mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{
		{Id: "pps-1", ProductPlanId: "pp-1", StaffId: "staff-1", WorkspaceId: "ws-1", Active: true},
		{Id: "pps-2", ProductPlanId: "pp-1", StaffId: "staff-2", WorkspaceId: "ws-1", Active: true},
	}}
	uc := newAssignSGPPSUC(sgpps, bothEligible, &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")

	// Clear staff-1 → sgpps-1 soft-deleted (the future corpse; still occupies the triple).
	if _, err := uc.Execute(ctx, assignReq("", "")); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	// Assign staff-2 fresh → a NEW active edge (no staff-2 corpse to reactivate).
	if _, err := uc.Execute(ctx, assignReq("staff-2", "primary")); err != nil {
		t.Fatalf("assign staff-2 failed: %v", err)
	}
	if sgpps.createCalls != 1 {
		t.Fatalf("assigning a never-before staff must CREATE once, got create=%d", sgpps.createCalls)
	}

	// Reassign back to staff-1: its soft-deleted corpse survives the (group, plan,
	// staff-1) triple → reactivate it, deactivate the staff-2 edge, NO new INSERT.
	resp, err := uc.Execute(ctx, assignReq("staff-1", "primary"))
	if err != nil {
		t.Fatalf("reassign back to staff-1 failed (C16 regression): %v", err)
	}
	if resp.Outcome != AssignOutcomeUpdated {
		t.Fatalf("expected updated outcome, got %q", resp.Outcome)
	}
	if sgpps.createCalls != 1 {
		t.Fatalf("reassign onto the corpse must NOT create a second row, got create=%d", sgpps.createCalls)
	}
	if len(sgpps.rows) != 2 {
		t.Fatalf("expected exactly two rows (corpse + staff-2 edge), got %d", len(sgpps.rows))
	}
	staff1Row := findRowByStaff(sgpps.rows, "staff-1")
	if staff1Row == nil || !staff1Row.GetActive() || staff1Row.GetId() != "sgpps-1" {
		t.Fatalf("expected the original sgpps-1 corpse reactivated for staff-1, got %+v", staff1Row)
	}
	staff2Row := findRowByStaff(sgpps.rows, "staff-2")
	if staff2Row == nil || staff2Row.GetActive() {
		t.Fatalf("expected the staff-2 edge soft-deactivated, got %+v", staff2Row)
	}
}

func findRowByID(rows []*pb.SubscriptionGroupProductPlanStaff, id string) *pb.SubscriptionGroupProductPlanStaff {
	for _, r := range rows {
		if r.GetId() == id {
			return r
		}
	}
	return nil
}

func findRowByStaff(rows []*pb.SubscriptionGroupProductPlanStaff, staffID string) *pb.SubscriptionGroupProductPlanStaff {
	for _, r := range rows {
		if r.GetStaffId() == staffID {
			return r
		}
	}
	return nil
}

// Missing target ids fail closed before any lookup.
func TestAssign_MissingTarget_Rejects(t *testing.T) {
	sgpps := &mockAssignSGPPSRepo{}
	uc := newAssignSGPPSUC(sgpps, eligiblePPS(), &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}}, &mockSGRepo{group: group("plan-1", "ws-1")})

	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), &AssignSubscriptionGroupProductPlanStaffRequest{WorkspaceID: "ws-1", StaffID: "staff-1"})
	if err == nil {
		t.Fatalf("expected rejection on missing subscription_group_id/product_plan_id")
	}
	if sgpps.createCalls != 0 || sgpps.updateCalls != 0 || sgpps.deleteCalls != 0 {
		t.Errorf("no write may run, got create=%d update=%d delete=%d", sgpps.createCalls, sgpps.updateCalls, sgpps.deleteCalls)
	}
}
