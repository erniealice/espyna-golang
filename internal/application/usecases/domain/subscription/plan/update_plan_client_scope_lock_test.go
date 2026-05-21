package plan

// Tests for the §3.1 client_id reassignment guard on UpdatePlan and the §3.2
// cascade to child PricePlans. Uses lightweight in-package mocks so the
// tests run without the mock_db / mock_auth build tags.

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// stubReferenceChecker — controllable per test. lockedIDs flags which plan
// IDs the §3.1 query would mark as in-use by an active subscription.
type stubReferenceChecker struct {
	lockedIDs map[string]bool
	err       error
}

func (s *stubReferenceChecker) GetPlanClientScopeLockedIDs(_ context.Context, ids []string) (map[string]bool, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := map[string]bool{}
	for _, id := range ids {
		if s.lockedIDs[id] {
			out[id] = true
		}
	}
	return out, nil
}

func (s *stubReferenceChecker) GetActiveSubscriptionCountForPricePlan(_ context.Context, _ string) (int, error) {
	return 0, nil
}

// ----- remaining reference.Checker stubs (return empty/nil) ---------------
//
// These exist solely to satisfy the interface; no test in this file exercises them.

func (s *stubReferenceChecker) GetLocationInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetRoleInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetCategoryInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetClientInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetProductInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetProductVariantInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetProductOptionValueInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetProductOptionInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetPlanInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetPriceListInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetPricePlanInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetPriceScheduleInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetAssetCategoryInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetAssetInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetPaymentTermInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetLineInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetLocationAreaInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetEventTagInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetSubscriptionInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetSupplierInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetJobInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetJobActivityInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetJobPhaseInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetJobTaskInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubReferenceChecker) GetJobTemplateInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, nil
}

// pricePlanRepoForCascade — mock that exposes a list of child PricePlans for
// the §3.2 cascade and captures every UpdatePricePlan call.
type pricePlanRepoForCascade struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	rows        []*priceplanpb.PricePlan
	updateCalls []*priceplanpb.PricePlan
}

func (m *pricePlanRepoForCascade) ListPricePlans(_ context.Context, _ *priceplanpb.ListPricePlansRequest) (*priceplanpb.ListPricePlansResponse, error) {
	return &priceplanpb.ListPricePlansResponse{Data: m.rows, Success: true}, nil
}

func (m *pricePlanRepoForCascade) UpdatePricePlan(_ context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	m.updateCalls = append(m.updateCalls, req.GetData())
	return &priceplanpb.UpdatePricePlanResponse{Data: []*priceplanpb.PricePlan{req.GetData()}, Success: true}, nil
}

func newUpdateUC(t *testing.T, planRepo *mockPlanRepo, pricePlanRepo *pricePlanRepoForCascade, refChecker ports.ReferenceChecker) *UpdatePlanUseCase {
	t.Helper()
	return NewUpdatePlanUseCase(
		UpdatePlanRepositories{Plan: planRepo, PricePlan: pricePlanRepo},
		UpdatePlanServices{
			Authorizer:       noOpAuth(),
			Transactor:       noTxn{},
			Translator:       noOpTranslation(),
			ReferenceChecker: refChecker,
		},
	)
}

// existing plan stamped with optional client_id.
func seedPlan(id, clientID string) *planpb.Plan {
	idCopy := id
	plan := &planpb.Plan{Id: &idCopy, Name: "Audit Engagement", Active: true}
	if clientID != "" {
		c := clientID
		plan.ClientId = &c
	}
	return plan
}

func TestUpdatePlan_MasterToClient_NoActiveSubs_Success(t *testing.T) {
	planRepo := newMockPlanRepo()
	planRepo.plans["plan-1"] = seedPlan("plan-1", "")
	pricePlanRepo := &pricePlanRepoForCascade{
		rows: []*priceplanpb.PricePlan{
			{Id: "pp-1", PlanId: "plan-1", Active: true, BillingCurrency: "PHP"},
		},
	}
	uc := newUpdateUC(t, planRepo, pricePlanRepo, &stubReferenceChecker{})

	id := "plan-1"
	clientID := "client-cruz"
	_, err := uc.Execute(context.Background(), &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{Id: &id, Name: "Audit Engagement", ClientId: &clientID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(pricePlanRepo.updateCalls); got != 1 {
		t.Errorf("expected exactly one cascade update, got %d", got)
	}
	if pricePlanRepo.updateCalls[0].GetClientId() != clientID {
		t.Errorf("expected cascaded client_id=%s, got %q", clientID, pricePlanRepo.updateCalls[0].GetClientId())
	}
}

func TestUpdatePlan_MasterToClient_WithActiveSub_Rejects(t *testing.T) {
	planRepo := newMockPlanRepo()
	planRepo.plans["plan-1"] = seedPlan("plan-1", "")
	pricePlanRepo := &pricePlanRepoForCascade{}
	refChecker := &stubReferenceChecker{lockedIDs: map[string]bool{"plan-1": true}}
	uc := newUpdateUC(t, planRepo, pricePlanRepo, refChecker)

	id := "plan-1"
	clientID := "client-cruz"
	_, err := uc.Execute(context.Background(), &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{Id: &id, Name: "Audit Engagement", ClientId: &clientID},
	})
	if err == nil {
		t.Fatalf("expected clientScopeLocked error")
	}
	if got := len(pricePlanRepo.updateCalls); got != 0 {
		t.Errorf("cascade should not run when client_id reassignment is locked, got %d updates", got)
	}
}

func TestUpdatePlan_ClientToOtherClient_WithActiveSub_Rejects(t *testing.T) {
	planRepo := newMockPlanRepo()
	planRepo.plans["plan-1"] = seedPlan("plan-1", "client-a")
	pricePlanRepo := &pricePlanRepoForCascade{}
	uc := newUpdateUC(t, planRepo, pricePlanRepo, &stubReferenceChecker{lockedIDs: map[string]bool{"plan-1": true}})

	id := "plan-1"
	other := "client-b"
	_, err := uc.Execute(context.Background(), &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{Id: &id, Name: "Audit Engagement", ClientId: &other},
	})
	if err == nil {
		t.Fatalf("expected lock error on cross-client reassignment")
	}
}

func TestUpdatePlan_ClientToMaster_WithActiveSub_Rejects(t *testing.T) {
	planRepo := newMockPlanRepo()
	planRepo.plans["plan-1"] = seedPlan("plan-1", "client-a")
	pricePlanRepo := &pricePlanRepoForCascade{}
	uc := newUpdateUC(t, planRepo, pricePlanRepo, &stubReferenceChecker{lockedIDs: map[string]bool{"plan-1": true}})

	id := "plan-1"
	_, err := uc.Execute(context.Background(), &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{Id: &id, Name: "Audit Engagement"}, // empty client_id = master
	})
	if err == nil {
		t.Fatalf("expected lock error on revert-to-master")
	}
}

func TestUpdatePlan_ClientToMaster_NoActiveSub_CascadesNullToChildren(t *testing.T) {
	planRepo := newMockPlanRepo()
	planRepo.plans["plan-1"] = seedPlan("plan-1", "client-a")
	currentClient := "client-a"
	pricePlanRepo := &pricePlanRepoForCascade{
		rows: []*priceplanpb.PricePlan{
			{Id: "pp-1", PlanId: "plan-1", Active: true, BillingCurrency: "PHP", ClientId: &currentClient},
		},
	}
	uc := newUpdateUC(t, planRepo, pricePlanRepo, &stubReferenceChecker{})

	id := "plan-1"
	_, err := uc.Execute(context.Background(), &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{Id: &id, Name: "Audit Engagement"}, // master
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(pricePlanRepo.updateCalls); got != 1 {
		t.Fatalf("expected one cascade, got %d", got)
	}
	if pricePlanRepo.updateCalls[0].GetClientId() != "" {
		t.Errorf("expected child PricePlan client_id reverted to NULL, got %q", pricePlanRepo.updateCalls[0].GetClientId())
	}
}

func TestUpdatePlan_NameOnlyChange_Allowed_RegardlessOfSubscriptionState(t *testing.T) {
	planRepo := newMockPlanRepo()
	planRepo.plans["plan-1"] = seedPlan("plan-1", "")
	pricePlanRepo := &pricePlanRepoForCascade{}
	uc := newUpdateUC(t, planRepo, pricePlanRepo, &stubReferenceChecker{lockedIDs: map[string]bool{"plan-1": true}})

	id := "plan-1"
	// client_id unchanged (still "" master). Lock should NOT fire even though
	// active subscriptions exist — because client_id isn't actually changing.
	_, err := uc.Execute(context.Background(), &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{Id: &id, Name: "Audit Engagement v2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(pricePlanRepo.updateCalls); got != 0 {
		t.Errorf("cascade should not run when client_id unchanged, got %d updates", got)
	}
}

func TestUpdatePlan_RefCheckerError_Bubbles(t *testing.T) {
	planRepo := newMockPlanRepo()
	planRepo.plans["plan-1"] = seedPlan("plan-1", "")
	pricePlanRepo := &pricePlanRepoForCascade{}
	uc := newUpdateUC(t, planRepo, pricePlanRepo, &stubReferenceChecker{err: errors.New("db down")})

	id := "plan-1"
	c := "client-x"
	_, err := uc.Execute(context.Background(), &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{Id: &id, Name: "Audit Engagement", ClientId: &c},
	})
	if err == nil {
		t.Fatalf("expected ref checker error to surface")
	}
}
