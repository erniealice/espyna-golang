package subscription_group_product_plan_staff

// v2 class-edge validation matrix (docs/plan/20260724-section-assignment-merged
// espyna.md §2): foreign pps, cross-plan offering, uncovered phase, phase
// variant mismatch, duplicate class-edge triple, and dual-write parity. Fully
// self-contained mocks (in-package, no build tags) — deliberately independent
// of create_subscription_group_product_plan_staff_test.go's mocks so this file
// never needs to touch that one.

import (
	"context"
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

// ----- v2-local mocks --------------------------------------------------------

type v2ClassRepo struct {
	sgpppb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	byID map[string]*sgpppb.SubscriptionGroupProductPlan
}

func (m *v2ClassRepo) ReadSubscriptionGroupProductPlan(_ context.Context, req *sgpppb.ReadSubscriptionGroupProductPlanRequest) (*sgpppb.ReadSubscriptionGroupProductPlanResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &sgpppb.ReadSubscriptionGroupProductPlanResponse{Success: true}, nil
	}
	return &sgpppb.ReadSubscriptionGroupProductPlanResponse{Success: true, Data: []*sgpppb.SubscriptionGroupProductPlan{row}}, nil
}

type v2PlanRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	byID map[string]*productplanpb.ProductPlan
}

func (m *v2PlanRepo) ReadProductPlan(_ context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &productplanpb.ReadProductPlanResponse{Success: true}, nil
	}
	return &productplanpb.ReadProductPlanResponse{Success: true, Data: []*productplanpb.ProductPlan{row}}, nil
}

type v2EligibilityRepo struct {
	productplanstaffpb.UnimplementedProductPlanStaffDomainServiceServer
	byID map[string]*productplanstaffpb.ProductPlanStaff
}

func (m *v2EligibilityRepo) ReadProductPlanStaff(_ context.Context, req *productplanstaffpb.ReadProductPlanStaffRequest) (*productplanstaffpb.ReadProductPlanStaffResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &productplanstaffpb.ReadProductPlanStaffResponse{Success: true}, nil
	}
	return &productplanstaffpb.ReadProductPlanStaffResponse{Success: true, Data: []*productplanstaffpb.ProductPlanStaff{row}}, nil
}

func (m *v2EligibilityRepo) ListProductPlanStaffs(_ context.Context, _ *productplanstaffpb.ListProductPlanStaffsRequest) (*productplanstaffpb.ListProductPlanStaffsResponse, error) {
	var rows []*productplanstaffpb.ProductPlanStaff
	for _, r := range m.byID {
		rows = append(rows, r)
	}
	return &productplanstaffpb.ListProductPlanStaffsResponse{Success: true, Data: rows}, nil
}

type v2GroupRepo struct {
	subscriptiongrouppb.UnimplementedSubscriptionGroupDomainServiceServer
	group *subscriptiongrouppb.SubscriptionGroup
}

func (m *v2GroupRepo) ReadSubscriptionGroup(_ context.Context, _ *subscriptiongrouppb.ReadSubscriptionGroupRequest) (*subscriptiongrouppb.ReadSubscriptionGroupResponse, error) {
	if m.group == nil {
		return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true}, nil
	}
	return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true, Data: []*subscriptiongrouppb.SubscriptionGroup{m.group}}, nil
}

type v2PhaseRepo struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	byID map[string]*jobtemplatephasepb.JobTemplatePhase
}

func (m *v2PhaseRepo) ReadJobTemplatePhase(_ context.Context, req *jobtemplatephasepb.ReadJobTemplatePhaseRequest) (*jobtemplatephasepb.ReadJobTemplatePhaseResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true}, nil
	}
	return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true, Data: []*jobtemplatephasepb.JobTemplatePhase{row}}, nil
}

type v2EdgeRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	createCalls int
	existing    []*pb.SubscriptionGroupProductPlanStaff
}

func (m *v2EdgeRepo) CreateSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	m.createCalls++
	return &pb.CreateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{req.GetData()}}, nil
}

func (m *v2EdgeRepo) ListSubscriptionGroupProductPlanStaffs(_ context.Context, _ *pb.ListSubscriptionGroupProductPlanStaffsRequest) (*pb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	return &pb.ListSubscriptionGroupProductPlanStaffsResponse{Success: true, Data: m.existing}, nil
}

type stubIDV2 struct{ id string }

func (s stubIDV2) GenerateID() string                   { return s.id }
func (s stubIDV2) GenerateIDWithPrefix(p string) string { return p + "-" + s.id }
func (s stubIDV2) IsEnabled() bool                      { return true }
func (s stubIDV2) GetProviderInfo() string              { return "stub" }

// ----- fixtures --------------------------------------------------------------

func v2class(id, sgID, ppID, templateID string) *sgpppb.SubscriptionGroupProductPlan {
	return &sgpppb.SubscriptionGroupProductPlan{
		Id: id, SubscriptionGroupId: sgID, ProductPlanId: ppID, JobTemplateId: templateID,
		Active: true, Status: sgpppb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_ACTIVE,
	}
}

func v2plan(id, planID string) *productplanpb.ProductPlan {
	return &productplanpb.ProductPlan{Id: id, PlanId: planID}
}

func v2planWithVariant(id, planID, variantID string) *productplanpb.ProductPlan {
	return &productplanpb.ProductPlan{Id: id, PlanId: planID, ProductVariantId: &variantID}
}

func v2eligibility(id, productPlanID, staffID string) *productplanstaffpb.ProductPlanStaff {
	return &productplanstaffpb.ProductPlanStaff{Id: id, ProductPlanId: productPlanID, StaffId: staffID, Active: true, WorkspaceId: "ws-1"}
}

func v2group(planID string) *subscriptiongrouppb.SubscriptionGroup {
	ws := "ws-1"
	return &subscriptiongrouppb.SubscriptionGroup{Id: "sg-1", PlanId: &planID, WorkspaceId: &ws, Active: true}
}

func v2phase(id, templateID string) *jobtemplatephasepb.JobTemplatePhase {
	return &jobtemplatephasepb.JobTemplatePhase{Id: id, JobTemplateId: templateID, Active: true}
}

func v2phaseWithVariant(id, templateID, variantID string) *jobtemplatephasepb.JobTemplatePhase {
	return &jobtemplatephasepb.JobTemplatePhase{Id: id, JobTemplateId: templateID, Active: true, OutputProductVariantId: &variantID}
}

type v2fixture struct {
	classRepo *v2ClassRepo
	planRepo  *v2PlanRepo
	ppsRepo   *v2EligibilityRepo
	groupRepo *v2GroupRepo
	phaseRepo *v2PhaseRepo
	edgeRepo  *v2EdgeRepo
}

func newV2Fixture() *v2fixture {
	return &v2fixture{
		classRepo: &v2ClassRepo{byID: map[string]*sgpppb.SubscriptionGroupProductPlan{}},
		planRepo:  &v2PlanRepo{byID: map[string]*productplanpb.ProductPlan{}},
		ppsRepo:   &v2EligibilityRepo{byID: map[string]*productplanstaffpb.ProductPlanStaff{}},
		groupRepo: &v2GroupRepo{},
		phaseRepo: &v2PhaseRepo{byID: map[string]*jobtemplatephasepb.JobTemplatePhase{}},
		edgeRepo:  &v2EdgeRepo{},
	}
}

func (f *v2fixture) createUC() *CreateSubscriptionGroupProductPlanStaffUseCase {
	return NewCreateSubscriptionGroupProductPlanStaffUseCase(
		CreateSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: f.edgeRepo,
			ProductPlanStaff:                  f.ppsRepo,
			ProductPlan:                       f.planRepo,
			SubscriptionGroup:                 f.groupRepo,
			SubscriptionGroupProductPlan:      f.classRepo,
			JobTemplatePhase:                  f.phaseRepo,
		},
		CreateSubscriptionGroupProductPlanStaffServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDV2{id: "sgpps-v2-new"},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func v2req(classID, ppsID, phaseID string) *pb.CreateSubscriptionGroupProductPlanStaffRequest {
	data := &pb.SubscriptionGroupProductPlanStaff{
		SubscriptionGroupProductPlanId: &classID,
		ProductPlanStaffId:             &ppsID,
	}
	if phaseID != "" {
		data.JobTemplatePhaseId = &phaseID
	}
	return &pb.CreateSubscriptionGroupProductPlanStaffRequest{Data: data}
}

// ----- tests ------------------------------------------------------------------

func TestCreateEdgeV2_Success_DualWritesLegacyFields(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-1", "tmpl-1")
	f.planRepo.byID["pp-1"] = v2plan("pp-1", "plan-1")
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-1", "staff-1")
	f.groupRepo.group = v2group("plan-1")

	uc := f.createUC()
	req := v2req("class-1", "pps-1", "")
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	if _, err := uc.Execute(ctx, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.edgeRepo.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", f.edgeRepo.createCalls)
	}
	// Dual-write (espyna.md §2): legacy f8/f9/f10 resolved from class+pps rows.
	if req.Data.GetSubscriptionGroupId() != "sg-1" {
		t.Errorf("f8 subscription_group_id = %q, want sg-1", req.Data.GetSubscriptionGroupId())
	}
	if req.Data.GetProductPlanId() != "pp-1" {
		t.Errorf("f9 product_plan_id = %q, want pp-1", req.Data.GetProductPlanId())
	}
	if req.Data.GetStaffId() != "staff-1" {
		t.Errorf("f10 staff_id = %q, want staff-1", req.Data.GetStaffId())
	}
}

func TestCreateEdgeV2_ForeignPPS_Rejects(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-1", "tmpl-1")
	// pps-missing is never registered in f.ppsRepo.byID.

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, v2req("class-1", "pps-missing", ""))
	if err == nil {
		t.Fatalf("expected rejection for a foreign/unresolvable product_plan_staff id")
	}
	if !strings.Contains(err.Error(), "eligibility") {
		t.Errorf("expected an eligibility-not-found error, got %q", err.Error())
	}
	if f.edgeRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.edgeRepo.createCalls)
	}
}

func TestCreateEdgeV2_CrossPlanOffering_Rejects(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-1", "tmpl-1") // class's offering = pp-1
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-OTHER", "staff-1")    // eligibility's offering = pp-OTHER

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, v2req("class-1", "pps-1", ""))
	if err == nil {
		t.Fatalf("expected rejection: pps.product_plan_id != class.product_plan_id (plan.md §1.2 #1)")
	}
	if !strings.Contains(err.Error(), "offering") {
		t.Errorf("expected an offering-mismatch error, got %q", err.Error())
	}
	if f.edgeRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.edgeRepo.createCalls)
	}
}

func TestCreateEdgeV2_PhaseNotInClassTemplate_Rejects(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-1", "tmpl-1")
	f.planRepo.byID["pp-1"] = v2plan("pp-1", "plan-1")
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-1", "staff-1")
	f.groupRepo.group = v2group("plan-1")
	f.phaseRepo.byID["phase-foreign"] = v2phase("phase-foreign", "tmpl-OTHER") // wrong template

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, v2req("class-1", "pps-1", "phase-foreign"))
	if err == nil {
		t.Fatalf("expected rejection: phase does not belong to the class's own job_template_id")
	}
	if !strings.Contains(err.Error(), "curriculum") {
		t.Errorf("expected a phase-not-in-template error, got %q", err.Error())
	}
	if f.edgeRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.edgeRepo.createCalls)
	}
}

// TestCreateEdgeV2_PhaseVariantMismatch_Rejects is the "Music class + Visual-
// Arts phase" example from plan.md §2.5's write-side validation: the phase
// belongs to the RIGHT template but its output_product_variant_id doesn't
// match the class's own strand variant.
func TestCreateEdgeV2_PhaseVariantMismatch_Rejects(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-music", "tmpl-1")
	f.planRepo.byID["pp-music"] = v2planWithVariant("pp-music", "plan-1", "variant-music")
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-music", "staff-1")
	f.groupRepo.group = v2group("plan-1")
	f.phaseRepo.byID["phase-va"] = v2phaseWithVariant("phase-va", "tmpl-1", "variant-visual-arts")

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, v2req("class-1", "pps-1", "phase-va"))
	if err == nil {
		t.Fatalf("expected rejection: phase variant does not match the class's strand variant")
	}
	if !strings.Contains(err.Error(), "strand") {
		t.Errorf("expected a strand/variant-mismatch error, got %q", err.Error())
	}
	if f.edgeRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.edgeRepo.createCalls)
	}
}

// TestCreateEdgeV2_PhaseVariantMatch_Succeeds is the positive twin: same
// strand class, but the phase's variant matches.
func TestCreateEdgeV2_PhaseVariantMatch_Succeeds(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-music", "tmpl-1")
	f.planRepo.byID["pp-music"] = v2planWithVariant("pp-music", "plan-1", "variant-music")
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-music", "staff-1")
	f.groupRepo.group = v2group("plan-1")
	f.phaseRepo.byID["phase-music"] = v2phaseWithVariant("phase-music", "tmpl-1", "variant-music")

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	if _, err := uc.Execute(ctx, v2req("class-1", "pps-1", "phase-music")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.edgeRepo.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", f.edgeRepo.createCalls)
	}
}

func TestCreateEdgeV2_DuplicateClassEdge_Rejects(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-1", "tmpl-1")
	f.planRepo.byID["pp-1"] = v2plan("pp-1", "plan-1")
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-1", "staff-1")
	f.groupRepo.group = v2group("plan-1")
	classID, ppsID := "class-1", "pps-1"
	f.edgeRepo.existing = []*pb.SubscriptionGroupProductPlanStaff{
		{Id: "existing-1", Active: true, SubscriptionGroupProductPlanId: &classID, ProductPlanStaffId: &ppsID},
	}

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, v2req("class-1", "pps-1", ""))
	if err == nil {
		t.Fatalf("expected duplicate-class-edge rejection (uq_sgpps_class_pps_phase, esqyma.md §2)")
	}
	if !strings.Contains(err.Error(), "already assigned") {
		t.Errorf("expected a friendly duplicate error, got %q", err.Error())
	}
	if f.edgeRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.edgeRepo.createCalls)
	}
}

// TestCreateEdgeV2_DistinctPhase_NotADuplicate confirms two rows sharing
// (class, pps) with DIFFERENT phases coexist (the multi-row-per-class-edge
// shape the phase drawer relies on).
func TestCreateEdgeV2_DistinctPhase_NotADuplicate(t *testing.T) {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-1", "tmpl-1")
	f.planRepo.byID["pp-1"] = v2plan("pp-1", "plan-1")
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-1", "staff-1")
	f.groupRepo.group = v2group("plan-1")
	f.phaseRepo.byID["phase-sem1"] = v2phase("phase-sem1", "tmpl-1")
	classID, ppsID, otherPhase := "class-1", "pps-1", "phase-sem2"
	f.edgeRepo.existing = []*pb.SubscriptionGroupProductPlanStaff{
		{Id: "existing-1", Active: true, SubscriptionGroupProductPlanId: &classID, ProductPlanStaffId: &ppsID, JobTemplatePhaseId: &otherPhase},
	}

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	if _, err := uc.Execute(ctx, v2req("class-1", "pps-1", "phase-sem1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.edgeRepo.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", f.edgeRepo.createCalls)
	}
}

func TestCreateEdgeV2_LegacyOnlyWrite_SkipsV2Resolution(t *testing.T) {
	// No f12/f13 anchors set — a pure legacy-shaped write (the pre-v2 Assign
	// flow) must not trigger v2 resolution and must behave exactly as before.
	f := newV2Fixture()
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-1", "staff-1")
	f.planRepo.byID["pp-1"] = v2plan("pp-1", "plan-1")
	f.groupRepo.group = v2group("plan-1")

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	req := &pb.CreateSubscriptionGroupProductPlanStaffRequest{
		Data: &pb.SubscriptionGroupProductPlanStaff{SubscriptionGroupId: "sg-1", ProductPlanId: "pp-1", StaffId: "staff-1"},
	}
	if _, err := uc.Execute(ctx, req); err != nil {
		t.Fatalf("unexpected error on a legacy-only write: %v", err)
	}
	if f.edgeRepo.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", f.edgeRepo.createCalls)
	}
}
