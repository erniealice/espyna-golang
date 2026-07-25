package subscription_group_product_plan_staff

// UpdateSubscriptionGroupProductPlanStaffUseCase — merge semantics and
// re-validation on the class-edge UPDATE path (coverage-audit-20260725 W-G2;
// also the update-side half of A-G7, plus the v2_anchors_required_together /
// eligibility_not_active reject branches of W-G4 that are only reachable here).
//
// What this file pins:
//   - effectiveEdge merges the update body OVER the persisted row, so a partial
//     body (e.g. role only) still re-derives the legacy f8/f9/f10 dual-write
//     from the persisted class + eligibility rows;
//   - firstSetPtr's contract on f14: a non-nil pointer to "" is an explicit
//     "clear back to All Phases" and MUST beat the persisted phase, whereas a
//     nil pointer inherits it. This is the upstream half of Bug #3 — the
//     adapter can only translate the cleared pointer to SQL NULL if the use
//     case hands it a non-nil-empty pointer in the first place;
//   - the full v2 re-validation (class/eligibility/phase/duplicate) runs again
//     on the MERGED edge, not just on the fields the caller happened to send;
//   - the repository is never called when validation fails.
//
// No database. Fully self-contained mocks (in-package, no build tags), matching
// the stub shapes in class_edge_v2_test.go / create_subscription_group_product_
// plan_staff_test.go but deliberately independent of both, per this package's
// existing convention — this file never needs either of them to change.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	sgpppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// ----- update-local mocks ----------------------------------------------------

type updClassRepo struct {
	sgpppb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	byID map[string]*sgpppb.SubscriptionGroupProductPlan
}

func (m *updClassRepo) ReadSubscriptionGroupProductPlan(_ context.Context, req *sgpppb.ReadSubscriptionGroupProductPlanRequest) (*sgpppb.ReadSubscriptionGroupProductPlanResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &sgpppb.ReadSubscriptionGroupProductPlanResponse{Success: true}, nil
	}
	return &sgpppb.ReadSubscriptionGroupProductPlanResponse{Success: true, Data: []*sgpppb.SubscriptionGroupProductPlan{row}}, nil
}

type updPlanRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	byID map[string]*productplanpb.ProductPlan
}

func (m *updPlanRepo) ReadProductPlan(_ context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &productplanpb.ReadProductPlanResponse{Success: true}, nil
	}
	return &productplanpb.ReadProductPlanResponse{Success: true, Data: []*productplanpb.ProductPlan{row}}, nil
}

type updEligibilityRepo struct {
	productplanstaffpb.UnimplementedProductPlanStaffDomainServiceServer
	byID map[string]*productplanstaffpb.ProductPlanStaff
}

func (m *updEligibilityRepo) ReadProductPlanStaff(_ context.Context, req *productplanstaffpb.ReadProductPlanStaffRequest) (*productplanstaffpb.ReadProductPlanStaffResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &productplanstaffpb.ReadProductPlanStaffResponse{Success: true}, nil
	}
	return &productplanstaffpb.ReadProductPlanStaffResponse{Success: true, Data: []*productplanstaffpb.ProductPlanStaff{row}}, nil
}

func (m *updEligibilityRepo) ListProductPlanStaffs(_ context.Context, _ *productplanstaffpb.ListProductPlanStaffsRequest) (*productplanstaffpb.ListProductPlanStaffsResponse, error) {
	var rows []*productplanstaffpb.ProductPlanStaff
	for _, r := range m.byID {
		rows = append(rows, r)
	}
	return &productplanstaffpb.ListProductPlanStaffsResponse{Success: true, Data: rows}, nil
}

type updGroupRepo struct {
	subscriptiongrouppb.UnimplementedSubscriptionGroupDomainServiceServer
	group *subscriptiongrouppb.SubscriptionGroup
}

func (m *updGroupRepo) ReadSubscriptionGroup(_ context.Context, _ *subscriptiongrouppb.ReadSubscriptionGroupRequest) (*subscriptiongrouppb.ReadSubscriptionGroupResponse, error) {
	if m.group == nil {
		return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true}, nil
	}
	return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true, Data: []*subscriptiongrouppb.SubscriptionGroup{m.group}}, nil
}

type updPhaseRepo struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	byID map[string]*jobtemplatephasepb.JobTemplatePhase
}

func (m *updPhaseRepo) ReadJobTemplatePhase(_ context.Context, req *jobtemplatephasepb.ReadJobTemplatePhaseRequest) (*jobtemplatephasepb.ReadJobTemplatePhaseResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true}, nil
	}
	return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true, Data: []*jobtemplatephasepb.JobTemplatePhase{row}}, nil
}

type stubIDUpd struct{}

func (stubIDUpd) GenerateID() string                   { return "sgpps-upd" }
func (stubIDUpd) GenerateIDWithPrefix(p string) string { return p + "-sgpps-upd" }
func (stubIDUpd) IsEnabled() bool                      { return true }
func (stubIDUpd) GetProviderInfo() string              { return "stub" }

// ----- update-local edge repo ------------------------------------------------

// updEdgeRepo records every call so a rejected update can be proven to have
// written nothing. Read returns `persisted` (nil = row not found); List returns
// `siblings` verbatim — the duplicate check's own client-side filtering is what
// is under test, so the mock deliberately does not pre-filter.
type updEdgeRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	persisted *pb.SubscriptionGroupProductPlanStaff
	siblings  []*pb.SubscriptionGroupProductPlanStaff
	listErr   error

	readCalls   int
	updateCalls int
	lastUpdate  *pb.SubscriptionGroupProductPlanStaff
}

func (m *updEdgeRepo) ReadSubscriptionGroupProductPlanStaff(_ context.Context, _ *pb.ReadSubscriptionGroupProductPlanStaffRequest) (*pb.ReadSubscriptionGroupProductPlanStaffResponse, error) {
	m.readCalls++
	if m.persisted == nil {
		return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Success: true}, nil
	}
	return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{m.persisted}}, nil
}

func (m *updEdgeRepo) ListSubscriptionGroupProductPlanStaffs(_ context.Context, _ *pb.ListSubscriptionGroupProductPlanStaffsRequest) (*pb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &pb.ListSubscriptionGroupProductPlanStaffsResponse{Success: true, Data: m.siblings}, nil
}

func (m *updEdgeRepo) UpdateSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.UpdateSubscriptionGroupProductPlanStaffRequest) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
	m.updateCalls++
	m.lastUpdate = req.GetData()
	return &pb.UpdateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{req.GetData()}}, nil
}

// ----- fixture ---------------------------------------------------------------

type updFixture struct {
	classRepo *updClassRepo
	planRepo  *updPlanRepo
	ppsRepo   *updEligibilityRepo
	groupRepo *updGroupRepo
	phaseRepo *updPhaseRepo
	edgeRepo  *updEdgeRepo
}

func strptr(s string) *string { return &s }

func updClass(id, sgID, ppID, templateID string) *sgpppb.SubscriptionGroupProductPlan {
	return &sgpppb.SubscriptionGroupProductPlan{
		Id: id, SubscriptionGroupId: sgID, ProductPlanId: ppID, JobTemplateId: templateID,
		Active: true, Status: sgpppb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_ACTIVE,
	}
}

func updPlan(id, planID string) *productplanpb.ProductPlan {
	return &productplanpb.ProductPlan{Id: id, PlanId: planID}
}

func updEligibility(id, productPlanID, staffID string) *productplanstaffpb.ProductPlanStaff {
	return &productplanstaffpb.ProductPlanStaff{Id: id, ProductPlanId: productPlanID, StaffId: staffID, Active: true, WorkspaceId: "ws-1"}
}

func updGroup(planID string) *subscriptiongrouppb.SubscriptionGroup {
	ws := "ws-1"
	return &subscriptiongrouppb.SubscriptionGroup{Id: "sg-1", PlanId: &planID, WorkspaceId: &ws, Active: true}
}

func updPhase(id, templateID string) *jobtemplatephasepb.JobTemplatePhase {
	return &jobtemplatephasepb.JobTemplatePhase{Id: id, JobTemplateId: templateID, Active: true}
}

// newUpdFixture builds the standard world:
//
//	class-1 = (sg-1 x pp-1) anchored to curriculum tmpl-1
//	pps-1   = ACTIVE eligibility (pp-1, staff-1) in ws-1
//	phase-a, phase-b belong to tmpl-1; phase-foreign belongs to tmpl-OTHER
//
// The persisted edge-1 row carries DELIBERATELY STALE legacy FKs
// (sg-STALE/pp-STALE/staff-STALE) so that any test asserting the dual-write can
// distinguish "re-derived from class+pps" from "inherited from the old row".
func newUpdFixture() *updFixture {
	f := &updFixture{
		classRepo: &updClassRepo{byID: map[string]*sgpppb.SubscriptionGroupProductPlan{}},
		planRepo:  &updPlanRepo{byID: map[string]*productplanpb.ProductPlan{}},
		ppsRepo:   &updEligibilityRepo{byID: map[string]*productplanstaffpb.ProductPlanStaff{}},
		groupRepo: &updGroupRepo{},
		phaseRepo: &updPhaseRepo{byID: map[string]*jobtemplatephasepb.JobTemplatePhase{}},
		edgeRepo:  &updEdgeRepo{},
	}
	f.classRepo.byID["class-1"] = updClass("class-1", "sg-1", "pp-1", "tmpl-1")
	f.planRepo.byID["pp-1"] = updPlan("pp-1", "plan-1")
	f.ppsRepo.byID["pps-1"] = updEligibility("pps-1", "pp-1", "staff-1")
	f.groupRepo.group = updGroup("plan-1")
	f.phaseRepo.byID["phase-a"] = updPhase("phase-a", "tmpl-1")
	f.phaseRepo.byID["phase-b"] = updPhase("phase-b", "tmpl-1")
	f.phaseRepo.byID["phase-foreign"] = updPhase("phase-foreign", "tmpl-OTHER")
	f.edgeRepo.persisted = &pb.SubscriptionGroupProductPlanStaff{
		Id:                             "edge-1",
		SubscriptionGroupId:            "sg-STALE",
		ProductPlanId:                  "pp-STALE",
		StaffId:                        "staff-STALE",
		Role:                           "primary",
		Active:                         true,
		SubscriptionGroupProductPlanId: strptr("class-1"),
		ProductPlanStaffId:             strptr("pps-1"),
		JobTemplatePhaseId:             strptr("phase-a"),
	}
	return f
}

func (f *updFixture) updateUC() *UpdateSubscriptionGroupProductPlanStaffUseCase {
	return NewUpdateSubscriptionGroupProductPlanStaffUseCase(
		UpdateSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: f.edgeRepo,
			ProductPlanStaff:                  f.ppsRepo,
			ProductPlan:                       f.planRepo,
			SubscriptionGroup:                 f.groupRepo,
			SubscriptionGroupProductPlan:      f.classRepo,
			JobTemplatePhase:                  f.phaseRepo,
		},
		UpdateSubscriptionGroupProductPlanStaffServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDUpd{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func updCtx() context.Context {
	return appcontext.WithWorkspaceID(context.Background(), "ws-1")
}

// updReq builds an update body. Fields left nil are OMITTED by the caller — the
// exact partial-body shape protojson produces when the drawer only sends the
// fields the user touched.
func updReq(id string, mutate func(d *pb.SubscriptionGroupProductPlanStaff)) *pb.UpdateSubscriptionGroupProductPlanStaffRequest {
	d := &pb.SubscriptionGroupProductPlanStaff{Id: id}
	if mutate != nil {
		mutate(d)
	}
	return &pb.UpdateSubscriptionGroupProductPlanStaffRequest{Data: d}
}

// ----- merge semantics --------------------------------------------------------

// A body that touches only `role` must still (a) inherit f12/f13/f14 from the
// persisted row and (b) re-derive the legacy f8/f9/f10 dual-write from the
// persisted class + eligibility rows — NOT pass the stale persisted triple
// through. update_subscription_group_product_plan_staff.go:74-79 states exactly
// this requirement.
func TestUpdateEdge_PartialBody_InheritsAnchorsAndRederivesLegacyFKs(t *testing.T) {
	f := newUpdFixture()
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.Role = "secondary" })

	if _, err := f.updateUC().Execute(updCtx(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.edgeRepo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", f.edgeRepo.updateCalls)
	}
	got := f.edgeRepo.lastUpdate
	if got.GetSubscriptionGroupProductPlanId() != "class-1" {
		t.Errorf("f12 = %q, want class-1 inherited from the persisted row", got.GetSubscriptionGroupProductPlanId())
	}
	if got.GetProductPlanStaffId() != "pps-1" {
		t.Errorf("f13 = %q, want pps-1 inherited from the persisted row", got.GetProductPlanStaffId())
	}
	if got.JobTemplatePhaseId == nil || got.GetJobTemplatePhaseId() != "phase-a" {
		t.Errorf("f14 = %v, want phase-a inherited from the persisted row", got.JobTemplatePhaseId)
	}
	// Dual-write re-derivation: the persisted row's stale triple must be replaced.
	if got.GetSubscriptionGroupId() != "sg-1" {
		t.Errorf("f8 = %q, want sg-1 re-derived from class-1 (stale value was sg-STALE)", got.GetSubscriptionGroupId())
	}
	if got.GetProductPlanId() != "pp-1" {
		t.Errorf("f9 = %q, want pp-1 re-derived from class-1 (stale value was pp-STALE)", got.GetProductPlanId())
	}
	if got.GetStaffId() != "staff-1" {
		t.Errorf("f10 = %q, want staff-1 re-derived from pps-1 (stale value was staff-STALE)", got.GetStaffId())
	}
	// The caller's actual change must survive the merge echo.
	if got.GetRole() != "secondary" {
		t.Errorf("role = %q, want the caller's secondary", got.GetRole())
	}
	if got.DateModified == nil || got.DateModifiedString == nil {
		t.Errorf("modified stamps must be set on the persisted body, got %v / %v", got.DateModified, got.DateModifiedString)
	}
}

// A caller-supplied phase beats the persisted one.
func TestUpdateEdge_ExplicitPhase_OverridesPersistedPhase(t *testing.T) {
	f := newUpdFixture() // persisted f14 = phase-a
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.JobTemplatePhaseId = strptr("phase-b") })

	if _, err := f.updateUC().Execute(updCtx(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.edgeRepo.lastUpdate.GetJobTemplatePhaseId() != "phase-b" {
		t.Errorf("f14 = %q, want the caller's phase-b", f.edgeRepo.lastUpdate.GetJobTemplatePhaseId())
	}
}

// firstSetPtr's whole reason to exist: a non-nil pointer to "" means "clear the
// phase back to All Phases" and must NOT fall through to the persisted phase.
// The pointer must also stay NON-NIL on the persisted body — the postgres
// adapter distinguishes nil (field absent, leave alone) from non-nil-empty
// (write SQL NULL). Collapsing it to nil silently keeps the old phase.
func TestUpdateEdge_ClearPhase_NonNilEmptyPointerWins(t *testing.T) {
	f := newUpdFixture() // persisted f14 = phase-a
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.JobTemplatePhaseId = strptr("") })

	if _, err := f.updateUC().Execute(updCtx(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := f.edgeRepo.lastUpdate
	if got.JobTemplatePhaseId == nil {
		t.Fatalf("f14 pointer was collapsed to nil; the adapter can no longer tell 'clear' from 'omitted'")
	}
	if v := got.GetJobTemplatePhaseId(); v != "" {
		t.Errorf("f14 = %q, want \"\" (All Phases); the persisted phase-a must not win over an explicit clear", v)
	}
}

// The mirror case: an omitted phase inherits, so "clear" and "omit" are
// genuinely different inputs rather than both landing on the same result.
func TestUpdateEdge_OmittedPhase_InheritsPersistedPhase(t *testing.T) {
	f := newUpdFixture()
	req := updReq("edge-1", nil) // JobTemplatePhaseId left nil

	if _, err := f.updateUC().Execute(updCtx(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.edgeRepo.lastUpdate.GetJobTemplatePhaseId() != "phase-a" {
		t.Errorf("f14 = %q, want the persisted phase-a", f.edgeRepo.lastUpdate.GetJobTemplatePhaseId())
	}
}

// f12/f13 use the same explicit-presence rule, so clearing exactly one of them
// leaves the pair half-set and must be refused (class_edge_v2.go:41-45) rather
// than silently inheriting the other anchor.
func TestUpdateEdge_ClearingOneAnchorOnly_Rejects(t *testing.T) {
	f := newUpdFixture()
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.SubscriptionGroupProductPlanId = strptr("") })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected rejection: f12 explicitly cleared while f13 stays set")
	}
	if !strings.Contains(err.Error(), "together") {
		t.Errorf("expected a v2_anchors_required_together error, got %q", err.Error())
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// ----- re-validation on the merged edge --------------------------------------

// A-G7: Create's invariants must be re-proven on Update. Re-pointing f13 at an
// unresolvable eligibility id is the update-side twin of
// TestCreateEdgeV2_ForeignPPS_Rejects.
func TestUpdateEdge_ForeignPPS_Rejects(t *testing.T) {
	f := newUpdFixture()
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.ProductPlanStaffId = strptr("pps-foreign") })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected rejection for a foreign/unresolvable product_plan_staff id on update")
	}
	if !strings.Contains(err.Error(), "eligibility") {
		t.Errorf("expected an eligibility-not-found error, got %q", err.Error())
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// A-G7: re-pointing f14 at a phase from another curriculum must be refused on
// update exactly as on create.
func TestUpdateEdge_UncoveredPhase_Rejects(t *testing.T) {
	f := newUpdFixture()
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.JobTemplatePhaseId = strptr("phase-foreign") })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected rejection: phase-foreign belongs to tmpl-OTHER, not the class's tmpl-1")
	}
	if !strings.Contains(err.Error(), "curriculum") {
		t.Errorf("expected a phase-not-in-template error, got %q", err.Error())
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// W-G4 branch reachable from the update path: a teacher whose eligibility was
// deactivated after the edge was created must not survive a partial update that
// does not even mention f13 (the TOCTOU guard the use case's header documents).
func TestUpdateEdge_DeactivatedEligibility_Rejects(t *testing.T) {
	f := newUpdFixture()
	f.ppsRepo.byID["pps-1"].Active = false
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.Role = "secondary" })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected rejection: the merged edge's eligibility row is no longer active")
	}
	if !strings.Contains(err.Error(), "not active") {
		t.Errorf("expected an eligibility-not-active error, got %q", err.Error())
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// An update whose class anchor no longer resolves must fail closed.
func TestUpdateEdge_ClassNotFound_Rejects(t *testing.T) {
	f := newUpdFixture()
	delete(f.classRepo.byID, "class-1")
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.Role = "secondary" })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected rejection: the merged edge's class no longer resolves")
	}
	if !strings.Contains(err.Error(), "class not found") {
		t.Errorf("expected a class-not-found error, got %q", err.Error())
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// ----- duplicate check with excludeID ----------------------------------------

// The row being updated is itself in the class's edge list; excluding it is what
// makes a no-op-anchors update (role-only edit) possible at all. Without the
// excludeID skip every such edit would 4xx as a duplicate of itself.
func TestUpdateEdge_SelfRow_NotADuplicate(t *testing.T) {
	f := newUpdFixture()
	f.edgeRepo.siblings = []*pb.SubscriptionGroupProductPlanStaff{
		{Id: "edge-1", Active: true, SubscriptionGroupProductPlanId: strptr("class-1"), ProductPlanStaffId: strptr("pps-1"), JobTemplatePhaseId: strptr("phase-a")},
	}
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.Role = "secondary" })

	if _, err := f.updateUC().Execute(updCtx(), req); err != nil {
		t.Fatalf("the row being updated must not collide with itself: %v", err)
	}
	if f.edgeRepo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", f.edgeRepo.updateCalls)
	}
}

// A DIFFERENT active row already occupying the merged (class, pps, phase)
// triple is a real collision — moving edge-1 onto phase-b when a sibling
// already holds it must be refused (uq_sgpps_class_pps_phase).
func TestUpdateEdge_SiblingHoldsTargetTriple_Rejects(t *testing.T) {
	f := newUpdFixture()
	f.edgeRepo.siblings = []*pb.SubscriptionGroupProductPlanStaff{
		{Id: "edge-1", Active: true, SubscriptionGroupProductPlanId: strptr("class-1"), ProductPlanStaffId: strptr("pps-1"), JobTemplatePhaseId: strptr("phase-a")},
		{Id: "edge-2", Active: true, SubscriptionGroupProductPlanId: strptr("class-1"), ProductPlanStaffId: strptr("pps-1"), JobTemplatePhaseId: strptr("phase-b")},
	}
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.JobTemplatePhaseId = strptr("phase-b") })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected duplicate rejection: edge-2 already holds (class-1, pps-1, phase-b)")
	}
	if !strings.Contains(err.Error(), "already assigned") {
		t.Errorf("expected a friendly duplicate error, got %q", err.Error())
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// An INACTIVE (cleared) sibling on the target triple is not a collision — the
// slot is free, and re-taking it must be allowed.
func TestUpdateEdge_InactiveSiblingOnTargetTriple_NotADuplicate(t *testing.T) {
	f := newUpdFixture()
	f.edgeRepo.siblings = []*pb.SubscriptionGroupProductPlanStaff{
		{Id: "edge-2", Active: false, SubscriptionGroupProductPlanId: strptr("class-1"), ProductPlanStaffId: strptr("pps-1"), JobTemplatePhaseId: strptr("phase-b")},
	}
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.JobTemplatePhaseId = strptr("phase-b") })

	if _, err := f.updateUC().Execute(updCtx(), req); err != nil {
		t.Fatalf("an inactive sibling must not block re-taking the slot: %v", err)
	}
	if f.edgeRepo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", f.edgeRepo.updateCalls)
	}
}

// A failing uniqueness List must fail CLOSED, never write.
func TestUpdateEdge_DuplicateCheckListError_FailsClosed(t *testing.T) {
	f := newUpdFixture()
	f.edgeRepo.listErr = errors.New("boom")
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.Role = "secondary" })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected fail-closed rejection when the uniqueness List errors")
	}
	if !strings.Contains(err.Error(), "uniqueness") {
		t.Errorf("expected a duplicate-check-failed error, got %q", err.Error())
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// ----- validation rejects never reach the repository -------------------------

func TestUpdateEdge_MalformedRequest_RepositoryNotTouched(t *testing.T) {
	cases := []struct {
		name string
		req  *pb.UpdateSubscriptionGroupProductPlanStaffRequest
	}{
		{"nil request", nil},
		{"nil data", &pb.UpdateSubscriptionGroupProductPlanStaffRequest{}},
		{"empty id", &pb.UpdateSubscriptionGroupProductPlanStaffRequest{Data: &pb.SubscriptionGroupProductPlanStaff{Role: "secondary"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUpdFixture()
			_, err := f.updateUC().Execute(updCtx(), tc.req)
			if err == nil {
				t.Fatalf("expected a data-required rejection")
			}
			if f.edgeRepo.readCalls != 0 {
				t.Errorf("no read may run on a malformed request, got %d", f.edgeRepo.readCalls)
			}
			if f.edgeRepo.updateCalls != 0 {
				t.Errorf("no update may run on a malformed request, got %d", f.edgeRepo.updateCalls)
			}
		})
	}
}

// A target id that does not resolve must 404-shape, not fall through to a write
// against a half-built merged edge.
func TestUpdateEdge_RowNotFound_Rejects(t *testing.T) {
	f := newUpdFixture()
	f.edgeRepo.persisted = nil
	req := updReq("edge-missing", func(d *pb.SubscriptionGroupProductPlanStaff) { d.Role = "secondary" })

	_, err := f.updateUC().Execute(updCtx(), req)
	if err == nil {
		t.Fatalf("expected a not-found rejection for an unresolvable edge id")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected a class-edge-not-found error, got %q", err.Error())
	}
	if f.edgeRepo.readCalls != 1 {
		t.Errorf("expected exactly 1 read attempt, got %d", f.edgeRepo.readCalls)
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}

// Workspace context is the tenancy proof for the merged edge; without it the
// eligibility guard must fail closed even though every FK resolves.
func TestUpdateEdge_MissingWorkspaceContext_Rejects(t *testing.T) {
	f := newUpdFixture()
	req := updReq("edge-1", func(d *pb.SubscriptionGroupProductPlanStaff) { d.Role = "secondary" })

	_, err := f.updateUC().Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected fail-closed rejection with empty workspace context")
	}
	if f.edgeRepo.updateCalls != 0 {
		t.Errorf("update must not run on rejection, got %d calls", f.edgeRepo.updateCalls)
	}
}
