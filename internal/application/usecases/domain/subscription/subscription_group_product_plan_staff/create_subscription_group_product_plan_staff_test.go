package subscription_group_product_plan_staff

// Eligibility fail-loud tests for the class-edge write path (red-team HIGH #5).
// In-package mocks keep the tests build-tag-free.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// ----- mocks ---------------------------------------------------------------

type mockSGPPSRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	createCalls int
}

func (m *mockSGPPSRepo) CreateSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	m.createCalls++
	return &pb.CreateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{req.GetData()}}, nil
}

type mockPPSRepo struct {
	productplanstaffpb.UnimplementedProductPlanStaffDomainServiceServer
	rows []*productplanstaffpb.ProductPlanStaff
}

func (m *mockPPSRepo) ListProductPlanStaffs(_ context.Context, _ *productplanstaffpb.ListProductPlanStaffsRequest) (*productplanstaffpb.ListProductPlanStaffsResponse, error) {
	return &productplanstaffpb.ListProductPlanStaffsResponse{Success: true, Data: m.rows}, nil
}

type mockPPRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	planByID map[string]string
}

func (m *mockPPRepo) ReadProductPlan(_ context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	id := req.GetData().GetId()
	plan, ok := m.planByID[id]
	if !ok {
		return &productplanpb.ReadProductPlanResponse{Success: true}, nil
	}
	return &productplanpb.ReadProductPlanResponse{Success: true, Data: []*productplanpb.ProductPlan{{Id: id, PlanId: plan}}}, nil
}

type mockSGRepo struct {
	subscriptiongrouppb.UnimplementedSubscriptionGroupDomainServiceServer
	group *subscriptiongrouppb.SubscriptionGroup
}

func (m *mockSGRepo) ReadSubscriptionGroup(_ context.Context, _ *subscriptiongrouppb.ReadSubscriptionGroupRequest) (*subscriptiongrouppb.ReadSubscriptionGroupResponse, error) {
	if m.group == nil {
		return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true}, nil
	}
	return &subscriptiongrouppb.ReadSubscriptionGroupResponse{Success: true, Data: []*subscriptiongrouppb.SubscriptionGroup{m.group}}, nil
}

// ----- helpers -------------------------------------------------------------

type stubIDSGPPS struct{}

func (stubIDSGPPS) GenerateID() string                   { return "sgpps-new" }
func (stubIDSGPPS) GenerateIDWithPrefix(p string) string { return p + "-new" }
func (stubIDSGPPS) IsEnabled() bool                      { return true }
func (stubIDSGPPS) GetProviderInfo() string              { return "stub" }

func eligibleRow(ws string) *productplanstaffpb.ProductPlanStaff {
	return &productplanstaffpb.ProductPlanStaff{Id: "pps-1", ProductPlanId: "pp-1", StaffId: "staff-1", WorkspaceId: ws, Active: true}
}

func group(planID, ws string) *subscriptiongrouppb.SubscriptionGroup {
	return &subscriptiongrouppb.SubscriptionGroup{Id: "sg-1", PlanId: &planID, WorkspaceId: &ws, Active: true}
}

func newCreateSGPPSUC(sgppsRepo *mockSGPPSRepo, pps *mockPPSRepo, pp *mockPPRepo, sg *mockSGRepo) *CreateSubscriptionGroupProductPlanStaffUseCase {
	return NewCreateSubscriptionGroupProductPlanStaffUseCase(
		CreateSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: sgppsRepo,
			ProductPlanStaff:                  pps,
			ProductPlan:                       pp,
			SubscriptionGroup:                 sg,
		},
		CreateSubscriptionGroupProductPlanStaffServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDSGPPS{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func edgeReq() *pb.CreateSubscriptionGroupProductPlanStaffRequest {
	return &pb.CreateSubscriptionGroupProductPlanStaffRequest{
		Data: &pb.SubscriptionGroupProductPlanStaff{SubscriptionGroupId: "sg-1", ProductPlanId: "pp-1", StaffId: "staff-1"},
	}
}

// ----- tests ---------------------------------------------------------------

func TestCreateEdge_Eligible_Success(t *testing.T) {
	sgppsRepo := &mockSGPPSRepo{}
	uc := newCreateSGPPSUC(
		sgppsRepo,
		&mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{eligibleRow("ws-1")}},
		&mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}},
		&mockSGRepo{group: group("plan-1", "ws-1")},
	)
	if _, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), edgeReq()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sgppsRepo.createCalls != 1 {
		t.Fatalf("expected create once, got %d", sgppsRepo.createCalls)
	}
}

func TestCreateEdge_MissingEligibility_Rejects(t *testing.T) {
	sgppsRepo := &mockSGPPSRepo{}
	uc := newCreateSGPPSUC(
		sgppsRepo,
		&mockPPSRepo{rows: nil}, // no eligibility row
		&mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}},
		&mockSGRepo{group: group("plan-1", "ws-1")},
	)
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), edgeReq())
	if err == nil {
		t.Fatalf("expected missing-eligibility rejection")
	}
	if !strings.Contains(err.Error(), "eligible") {
		t.Errorf("expected eligibility error, got %q", err.Error())
	}
	if sgppsRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", sgppsRepo.createCalls)
	}
}

func TestCreateEdge_PlanMismatch_Rejects(t *testing.T) {
	sgppsRepo := &mockSGPPSRepo{}
	uc := newCreateSGPPSUC(
		sgppsRepo,
		&mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{eligibleRow("ws-1")}},
		&mockPPRepo{planByID: map[string]string{"pp-1": "plan-2"}}, // product_plan.plan_id = plan-2
		&mockSGRepo{group: group("plan-1", "ws-1")},                // group.plan_id = plan-1
	)
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), edgeReq())
	if err == nil {
		t.Fatalf("expected plan-mismatch rejection")
	}
	if !strings.Contains(err.Error(), "plan") {
		t.Errorf("expected plan-mismatch error, got %q", err.Error())
	}
	if sgppsRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", sgppsRepo.createCalls)
	}
}

func TestCreateEdge_CrossWorkspaceGroup_Rejects(t *testing.T) {
	sgppsRepo := &mockSGPPSRepo{}
	uc := newCreateSGPPSUC(
		sgppsRepo,
		&mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{eligibleRow("ws-1")}},
		&mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}},
		&mockSGRepo{group: group("plan-1", "ws-2")}, // group belongs to another workspace
	)
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), edgeReq())
	if err == nil {
		t.Fatalf("expected cross-workspace rejection")
	}
	if sgppsRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", sgppsRepo.createCalls)
	}
}

func TestCreateEdge_MissingWorkspaceContext_Rejects(t *testing.T) {
	sgppsRepo := &mockSGPPSRepo{}
	uc := newCreateSGPPSUC(
		sgppsRepo,
		&mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{eligibleRow("")}},
		&mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}},
		&mockSGRepo{group: group("plan-1", "")},
	)
	_, err := uc.Execute(context.Background(), edgeReq())
	if err == nil {
		t.Fatalf("expected fail-closed rejection with empty workspace context")
	}
	if sgppsRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", sgppsRepo.createCalls)
	}
}
