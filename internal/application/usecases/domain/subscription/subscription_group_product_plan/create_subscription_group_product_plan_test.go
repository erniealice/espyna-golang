package subscription_group_product_plan

// Validation matrix for the class-invariant guard (plan.md §1.2 #1-3): plan
// mismatch, template mismatch, cross-workspace, duplicate class, and the
// happy path. In-package mocks, no build tags — mirrors the sgpps eligibility
// test idiom (subscription_group_product_plan_staff/create_..._test.go).

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

// ----- mocks -----------------------------------------------------------------

type mockClassRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	createCalls int
	existing    []*pb.SubscriptionGroupProductPlan
}

func (m *mockClassRepo) CreateSubscriptionGroupProductPlan(_ context.Context, req *pb.CreateSubscriptionGroupProductPlanRequest) (*pb.CreateSubscriptionGroupProductPlanResponse, error) {
	m.createCalls++
	return &pb.CreateSubscriptionGroupProductPlanResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlan{req.GetData()}}, nil
}

func (m *mockClassRepo) ListSubscriptionGroupProductPlans(_ context.Context, _ *pb.ListSubscriptionGroupProductPlansRequest) (*pb.ListSubscriptionGroupProductPlansResponse, error) {
	return &pb.ListSubscriptionGroupProductPlansResponse{Success: true, Data: m.existing}, nil
}

type mockPlanRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	byID map[string]*productplanpb.ProductPlan
}

func (m *mockPlanRepo) ReadProductPlan(_ context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &productplanpb.ReadProductPlanResponse{Success: true}, nil
	}
	return &productplanpb.ReadProductPlanResponse{Success: true, Data: []*productplanpb.ProductPlan{row}}, nil
}

type mockGroupRepo struct {
	subscriptiongrouppb.UnimplementedSubscriptionGroupDomainServiceServer
	group *subscriptiongrouppb.SubscriptionGroup
}

func (m *mockGroupRepo) ReadSubscriptionGroup(_ context.Context, _ *subscriptiongrouppb.ReadSubscriptionGroupRequest) (*subscriptiongrouppb.ReadSubscriptionGroupResponse, error) {
	if m.group == nil {
		return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true}, nil
	}
	return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true, Data: []*subscriptiongrouppb.SubscriptionGroup{m.group}}, nil
}

type mockJobTemplateRepo struct {
	jobtemplatepb.UnimplementedJobTemplateDomainServiceServer
	byID map[string]*jobtemplatepb.JobTemplate
}

func (m *mockJobTemplateRepo) ReadJobTemplate(_ context.Context, req *jobtemplatepb.ReadJobTemplateRequest) (*jobtemplatepb.ReadJobTemplateResponse, error) {
	row, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return &jobtemplatepb.ReadJobTemplateResponse{Success: true}, nil
	}
	return &jobtemplatepb.ReadJobTemplateResponse{Success: true, Data: []*jobtemplatepb.JobTemplate{row}}, nil
}

type stubIDClass struct{}

func (stubIDClass) GenerateID() string                   { return "class-new" }
func (stubIDClass) GenerateIDWithPrefix(p string) string { return p + "-new" }
func (stubIDClass) IsEnabled() bool                      { return true }
func (stubIDClass) GetProviderInfo() string              { return "stub" }

// ----- fixtures ---------------------------------------------------------------

func classGroup(planID, ws string) *subscriptiongrouppb.SubscriptionGroup {
	return &subscriptiongrouppb.SubscriptionGroup{Id: "sg-1", PlanId: &planID, WorkspaceId: &ws, Active: true}
}

func classPlan(id, planID, productID string) *productplanpb.ProductPlan {
	return &productplanpb.ProductPlan{Id: id, PlanId: planID, ProductId: productID}
}

func classTemplate(id, outputProductID string) *jobtemplatepb.JobTemplate {
	return &jobtemplatepb.JobTemplate{Id: id, OutputProductId: &outputProductID}
}

type classFixture struct {
	classRepo    *mockClassRepo
	planRepo     *mockPlanRepo
	groupRepo    *mockGroupRepo
	templateRepo *mockJobTemplateRepo
}

func newClassFixture() *classFixture {
	return &classFixture{
		classRepo:    &mockClassRepo{},
		planRepo:     &mockPlanRepo{byID: map[string]*productplanpb.ProductPlan{}},
		groupRepo:    &mockGroupRepo{},
		templateRepo: &mockJobTemplateRepo{byID: map[string]*jobtemplatepb.JobTemplate{}},
	}
}

func (f *classFixture) createUC() *CreateSubscriptionGroupProductPlanUseCase {
	return NewCreateSubscriptionGroupProductPlanUseCase(
		CreateSubscriptionGroupProductPlanRepositories{
			SubscriptionGroupProductPlan: f.classRepo,
			ProductPlan:                  f.planRepo,
			SubscriptionGroup:            f.groupRepo,
			JobTemplate:                  f.templateRepo,
		},
		CreateSubscriptionGroupProductPlanServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDClass{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func classReq(sgID, ppID, templateID string) *pb.CreateSubscriptionGroupProductPlanRequest {
	return &pb.CreateSubscriptionGroupProductPlanRequest{
		Data: &pb.SubscriptionGroupProductPlan{SubscriptionGroupId: sgID, ProductPlanId: ppID, JobTemplateId: templateID},
	}
}

// ----- tests -------------------------------------------------------------------

func TestCreateClass_Success_DefaultsActiveStatus(t *testing.T) {
	f := newClassFixture()
	f.planRepo.byID["pp-1"] = classPlan("pp-1", "plan-1", "product-1")
	f.groupRepo.group = classGroup("plan-1", "ws-1")
	f.templateRepo.byID["tmpl-1"] = classTemplate("tmpl-1", "product-1")

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	req := classReq("sg-1", "pp-1", "tmpl-1")
	if _, err := uc.Execute(ctx, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.classRepo.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", f.classRepo.createCalls)
	}
	if !req.Data.GetActive() {
		t.Errorf("expected Active=true after enrich")
	}
	if req.Data.GetStatus() != pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_ACTIVE {
		t.Errorf("expected default status ACTIVE, got %v", req.Data.GetStatus())
	}
}

func TestCreateClass_PlanMismatch_Rejects(t *testing.T) {
	f := newClassFixture()
	f.planRepo.byID["pp-1"] = classPlan("pp-1", "plan-2", "product-1") // offering belongs to plan-2
	f.groupRepo.group = classGroup("plan-1", "ws-1")                   // section belongs to plan-1
	f.templateRepo.byID["tmpl-1"] = classTemplate("tmpl-1", "product-1")

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, classReq("sg-1", "pp-1", "tmpl-1"))
	if err == nil {
		t.Fatalf("expected rejection: pp.plan_id != sg.plan_id (plan.md §1.2 #3)")
	}
	if !strings.Contains(err.Error(), "plan") {
		t.Errorf("expected a plan-mismatch error, got %q", err.Error())
	}
	if f.classRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.classRepo.createCalls)
	}
}

func TestCreateClass_TemplateMismatch_Rejects(t *testing.T) {
	f := newClassFixture()
	f.planRepo.byID["pp-1"] = classPlan("pp-1", "plan-1", "product-1")
	f.groupRepo.group = classGroup("plan-1", "ws-1")
	f.templateRepo.byID["tmpl-1"] = classTemplate("tmpl-1", "product-OTHER") // template delivers a different product

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, classReq("sg-1", "pp-1", "tmpl-1"))
	if err == nil {
		t.Fatalf("expected rejection: template.output_product_id != pp.product_id (plan.md §1.2 #2)")
	}
	if !strings.Contains(err.Error(), "template") {
		t.Errorf("expected a template-mismatch error, got %q", err.Error())
	}
	if f.classRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.classRepo.createCalls)
	}
}

func TestCreateClass_CrossWorkspaceGroup_Rejects(t *testing.T) {
	f := newClassFixture()
	f.planRepo.byID["pp-1"] = classPlan("pp-1", "plan-1", "product-1")
	f.groupRepo.group = classGroup("plan-1", "ws-2") // group belongs to another workspace
	f.templateRepo.byID["tmpl-1"] = classTemplate("tmpl-1", "product-1")

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, classReq("sg-1", "pp-1", "tmpl-1"))
	if err == nil {
		t.Fatalf("expected cross-workspace rejection")
	}
	if f.classRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.classRepo.createCalls)
	}
}

func TestCreateClass_DuplicateOffering_Rejects(t *testing.T) {
	f := newClassFixture()
	f.planRepo.byID["pp-1"] = classPlan("pp-1", "plan-1", "product-1")
	f.groupRepo.group = classGroup("plan-1", "ws-1")
	f.templateRepo.byID["tmpl-1"] = classTemplate("tmpl-1", "product-1")
	f.classRepo.existing = []*pb.SubscriptionGroupProductPlan{
		{Id: "existing-1", Active: true, SubscriptionGroupId: "sg-1", ProductPlanId: "pp-1", WorkspaceId: "ws-1"},
	}

	uc := f.createUC()
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	_, err := uc.Execute(ctx, classReq("sg-1", "pp-1", "tmpl-1"))
	if err == nil {
		t.Fatalf("expected duplicate-class rejection (uq_subscription_group_product_plan_1)")
	}
	if !strings.Contains(err.Error(), "already has a class") {
		t.Errorf("expected a friendly duplicate error, got %q", err.Error())
	}
	if f.classRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", f.classRepo.createCalls)
	}
}

// TestCreateClass_NilJobTemplateRepo_SkipsTemplateLegOnly confirms the
// best-effort tolerance documented in validation.go: a nil JobTemplate repo
// (composition gap) skips leg (2) only — legs (1)/(3) still enforce.
func TestCreateClass_NilJobTemplateRepo_SkipsTemplateLegOnly(t *testing.T) {
	f := newClassFixture()
	f.planRepo.byID["pp-1"] = classPlan("pp-1", "plan-1", "product-1")
	f.groupRepo.group = classGroup("plan-1", "ws-1")
	// templateRepo intentionally left with no wiring at the use-case level.
	uc := NewCreateSubscriptionGroupProductPlanUseCase(
		CreateSubscriptionGroupProductPlanRepositories{
			SubscriptionGroupProductPlan: f.classRepo,
			ProductPlan:                  f.planRepo,
			SubscriptionGroup:            f.groupRepo,
			JobTemplate:                  nil,
		},
		CreateSubscriptionGroupProductPlanServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDClass{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
	ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
	if _, err := uc.Execute(ctx, classReq("sg-1", "pp-1", "tmpl-1")); err != nil {
		t.Fatalf("unexpected error with nil JobTemplate repo: %v", err)
	}
	if f.classRepo.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", f.classRepo.createCalls)
	}
}
