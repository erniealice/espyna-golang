package revenue

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ---------------------------------------------------------------------------
// Mock repositories
// ---------------------------------------------------------------------------

type recognizeMocks struct {
	subscription      *subscriptionpb.Subscription
	pricePlan         *priceplanpb.PricePlan
	productPricePlans []*productpriceplanpb.ProductPricePlan
	client            *clientpb.Client
	priceSchedule     *priceschedulepb.PriceSchedule

	priorRevenues  []*revenuepb.Revenue
	priorLineItems map[string][]*revenuelineitempb.RevenueLineItem // keyed by revenue_id

	createdRevenue   *revenuepb.Revenue
	createdLineItems []*revenuelineitempb.RevenueLineItem
}

type mockSubscriptionRepo struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	sub *subscriptionpb.Subscription
}

func (m *mockSubscriptionRepo) ReadSubscription(_ context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if m.sub == nil || (req != nil && req.GetData() != nil && req.GetData().GetId() != m.sub.GetId() && m.sub.GetId() != "") {
		return &subscriptionpb.ReadSubscriptionResponse{Data: nil, Success: true}, nil
	}
	return &subscriptionpb.ReadSubscriptionResponse{
		Data:    []*subscriptionpb.Subscription{m.sub},
		Success: true,
	}, nil
}

type mockPricePlanRepo struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	pp *priceplanpb.PricePlan
}

func (m *mockPricePlanRepo) ReadPricePlan(_ context.Context, _ *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if m.pp == nil {
		return &priceplanpb.ReadPricePlanResponse{Data: nil, Success: true}, nil
	}
	return &priceplanpb.ReadPricePlanResponse{
		Data:    []*priceplanpb.PricePlan{m.pp},
		Success: true,
	}, nil
}

type mockProductPricePlanRepo struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	rows []*productpriceplanpb.ProductPricePlan
}

func (m *mockProductPricePlanRepo) ListProductPricePlans(_ context.Context, _ *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	return &productpriceplanpb.ListProductPricePlansResponse{
		Data:    m.rows,
		Success: true,
	}, nil
}

type mockPriceScheduleRepo struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	sched *priceschedulepb.PriceSchedule
}

func (m *mockPriceScheduleRepo) ReadPriceSchedule(_ context.Context, _ *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if m.sched == nil {
		return &priceschedulepb.ReadPriceScheduleResponse{Data: nil, Success: true}, nil
	}
	return &priceschedulepb.ReadPriceScheduleResponse{
		Data:    []*priceschedulepb.PriceSchedule{m.sched},
		Success: true,
	}, nil
}

type mockClientRepo struct {
	clientpb.UnimplementedClientDomainServiceServer
	c *clientpb.Client
}

func (m *mockClientRepo) ReadClient(_ context.Context, _ *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if m.c == nil {
		return &clientpb.ReadClientResponse{Data: nil, Success: true}, nil
	}
	return &clientpb.ReadClientResponse{
		Data:    []*clientpb.Client{m.c},
		Success: true,
	}, nil
}

type mockRevenueRepo struct {
	revenuepb.UnimplementedRevenueDomainServiceServer
	priorRevenues []*revenuepb.Revenue
	created       *revenuepb.Revenue
	createErr     error
	failOnCreate  bool
}

func (m *mockRevenueRepo) ListRevenues(_ context.Context, _ *revenuepb.ListRevenuesRequest) (*revenuepb.ListRevenuesResponse, error) {
	return &revenuepb.ListRevenuesResponse{
		Data:    m.priorRevenues,
		Success: true,
	}, nil
}

func (m *mockRevenueRepo) CreateRevenue(_ context.Context, req *revenuepb.CreateRevenueRequest) (*revenuepb.CreateRevenueResponse, error) {
	if m.failOnCreate {
		return nil, errors.New("simulated create failure")
	}
	if m.createErr != nil {
		return nil, m.createErr
	}
	out := req.GetData()
	if out.GetId() == "" {
		out.Id = "rev-new"
	}
	m.created = out
	return &revenuepb.CreateRevenueResponse{
		Data:    []*revenuepb.Revenue{out},
		Success: true,
	}, nil
}

type mockRevenueLineItemRepo struct {
	revenuelineitempb.UnimplementedRevenueLineItemDomainServiceServer
	priorLineItems map[string][]*revenuelineitempb.RevenueLineItem
	created        []*revenuelineitempb.RevenueLineItem
}

func (m *mockRevenueLineItemRepo) ListRevenueLineItems(_ context.Context, req *revenuelineitempb.ListRevenueLineItemsRequest) (*revenuelineitempb.ListRevenueLineItemsResponse, error) {
	id := ""
	if req != nil && req.RevenueId != nil {
		id = *req.RevenueId
	}
	rows := m.priorLineItems[id]
	return &revenuelineitempb.ListRevenueLineItemsResponse{
		Data:    rows,
		Success: true,
	}, nil
}

func (m *mockRevenueLineItemRepo) CreateRevenueLineItem(_ context.Context, req *revenuelineitempb.CreateRevenueLineItemRequest) (*revenuelineitempb.CreateRevenueLineItemResponse, error) {
	out := req.GetData()
	if out.GetId() == "" {
		out.Id = "rli-new"
	}
	m.created = append(m.created, out)
	return &revenuelineitempb.CreateRevenueLineItemResponse{
		Data:    []*revenuelineitempb.RevenueLineItem{out},
		Success: true,
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildUseCase(t *testing.T, m *recognizeMocks) (*RecognizeRevenueFromSubscriptionUseCase, *mockRevenueRepo, *mockRevenueLineItemRepo) {
	t.Helper()

	revenueRepo := &mockRevenueRepo{priorRevenues: m.priorRevenues}
	rliRepo := &mockRevenueLineItemRepo{priorLineItems: m.priorLineItems}

	repos := RecognizeRevenueFromSubscriptionRepositories{
		Revenue:          revenueRepo,
		RevenueLineItem:  rliRepo,
		Subscription:     &mockSubscriptionRepo{sub: m.subscription},
		PricePlan:        &mockPricePlanRepo{pp: m.pricePlan},
		ProductPricePlan: &mockProductPricePlanRepo{rows: m.productPricePlans},
		PriceSchedule:    &mockPriceScheduleRepo{sched: m.priceSchedule},
		Client:           &mockClientRepo{c: m.client},
		PaymentTerm:      paymenttermpb.UnimplementedPaymentTermDomainServiceServer{},
	}
	services := RecognizeRevenueFromSubscriptionServices{
		Authorizer:  ports.NewNoOpAuthorizer(),
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}
	return NewRecognizeRevenueFromSubscriptionUseCase(repos, services), revenueRepo, rliRepo
}

func ppp(id, pricePlanID string, amount int64, treatment productpriceplanpb.BillingTreatment, productName string) *productpriceplanpb.ProductPricePlan {
	out := &productpriceplanpb.ProductPricePlan{
		Id:               id,
		PricePlanId:      pricePlanID,
		BillingAmount:    amount,
		BillingCurrency:  "PHP",
		BillingTreatment: treatment,
	}
	if productName != "" {
		out.ProductPlan = &productplanpb.ProductPlan{
			Product: &productpb.Product{Name: productName},
		}
	}
	return out
}

func activeSubscription(id, pricePlanID, clientID string) *subscriptionpb.Subscription {
	return &subscriptionpb.Subscription{
		Id:          id,
		Active:      true,
		Name:        "Engagement [TEST123]",
		ClientId:    clientID,
		PricePlanId: pricePlanID,
	}
}

func basicReq(subID string) *revenuepb.CreateRevenueWithLineItemsRequest {
	subIDLocal := subID
	return &revenuepb.CreateRevenueWithLineItemsRequest{
		Data:           &revenuepb.Revenue{},
		SubscriptionId: &subIDLocal,
	}
}

// ---------------------------------------------------------------------------
// Tests — 11 cases per plan §5 Phase B
// ---------------------------------------------------------------------------

// 1. ONE_TIME plan → all PPP lines included.
func TestRecognize_OneTimePlan_AllLinesIncluded(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-1", "pp-1", "client-1"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-1",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
			BillingCurrency: "PHP",
			BillingAmount:   100000,
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-a", "pp-1", 60000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Item A"),
			ppp("ppp-b", "pp-1", 40000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL, "Item B"),
		},
	}
	uc, _, rli := buildUseCase(t, mocks)
	resp, err := uc.Execute(context.Background(), basicReq("sub-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatal("expected success")
	}
	if len(rli.created) != 2 {
		t.Fatalf("expected 2 line items, got %d", len(rli.created))
	}
	if len(resp.GetPreviewLines()) != 2 {
		t.Fatalf("expected 2 preview lines, got %d", len(resp.GetPreviewLines()))
	}
	for _, p := range resp.GetPreviewLines() {
		if p.GetTreatment() != treatmentOneTime {
			t.Errorf("expected one_time treatment, got %q", p.GetTreatment())
		}
	}
}

// 2. RECURRING plan, first cycle → RECURRING + ONE_TIME_INITIAL lines included; USAGE_BASED skipped.
func TestRecognize_RecurringPlan_FirstCycle(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-2", "pp-2", "client-2"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-2",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-r", "pp-2", 50000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Audit Hours"),
			ppp("ppp-i", "pp-2", 10000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL, "Setup Fee"),
			ppp("ppp-u", "pp-2", 0, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED, "Usage"),
		},
	}
	uc, _, rli := buildUseCase(t, mocks)
	resp, err := uc.Execute(context.Background(), basicReq("sub-2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 2 {
		t.Fatalf("expected 2 lines (recurring+initial), got %d", len(rli.created))
	}
	gotTreatments := map[string]bool{}
	for _, p := range resp.GetPreviewLines() {
		gotTreatments[p.GetTreatment()] = true
	}
	if !gotTreatments[treatmentRecurring] {
		t.Error("expected recurring treatment present")
	}
	if !gotTreatments[treatmentFirstCycle] {
		t.Error("expected first_cycle treatment present")
	}
	if len(resp.GetWarnings()) == 0 {
		t.Error("expected at least one warning for the skipped usage-based line")
	}
}

// 3. RECURRING plan, second cycle → only RECURRING lines included.
func TestRecognize_RecurringPlan_SecondCycle_SuppressesOneTimeInitial(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-3", "pp-3", "client-3"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-3",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-r2", "pp-3", 50000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Audit Hours"),
			ppp("ppp-i2", "pp-3", 10000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL, "Setup Fee"),
		},
		priorRevenues: []*revenuepb.Revenue{
			{Id: "rev-prior", Status: "complete"},
		},
		priorLineItems: map[string][]*revenuelineitempb.RevenueLineItem{
			"rev-prior": {
				{Id: "rli-prior-i", ProductPricePlanId: stringPtrTest("ppp-i2")},
			},
		},
	}
	uc, _, rli := buildUseCase(t, mocks)
	resp, err := uc.Execute(context.Background(), basicReq("sub-3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 line (recurring only), got %d", len(rli.created))
	}
	if resp.GetPreviewLines()[0].GetTreatment() != treatmentRecurring {
		t.Errorf("expected recurring treatment, got %q", resp.GetPreviewLines()[0].GetTreatment())
	}
}

// 4. CONTRACT + TOTAL_PACKAGE → behaves like RECURRING-with-CONTRACT (all lines in single cycle).
func TestRecognize_ContractTotalPackage(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-4", "pp-4", "client-4"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-4",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_CONTRACT,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
			BillingCurrency: "PHP",
			BillingAmount:   120000,
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-c1", "pp-4", 70000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Hours"),
			ppp("ppp-c2", "pp-4", 50000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL, "Setup"),
		},
	}
	uc, _, rli := buildUseCase(t, mocks)
	resp, err := uc.Execute(context.Background(), basicReq("sub-4"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(rli.created))
	}
	if resp.GetData()[0].GetTotalAmount() != 120000 {
		t.Errorf("expected header total 120000, got %d", resp.GetData()[0].GetTotalAmount())
	}
}

// 5. CONTRACT + PER_CYCLE → behaves like RECURRING (suppresses one-time-initial on second cycle).
func TestRecognize_ContractPerCycle_SecondCycle(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-5", "pp-5", "client-5"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-5",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_CONTRACT,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-cr", "pp-5", 30000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Hours"),
			ppp("ppp-ci", "pp-5", 5000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL, "Onboarding"),
		},
		priorRevenues: []*revenuepb.Revenue{{Id: "rev-prev", Status: "complete"}},
		priorLineItems: map[string][]*revenuelineitempb.RevenueLineItem{
			"rev-prev": {{Id: "rli-prev", ProductPricePlanId: stringPtrTest("ppp-ci")}},
		},
	}
	uc, _, rli := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-5")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 recurring line, got %d", len(rli.created))
	}
}

// 6. DERIVED_FROM_LINES → header total = sum of lines (PricePlan amount ignored).
func TestRecognize_DerivedFromLines(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-6", "pp-6", "client-6"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-6",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_DERIVED_FROM_LINES,
			BillingCurrency: "PHP",
			BillingAmount:   999999, // should be ignored
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-d1", "pp-6", 12000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "L1"),
			ppp("ppp-d2", "pp-6", 8000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "L2"),
		},
	}
	uc, revRepo, _ := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-6")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := revRepo.created.GetTotalAmount(); got != 20000 {
		t.Errorf("expected derived-from-lines total 20000, got %d", got)
	}
}

// 7. Currency mismatch → hard block (per plan §11.4).
func TestRecognize_CurrencyMismatch_Blocks(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-7", "pp-7", "client-7"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-7",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		client: &clientpb.Client{Id: "client-7", BillingCurrency: stringPtrTest("USD")},
	}
	uc, _, _ := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-7")); err == nil {
		t.Fatal("expected currency mismatch error, got nil")
	}
}

// 8. Idempotency conflict → returns existing-revenue error, does NOT double-create.
func TestRecognize_IdempotencyConflict_Blocks(t *testing.T) {
	periodStart := "2026-04-01T00:00:00Z"
	periodEnd := "2026-04-30T23:59:59Z"
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-8", "pp-8", "client-8"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-8",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-x", "pp-8", 50000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Hours"),
		},
		priorRevenues: []*revenuepb.Revenue{
			{
				Id:     "rev-existing",
				Status: "draft",
				Notes:  stringPtrTest("Period: " + periodStart + " → " + periodEnd),
			},
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	req := basicReq("sub-8")
	req.PeriodStart = stringPtrTest(periodStart)
	req.PeriodEnd = stringPtrTest(periodEnd)
	resp, err := uc.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("expected idempotency error, got nil")
	}
	if resp == nil || resp.GetConflictingRevenueId() != "rev-existing" {
		t.Errorf("expected conflicting_revenue_id=rev-existing, got %v", resp.GetConflictingRevenueId())
	}
	if revRepo.created != nil {
		t.Error("expected no revenue to be created on conflict")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected no line items created, got %d", len(rli.created))
	}
}

// 9. Subscription inactive → error subscription_inactive.
func TestRecognize_SubscriptionInactive(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: &subscriptionpb.Subscription{
			Id: "sub-9", Active: false, PricePlanId: "pp-9",
		},
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-9",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
	}
	uc, _, _ := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-9")); err == nil {
		t.Fatal("expected error for inactive subscription, got nil")
	}
}

// 10. Subscription with no price_plan_id → error price_plan_required.
func TestRecognize_NoPricePlan(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: &subscriptionpb.Subscription{
			Id: "sub-10", Active: true, PricePlanId: "",
		},
	}
	uc, _, _ := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-10")); err == nil {
		t.Fatal("expected error for missing price_plan_id, got nil")
	}
}

// 11. Empty PPP list + TOTAL_PACKAGE → single header line uses PricePlan.billing_amount.
func TestRecognize_EmptyPPPs_TotalPackage_BundleLine(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-11", "pp-11", "client-11"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-11",
			Name:            stringPtrTest("Audit Bundle"),
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
			BillingCurrency: "PHP",
			BillingAmount:   80000,
		},
		productPricePlans: nil,
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-11")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 bundle line, got %d", len(rli.created))
	}
	if rli.created[0].GetUnitPrice() != 80000 {
		t.Errorf("expected unit_price 80000, got %d", rli.created[0].GetUnitPrice())
	}
	if revRepo.created.GetTotalAmount() != 80000 {
		t.Errorf("expected header total 80000, got %d", revRepo.created.GetTotalAmount())
	}
}

// ---------------------------------------------------------------------------
// Tests — scenarios.md additional coverage (2026-04-28)
// ---------------------------------------------------------------------------

// E-2 · Free trial / all-zero cycle — generates a valid draft Revenue with total = 0.
func TestRecognize_FreeTrialZeroAmount(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-zero", "pp-zero", "client-zero"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-zero",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-zero", "pp-zero", 0, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Trial Visit"),
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-zero")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 line, got %d", len(rli.created))
	}
	if rli.created[0].GetTotalPrice() != 0 {
		t.Errorf("expected total_price 0, got %d", rli.created[0].GetTotalPrice())
	}
	if revRepo.created.GetTotalAmount() != 0 {
		t.Errorf("expected header total 0, got %d", revRepo.created.GetTotalAmount())
	}
	if revRepo.created.GetStatus() != "draft" {
		t.Errorf("expected status=draft, got %q", revRepo.created.GetStatus())
	}
}

// E-5 · Subscription / plan client drift — guards the recognition flow from a
// desynced client identity (sub.client_id ≠ pricePlan.client_id).
func TestRecognize_SubscriptionPlanClientDrift(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-drift", "pp-drift", "client-A"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-drift",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
			ClientId:        stringPtrTest("client-B"),
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-drift", "pp-drift", 50000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Hours"),
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-drift")); err == nil {
		t.Fatal("expected client-drift error, got nil")
	}
	if revRepo.created != nil {
		t.Error("expected no revenue created on client drift")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected no lines on client drift, got %d", len(rli.created))
	}
}

// E-8 · Operator removes a line via override — that PPP is filtered out before
// the treatment switch so neither the line nor a treatment badge is emitted.
func TestRecognize_OperatorRemovesLine(t *testing.T) {
	removed := true
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-rm", "pp-rm", "client-rm"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-rm",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-keep", "pp-rm", 30000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Keep"),
			ppp("ppp-drop", "pp-rm", 20000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Drop"),
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	req := basicReq("sub-rm")
	req.Overrides = []*revenuepb.LineItemOverride{
		{ProductPricePlanId: "ppp-drop", Removed: &removed},
	}
	if _, err := uc.Execute(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 line after removal, got %d", len(rli.created))
	}
	if got := rli.created[0].GetProductPricePlanId(); got != "ppp-keep" {
		t.Errorf("expected only ppp-keep, got %q", got)
	}
	if revRepo.created.GetTotalAmount() != 30000 {
		t.Errorf("expected header total 30000, got %d", revRepo.created.GetTotalAmount())
	}
}

// E-9 · Operator overrides amount + quantity — total_price = int64(unit*qty).
// 12000 × 1.5 = 18000 (no rounding loss at this scale).
func TestRecognize_OperatorOverridesAmountAndQuantity(t *testing.T) {
	unit := int64(12000)
	qty := 1.5
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-ov", "pp-ov", "client-ov"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-ov",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-ov", "pp-ov", 10000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Item"),
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	req := basicReq("sub-ov")
	req.Overrides = []*revenuepb.LineItemOverride{
		{ProductPricePlanId: "ppp-ov", UnitPrice: &unit, Quantity: &qty},
	}
	if _, err := uc.Execute(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 line, got %d", len(rli.created))
	}
	got := rli.created[0]
	if got.GetUnitPrice() != 12000 {
		t.Errorf("expected unit 12000, got %d", got.GetUnitPrice())
	}
	if got.GetQuantity() != 1.5 {
		t.Errorf("expected qty 1.5, got %f", got.GetQuantity())
	}
	if got.GetTotalPrice() != 18000 {
		t.Errorf("expected total 18000, got %d", got.GetTotalPrice())
	}
	if revRepo.created.GetTotalAmount() != 18000 {
		t.Errorf("expected header total 18000, got %d", revRepo.created.GetTotalAmount())
	}
}

// E-10 · Skip-header path (manual revenue-add → autoPopulateLineItems delegate).
// Caller has already created the Revenue header; only line items are written,
// and the idempotency check is bypassed even with a colliding period marker.
func TestRecognize_SkipHeaderPath(t *testing.T) {
	skip := true
	existingID := "rev-existing-header"
	periodStart := "2026-04-01T00:00:00Z"
	periodEnd := "2026-04-30T23:59:59Z"
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-skip", "pp-skip", "client-skip"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-skip",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-skip", "pp-skip", 25000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Hours"),
		},
		// A prior revenue with a colliding period marker would normally trip the
		// idempotency block, but skip_header bypasses the check.
		priorRevenues: []*revenuepb.Revenue{
			{
				Id:     "rev-collide",
				Status: "draft",
				Notes:  stringPtrTest("Period: " + periodStart + " → " + periodEnd),
			},
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	req := basicReq("sub-skip")
	req.SkipHeader = &skip
	req.ExistingRevenueId = &existingID
	req.PeriodStart = &periodStart
	req.PeriodEnd = &periodEnd
	resp, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatal("expected success")
	}
	if revRepo.created != nil {
		t.Error("expected no header insert under skip_header=true")
	}
	if len(resp.GetData()) != 0 {
		t.Errorf("expected no Revenue rows in response, got %d", len(resp.GetData()))
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 line, got %d", len(rli.created))
	}
	if rli.created[0].GetRevenueId() != existingID {
		t.Errorf("expected revenue_id=%q, got %q", existingID, rli.created[0].GetRevenueId())
	}
}

// E-11 · Dry-run preview — gating runs, preview is returned, nothing written.
func TestRecognize_DryRunPreview(t *testing.T) {
	dry := true
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-dry", "pp-dry", "client-dry"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-dry",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-dry", "pp-dry", 40000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Hours"),
			ppp("ppp-dry-i", "pp-dry", 5000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL, "Setup"),
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	req := basicReq("sub-dry")
	req.DryRun = &dry
	resp, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatal("expected success")
	}
	if revRepo.created != nil {
		t.Error("expected no Revenue insert under dry_run")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected no line items written, got %d", len(rli.created))
	}
	if len(resp.GetPreviewLines()) != 2 {
		t.Fatalf("expected 2 preview lines, got %d", len(resp.GetPreviewLines()))
	}
}

// S-3 · ONE_TIME × DERIVED_FROM_LINES — all PPPs included regardless of
// billing_treatment; every line gets the `one_time` badge; header total = sum.
func TestRecognize_OneTimeDerivedFromLines(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-otd", "pp-otd", "client-otd"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-otd",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_DERIVED_FROM_LINES,
			BillingCurrency: "PHP",
			BillingAmount:   999999, // ignored under DERIVED_FROM_LINES
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-photo", "pp-otd", 5000000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Photography"),
			ppp("ppp-cater", "pp-otd", 15000000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL, "Catering"),
			ppp("ppp-venue", "pp-otd", 8000000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Venue"),
		},
	}
	uc, revRepo, _ := buildUseCase(t, mocks)
	resp, err := uc.Execute(context.Background(), basicReq("sub-otd"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetPreviewLines()) != 3 {
		t.Fatalf("expected 3 preview lines, got %d", len(resp.GetPreviewLines()))
	}
	for i, p := range resp.GetPreviewLines() {
		if p.GetTreatment() != treatmentOneTime {
			t.Errorf("line %d: expected one_time, got %q", i, p.GetTreatment())
		}
	}
	if got := revRepo.created.GetTotalAmount(); got != 28000000 {
		t.Errorf("expected header total 28000000 (sum), got %d", got)
	}
}

// Followups §1 integration · ONE_TIME × PER_CYCLE legacy plan with a stale
// billing_cycle should normalize to TOTAL_PACKAGE inside Execute and produce
// the canonical S-1 single-bundle-line output.
func TestRecognize_OneTimePerCycle_NormalizesToBundleLine(t *testing.T) {
	cycleVal := int32(1)
	cycleUnit := "month"
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-norm", "pp-norm", "client-norm"),
		pricePlan: &priceplanpb.PricePlan{
			Id:                "pp-norm",
			Name:              stringPtrTest("Laser 6-Session"),
			BillingKind:       priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
			AmountBasis:       priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE, // legacy / incoherent
			BillingCurrency:   "PHP",
			BillingAmount:     120000000,
			BillingCycleValue: &cycleVal,  // stale
			BillingCycleUnit:  &cycleUnit, // stale
		},
		productPricePlans: nil, // empty PPP list → bundle fallback should fire after coercion
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-norm")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mocks.pricePlan.GetAmountBasis() != priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		t.Errorf("expected basis coerced to TOTAL_PACKAGE, got %v", mocks.pricePlan.GetAmountBasis())
	}
	if mocks.pricePlan.BillingCycleValue != nil || mocks.pricePlan.BillingCycleUnit != nil {
		t.Error("expected billing_cycle_* cleared by normalization")
	}
	if len(rli.created) != 1 {
		t.Fatalf("expected 1 bundle line after coercion, got %d", len(rli.created))
	}
	if rli.created[0].GetUnitPrice() != 120000000 {
		t.Errorf("expected unit_price 120000000, got %d", rli.created[0].GetUnitPrice())
	}
	if revRepo.created.GetTotalAmount() != 120000000 {
		t.Errorf("expected header total 120000000, got %d", revRepo.created.GetTotalAmount())
	}
}

// Followups §2 · Empty PPPs under non-TOTAL_PACKAGE basis must reject rather
// than silently writing a zero-line, zero-total Revenue.
func TestRecognize_EmptyLines_NonTotalPackage_Rejects(t *testing.T) {
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-empty", "pp-empty", "client-empty"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-empty",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_DERIVED_FROM_LINES,
			BillingCurrency: "PHP",
		},
		productPricePlans: nil,
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-empty")); err == nil {
		t.Fatal("expected no_lines_to_invoice error, got nil")
	}
	if revRepo.created != nil {
		t.Error("expected no Revenue created on empty-line rejection")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected zero line items, got %d", len(rli.created))
	}
}

// Followups §2 · All operator overrides removed → every PPP filtered → reject.
func TestRecognize_EmptyLines_AfterAllOperatorRemovals_Rejects(t *testing.T) {
	removed := true
	mocks := &recognizeMocks{
		subscription: activeSubscription("sub-allrm", "pp-allrm", "client-allrm"),
		pricePlan: &priceplanpb.PricePlan{
			Id:              "pp-allrm",
			BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE,
			BillingCurrency: "PHP",
		},
		productPricePlans: []*productpriceplanpb.ProductPricePlan{
			ppp("ppp-only", "pp-allrm", 30000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Only line"),
		},
	}
	uc, revRepo, rli := buildUseCase(t, mocks)
	req := basicReq("sub-allrm")
	req.Overrides = []*revenuepb.LineItemOverride{
		{ProductPricePlanId: "ppp-only", Removed: &removed},
	}
	if _, err := uc.Execute(context.Background(), req); err == nil {
		t.Fatal("expected no_lines_to_invoice error after all-removed, got nil")
	}
	if revRepo.created != nil {
		t.Error("expected no Revenue created when all lines removed")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected zero line items, got %d", len(rli.created))
	}
}

// stringPtrTest returns &s — local helper to avoid importing the package's
// stringPtrLocal across test+source boundaries.
func stringPtrTest(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// MILESTONE branch — unit tests (milestone-billing plan §7)
// ---------------------------------------------------------------------------

type mockBillingEventRepo struct {
	billingeventpb.UnimplementedBillingEventDomainServiceServer
	events       map[string]*billingeventpb.BillingEvent
	createdChild *billingeventpb.BillingEvent
	bySubscript  map[string][]*billingeventpb.BillingEvent
	updates      []*billingeventpb.BillingEvent
}

func (m *mockBillingEventRepo) ReadBillingEvent(_ context.Context, req *billingeventpb.ReadBillingEventRequest) (*billingeventpb.ReadBillingEventResponse, error) {
	if req == nil || req.Data == nil {
		return &billingeventpb.ReadBillingEventResponse{Success: true}, nil
	}
	ev, ok := m.events[req.Data.GetId()]
	if !ok {
		return &billingeventpb.ReadBillingEventResponse{Success: true}, nil
	}
	return &billingeventpb.ReadBillingEventResponse{
		Data:    []*billingeventpb.BillingEvent{ev},
		Success: true,
	}, nil
}

func (m *mockBillingEventRepo) ListBySubscription(_ context.Context, req *billingeventpb.ListBillingEventsBySubscriptionRequest) (*billingeventpb.ListBillingEventsBySubscriptionResponse, error) {
	out := m.bySubscript[req.GetSubscriptionId()]
	return &billingeventpb.ListBillingEventsBySubscriptionResponse{
		BillingEvents: out,
		Success:       true,
	}, nil
}

func (m *mockBillingEventRepo) UpdateBillingEvent(_ context.Context, req *billingeventpb.UpdateBillingEventRequest) (*billingeventpb.UpdateBillingEventResponse, error) {
	m.updates = append(m.updates, req.GetData())
	if m.events == nil {
		m.events = map[string]*billingeventpb.BillingEvent{}
	}
	m.events[req.GetData().GetId()] = req.GetData()
	return &billingeventpb.UpdateBillingEventResponse{
		Data:    []*billingeventpb.BillingEvent{req.GetData()},
		Success: true,
	}, nil
}

func (m *mockBillingEventRepo) CreateBillingEvent(_ context.Context, req *billingeventpb.CreateBillingEventRequest) (*billingeventpb.CreateBillingEventResponse, error) {
	out := req.GetData()
	if out.GetId() == "" {
		out.Id = "be-child"
	}
	m.createdChild = out
	return &billingeventpb.CreateBillingEventResponse{
		Data:    []*billingeventpb.BillingEvent{out},
		Success: true,
	}, nil
}

type mockJobTemplatePhaseRepo struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	phases map[string]*jobtemplatephasepb.JobTemplatePhase
}

func (m *mockJobTemplatePhaseRepo) ReadJobTemplatePhase(_ context.Context, req *jobtemplatephasepb.ReadJobTemplatePhaseRequest) (*jobtemplatephasepb.ReadJobTemplatePhaseResponse, error) {
	if req == nil || req.Data == nil {
		return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true}, nil
	}
	p, ok := m.phases[req.Data.GetId()]
	if !ok {
		return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true}, nil
	}
	return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{
		Data:    []*jobtemplatephasepb.JobTemplatePhase{p},
		Success: true,
	}, nil
}

// milestoneMocks holds the additional mock state for the MILESTONE branch on
// top of the existing recognizeMocks shape.
type milestoneMocks struct {
	*recognizeMocks
	billingEvent     *mockBillingEventRepo
	jobTemplatePhase *mockJobTemplatePhaseRepo
}

func buildMilestoneUseCase(t *testing.T, m *milestoneMocks) (*RecognizeRevenueFromSubscriptionUseCase, *mockRevenueRepo, *mockRevenueLineItemRepo) {
	t.Helper()

	revenueRepo := &mockRevenueRepo{priorRevenues: m.priorRevenues}
	rliRepo := &mockRevenueLineItemRepo{priorLineItems: m.priorLineItems}

	repos := RecognizeRevenueFromSubscriptionRepositories{
		Revenue:          revenueRepo,
		RevenueLineItem:  rliRepo,
		Subscription:     &mockSubscriptionRepo{sub: m.subscription},
		PricePlan:        &mockPricePlanRepo{pp: m.pricePlan},
		ProductPricePlan: &mockProductPricePlanRepo{rows: m.productPricePlans},
		PriceSchedule:    &mockPriceScheduleRepo{sched: m.priceSchedule},
		Client:           &mockClientRepo{c: m.client},
		PaymentTerm:      paymenttermpb.UnimplementedPaymentTermDomainServiceServer{},

		BillingEvent:     m.billingEvent,
		JobTemplatePhase: m.jobTemplatePhase,
	}
	services := RecognizeRevenueFromSubscriptionServices{
		Authorizer:  ports.NewNoOpAuthorizer(),
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}
	return NewRecognizeRevenueFromSubscriptionUseCase(repos, services), revenueRepo, rliRepo
}

func milestonePricePlan(id string) *priceplanpb.PricePlan {
	return &priceplanpb.PricePlan{
		Id:              id,
		BillingKind:     priceplanpb.BillingKind_BILLING_KIND_MILESTONE,
		AmountBasis:     priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		BillingCurrency: "PHP",
		BillingAmount:   50000000, // ₱500,000
	}
}

func readyEvent(id, subID, jtpID string, amount int64) *billingeventpb.BillingEvent {
	jtp := jtpID
	return &billingeventpb.BillingEvent{
		Id:                 id,
		Active:             true,
		SubscriptionId:     subID,
		BillableAmount:     amount,
		BillingCurrency:    "PHP",
		Status:             billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_READY,
		Trigger:            billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_PHASE_COMPLETED,
		JobTemplatePhaseId: &jtp,
	}
}

// MILESTONE-1: MILESTONE plan, billing_event_id missing → milestone_required.
func TestRecognize_Milestone_NoBillingEventID_Rejects(t *testing.T) {
	mocks := &milestoneMocks{
		recognizeMocks: &recognizeMocks{
			subscription: activeSubscription("sub-m1", "pp-m1", "client-m1"),
			pricePlan:    milestonePricePlan("pp-m1"),
		},
		billingEvent:     &mockBillingEventRepo{events: map[string]*billingeventpb.BillingEvent{}},
		jobTemplatePhase: &mockJobTemplatePhaseRepo{phases: map[string]*jobtemplatephasepb.JobTemplatePhase{}},
	}
	uc, revRepo, rli := buildMilestoneUseCase(t, mocks)
	if _, err := uc.Execute(context.Background(), basicReq("sub-m1")); err == nil {
		t.Fatal("expected milestone_required error, got nil")
	}
	if revRepo.created != nil {
		t.Error("expected no Revenue created on milestone_required reject")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected zero line items, got %d", len(rli.created))
	}
}

// MILESTONE-2: MILESTONE plan, event ready → header total = event amount;
// lines filtered to PPP rows tagged with this template phase.
func TestRecognize_Milestone_EventReady_Success(t *testing.T) {
	jtpM1 := "jtp-m1"
	bID := "be-001"
	mocks := &milestoneMocks{
		recognizeMocks: &recognizeMocks{
			subscription: activeSubscription("sub-m2", "pp-m2", "client-m2"),
			pricePlan:    milestonePricePlan("pp-m2"),
			productPricePlans: []*productpriceplanpb.ProductPricePlan{
				{
					Id:                 "ppp-design",
					PricePlanId:        "pp-m2",
					BillingAmount:      10000000,
					BillingCurrency:    "PHP",
					JobTemplatePhaseId: stringPtrTest(jtpM1),
					ProductPlan:        &productplanpb.ProductPlan{Product: &productpb.Product{Name: "Design fee"}},
				},
				{
					Id:                 "ppp-permit",
					PricePlanId:        "pp-m2",
					BillingAmount:      5000000,
					BillingCurrency:    "PHP",
					JobTemplatePhaseId: stringPtrTest(jtpM1),
					ProductPlan:        &productplanpb.ProductPlan{Product: &productpb.Product{Name: "Permits"}},
				},
				{
					// Different phase — must be excluded from M1's lines.
					Id:                 "ppp-other",
					PricePlanId:        "pp-m2",
					BillingAmount:      99999999,
					BillingCurrency:    "PHP",
					JobTemplatePhaseId: stringPtrTest("jtp-m2"),
					ProductPlan:        &productplanpb.ProductPlan{Product: &productpb.Product{Name: "Other phase"}},
				},
			},
		},
		billingEvent: &mockBillingEventRepo{
			events:      map[string]*billingeventpb.BillingEvent{bID: readyEvent(bID, "sub-m2", jtpM1, 15000000)},
			bySubscript: map[string][]*billingeventpb.BillingEvent{},
		},
		jobTemplatePhase: &mockJobTemplatePhaseRepo{
			phases: map[string]*jobtemplatephasepb.JobTemplatePhase{
				jtpM1: {Id: jtpM1, BillingPercentBps: int32Ptr(3000)},
			},
		},
	}
	uc, revRepo, rli := buildMilestoneUseCase(t, mocks)
	req := basicReq("sub-m2")
	req.BillingEventId = stringPtrTest(bID)
	resp, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatal("expected success")
	}
	if got := revRepo.created.GetTotalAmount(); got != 15000000 {
		t.Errorf("expected header total 15000000, got %d", got)
	}
	if revRepo.created.GetBillingEventId() != bID {
		t.Errorf("expected billing_event_id=%q on revenue header, got %q", bID, revRepo.created.GetBillingEventId())
	}
	if len(rli.created) != 2 {
		t.Fatalf("expected 2 lines (filtered to phase M1), got %d", len(rli.created))
	}
	for _, l := range rli.created {
		if l.GetProductPricePlanId() == "ppp-other" {
			t.Error("expected ppp-other to be excluded from M1 lines")
		}
	}
	if len(mocks.billingEvent.updates) != 1 ||
		mocks.billingEvent.updates[0].GetStatus() != billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_BILLED {
		t.Error("expected billing_event update to BILLED")
	}
}

// MILESTONE-3: billing_event_id set on a non-MILESTONE plan → milestone_not_applicable.
func TestRecognize_Milestone_NotApplicable_OnRecurringPlan(t *testing.T) {
	mocks := &milestoneMocks{
		recognizeMocks: &recognizeMocks{
			subscription: activeSubscription("sub-m3", "pp-m3", "client-m3"),
			pricePlan: &priceplanpb.PricePlan{
				Id:              "pp-m3",
				BillingKind:     priceplanpb.BillingKind_BILLING_KIND_RECURRING,
				BillingCurrency: "PHP",
			},
			productPricePlans: []*productpriceplanpb.ProductPricePlan{
				ppp("ppp-r3", "pp-m3", 50000, productpriceplanpb.BillingTreatment_BILLING_TREATMENT_RECURRING, "Hours"),
			},
		},
		billingEvent:     &mockBillingEventRepo{},
		jobTemplatePhase: &mockJobTemplatePhaseRepo{},
	}
	uc, revRepo, rli := buildMilestoneUseCase(t, mocks)
	req := basicReq("sub-m3")
	req.BillingEventId = stringPtrTest("be-stray")
	if _, err := uc.Execute(context.Background(), req); err == nil {
		t.Fatal("expected milestone_not_applicable error, got nil")
	}
	if revRepo.created != nil {
		t.Error("expected no Revenue created on milestone_not_applicable")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected no line items, got %d", len(rli.created))
	}
}

// MILESTONE-4: non-existent billing_event_id → reject.
func TestRecognize_Milestone_UnknownEventID_Rejects(t *testing.T) {
	mocks := &milestoneMocks{
		recognizeMocks: &recognizeMocks{
			subscription: activeSubscription("sub-m4", "pp-m4", "client-m4"),
			pricePlan:    milestonePricePlan("pp-m4"),
		},
		billingEvent:     &mockBillingEventRepo{events: map[string]*billingeventpb.BillingEvent{}},
		jobTemplatePhase: &mockJobTemplatePhaseRepo{},
	}
	uc, revRepo, _ := buildMilestoneUseCase(t, mocks)
	req := basicReq("sub-m4")
	req.BillingEventId = stringPtrTest("be-missing")
	if _, err := uc.Execute(context.Background(), req); err == nil {
		t.Fatal("expected billing_event_not_found error, got nil")
	}
	if revRepo.created != nil {
		t.Error("expected no Revenue created when event missing")
	}
}

// MILESTONE-5: idempotency — second call with same event_id returns
// conflicting_revenue_id.
func TestRecognize_Milestone_Idempotency_ReturnsConflict(t *testing.T) {
	jtpID := "jtp-m5"
	evID := "be-005"
	mocks := &milestoneMocks{
		recognizeMocks: &recognizeMocks{
			subscription: activeSubscription("sub-m5", "pp-m5", "client-m5"),
			pricePlan:    milestonePricePlan("pp-m5"),
			productPricePlans: []*productpriceplanpb.ProductPricePlan{
				{
					Id:                 "ppp-m5",
					PricePlanId:        "pp-m5",
					BillingAmount:      15000000,
					BillingCurrency:    "PHP",
					JobTemplatePhaseId: stringPtrTest(jtpID),
				},
			},
			priorRevenues: []*revenuepb.Revenue{
				{
					Id:             "rev-existing-milestone",
					Status:         "draft",
					BillingEventId: stringPtrTest(evID),
				},
			},
		},
		billingEvent: &mockBillingEventRepo{
			events: map[string]*billingeventpb.BillingEvent{evID: readyEvent(evID, "sub-m5", jtpID, 15000000)},
		},
		jobTemplatePhase: &mockJobTemplatePhaseRepo{
			phases: map[string]*jobtemplatephasepb.JobTemplatePhase{
				jtpID: {Id: jtpID, BillingAmount: int64Ptr(15000000)},
			},
		},
	}
	uc, revRepo, rli := buildMilestoneUseCase(t, mocks)
	req := basicReq("sub-m5")
	req.BillingEventId = stringPtrTest(evID)
	resp, err := uc.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("expected idempotency error, got nil")
	}
	if resp == nil || resp.GetConflictingRevenueId() != "rev-existing-milestone" {
		t.Errorf("expected conflicting_revenue_id=rev-existing-milestone, got %v", resp.GetConflictingRevenueId())
	}
	if revRepo.created != nil {
		t.Error("expected no second Revenue insert on conflict")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected no line items written on conflict, got %d", len(rli.created))
	}
}

// MILESTONE-6: over-billing rejection — sum of committed events > template_total.
func TestRecognize_Milestone_OverBilling_Rejects(t *testing.T) {
	jtpID := "jtp-m6"
	subID := "sub-m6"
	evID := "be-006"
	otherEvID := "be-other"
	templateAmount := int64(20000000)
	otherCommitted := int64(15000000)
	thisAttempt := int64(10000000)
	mocks := &milestoneMocks{
		recognizeMocks: &recognizeMocks{
			subscription: activeSubscription(subID, "pp-m6", "client-m6"),
			pricePlan:    milestonePricePlan("pp-m6"),
			productPricePlans: []*productpriceplanpb.ProductPricePlan{
				{
					Id:                 "ppp-m6",
					PricePlanId:        "pp-m6",
					BillingAmount:      thisAttempt,
					BillingCurrency:    "PHP",
					JobTemplatePhaseId: stringPtrTest(jtpID),
				},
			},
		},
		billingEvent: &mockBillingEventRepo{
			events: map[string]*billingeventpb.BillingEvent{
				evID: readyEvent(evID, subID, jtpID, thisAttempt),
			},
			bySubscript: map[string][]*billingeventpb.BillingEvent{
				subID: {
					readyEvent(evID, subID, jtpID, thisAttempt),
					{
						Id:                 otherEvID,
						SubscriptionId:     subID,
						BillableAmount:     otherCommitted,
						BillingCurrency:    "PHP",
						JobTemplatePhaseId: stringPtrTest(jtpID),
						Status:             billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_BILLED,
					},
				},
			},
		},
		jobTemplatePhase: &mockJobTemplatePhaseRepo{
			phases: map[string]*jobtemplatephasepb.JobTemplatePhase{
				jtpID: {Id: jtpID, BillingAmount: int64Ptr(templateAmount)},
			},
		},
	}
	uc, revRepo, rli := buildMilestoneUseCase(t, mocks)
	req := basicReq(subID)
	req.BillingEventId = stringPtrTest(evID)
	if _, err := uc.Execute(context.Background(), req); err == nil {
		t.Fatal("expected over_billing_rejected error, got nil")
	}
	if revRepo.created != nil {
		t.Error("expected no Revenue created on over-billing reject")
	}
	if len(rli.created) != 0 {
		t.Errorf("expected no line items, got %d", len(rli.created))
	}
}

func int32Ptr(v int32) *int32 { return &v }
func int64Ptr(v int64) *int64 { return &v }
