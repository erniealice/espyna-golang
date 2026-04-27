package price_plan

// Tests for the §3.2 server-side coercion that cascades a parent Plan's
// client_id onto a freshly-created PricePlan. Body-supplied client_id is
// always overwritten — the denormalized invariant
// `price_plan.client_id == plan.client_id` must hold on every write.

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// ----- mocks ---------------------------------------------------------------

type mockPlanRepoForCreate struct {
	planpb.UnimplementedPlanDomainServiceServer
	plans map[string]*planpb.Plan
}

func (m *mockPlanRepoForCreate) ReadPlan(_ context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	id := ""
	if req.GetData() != nil && req.GetData().Id != nil {
		id = *req.GetData().Id
	}
	if p, ok := m.plans[id]; ok {
		return &planpb.ReadPlanResponse{Data: []*planpb.Plan{p}, Success: true}, nil
	}
	return &planpb.ReadPlanResponse{Data: nil, Success: true}, nil
}

type mockPricePlanRepoCreate struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	captured *priceplanpb.PricePlan
}

func (m *mockPricePlanRepoCreate) CreatePricePlan(_ context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	m.captured = req.GetData()
	return &priceplanpb.CreatePricePlanResponse{Data: []*priceplanpb.PricePlan{req.GetData()}, Success: true}, nil
}

type stubIDSvc struct{ count int }

func (s *stubIDSvc) GenerateID() string {
	s.count++
	return "id-" + itoaSimple(s.count)
}
func (s *stubIDSvc) GenerateIDWithPrefix(p string) string { s.count++; return p + "-" + itoaSimple(s.count) }
func (s *stubIDSvc) IsEnabled() bool                      { return true }
func (s *stubIDSvc) GetProviderInfo() string              { return "stub" }

func itoaSimple(i int) string {
	if i == 0 {
		return "0"
	}
	d := []byte{}
	for i > 0 {
		d = append([]byte{byte('0' + i%10)}, d...)
		i /= 10
	}
	return string(d)
}

type noTxnCreate struct{}

func (noTxnCreate) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (noTxnCreate) SupportsTransactions() bool          { return false }
func (noTxnCreate) IsTransactionActive(context.Context) bool { return false }

// ----- fixture --------------------------------------------------------------

func newCreateFixture(t *testing.T) (*CreatePricePlanUseCase, *mockPlanRepoForCreate, *mockPricePlanRepoCreate) {
	t.Helper()
	planRepo := &mockPlanRepoForCreate{plans: map[string]*planpb.Plan{}}
	ppRepo := &mockPricePlanRepoCreate{}
	uc := NewCreatePricePlanUseCase(
		CreatePricePlanRepositories{PricePlan: ppRepo, Plan: planRepo},
		CreatePricePlanServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TransactionService:   noTxnCreate{},
			TranslationService:   ports.NewNoOpTranslationService(),
			IDService:            &stubIDSvc{},
		},
	)
	return uc, planRepo, ppRepo
}

func seedPlanForCreate(id, clientID string) *planpb.Plan {
	idCopy := id
	p := &planpb.Plan{Id: &idCopy, Name: "Audit Engagement", Active: true}
	if clientID != "" {
		c := clientID
		p.ClientId = &c
	}
	return p
}

func basePricePlanRequest(planID string) *priceplanpb.CreatePricePlanRequest {
	return &priceplanpb.CreatePricePlanRequest{
		Data: &priceplanpb.PricePlan{
			PlanId:          planID,
			BillingAmount:   100000,
			BillingCurrency: "PHP",
		},
	}
}

// ----- tests ---------------------------------------------------------------

func TestCreatePricePlan_ClientScopedParent_ChildInheritsClientID(t *testing.T) {
	uc, planRepo, ppRepo := newCreateFixture(t)
	planRepo.plans["plan-1"] = seedPlanForCreate("plan-1", "client-cruz")

	if _, err := uc.Execute(context.Background(), basePricePlanRequest("plan-1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ppRepo.captured == nil {
		t.Fatalf("CreatePricePlan was not invoked")
	}
	if got := ppRepo.captured.GetClientId(); got != "client-cruz" {
		t.Errorf("expected child client_id=client-cruz, got %q", got)
	}
}

func TestCreatePricePlan_MasterParent_ChildIsMaster(t *testing.T) {
	uc, planRepo, ppRepo := newCreateFixture(t)
	planRepo.plans["plan-1"] = seedPlanForCreate("plan-1", "")

	if _, err := uc.Execute(context.Background(), basePricePlanRequest("plan-1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ppRepo.captured.GetClientId(); got != "" {
		t.Errorf("expected child client_id=NULL/empty, got %q", got)
	}
}

func TestCreatePricePlan_BodyClientID_Overwritten(t *testing.T) {
	uc, planRepo, ppRepo := newCreateFixture(t)
	planRepo.plans["plan-1"] = seedPlanForCreate("plan-1", "client-cruz")

	body := basePricePlanRequest("plan-1")
	other := "client-other"
	body.Data.ClientId = &other // operator tries to spoof a different client

	if _, err := uc.Execute(context.Background(), body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ppRepo.captured.GetClientId(); got != "client-cruz" {
		t.Errorf("server-coerce failed: got client_id=%q want client-cruz", got)
	}
}
