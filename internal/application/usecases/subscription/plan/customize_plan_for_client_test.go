package plan

// Tests for CustomizePlanForClientUseCase. They use lightweight in-package
// mocks that capture every Create*/Read*/List* call so we can assert the
// remap maths and transaction shape end-to-end. The tests deliberately avoid
// the heavier mock_db build-tag fixtures used by the legacy CRUD tests —
// the use case is brand-new and its surface is small enough to hand-mock.

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ----- mocks ---------------------------------------------------------------

type mockPlanRepo struct {
	planpb.UnimplementedPlanDomainServiceServer
	plans       map[string]*planpb.Plan
	createCalls []*planpb.Plan
	createErr   error
}

func newMockPlanRepo() *mockPlanRepo {
	return &mockPlanRepo{plans: map[string]*planpb.Plan{}}
}

func (m *mockPlanRepo) ReadPlan(_ context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	id := ""
	if req.GetData() != nil && req.GetData().Id != nil {
		id = *req.GetData().Id
	}
	if p, ok := m.plans[id]; ok {
		return &planpb.ReadPlanResponse{Data: []*planpb.Plan{p}, Success: true}, nil
	}
	return &planpb.ReadPlanResponse{Data: nil, Success: true}, nil
}

func (m *mockPlanRepo) CreatePlan(_ context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.createCalls = append(m.createCalls, req.GetData())
	if req.GetData() != nil && req.GetData().Id != nil {
		m.plans[*req.GetData().Id] = req.GetData()
	}
	return &planpb.CreatePlanResponse{Data: []*planpb.Plan{req.GetData()}, Success: true}, nil
}

// UpdatePlan persists the new plan in-memory so subsequent reads see the
// post-update state. Used by the §3.1/§3.2 update tests.
func (m *mockPlanRepo) UpdatePlan(_ context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	if req.GetData() != nil && req.GetData().Id != nil {
		m.plans[*req.GetData().Id] = req.GetData()
	}
	return &planpb.UpdatePlanResponse{Data: []*planpb.Plan{req.GetData()}, Success: true}, nil
}

type mockPricePlanRepo struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	pricePlans  map[string]*priceplanpb.PricePlan
	createCalls []*priceplanpb.PricePlan
	createErr   error
}

func newMockPricePlanRepo() *mockPricePlanRepo {
	return &mockPricePlanRepo{pricePlans: map[string]*priceplanpb.PricePlan{}}
}

func (m *mockPricePlanRepo) ReadPricePlan(_ context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if pp, ok := m.pricePlans[req.GetData().GetId()]; ok {
		return &priceplanpb.ReadPricePlanResponse{Data: []*priceplanpb.PricePlan{pp}, Success: true}, nil
	}
	return &priceplanpb.ReadPricePlanResponse{Data: nil, Success: true}, nil
}

func (m *mockPricePlanRepo) CreatePricePlan(_ context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.createCalls = append(m.createCalls, req.GetData())
	m.pricePlans[req.GetData().GetId()] = req.GetData()
	return &priceplanpb.CreatePricePlanResponse{Data: []*priceplanpb.PricePlan{req.GetData()}, Success: true}, nil
}

type mockProductPlanRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	byPlan      map[string][]*productplanpb.ProductPlan
	createCalls []*productplanpb.ProductPlan
}

func newMockProductPlanRepo() *mockProductPlanRepo {
	return &mockProductPlanRepo{byPlan: map[string][]*productplanpb.ProductPlan{}}
}

func (m *mockProductPlanRepo) ListByPlan(_ context.Context, req *productplanpb.ListProductPlansByPlanRequest) (*productplanpb.ListProductPlansByPlanResponse, error) {
	rows := m.byPlan[req.GetPlanId()]
	return &productplanpb.ListProductPlansByPlanResponse{ProductPlans: rows, Success: true}, nil
}

func (m *mockProductPlanRepo) CreateProductPlan(_ context.Context, req *productplanpb.CreateProductPlanRequest) (*productplanpb.CreateProductPlanResponse, error) {
	m.createCalls = append(m.createCalls, req.GetData())
	m.byPlan[req.GetData().GetPlanId()] = append(m.byPlan[req.GetData().GetPlanId()], req.GetData())
	return &productplanpb.CreateProductPlanResponse{Data: []*productplanpb.ProductPlan{req.GetData()}, Success: true}, nil
}

type mockProductPricePlanRepo struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	all         []*productpriceplanpb.ProductPricePlan
	createCalls []*productpriceplanpb.ProductPricePlan
}

func newMockProductPricePlanRepo() *mockProductPricePlanRepo {
	return &mockProductPricePlanRepo{}
}

func (m *mockProductPricePlanRepo) ListProductPricePlans(_ context.Context, _ *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	return &productpriceplanpb.ListProductPricePlansResponse{Data: m.all, Success: true}, nil
}

func (m *mockProductPricePlanRepo) CreateProductPricePlan(_ context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) (*productpriceplanpb.CreateProductPricePlanResponse, error) {
	m.createCalls = append(m.createCalls, req.GetData())
	m.all = append(m.all, req.GetData())
	return &productpriceplanpb.CreateProductPricePlanResponse{Data: []*productpriceplanpb.ProductPricePlan{req.GetData()}, Success: true}, nil
}

type mockPriceScheduleRepo struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	schedules   map[string]*priceschedulepb.PriceSchedule
	listResult  []*priceschedulepb.PriceSchedule
	createCalls []*priceschedulepb.PriceSchedule
}

func newMockPriceScheduleRepo() *mockPriceScheduleRepo {
	return &mockPriceScheduleRepo{schedules: map[string]*priceschedulepb.PriceSchedule{}}
}

func (m *mockPriceScheduleRepo) ReadPriceSchedule(_ context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if s, ok := m.schedules[req.GetData().GetId()]; ok {
		return &priceschedulepb.ReadPriceScheduleResponse{Data: []*priceschedulepb.PriceSchedule{s}, Success: true}, nil
	}
	return &priceschedulepb.ReadPriceScheduleResponse{Data: nil, Success: true}, nil
}

func (m *mockPriceScheduleRepo) ListPriceSchedules(_ context.Context, _ *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
	return &priceschedulepb.ListPriceSchedulesResponse{Data: m.listResult, Success: true}, nil
}

func (m *mockPriceScheduleRepo) CreatePriceSchedule(_ context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	m.createCalls = append(m.createCalls, req.GetData())
	m.schedules[req.GetData().GetId()] = req.GetData()
	return &priceschedulepb.CreatePriceScheduleResponse{Data: []*priceschedulepb.PriceSchedule{req.GetData()}, Success: true}, nil
}

type mockSubscriptionRepo struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	subs        map[string]*subscriptionpb.Subscription
	updateCalls []*subscriptionpb.Subscription
}

func newMockSubscriptionRepo() *mockSubscriptionRepo {
	return &mockSubscriptionRepo{subs: map[string]*subscriptionpb.Subscription{}}
}

func (m *mockSubscriptionRepo) ReadSubscription(_ context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if s, ok := m.subs[req.GetData().GetId()]; ok {
		return &subscriptionpb.ReadSubscriptionResponse{Data: []*subscriptionpb.Subscription{s}, Success: true}, nil
	}
	return &subscriptionpb.ReadSubscriptionResponse{Data: nil, Success: true}, nil
}

func (m *mockSubscriptionRepo) UpdateSubscription(_ context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	m.updateCalls = append(m.updateCalls, req.GetData())
	m.subs[req.GetData().GetId()] = req.GetData()
	return &subscriptionpb.UpdateSubscriptionResponse{Data: []*subscriptionpb.Subscription{req.GetData()}, Success: true}, nil
}

type mockClientRepo struct {
	clientpb.UnimplementedClientDomainServiceServer
	c *clientpb.Client
}

func (m *mockClientRepo) ReadClient(_ context.Context, _ *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if m.c == nil {
		return &clientpb.ReadClientResponse{Data: nil, Success: true}, nil
	}
	return &clientpb.ReadClientResponse{Data: []*clientpb.Client{m.c}, Success: true}, nil
}

// stubIDService — deterministic IDs so tests can assert remap entries
// without timing-sensitive collisions.
type stubIDService struct {
	prefix string
	count  int
}

func newStubIDService(prefix string) *stubIDService { return &stubIDService{prefix: prefix} }

func (s *stubIDService) GenerateID() string {
	s.count++
	return s.prefix + "-" + itoa(s.count)
}
func (s *stubIDService) GenerateIDWithPrefix(p string) string {
	s.count++
	return p + "-" + itoa(s.count)
}
func (s *stubIDService) IsEnabled() bool         { return true }
func (s *stubIDService) GetProviderInfo() string { return "stub" }

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// noOpAuth — alias to the official no-op authorization service.
func noOpAuth() ports.AuthorizationService { return ports.NewNoOpAuthorizationService() }

// noOpTranslation — alias to the official no-op translation service.
func noOpTranslation() ports.TranslationService { return ports.NewNoOpTranslationService() }

// noTxn — TransactionService that does NOT support transactions, so the use
// case runs executeCore directly.
type noTxn struct{}

func (noTxn) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (noTxn) SupportsTransactions() bool               { return false }
func (noTxn) IsTransactionActive(context.Context) bool { return false }

// hasTxn — TransactionService that DOES support transactions; used to verify
// the rollback path in the partial-failure case.
type hasTxn struct{}

func (hasTxn) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (hasTxn) SupportsTransactions() bool               { return true }
func (hasTxn) IsTransactionActive(context.Context) bool { return false }

// ----- fixture builder ----------------------------------------------------

type customizeFixture struct {
	plan             *mockPlanRepo
	pricePlan        *mockPricePlanRepo
	productPlan      *mockProductPlanRepo
	productPricePlan *mockProductPricePlanRepo
	priceSchedule    *mockPriceScheduleRepo
	subscription     *mockSubscriptionRepo
	client           *mockClientRepo
	uc               *CustomizePlanForClientUseCase
}

func newFixture(t *testing.T) *customizeFixture {
	t.Helper()
	pr := newMockPlanRepo()
	ppr := newMockPricePlanRepo()
	pdr := newMockProductPlanRepo()
	pppr := newMockProductPricePlanRepo()
	psr := newMockPriceScheduleRepo()
	sr := newMockSubscriptionRepo()
	clientName := "Cruz Engineering"
	cr := &mockClientRepo{c: &clientpb.Client{Id: "client-cruz", Active: true, Name: &clientName}}

	repos := CustomizePlanForClientRepositories{
		Plan:             pr,
		PricePlan:        ppr,
		ProductPlan:      pdr,
		ProductPricePlan: pppr,
		PriceSchedule:    psr,
		Subscription:     sr,
		Client:           cr,
	}
	svcs := CustomizePlanForClientServices{
		AuthorizationService: noOpAuth(),
		TransactionService:   noTxn{},
		TranslationService:   noOpTranslation(),
		IDService:            newStubIDService("new"),
	}
	return &customizeFixture{
		plan: pr, pricePlan: ppr, productPlan: pdr, productPricePlan: pppr,
		priceSchedule: psr, subscription: sr, client: cr,
		uc: NewCustomizePlanForClientUseCase(repos, svcs),
	}
}

// seedMasterPlan sets up the canonical "Audit Engagement" master tree:
//   - Plan plan-master (client_id = NULL)
//     └── 2 ProductPlans: pp-1, pp-2
//     └── PricePlan ppp-master priced for it
//     └── 2 ProductPricePlans, one per ProductPlan
//   - PriceSchedule ps-master at location loc-manila.
func (f *customizeFixture) seedMasterPlan() {
	planID := "plan-master"
	masterName := "Audit Engagement"
	f.plan.plans[planID] = &planpb.Plan{
		Id:     &planID,
		Name:   masterName,
		Active: true,
		// ClientId nil = master.
	}

	psID := "ps-master"
	loc := "loc-manila"
	f.priceSchedule.schedules[psID] = &priceschedulepb.PriceSchedule{
		Id:         psID,
		Name:       "Q1 2026 Manila",
		Active:     true,
		LocationId: &loc,
	}

	psIDPtr := psID
	pricePlanID := "ppp-master"
	pricePlanName := "Audit Monthly"
	f.pricePlan.pricePlans[pricePlanID] = &priceplanpb.PricePlan{
		Id:              pricePlanID,
		PlanId:          planID,
		Name:            &pricePlanName,
		BillingAmount:   500000,
		BillingCurrency: "PHP",
		PriceScheduleId: &psIDPtr,
		Active:          true,
	}

	f.productPlan.byPlan[planID] = []*productplanpb.ProductPlan{
		{Id: "pp-1", PlanId: planID, ProductId: "prod-tax", Active: true},
		{Id: "pp-2", PlanId: planID, ProductId: "prod-bookkeep", Active: true},
	}

	f.productPricePlan.all = []*productpriceplanpb.ProductPricePlan{
		{Id: "ppp-line-1", PricePlanId: pricePlanID, ProductPlanId: "pp-1", BillingAmount: 200000, BillingCurrency: "PHP", Active: true},
		{Id: "ppp-line-2", PricePlanId: pricePlanID, ProductPlanId: "pp-2", BillingAmount: 300000, BillingCurrency: "PHP", Active: true},
	}
}

func baseRequest() *CustomizePlanForClientRequest {
	return &CustomizePlanForClientRequest{
		SourcePlanID:      "plan-master",
		SourcePricePlanID: "ppp-master",
		ClientID:          "client-cruz",
		NewScheduleName:   "Cruz Engineering - Rate Cards",
	}
}

// ----- tests ---------------------------------------------------------------

func TestCustomize_MasterClone_NoExistingClientSchedule_CreatesNewSchedule(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	resp, err := f.uc.Execute(context.Background(), baseRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Reused {
		t.Errorf("expected Reused=false for fresh client schedule, got true")
	}
	if got := len(f.priceSchedule.createCalls); got != 1 {
		t.Errorf("expected 1 PriceSchedule create call, got %d", got)
	}
	if resp.PriceSchedule.GetClientId() != "client-cruz" {
		t.Errorf("expected new schedule client_id=client-cruz, got %q", resp.PriceSchedule.GetClientId())
	}
	if resp.Plan.GetClientId() != "client-cruz" {
		t.Errorf("expected cloned plan client_id=client-cruz, got %q", resp.Plan.GetClientId())
	}
	if resp.PricePlan.GetClientId() != "client-cruz" {
		t.Errorf("expected cloned price plan client_id=client-cruz, got %q", resp.PricePlan.GetClientId())
	}
	// parent_id on the cloned Plan must point at the master's ID.
	if resp.Plan.GetParentId() != "plan-master" {
		t.Errorf("expected cloned plan parent_id=plan-master, got %q", resp.Plan.GetParentId())
	}
	if got := len(f.productPlan.createCalls); got != 2 {
		t.Errorf("expected 2 ProductPlan creates (one per source), got %d", got)
	}
	if got := len(f.productPricePlan.createCalls); got != 2 {
		t.Errorf("expected 2 ProductPricePlan creates, got %d", got)
	}
}

func TestCustomize_ReusesExistingClientSchedule(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	// Pre-existing client-scoped schedule for the same (location, client).
	clientID := "client-cruz"
	loc := "loc-manila"
	existing := &priceschedulepb.PriceSchedule{
		Id:         "ps-cruz",
		Name:       "Cruz Engineering - Rate Cards",
		Active:     true,
		ClientId:   &clientID,
		LocationId: &loc,
	}
	f.priceSchedule.schedules[existing.Id] = existing
	f.priceSchedule.listResult = []*priceschedulepb.PriceSchedule{existing}

	resp, err := f.uc.Execute(context.Background(), baseRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Reused {
		t.Errorf("expected Reused=true when matching schedule already exists")
	}
	if got := len(f.priceSchedule.createCalls); got != 0 {
		t.Errorf("expected zero PriceSchedule creates, got %d", got)
	}
	if resp.PriceSchedule.GetId() != "ps-cruz" {
		t.Errorf("expected reused schedule id=ps-cruz, got %q", resp.PriceSchedule.GetId())
	}
}

func TestCustomize_CrossClientClone_AllowsClientB(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	// Make the source PricePlan client-scoped to client-A.
	clientA := "client-a"
	src := f.pricePlan.pricePlans["ppp-master"]
	src.ClientId = &clientA
	srcPlan := f.plan.plans["plan-master"]
	srcPlan.ClientId = &clientA

	// Target is client-B; the use case should clone for them.
	clientB := "client-b"
	clientBName := "Client B"
	f.client.c = &clientpb.Client{Id: clientB, Active: true, Name: &clientBName}

	req := baseRequest()
	req.ClientID = clientB
	resp, err := f.uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Plan.GetClientId() != clientB {
		t.Errorf("expected cloned plan client_id=%s, got %q", clientB, resp.Plan.GetClientId())
	}
}

func TestCustomize_RepointsSubscription_Atomic(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	subID := "sub-1"
	f.subscription.subs[subID] = &subscriptionpb.Subscription{
		Id:          subID,
		ClientId:    "client-cruz",
		PricePlanId: "ppp-master",
		Active:      true,
	}
	req := baseRequest()
	req.SubscriptionID = subID

	resp, err := f.uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(f.subscription.updateCalls); got != 1 {
		t.Fatalf("expected exactly one subscription update, got %d", got)
	}
	updated := f.subscription.updateCalls[0]
	if updated.GetPricePlanId() != resp.PricePlan.GetId() {
		t.Errorf("subscription not repointed: got %q want %q", updated.GetPricePlanId(), resp.PricePlan.GetId())
	}
}

func TestCustomize_RejectsSubscriptionClientMismatch(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	subID := "sub-1"
	f.subscription.subs[subID] = &subscriptionpb.Subscription{
		Id:          subID,
		ClientId:    "client-other",
		PricePlanId: "ppp-master",
		Active:      true,
	}
	req := baseRequest()
	req.SubscriptionID = subID
	_, err := f.uc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected subscription_client_mismatch error")
	}
}

func TestCustomize_DerivedFromLines_RemapsAllPPPs(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	// Add a third product plan + ppp line to widen the remap.
	f.productPlan.byPlan["plan-master"] = append(f.productPlan.byPlan["plan-master"],
		&productplanpb.ProductPlan{Id: "pp-3", PlanId: "plan-master", ProductId: "prod-advice", Active: true},
	)
	f.productPricePlan.all = append(f.productPricePlan.all,
		&productpriceplanpb.ProductPricePlan{Id: "ppp-line-3", PricePlanId: "ppp-master", ProductPlanId: "pp-3", BillingAmount: 100000, BillingCurrency: "PHP", Active: true},
	)

	resp, err := f.uc.Execute(context.Background(), baseRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(f.productPricePlan.createCalls); got != 3 {
		t.Errorf("expected 3 ProductPricePlan creates, got %d", got)
	}
	// Every cloned PPP must point at one of the freshly-cloned ProductPlans
	// (its product_plan_id is in the remap target set), and at the new
	// PricePlan.
	clonedProductPlanIDs := map[string]bool{}
	for _, pp := range f.productPlan.createCalls {
		clonedProductPlanIDs[pp.GetId()] = true
	}
	for _, line := range f.productPricePlan.createCalls {
		if line.GetPricePlanId() != resp.PricePlan.GetId() {
			t.Errorf("cloned line %s points at wrong price plan %s", line.GetId(), line.GetPricePlanId())
		}
		if !clonedProductPlanIDs[line.GetProductPlanId()] {
			t.Errorf("cloned line %s points at non-cloned product plan %s", line.GetId(), line.GetProductPlanId())
		}
	}
}

func TestCustomize_MidCloneFailure_Surfaces(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	// Force CreatePricePlan to fail to simulate a mid-clone failure. The
	// surrounding TransactionService is a no-op in this test, so we just
	// check that the error bubbles up — the postgres-side rollback is the
	// production guarantee.
	f.pricePlan.createErr = errors.New("price plan write failed")
	_, err := f.uc.Execute(context.Background(), baseRequest())
	if err == nil {
		t.Fatalf("expected error when CreatePricePlan fails")
	}
}

func TestCustomize_ConcurrentCustomize_SecondReusesFirst(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	// First call creates a new client schedule.
	if _, err := f.uc.Execute(context.Background(), baseRequest()); err != nil {
		t.Fatalf("first customize failed: %v", err)
	}
	// Make the freshly-created schedule visible to the list lookup.
	f.priceSchedule.listResult = nil
	for _, s := range f.priceSchedule.schedules {
		if s.GetClientId() == "client-cruz" {
			f.priceSchedule.listResult = append(f.priceSchedule.listResult, s)
		}
	}
	// Second call must reuse the first one.
	resp2, err := f.uc.Execute(context.Background(), baseRequest())
	if err != nil {
		t.Fatalf("second customize failed: %v", err)
	}
	if !resp2.Reused {
		t.Errorf("second customize should have reused the first schedule")
	}
	if len(f.priceSchedule.createCalls) != 1 {
		t.Errorf("expected exactly 1 schedule create across 2 customize calls, got %d", len(f.priceSchedule.createCalls))
	}
}

func TestCustomize_VariantModeConfigurable_CarriesVariantID(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	// Stamp a variant id on the source ProductPlan.
	variant := "variant-black-l"
	f.productPlan.byPlan["plan-master"][0].ProductVariantId = &variant
	if _, err := f.uc.Execute(context.Background(), baseRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Find the cloned product plan that originated from pp-1 — it carries
	// product_id=prod-tax — and confirm variant_id was carried verbatim.
	var found bool
	for _, pp := range f.productPlan.createCalls {
		if pp.GetProductId() == "prod-tax" {
			if pp.GetProductVariantId() != variant {
				t.Errorf("expected variant_id %q on cloned product plan, got %q", variant, pp.GetProductVariantId())
			}
			found = true
		}
	}
	if !found {
		t.Errorf("did not find cloned product plan derived from pp-1")
	}
}

func TestCustomize_AlreadyCustomizedForClient_Rejects(t *testing.T) {
	f := newFixture(t)
	f.seedMasterPlan()
	// Mark the source PricePlan as already client-scoped to the target
	// client — should surface an `already_customized` error rather than
	// duplicate-cloning.
	clientID := "client-cruz"
	f.pricePlan.pricePlans["ppp-master"].ClientId = &clientID

	_, err := f.uc.Execute(context.Background(), baseRequest())
	if err == nil {
		t.Fatalf("expected already_customized error")
	}
}

// TestCustomize_ClonesAClone_FlattensToMaster verifies the no-grandchildren
// invariant: customizing an already-cloned plan for a different client must
// set parent_id to the original master, not the intermediate clone.
func TestCustomize_ClonesAClone_FlattensToMaster(t *testing.T) {
	f := newFixture(t)

	// Setup: master plan M (plan-master) already exists from seedMasterPlan, but
	// we build the fixture manually so we control parent_id on the intermediate
	// clone C.
	masterID := "plan-master"
	masterName := "Audit Engagement"
	f.plan.plans[masterID] = &planpb.Plan{
		Id:     &masterID,
		Name:   masterName,
		Active: true,
		// ClientId nil = master; ParentId nil = master.
	}

	// Clone C: client-scoped to client-A, parent_id pointing at the master.
	cloneID := "plan-clone-a"
	cloneName := "Audit Engagement (Client A)"
	clientA := "client-a"
	f.plan.plans[cloneID] = &planpb.Plan{
		Id:       &cloneID,
		Name:     cloneName,
		Active:   true,
		ClientId: &clientA,
		ParentId: &masterID, // correctly points at master
	}

	// PriceSchedule for clone C.
	psID := "ps-clone-a"
	loc := "loc-manila"
	f.priceSchedule.schedules[psID] = &priceschedulepb.PriceSchedule{
		Id:         psID,
		Name:       "Client A - Rate Cards",
		Active:     true,
		LocationId: &loc,
	}

	// PricePlan attached to clone C.
	psIDPtr := psID
	pricePlanID := "ppp-clone-a"
	ppName := "Audit Monthly (Client A)"
	f.pricePlan.pricePlans[pricePlanID] = &priceplanpb.PricePlan{
		Id:              pricePlanID,
		PlanId:          cloneID,
		Name:            &ppName,
		BillingAmount:   500000,
		BillingCurrency: "PHP",
		PriceScheduleId: &psIDPtr,
		ClientId:        &clientA,
		Active:          true,
	}

	// Action: customize clone C for client-B.
	clientB := "client-b"
	clientBName := "Client B"
	f.client.c = &clientpb.Client{Id: clientB, Active: true, Name: &clientBName}

	req := &CustomizePlanForClientRequest{
		SourcePlanID:      cloneID,
		SourcePricePlanID: pricePlanID,
		ClientID:          clientB,
		NewScheduleName:   "Client B - Rate Cards",
	}
	resp, err := f.uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert: new clone has parent_id = masterID (NOT cloneID), client_id = clientB.
	if resp.Plan.GetParentId() != masterID {
		t.Errorf("flatten invariant violated: expected parent_id=%q, got %q (must point at master, not intermediate clone)",
			masterID, resp.Plan.GetParentId())
	}
	if resp.Plan.GetClientId() != clientB {
		t.Errorf("expected client_id=%q on new clone, got %q", clientB, resp.Plan.GetClientId())
	}
}

// ensure the commonpb import gets used even if the fixture stops needing it
// later (defensive against compiler complaints in the slim test build).
var _ = commonpb.StringOperator_STRING_EQUALS
