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

// ----- resolveClassEdgeV2 reject / fail-closed branch matrix -----------------
//
// W-G4 (coverage-audit-20260725.md §3): five of resolveClassEdgeV2's eight
// reject branches had no coverage at all — class-not-found, class-not-active,
// eligibility-not-active, phase-not-found, and both nil-repo fail-closed
// guards; the "exactly one of f12/f13" anchor guard was likewise untested
// (table 2.2). These cases drive the function DIRECTLY rather than through the
// use case so each assertion pins the SPECIFIC rejection reason: a fail-closed
// path that fails for the wrong reason is still a bug, and several of these
// messages ("eligibility row not found" vs "eligibility row is not active")
// are only one word apart. The use-case-level companions below re-prove the
// same branches also stop the write (createCalls == 0).

func v2data(classID, ppsID, phaseID string) *pb.SubscriptionGroupProductPlanStaff {
	data := &pb.SubscriptionGroupProductPlanStaff{}
	if classID != "" {
		data.SubscriptionGroupProductPlanId = &classID
	}
	if ppsID != "" {
		data.ProductPlanStaffId = &ppsID
	}
	if phaseID != "" {
		data.JobTemplatePhaseId = &phaseID
	}
	return data
}

// repos wires the fixture's mocks into the v2Repos bundle resolveClassEdgeV2
// actually consumes (the use case builds the same bundle from its own repo
// struct).
func (f *v2fixture) repos() v2Repos {
	return v2Repos{
		SubscriptionGroupProductPlan: f.classRepo,
		ProductPlanStaff:             f.ppsRepo,
		ProductPlan:                  f.planRepo,
		JobTemplatePhase:             f.phaseRepo,
	}
}

// v2wellFormed is the fully-valid graph every reject case below perturbs by
// exactly one fact, so a failure names the perturbation and nothing else.
func v2wellFormed() *v2fixture {
	f := newV2Fixture()
	f.classRepo.byID["class-1"] = v2class("class-1", "sg-1", "pp-1", "tmpl-1")
	f.planRepo.byID["pp-1"] = v2plan("pp-1", "plan-1")
	f.ppsRepo.byID["pps-1"] = v2eligibility("pps-1", "pp-1", "staff-1")
	f.groupRepo.group = v2group("plan-1")
	f.phaseRepo.byID["phase-1"] = v2phase("phase-1", "tmpl-1")
	return f
}

func TestResolveClassEdgeV2_RejectBranches(t *testing.T) {
	tr := ports.NewNoOpTranslator()

	tests := []struct {
		name string
		// setup perturbs the well-formed fixture and returns the repo bundle,
		// so a case can also blank out a repo to hit a fail-closed guard.
		setup   func(f *v2fixture) v2Repos
		data    *pb.SubscriptionGroupProductPlanStaff
		wantErr string
	}{
		{
			name:    "class anchor set without eligibility anchor",
			setup:   func(f *v2fixture) v2Repos { return f.repos() },
			data:    v2data("class-1", "", ""),
			wantErr: "must be set together",
		},
		{
			name:    "eligibility anchor set without class anchor",
			setup:   func(f *v2fixture) v2Repos { return f.repos() },
			data:    v2data("", "pps-1", ""),
			wantErr: "must be set together",
		},
		{
			name: "nil class repo fails closed",
			setup: func(f *v2fixture) v2Repos {
				r := f.repos()
				r.SubscriptionGroupProductPlan = nil
				return r
			},
			data:    v2data("class-1", "pps-1", ""),
			wantErr: "class-edge validation is not configured",
		},
		{
			name: "nil eligibility repo fails closed",
			setup: func(f *v2fixture) v2Repos {
				r := f.repos()
				r.ProductPlanStaff = nil
				return r
			},
			data:    v2data("class-1", "pps-1", ""),
			wantErr: "class-edge validation is not configured",
		},
		{
			name:    "class id unresolvable",
			setup:   func(f *v2fixture) v2Repos { return f.repos() },
			data:    v2data("class-MISSING", "pps-1", ""),
			wantErr: "class not found",
		},
		{
			name: "class row is inactive",
			setup: func(f *v2fixture) v2Repos {
				f.classRepo.byID["class-1"].Active = false
				return f.repos()
			},
			data:    v2data("class-1", "pps-1", ""),
			wantErr: "class is not active",
		},
		{
			name:    "eligibility id unresolvable",
			setup:   func(f *v2fixture) v2Repos { return f.repos() },
			data:    v2data("class-1", "pps-MISSING", ""),
			wantErr: "eligibility row not found",
		},
		{
			name: "eligibility row is inactive",
			setup: func(f *v2fixture) v2Repos {
				f.ppsRepo.byID["pps-1"].Active = false
				return f.repos()
			},
			data:    v2data("class-1", "pps-1", ""),
			wantErr: "eligibility row is not active",
		},
		{
			name: "eligibility offering differs from the class offering",
			setup: func(f *v2fixture) v2Repos {
				f.ppsRepo.byID["pps-1"].ProductPlanId = "pp-OTHER"
				return f.repos()
			},
			data:    v2data("class-1", "pps-1", ""),
			wantErr: "does not match the class's offering",
		},
		{
			// plan.md §1.2 #1 compares two ids; an empty id on either side must
			// reject rather than compare-equal into an accept.
			name: "eligibility offering is empty",
			setup: func(f *v2fixture) v2Repos {
				f.ppsRepo.byID["pps-1"].ProductPlanId = ""
				return f.repos()
			},
			data:    v2data("class-1", "pps-1", ""),
			wantErr: "does not match the class's offering",
		},
		{
			name: "class offering is empty",
			setup: func(f *v2fixture) v2Repos {
				f.classRepo.byID["class-1"].ProductPlanId = ""
				f.ppsRepo.byID["pps-1"].ProductPlanId = ""
				return f.repos()
			},
			data:    v2data("class-1", "pps-1", ""),
			wantErr: "does not match the class's offering",
		},
		{
			name: "phase set but phase repo is nil fails closed",
			setup: func(f *v2fixture) v2Repos {
				r := f.repos()
				r.JobTemplatePhase = nil
				return r
			},
			data:    v2data("class-1", "pps-1", "phase-1"),
			wantErr: "phase validation is not configured",
		},
		{
			name:    "phase id unresolvable",
			setup:   func(f *v2fixture) v2Repos { return f.repos() },
			data:    v2data("class-1", "pps-1", "phase-MISSING"),
			wantErr: "phase not found",
		},
		{
			name: "phase belongs to a foreign template",
			setup: func(f *v2fixture) v2Repos {
				f.phaseRepo.byID["phase-foreign"] = v2phase("phase-foreign", "tmpl-OTHER")
				return f.repos()
			},
			data:    v2data("class-1", "pps-1", "phase-foreign"),
			wantErr: "does not belong to the class's curriculum",
		},
		{
			name: "phase strand variant differs from the class variant",
			setup: func(f *v2fixture) v2Repos {
				f.planRepo.byID["pp-1"] = v2planWithVariant("pp-1", "plan-1", "variant-music")
				f.phaseRepo.byID["phase-va"] = v2phaseWithVariant("phase-va", "tmpl-1", "variant-visual-arts")
				return f.repos()
			},
			data:    v2data("class-1", "pps-1", "phase-va"),
			wantErr: "not compatible with the class's strand",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := v2wellFormed()
			repos := tc.setup(f)

			err := resolveClassEdgeV2(context.Background(), repos, tr, tc.data)
			if err == nil {
				t.Fatalf("expected rejection %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("wrong rejection reason:\n got: %q\nwant substring: %q", err.Error(), tc.wantErr)
			}
			// A rejected edge must leave the legacy dual-write fields untouched
			// — a half-resolved f8/f9/f10 on a rejected payload is exactly the
			// parity drift the dual-write exists to prevent.
			if got := tc.data.GetSubscriptionGroupId(); got != "" {
				t.Errorf("rejected edge dual-wrote f8 subscription_group_id = %q, want empty", got)
			}
			if got := tc.data.GetProductPlanId(); got != "" {
				t.Errorf("rejected edge dual-wrote f9 product_plan_id = %q, want empty", got)
			}
			if got := tc.data.GetStaffId(); got != "" {
				t.Errorf("rejected edge dual-wrote f10 staff_id = %q, want empty", got)
			}
		})
	}
}

// TestResolveClassEdgeV2_WellFormed_Accepts is the positive control for the
// matrix above: it proves v2wellFormed() really is well-formed, so every
// rejection there is attributable to that case's single perturbation and not
// to a broken fixture.
func TestResolveClassEdgeV2_WellFormed_Accepts(t *testing.T) {
	f := v2wellFormed()
	data := v2data("class-1", "pps-1", "phase-1")

	if err := resolveClassEdgeV2(context.Background(), f.repos(), ports.NewNoOpTranslator(), data); err != nil {
		t.Fatalf("well-formed class edge rejected: %v", err)
	}
	if got := data.GetSubscriptionGroupId(); got != "sg-1" {
		t.Errorf("f8 subscription_group_id = %q, want sg-1", got)
	}
	if got := data.GetProductPlanId(); got != "pp-1" {
		t.Errorf("f9 product_plan_id = %q, want pp-1", got)
	}
	if got := data.GetStaffId(); got != "staff-1" {
		t.Errorf("f10 staff_id = %q, want staff-1", got)
	}
}

// TestResolveClassEdgeV2_NilProductPlanRepo_WithPhase_FailsOpen documents
// W-G6, an ASYMMETRY, not an endorsement: the phase strand-variant check at
// class_edge_v2.go:112 is skipped entirely when the ProductPlan repo is
// absent, while the three sibling guards in the same function (nil sgpp, nil
// pps, nil phase repo) all fail CLOSED. The fixture here is a genuine
// "Music class + Visual-Arts phase" pair that the matrix case above rejects;
// with only the ProductPlan repo removed it is ACCEPTED.
//
// If the owner decides this should fail closed like its siblings, this test
// must be INVERTED (expect an error), not deleted — it exists to make the
// decision visible rather than to bless the current behaviour.
func TestResolveClassEdgeV2_NilProductPlanRepo_WithPhase_FailsOpen(t *testing.T) {
	f := v2wellFormed()
	f.planRepo.byID["pp-1"] = v2planWithVariant("pp-1", "plan-1", "variant-music")
	f.phaseRepo.byID["phase-va"] = v2phaseWithVariant("phase-va", "tmpl-1", "variant-visual-arts")
	repos := f.repos()
	repos.ProductPlan = nil

	data := v2data("class-1", "pps-1", "phase-va")
	err := resolveClassEdgeV2(context.Background(), repos, ports.NewNoOpTranslator(), data)
	if err != nil {
		t.Fatalf("W-G6 characterization: expected the variant check to be SKIPPED with a nil ProductPlan repo, got %q "+
			"— if the fail-open was deliberately fixed, invert this test", err.Error())
	}
	if got := data.GetStaffId(); got != "staff-1" {
		t.Errorf("fail-open path still must dual-write f10 staff_id, got %q", got)
	}
}

// ----- use-case-level companions: the reject must also stop the write --------

// TestCreateEdgeV2_RejectBranches_DoNotWrite re-drives the previously
// untested reject branches through CreateSubscriptionGroupProductPlanStaffUseCase
// to prove the rejection actually reaches the caller and no row is created.
// (The nil-repo guards are unreachable here — the use case always supplies
// every repo — so they are covered directly in the matrix above.)
func TestCreateEdgeV2_RejectBranches_DoNotWrite(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(f *v2fixture)
		req     *pb.CreateSubscriptionGroupProductPlanStaffRequest
		wantErr string
	}{
		{
			name:    "class anchor without eligibility anchor",
			setup:   func(f *v2fixture) {},
			req:     &pb.CreateSubscriptionGroupProductPlanStaffRequest{Data: v2data("class-1", "", "")},
			wantErr: "must be set together",
		},
		{
			name:    "class id unresolvable",
			setup:   func(f *v2fixture) {},
			req:     &pb.CreateSubscriptionGroupProductPlanStaffRequest{Data: v2data("class-MISSING", "pps-1", "")},
			wantErr: "class not found",
		},
		{
			name:    "class row is inactive",
			setup:   func(f *v2fixture) { f.classRepo.byID["class-1"].Active = false },
			req:     &pb.CreateSubscriptionGroupProductPlanStaffRequest{Data: v2data("class-1", "pps-1", "")},
			wantErr: "class is not active",
		},
		{
			name:    "eligibility row is inactive",
			setup:   func(f *v2fixture) { f.ppsRepo.byID["pps-1"].Active = false },
			req:     &pb.CreateSubscriptionGroupProductPlanStaffRequest{Data: v2data("class-1", "pps-1", "")},
			wantErr: "eligibility row is not active",
		},
		{
			name:    "phase id unresolvable",
			setup:   func(f *v2fixture) {},
			req:     &pb.CreateSubscriptionGroupProductPlanStaffRequest{Data: v2data("class-1", "pps-1", "phase-MISSING")},
			wantErr: "phase not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := v2wellFormed()
			tc.setup(f)

			ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
			_, err := f.createUC().Execute(ctx, tc.req)
			if err == nil {
				t.Fatalf("expected rejection %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("wrong rejection reason:\n got: %q\nwant substring: %q", err.Error(), tc.wantErr)
			}
			if f.edgeRepo.createCalls != 0 {
				t.Errorf("create must not run on rejection, got %d calls", f.edgeRepo.createCalls)
			}
		})
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
