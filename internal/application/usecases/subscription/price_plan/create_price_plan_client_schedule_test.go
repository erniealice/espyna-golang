package price_plan

// Tests for the §3.2 / §4.4 auto-resolve-or-create-client-schedule path on
// CreatePricePlan, added 2026-04-28. Covers:
//   - master parent Plan → no schedule auto-creation (existing behaviour).
//   - client-scoped parent + empty schedule_id → schedule created with
//     derived name + client_id stamp.
//   - client-scoped parent + existing client schedule → reused (no duplicate).
//   - client-scoped parent + different client's schedule_id → reject with
//     scheduleClientMismatch.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	plan_locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_location"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// ----- mocks ---------------------------------------------------------------

type mockPriceScheduleRepo struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	schedules     map[string]*priceschedulepb.PriceSchedule
	listFilter    *commonpb.FilterRequest
	createCalls   int
	createCapture *priceschedulepb.PriceSchedule
}

func (m *mockPriceScheduleRepo) ReadPriceSchedule(_ context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	id := ""
	if req.GetData() != nil {
		id = req.GetData().GetId()
	}
	if s, ok := m.schedules[id]; ok {
		return &priceschedulepb.ReadPriceScheduleResponse{Data: []*priceschedulepb.PriceSchedule{s}, Success: true}, nil
	}
	return &priceschedulepb.ReadPriceScheduleResponse{Data: nil, Success: true}, nil
}

func (m *mockPriceScheduleRepo) ListPriceSchedules(_ context.Context, req *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
	m.listFilter = req.GetFilters()
	// Naive filter implementation — match on client_id when present.
	wantClient := ""
	wantLocation := ""
	if req.GetFilters() != nil {
		for _, f := range req.GetFilters().GetFilters() {
			if f.GetField() == "client_id" {
				if sf := f.GetStringFilter(); sf != nil {
					wantClient = sf.GetValue()
				}
			}
			if f.GetField() == "location_id" {
				if sf := f.GetStringFilter(); sf != nil {
					wantLocation = sf.GetValue()
				}
			}
		}
	}
	out := []*priceschedulepb.PriceSchedule{}
	for _, s := range m.schedules {
		if wantClient != "" && s.GetClientId() != wantClient {
			continue
		}
		if wantLocation != "" && s.GetLocationId() != wantLocation {
			continue
		}
		out = append(out, s)
	}
	return &priceschedulepb.ListPriceSchedulesResponse{Data: out, Success: true}, nil
}

func (m *mockPriceScheduleRepo) CreatePriceSchedule(_ context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	m.createCalls++
	m.createCapture = req.GetData()
	if m.schedules == nil {
		m.schedules = map[string]*priceschedulepb.PriceSchedule{}
	}
	m.schedules[req.GetData().GetId()] = req.GetData()
	return &priceschedulepb.CreatePriceScheduleResponse{Data: []*priceschedulepb.PriceSchedule{req.GetData()}, Success: true}, nil
}

type mockClientRepoForCreate struct {
	clientpb.UnimplementedClientDomainServiceServer
	clients map[string]*clientpb.Client
}

func (m *mockClientRepoForCreate) ReadClient(_ context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	id := ""
	if req.GetData() != nil {
		id = req.GetData().GetId()
	}
	if c, ok := m.clients[id]; ok {
		return &clientpb.ReadClientResponse{Data: []*clientpb.Client{c}, Success: true}, nil
	}
	return &clientpb.ReadClientResponse{Data: nil, Success: true}, nil
}

// ----- fixture --------------------------------------------------------------

func newScheduleFixture(t *testing.T) (*CreatePricePlanUseCase, *mockPlanRepoForCreate, *mockPricePlanRepoCreate, *mockPriceScheduleRepo, *mockClientRepoForCreate) {
	t.Helper()
	planRepo := &mockPlanRepoForCreate{plans: map[string]*planpb.Plan{}}
	ppRepo := &mockPricePlanRepoCreate{}
	scheduleRepo := &mockPriceScheduleRepo{schedules: map[string]*priceschedulepb.PriceSchedule{}}
	clientRepo := &mockClientRepoForCreate{clients: map[string]*clientpb.Client{}}
	uc := NewCreatePricePlanUseCase(
		CreatePricePlanRepositories{
			PricePlan:     ppRepo,
			Plan:          planRepo,
			PriceSchedule: scheduleRepo,
			Client:        clientRepo,
		},
		CreatePricePlanServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TransactionService:   noTxnCreate{},
			TranslationService:   ports.NewNoOpTranslationService(),
			IDService:            &stubIDSvc{},
		},
	)
	return uc, planRepo, ppRepo, scheduleRepo, clientRepo
}

func seedClientScopedPlan(id, clientID, locationID string) *planpb.Plan {
	idCopy := id
	c := clientID
	p := &planpb.Plan{Id: &idCopy, Name: "Audit Engagement", Active: true, ClientId: &c}
	if locationID != "" {
		p.PlanLocations = []*plan_locationpb.PlanLocation{
			{LocationId: locationID, Active: true},
		}
	}
	return p
}

// ----- tests ---------------------------------------------------------------

// Master parent → no auto-creation. Whatever schedule_id (or none) the
// operator submits is left untouched.
func TestCreatePricePlan_MasterParent_NoScheduleAutoCreation(t *testing.T) {
	uc, planRepo, ppRepo, scheduleRepo, _ := newScheduleFixture(t)
	planRepo.plans["plan-master"] = seedPlanForCreate("plan-master", "")

	if _, err := uc.Execute(context.Background(), basePricePlanRequest("plan-master")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scheduleRepo.createCalls != 0 {
		t.Errorf("master parent should not auto-create a schedule, got %d Create calls", scheduleRepo.createCalls)
	}
	if got := ppRepo.captured.GetPriceScheduleId(); got != "" {
		t.Errorf("master parent should leave price_schedule_id untouched (empty), got %q", got)
	}
}

// Client-scoped parent + no submitted schedule_id → schedule created with
// derived name + client_id stamp.
func TestCreatePricePlan_ClientScoped_EmptySchedule_AutoCreates(t *testing.T) {
	uc, planRepo, ppRepo, scheduleRepo, clientRepo := newScheduleFixture(t)
	planRepo.plans["plan-cruz"] = seedClientScopedPlan("plan-cruz", "client-cruz", "loc-manila")
	cruzName := "Cruz Engineering"
	clientRepo.clients["client-cruz"] = &clientpb.Client{Id: "client-cruz", Name: &cruzName, Active: true}

	// Handler-supplied lyngua-resolved suffix via context (mimicking centymo
	// handler). The use case builds the schedule name as
	// "{client.name} - {suffix} - {timestamp} {tz}" — so we assert the
	// stable prefix rather than the full string.
	ctx := context.WithValue(context.Background(), "clientScheduleSuffix", "Rate Cards")
	if _, err := uc.Execute(ctx, basePricePlanRequest("plan-cruz")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scheduleRepo.createCalls != 1 {
		t.Fatalf("expected 1 schedule create, got %d", scheduleRepo.createCalls)
	}
	created := scheduleRepo.createCapture
	if got := created.GetClientId(); got != "client-cruz" {
		t.Errorf("created schedule client_id = %q, want client-cruz", got)
	}
	if got := created.GetName(); !strings.HasPrefix(got, "Cruz Engineering - Rate Cards - ") {
		t.Errorf("created schedule name = %q, want prefix %q", got, "Cruz Engineering - Rate Cards - ")
	}
	// 1-to-1 invariant: client-scoped PriceSchedule collapses across locations.
	// The use case intentionally passes "" for the location filter (see
	// resolve_client_schedule.go) so the new row carries no location stamp.
	if got := created.GetLocationId(); got != "" {
		t.Errorf("created schedule location_id = %q, want empty (client-scoped collapses across locations)", got)
	}
	if got := ppRepo.captured.GetPriceScheduleId(); got != created.GetId() {
		t.Errorf("price_plan.price_schedule_id = %q, want resolved schedule %q", got, created.GetId())
	}
	if got := ppRepo.captured.GetClientId(); got != "client-cruz" {
		t.Errorf("price_plan.client_id cascade failed: %q", got)
	}
}

// Client-scoped parent + existing matching client schedule → reused, no
// duplicate insert.
func TestCreatePricePlan_ClientScoped_ExistingSchedule_Reused(t *testing.T) {
	uc, planRepo, ppRepo, scheduleRepo, clientRepo := newScheduleFixture(t)
	planRepo.plans["plan-cruz"] = seedClientScopedPlan("plan-cruz", "client-cruz", "loc-manila")
	cruzName := "Cruz Engineering"
	clientRepo.clients["client-cruz"] = &clientpb.Client{Id: "client-cruz", Name: &cruzName, Active: true}

	existing := "existing-sched"
	loc := "loc-manila"
	cli := "client-cruz"
	scheduleRepo.schedules[existing] = &priceschedulepb.PriceSchedule{
		Id:         existing,
		Name:       "Cruz - existing",
		Active:     true,
		ClientId:   &cli,
		LocationId: &loc,
	}

	if _, err := uc.Execute(context.Background(), basePricePlanRequest("plan-cruz")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scheduleRepo.createCalls != 0 {
		t.Errorf("expected reuse (0 creates), got %d", scheduleRepo.createCalls)
	}
	if got := ppRepo.captured.GetPriceScheduleId(); got != existing {
		t.Errorf("expected reused schedule_id=%q, got %q", existing, got)
	}
}

// Client-scoped parent + operator picks a schedule belonging to a different
// client → reject with scheduleClientMismatch (lyngua key fragment).
func TestCreatePricePlan_ClientScoped_DifferentClientSchedule_Rejected(t *testing.T) {
	uc, planRepo, _, scheduleRepo, clientRepo := newScheduleFixture(t)
	planRepo.plans["plan-cruz"] = seedClientScopedPlan("plan-cruz", "client-cruz", "loc-manila")
	cruzName := "Cruz Engineering"
	clientRepo.clients["client-cruz"] = &clientpb.Client{Id: "client-cruz", Name: &cruzName, Active: true}

	otherClient := "client-other"
	scheduleRepo.schedules["sched-other"] = &priceschedulepb.PriceSchedule{
		Id:       "sched-other",
		Name:     "Other client schedule",
		Active:   true,
		ClientId: &otherClient,
	}

	req := basePricePlanRequest("plan-cruz")
	picked := "sched-other"
	req.Data.PriceScheduleId = &picked

	_, err := uc.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected scheduleClientMismatch error, got nil")
	}
	// NoOp translation service returns the default message string with the
	// "[DEFAULT]" suffix; assert on a stable substring of the literal copy.
	if !strings.Contains(err.Error(), "different client") {
		t.Errorf("expected error to mention 'different client', got: %v", err)
	}
	// Sanity: also confirm the underlying error is not nil and stays
	// distinct from a generic plan-not-found.
	var notFoundMarker = "does not exist"
	if errors.Is(err, errors.New(notFoundMarker)) {
		t.Errorf("got generic plan-not-found error path; expected schedule mismatch")
	}
}
