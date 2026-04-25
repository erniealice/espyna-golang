package revenue

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
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

	priorRevenues   []*revenuepb.Revenue
	priorLineItems  map[string][]*revenuelineitempb.RevenueLineItem // keyed by revenue_id

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
	priorRevenues   []*revenuepb.Revenue
	created         *revenuepb.Revenue
	createErr       error
	failOnCreate    bool
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
		AuthorizationService: ports.NewNoOpAuthorizationService(),
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
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

// stringPtrTest returns &s — local helper to avoid importing the package's
// stringPtrLocal across test+source boundaries.
func stringPtrTest(s string) *string {
	return &s
}
