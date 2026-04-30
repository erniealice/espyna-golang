package subscription

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ---- BillingEvent stub (AD_HOC × PER_OCCURRENCE) ----

type stubBillingEventRepo struct {
	billingeventpb.UnimplementedBillingEventDomainServiceServer
	created []*billingeventpb.BillingEvent
}

func (r *stubBillingEventRepo) CreateBillingEvent(_ context.Context, req *billingeventpb.CreateBillingEventRequest) (*billingeventpb.CreateBillingEventResponse, error) {
	r.created = append(r.created, req.Data)
	return &billingeventpb.CreateBillingEventResponse{Data: []*billingeventpb.BillingEvent{req.Data}, Success: true}, nil
}

// ============================================================================
// Stub repositories for the cyclic-instance use case.
//
// These extend the patterns in materialize_jobs_for_subscription_test.go but
// add a ListJobs stub so the use case can find existing engagement / cycle
// Jobs (idempotency, cycle index counting, "first ever call" detection).
// ============================================================================

// stubInstanceJobRepo extends stubJobRepo behaviour with ListJobs support.
type stubInstanceJobRepo struct {
	jobpb.UnimplementedJobDomainServiceServer
	created []*jobpb.Job
	// preExisting rows are returned by ListJobs *before* anything is created.
	// Tests use this to model a subscription that already has an engagement
	// shell or some cycle Jobs.
	preExisting []*jobpb.Job
	// failOnCreate optionally aborts after N successful creates.
	failOnCreate int
}

func (r *stubInstanceJobRepo) CreateJob(_ context.Context, req *jobpb.CreateJobRequest) (*jobpb.CreateJobResponse, error) {
	if r.failOnCreate > 0 && len(r.created) >= r.failOnCreate {
		return nil, errors.New("simulated CreateJob failure")
	}
	r.created = append(r.created, req.Data)
	return &jobpb.CreateJobResponse{Data: []*jobpb.Job{req.Data}, Success: true}, nil
}

func (r *stubInstanceJobRepo) ListJobs(_ context.Context, req *jobpb.ListJobsRequest) (*jobpb.ListJobsResponse, error) {
	// Filter by origin_id if the test passed one.
	wantOriginID := ""
	if req != nil && req.GetFilters() != nil {
		for _, f := range req.GetFilters().GetFilters() {
			if f.GetField() == "origin_id" {
				if sf := f.GetStringFilter(); sf != nil {
					wantOriginID = sf.GetValue()
				}
			}
		}
	}
	all := append([]*jobpb.Job{}, r.preExisting...)
	all = append(all, r.created...)
	out := all
	if wantOriginID != "" {
		out = nil
		for _, j := range all {
			if j.GetOriginId() == wantOriginID {
				out = append(out, j)
			}
		}
	}
	return &jobpb.ListJobsResponse{Data: out, Success: true}, nil
}

// ---- Fixture builder ----

type instFixture struct {
	uc       *MaterializeInstanceJobsForSubscriptionUseCase
	jobs     *stubInstanceJobRepo
	phases   *stubJobPhaseRepo
	tasks    *stubJobTaskRepo
	events   *stubBillingEventRepo
	subRepo  *stubSubscriptionRepo
}

type instFixtureOpts struct {
	subActive            bool
	subDateTimeStart     time.Time
	subName              string
	billingKind          priceplanpb.BillingKind
	amountBasis          priceplanpb.AmountBasis
	billingAmount        int64
	billingCycleValue    int32
	billingCycleUnit     string
	visitsPerCycle       int32
	planJobTemplateID    string
	// AD_HOC × TOTAL_PACKAGE knobs (codex MAJ-1).
	entitledOccurrences         int32
	entitledOccurrencesOverride int32
	// AD_HOC × PER_OCCURRENCE: omit BillingEvent repo to exercise the
	// repo-required error path.
	omitBillingEventRepo bool
	templates            map[string]*jobtemplatepb.JobTemplate
	phasesByTpl          map[string][]*jobtemplatephasepb.JobTemplatePhase
	tasksByPhase         map[string][]*jobtemplatetaskpb.JobTemplateTask
	relations            []*jobtemplaterelationpb.JobTemplateRelation
	preExistingJobs      []*jobpb.Job
}

func newInstFixture(t *testing.T, opts instFixtureOpts) *instFixture {
	t.Helper()
	subID := "sub-1"
	planID := "plan-1"
	pricePlanID := "pp-1"

	if opts.subDateTimeStart.IsZero() {
		opts.subDateTimeStart = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	}
	subActive := true
	if !opts.subActive && opts.subActive == false {
		subActive = opts.subActive
		// We want default true — opts.subActive defaults to false so check explicitly.
	}
	// Re-default: if caller did not set it, use true.
	subActive = true
	if opts.subActive == false && hasFieldExplicitlyFalse(opts) {
		subActive = false
	}

	subName := opts.subName
	if subName == "" {
		subName = "TestSub"
	}

	subRow := &subscriptionpb.Subscription{
		Id:            subID,
		Active:        subActive,
		ClientId:      "client-1",
		PricePlanId:   pricePlanID,
		Name:          subName,
		DateTimeStart: timestamppb.New(opts.subDateTimeStart),
	}
	if opts.entitledOccurrencesOverride > 0 {
		v := opts.entitledOccurrencesOverride
		subRow.EntitledOccurrencesOverride = &v
	}
	subRepo := &stubSubscriptionRepo{rows: map[string]*subscriptionpb.Subscription{subID: subRow}}

	cycleVal := opts.billingCycleValue
	cycleUnit := opts.billingCycleUnit
	pp := &priceplanpb.PricePlan{
		Id:              pricePlanID,
		Active:          true,
		PlanId:          planID,
		BillingKind:     opts.billingKind,
		AmountBasis:     opts.amountBasis,
		BillingAmount:   opts.billingAmount,
		BillingCurrency: "PHP",
	}
	if cycleVal > 0 {
		v := cycleVal
		pp.BillingCycleValue = &v
	}
	if cycleUnit != "" {
		pp.BillingCycleUnit = &cycleUnit
	}
	if opts.entitledOccurrences > 0 {
		v := opts.entitledOccurrences
		pp.EntitledOccurrences = &v
	}
	ppRepo := &stubPricePlanRepo{rows: map[string]*priceplanpb.PricePlan{pricePlanID: pp}}

	plan := &planpb.Plan{}
	if opts.planJobTemplateID != "" {
		v := opts.planJobTemplateID
		plan.JobTemplateId = &v
	}
	if opts.visitsPerCycle > 0 {
		v := opts.visitsPerCycle
		plan.VisitsPerCycle = &v
	}
	id := planID
	plan.Id = &id
	planRepo := &stubPlanRepo{rows: map[string]*planpb.Plan{planID: plan}}

	tplRepo := &stubJobTemplateRepo{rows: opts.templates}
	if tplRepo.rows == nil {
		tplRepo.rows = map[string]*jobtemplatepb.JobTemplate{}
	}
	phaseRepo := &stubJobTemplatePhaseRepo{byTemplate: opts.phasesByTpl}
	if phaseRepo.byTemplate == nil {
		phaseRepo.byTemplate = map[string][]*jobtemplatephasepb.JobTemplatePhase{}
	}
	taskRepo := &stubJobTemplateTaskRepo{byPhase: opts.tasksByPhase}
	if taskRepo.byPhase == nil {
		taskRepo.byPhase = map[string][]*jobtemplatetaskpb.JobTemplateTask{}
	}

	var relRepo *stubJobTemplateRelationRepo
	byParent := map[string][]*jobtemplaterelationpb.JobTemplateRelation{}
	for _, rel := range opts.relations {
		byParent[rel.GetParentTemplateId()] = append(byParent[rel.GetParentTemplateId()], rel)
	}
	relRepo = &stubJobTemplateRelationRepo{byParent: byParent}

	jobRepo := &stubInstanceJobRepo{preExisting: opts.preExistingJobs}
	jobPhaseRepo := &stubJobPhaseRepo{}
	jobTaskRepo := &stubJobTaskRepo{}
	eventRepo := &stubBillingEventRepo{}

	repos := MaterializeInstanceJobsForSubscriptionRepositories{
		Subscription:        subRepo,
		PricePlan:           ppRepo,
		Plan:                planRepo,
		JobTemplate:         tplRepo,
		JobTemplatePhase:    phaseRepo,
		JobTemplateTask:     taskRepo,
		JobTemplateRelation: relRepo,
		Job:                 jobRepo,
		JobPhase:            jobPhaseRepo,
		JobTask:             jobTaskRepo,
	}
	if !opts.omitBillingEventRepo {
		repos.BillingEvent = eventRepo
	}
	uc := NewMaterializeInstanceJobsForSubscriptionUseCase(
		repos,
		MaterializeInstanceJobsForSubscriptionServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TransactionService:   stubTxService{},
			TranslationService:   ports.NewNoOpTranslationService(),
			IDService:            ports.NewNoOpIDService(),
		},
	)
	return &instFixture{uc: uc, jobs: jobRepo, phases: jobPhaseRepo, tasks: jobTaskRepo, events: eventRepo, subRepo: subRepo}
}

// hasFieldExplicitlyFalse is a tiny helper so callers can distinguish "default
// (zero value, treat as true)" from "explicitly set to false". In practice the
// inactive-subscription test passes a custom subscription record below.
// This stub keeps the fixture path uniform.
func hasFieldExplicitlyFalse(opts instFixtureOpts) bool {
	// Heuristic: only a test that explicitly sets cycle data with the inactive
	// flag relies on this. We treat explicit false plus any other field set as
	// "wants inactive".
	return opts.billingKind != priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED &&
		!opts.subActive
}

// ---- Test cases (cyclic-subscription-jobs plan §11.1, 15 cases) ----

// Case 1: spawns one cycle with 1 visit_per_cycle (canonical).
func TestMaterializeInstanceJobs_Case1_OneCycleSingleVisit(t *testing.T) {
	rootID := "tpl-cleaning"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Cleaning Visit", true)},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SkippedReason != "" {
		t.Fatalf("expected no skip, got %q", resp.SkippedReason)
	}
	if got := len(resp.SpawnedCycles); got != 1 {
		t.Fatalf("want 1 cycle, got %d", got)
	}
	cycle := resp.SpawnedCycles[0]
	if cycle.CycleIndex != 1 {
		t.Errorf("cycle_index want 1, got %d", cycle.CycleIndex)
	}
	if got := len(cycle.Jobs); got != 1 {
		t.Errorf("want 1 cycle Job (visits_per_cycle=1), got %d", got)
	}
	if cycle.CyclePeriodStart != "2026-05-01" {
		t.Errorf("period_start want 2026-05-01, got %s", cycle.CyclePeriodStart)
	}
	if cycle.CyclePeriodEnd != "2026-05-31" {
		t.Errorf("period_end want 2026-05-31, got %s", cycle.CyclePeriodEnd)
	}
	// Engagement Job auto-created.
	if resp.EngagementJob == nil {
		t.Fatalf("engagement job missing")
	}
	if !resp.EngagementWasNewlyCreated {
		t.Errorf("engagement should be newly created")
	}
}

// Case 2: spawns multi-visit cycle (visits_per_cycle=2).
func TestMaterializeInstanceJobs_Case2_MultiVisitCycle(t *testing.T) {
	rootID := "tpl-cleaning"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    2,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Cleaning Visit", true)},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.SpawnedCycles); got != 1 {
		t.Fatalf("want 1 billing-cycle group, got %d", got)
	}
	cycle := resp.SpawnedCycles[0]
	if got := len(cycle.Jobs); got != 2 {
		t.Errorf("want 2 sub-cycle Jobs (visits_per_cycle=2), got %d", got)
	}
	// cycle_index should be monotone — first sub gets 1, second gets 2.
	if cycle.Jobs[0].GetCycleIndex() != 1 {
		t.Errorf("first sub cycle_index want 1, got %d", cycle.Jobs[0].GetCycleIndex())
	}
	if cycle.Jobs[1].GetCycleIndex() != 2 {
		t.Errorf("second sub cycle_index want 2, got %d", cycle.Jobs[1].GetCycleIndex())
	}
}

// Case 3: idempotency — re-running for the same period returns existing cycle
// without creating duplicates.
func TestMaterializeInstanceJobs_Case3_IdempotencySamePeriod(t *testing.T) {
	rootID := "tpl-cleaning"
	// Pre-existing engagement + one cycle Job for 2026-05-01.
	engagementID := "eng-1"
	cycleID := "cyc-1"
	subID := "sub-1"
	parentID := engagementID
	cycleIdx := int32(1)
	startStr := "2026-05-01"
	endStr := "2026-05-31"
	preExisting := []*jobpb.Job{
		{
			Id: engagementID, OriginType: enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
			OriginId: stringPtr(subID), Active: true,
		},
		{
			Id: cycleID, OriginType: enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
			OriginId: stringPtr(subID), Active: true,
			ParentJobId: &parentID, CycleIndex: &cycleIdx,
			CyclePeriodStart: &startStr, CyclePeriodEnd: &endStr,
		},
	}
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Cleaning Visit", true)},
		preExistingJobs:   preExisting,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   subID,
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SkippedReason != InstanceSkipReasonNoPendingCycles {
		t.Errorf("want skip=%q, got %q", InstanceSkipReasonNoPendingCycles, resp.SkippedReason)
	}
	// No new Job rows created (engagement existed; cycle existed).
	if got := len(f.jobs.created); got != 0 {
		t.Errorf("idempotency must not create new Jobs, got %d created", got)
	}
}

// Case 4: ONCE_AT_ENGAGEMENT_START fires only on cycle_index=0 (first-ever call).
func TestMaterializeInstanceJobs_Case4_OnceAtStartFiresOnFirstCall(t *testing.T) {
	rootID := "tpl-cleaning"
	onbID := "tpl-cleaning-onb"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID: makeTemplate(rootID, "Cleaning Visit", true),
			onbID:  makeTemplate(onbID, "Onboarding", true),
		},
		relations: []*jobtemplaterelationpb.JobTemplateRelation{
			makeOnceAtStartRelation(rootID, onbID, 1),
		},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.OnceAtStartJobs); got != 1 {
		t.Errorf("want 1 onboarding job, got %d", got)
	}
	if onb := resp.OnceAtStartJobs[0]; onb.GetCycleIndex() != 0 {
		t.Errorf("onboarding job cycle_index must be 0/NULL; got %d", onb.GetCycleIndex())
	}
}

// Case 5: PER_VISIT (default SUB_TEMPLATE) does NOT fire from this use case
// — composition with materialize-jobs handles it. Cyclic spawn must not pull
// in SUB_TEMPLATE relations as cycle Jobs.
func TestMaterializeInstanceJobs_Case5_SubTemplateRelationIgnored(t *testing.T) {
	rootID := "tpl-cleaning"
	subTplID := "tpl-cleaning-checklist"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID:   makeTemplate(rootID, "Cleaning Visit", true),
			subTplID: makeTemplate(subTplID, "Checklist", true),
		},
		relations: []*jobtemplaterelationpb.JobTemplateRelation{
			makeRelation(rootID, subTplID, 1), // SUB_TEMPLATE — default relation_type
		},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.OnceAtStartJobs); got != 0 {
		t.Errorf("SUB_TEMPLATE must not spawn as once-at-start; got %d", got)
	}
	if got := len(resp.SpawnedCycles); got != 1 || len(resp.SpawnedCycles[0].Jobs) != 1 {
		t.Errorf("expected single cycle with single root Job; got cycles=%d", got)
	}
}

// Case 6: Mixed relation types — only ONCE_AT_ENGAGEMENT_START spawns.
func TestMaterializeInstanceJobs_Case6_MixedRelationTypes(t *testing.T) {
	rootID := "tpl-cleaning"
	onbID := "tpl-onb"
	subTplID := "tpl-checklist"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID:   makeTemplate(rootID, "Cleaning Visit", true),
			onbID:    makeTemplate(onbID, "Onboarding", true),
			subTplID: makeTemplate(subTplID, "Checklist", true),
		},
		relations: []*jobtemplaterelationpb.JobTemplateRelation{
			makeOnceAtStartRelation(rootID, onbID, 1),
			makeRelation(rootID, subTplID, 2), // SUB_TEMPLATE — ignored here
		},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.OnceAtStartJobs); got != 1 {
		t.Errorf("want exactly 1 onboarding (only ONCE_AT_ENGAGEMENT_START), got %d", got)
	}
}

// Case 7: Eligibility false for non-cyclic price plan (ONE_TIME).
func TestMaterializeInstanceJobs_Case7_NonCyclicSkip(t *testing.T) {
	rootID := "tpl-once"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Setup", true)},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SkippedReason != InstanceSkipReasonNonCyclicPlan {
		t.Errorf("want skip=%q, got %q", InstanceSkipReasonNonCyclicPlan, resp.SkippedReason)
	}
	if len(f.jobs.created) != 0 {
		t.Errorf("non-cyclic must not create Jobs, got %d", len(f.jobs.created))
	}
}

// Case 8: Cycle period boundaries — month, week, quarter all compute correctly.
func TestMaterializeInstanceJobs_Case8_CyclePeriodBoundaries(t *testing.T) {
	cases := []struct {
		name      string
		unit      string
		value     int32
		start     string
		wantEnd   string
	}{
		{name: "monthly", unit: "month", value: 1, start: "2026-05-01", wantEnd: "2026-05-31"},
		{name: "weekly", unit: "week", value: 1, start: "2026-05-01", wantEnd: "2026-05-07"},
		{name: "quarterly", unit: "quarter", value: 1, start: "2026-05-01", wantEnd: "2026-07-31"},
		{name: "yearly", unit: "year", value: 1, start: "2026-05-01", wantEnd: "2027-04-30"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rootID := "tpl-x"
			f := newInstFixture(t, instFixtureOpts{
				subActive:         true,
				billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
				billingCycleValue: tc.value,
				billingCycleUnit:  tc.unit,
				visitsPerCycle:    1,
				planJobTemplateID: rootID,
				templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "X", true)},
			})
			resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
				SubscriptionId:   "sub-1",
				CyclePeriodStart: tc.start,
			})
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if resp.SpawnedCycles[0].CyclePeriodEnd != tc.wantEnd {
				t.Errorf("end want %s, got %s", tc.wantEnd, resp.SpawnedCycles[0].CyclePeriodEnd)
			}
		})
	}
}

// Case 9: Backfill mode — multiple historical cycles, all spawned in one call.
func TestMaterializeInstanceJobs_Case9_BackfillHistoricalCycles(t *testing.T) {
	rootID := "tpl-cleaning"
	// Subscription started 6 months ago; nothing exists yet. Use a date well
	// in the past so today is past it regardless of when the test runs.
	subStart := time.Now().AddDate(0, -5, 0).UTC().Truncate(24 * time.Hour)
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		subDateTimeStart:  subStart,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Cleaning Visit", true)},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId: "sub-1",
		Backfill:       true,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// Should spawn at least 5 cycles (could be 6 depending on exact day).
	if got := len(resp.SpawnedCycles); got < 5 {
		t.Errorf("backfill 6mo: want >=5 cycles, got %d", got)
	}
	// cycle indices must be monotone starting at 1.
	for i, c := range resp.SpawnedCycles {
		want := int32(i + 1)
		if c.CycleIndex != want {
			t.Errorf("cycle %d index want %d, got %d", i, want, c.CycleIndex)
		}
	}
}

// Case 10: Cycle index monotonicity — second call appends, doesn't reset.
func TestMaterializeInstanceJobs_Case10_CycleIndexMonotone(t *testing.T) {
	rootID := "tpl-cleaning"
	// Pre-existing engagement + cycle 1.
	engagementID := "eng-1"
	subID := "sub-1"
	parentID := engagementID
	cycleIdx := int32(1)
	startStr := "2026-05-01"
	endStr := "2026-05-31"
	preExisting := []*jobpb.Job{
		{Id: engagementID, OriginType: enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION, OriginId: stringPtr(subID), Active: true},
		{Id: "cyc-1", OriginType: enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION, OriginId: stringPtr(subID), Active: true,
			ParentJobId: &parentID, CycleIndex: &cycleIdx, CyclePeriodStart: &startStr, CyclePeriodEnd: &endStr},
	}
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "X", true)},
		preExistingJobs:   preExisting,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   subID,
		CyclePeriodStart: "2026-06-01",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if resp.SpawnedCycles[0].CycleIndex != 2 {
		t.Errorf("want cycle_index=2 (after pre-existing 1), got %d", resp.SpawnedCycles[0].CycleIndex)
	}
}

// Case 11: parent_job_id correctly set on every cycle Job.
func TestMaterializeInstanceJobs_Case11_ParentJobIdSet(t *testing.T) {
	rootID := "tpl-cleaning"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    2,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "X", true)},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	engID := resp.EngagementJob.GetId()
	for _, j := range resp.SpawnedCycles[0].Jobs {
		if j.GetParentJobId() != engID {
			t.Errorf("cycle Job parent_job_id want %s, got %s", engID, j.GetParentJobId())
		}
		if j.GetOriginType() != enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION {
			t.Errorf("origin_type must be SUBSCRIPTION; got %v", j.GetOriginType())
		}
		if j.GetBillingRuleType() != enumspb.BillingRuleType_BILLING_RULE_TYPE_NON_BILLABLE {
			t.Errorf("cycle Job billing_rule_type must be NON_BILLABLE; got %v", j.GetBillingRuleType())
		}
	}
}

// Case 12: Subscription must exist.
func TestMaterializeInstanceJobs_Case12_SubscriptionMustExist(t *testing.T) {
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
	})
	_, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-does-not-exist",
		CyclePeriodStart: "2026-05-01",
	})
	if err == nil {
		t.Fatalf("expected error for missing subscription")
	}
}

// Case 13: Subscription must be active.
func TestMaterializeInstanceJobs_Case13_SubscriptionMustBeActive(t *testing.T) {
	rootID := "tpl-x"
	subRepo := &stubSubscriptionRepo{rows: map[string]*subscriptionpb.Subscription{
		"sub-1": {
			Id: "sub-1", Active: false, ClientId: "client-1", PricePlanId: "pp-1",
			DateTimeStart: timestamppb.New(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)),
		},
	}}
	cycleVal := int32(1)
	cycleUnit := "month"
	pp := &priceplanpb.PricePlan{
		Id: "pp-1", Active: true, PlanId: "plan-1",
		BillingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		BillingCurrency:   "PHP",
		BillingCycleValue: &cycleVal, BillingCycleUnit: &cycleUnit,
	}
	ppRepo := &stubPricePlanRepo{rows: map[string]*priceplanpb.PricePlan{"pp-1": pp}}
	planID := "plan-1"
	v := rootID
	plan := &planpb.Plan{Id: &planID, JobTemplateId: &v}
	planRepo := &stubPlanRepo{rows: map[string]*planpb.Plan{planID: plan}}
	tplRepo := &stubJobTemplateRepo{rows: map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "X", true)}}
	uc := NewMaterializeInstanceJobsForSubscriptionUseCase(
		MaterializeInstanceJobsForSubscriptionRepositories{
			Subscription:        subRepo,
			PricePlan:           ppRepo,
			Plan:                planRepo,
			JobTemplate:         tplRepo,
			JobTemplatePhase:    &stubJobTemplatePhaseRepo{byTemplate: map[string][]*jobtemplatephasepb.JobTemplatePhase{}},
			JobTemplateTask:     &stubJobTemplateTaskRepo{byPhase: map[string][]*jobtemplatetaskpb.JobTemplateTask{}},
			JobTemplateRelation: &stubJobTemplateRelationRepo{byParent: map[string][]*jobtemplaterelationpb.JobTemplateRelation{}},
			Job:                 &stubInstanceJobRepo{},
			JobPhase:            &stubJobPhaseRepo{},
			JobTask:             &stubJobTaskRepo{},
		},
		MaterializeInstanceJobsForSubscriptionServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TransactionService:   stubTxService{},
			TranslationService:   ports.NewNoOpTranslationService(),
			IDService:            ports.NewNoOpIDService(),
		},
	)
	_, err := uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err == nil {
		t.Fatalf("expected error for inactive subscription")
	}
	if !strings.Contains(err.Error(), "inactive") {
		t.Errorf("expected 'inactive' in error message, got %q", err.Error())
	}
}

// Case 14: cycle_period_start/end strings are properly formatted (ISO 8601).
func TestMaterializeInstanceJobs_Case14_PeriodFormatISO8601(t *testing.T) {
	rootID := "tpl-x"
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "month",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "X", true)},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		CyclePeriodStart: "2026-05-01",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	cycle := resp.SpawnedCycles[0]
	// Format check: YYYY-MM-DD, length 10, two dashes.
	if len(cycle.CyclePeriodStart) != 10 || strings.Count(cycle.CyclePeriodStart, "-") != 2 {
		t.Errorf("period_start not ISO 8601: %s", cycle.CyclePeriodStart)
	}
	if len(cycle.CyclePeriodEnd) != 10 || strings.Count(cycle.CyclePeriodEnd, "-") != 2 {
		t.Errorf("period_end not ISO 8601: %s", cycle.CyclePeriodEnd)
	}
	// And the cycle Job carries them.
	if cycle.Jobs[0].GetCyclePeriodStart() != cycle.CyclePeriodStart {
		t.Errorf("Job's cycle_period_start mismatch")
	}
	if cycle.Jobs[0].GetCyclePeriodEnd() != cycle.CyclePeriodEnd {
		t.Errorf("Job's cycle_period_end mismatch")
	}
}

// Case 15: backfill cap (24 cycles per request) is enforced.
func TestMaterializeInstanceJobs_Case15_BackfillCap(t *testing.T) {
	rootID := "tpl-x"
	// Subscription started 5 years ago; weekly cadence → ~260 cycles.
	subStart := time.Now().AddDate(-5, 0, 0).UTC().Truncate(24 * time.Hour)
	f := newInstFixture(t, instFixtureOpts{
		subActive:         true,
		subDateTimeStart:  subStart,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		billingCycleValue: 1, billingCycleUnit: "week",
		visitsPerCycle:    1,
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "X", true)},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId: "sub-1",
		Backfill:       true,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := int32(len(resp.SpawnedCycles)); got != MaxBackfillCycles {
		t.Errorf("backfill must cap at %d, got %d cycles", MaxBackfillCycles, got)
	}
	if resp.BackfillCappedAt != MaxBackfillCycles {
		t.Errorf("BackfillCappedAt want %d, got %d", MaxBackfillCycles, resp.BackfillCappedAt)
	}
}

// ---- Composition tests with MaterializeJobsForSubscription (plan §11.2) ----

// C1: Cyclic Plan + Subscription.Create — spawns engagement-shell ONLY,
// no cycle Jobs, no phases on the engagement.
func TestMaterializeJobs_C1_CyclicSpawnsEngagementShellOnly(t *testing.T) {
	rootID := "tpl-cleaning"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Cleaning Visit", true)},
		phasesByTpl: map[string][]*jobtemplatephasepb.JobTemplatePhase{
			rootID: {makePhase("p1", rootID, "Visit", 1, "")},
		},
	})
	// Override the PricePlan repo to set BillingCycleValue (newFixture builds
	// a PricePlan without cycle metadata; cyclic detection needs cycle_value
	// or RECURRING kind).
	resp, err := f.uc.Execute(context.Background(), MaterializeJobsForSubscriptionRequest{
		SubscriptionId: "sub-1",
		SpawnJobs:      true,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := len(resp.SpawnedJobs); got != 1 {
		t.Fatalf("want 1 spawned Job (engagement shell only), got %d", got)
	}
	eng := resp.SpawnedJobs[0]
	if eng.GetJobTemplateId() != "" {
		t.Errorf("engagement shell must NOT carry job_template_id; got %s", eng.GetJobTemplateId())
	}
	if eng.GetParentJobId() != "" {
		t.Errorf("engagement must have no parent")
	}
	if eng.GetStatus() != enumspb.JobStatus_JOB_STATUS_ACTIVE {
		t.Errorf("engagement status want ACTIVE, got %v", eng.GetStatus())
	}
	if eng.GetBillingRuleType() != enumspb.BillingRuleType_BILLING_RULE_TYPE_NON_BILLABLE {
		t.Errorf("engagement billing_rule_type want NON_BILLABLE, got %v", eng.GetBillingRuleType())
	}
	// No phases — the cyclic branch skips them.
	if got := len(f.phs.created); got != 0 {
		t.Errorf("cyclic engagement must have NO phases on the shell; got %d", got)
	}
}

// C2: Non-cyclic Plan creates full job graph (existing behavior). Canary for
// regression — phase4-subscription specs 08-15 depend on this path.
func TestMaterializeJobs_C2_NonCyclicUnaffected(t *testing.T) {
	rootID := "tpl-tower-audit"
	tplPhases := []*jobtemplatephasepb.JobTemplatePhase{
		makePhase("p1", rootID, "Plan", 1, ""),
		makePhase("p2", rootID, "Build", 2, "p1"),
	}
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Tower Audit", true)},
		phasesByTpl:       map[string][]*jobtemplatephasepb.JobTemplatePhase{rootID: tplPhases},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeJobsForSubscriptionRequest{
		SubscriptionId: "sub-1",
		SpawnJobs:      true,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := len(resp.SpawnedJobs); got != 1 {
		t.Fatalf("non-cyclic: want 1 lifetime engagement Job, got %d", got)
	}
	eng := resp.SpawnedJobs[0]
	if eng.GetJobTemplateId() != rootID {
		t.Errorf("non-cyclic engagement must carry job_template_id; got %s", eng.GetJobTemplateId())
	}
	if got := len(f.phs.created); got != 2 {
		t.Errorf("non-cyclic engagement must have phases (2); got %d", got)
	}
}

// C3: Cyclic Plan with ONCE_AT_ENGAGEMENT_START template — fires once at
// engagement start (via materialize-jobs for the cyclic branch).
func TestMaterializeJobs_C3_CyclicWithOnceAtStart(t *testing.T) {
	rootID := "tpl-cleaning"
	onbID := "tpl-onb"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID: makeTemplate(rootID, "Cleaning Visit", true),
			onbID:  makeTemplate(onbID, "Onboarding", true),
		},
		relations: []*jobtemplaterelationpb.JobTemplateRelation{
			makeOnceAtStartRelation(rootID, onbID, 1),
		},
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeJobsForSubscriptionRequest{
		SubscriptionId: "sub-1",
		SpawnJobs:      true,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// Cyclic branch in MaterializeJobsForSubscription spawns engagement +
	// onboarding (ONCE_AT_ENGAGEMENT_START fires alongside the engagement
	// shell). It does NOT spawn cycle Jobs — those come from
	// MaterializeInstanceJobsForSubscription.
	if got := len(resp.SpawnedJobs); got != 2 {
		t.Fatalf("want 2 jobs (engagement + onboarding), got %d", got)
	}
	eng := resp.SpawnedJobs[0]
	onb := resp.SpawnedJobs[1]
	if eng.GetParentJobId() != "" {
		t.Errorf("engagement must have no parent")
	}
	if onb.GetParentJobId() != eng.GetId() {
		t.Errorf("onboarding must be parented to engagement; got %s", onb.GetParentJobId())
	}
	// Onboarding must have cycle_index = 0/NULL (it's not a cycle).
	if onb.GetCycleIndex() != 0 {
		t.Errorf("onboarding cycle_index must be 0/NULL; got %d", onb.GetCycleIndex())
	}
}

// ---- helpers ----

func makeOnceAtStartRelation(parent, child string, seq int32) *jobtemplaterelationpb.JobTemplateRelation {
	return &jobtemplaterelationpb.JobTemplateRelation{
		Id: parent + "->" + child + "@once", ParentTemplateId: parent, ChildTemplateId: child,
		SequenceOrder: seq, Active: true,
		RelationType: jobtemplaterelationpb.JobTemplateRelationType_JOB_TEMPLATE_RELATION_TYPE_ONCE_AT_ENGAGEMENT_START,
	}
}

func stringPtr(s string) *string { return &s }

// ---- Sanity: predicate ----

func TestEligibleForInstanceSpawn(t *testing.T) {
	cases := []struct {
		name string
		pp   *priceplanpb.PricePlan
		want bool
	}{
		{"recurring", &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING}, true},
		{"contract_with_cycle", &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT, BillingCycleValue: func() *int32 { v := int32(1); return &v }()}, true},
		{"contract_without_cycle", &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT}, false},
		{"one_time", &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME}, false},
		{"milestone", &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_MILESTONE}, false},
		{"ad_hoc_total_package", &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_AD_HOC, AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE}, true},
		{"ad_hoc_per_occurrence", &priceplanpb.PricePlan{BillingKind: priceplanpb.BillingKind_BILLING_KIND_AD_HOC, AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE}, true},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := eligibleForInstanceSpawn(tc.pp); got != tc.want {
				t.Errorf("eligibleForInstanceSpawn want %v, got %v", tc.want, got)
			}
		})
	}
}

// ============================================================================
// AD_HOC test cases (ad-hoc-subscription-billing plan §10.1)
// ============================================================================

// Common fixture for AD_HOC tests.
func adHocFixture(t *testing.T, opts instFixtureOpts) *instFixture {
	t.Helper()
	rootID := "tpl-aircon"
	if opts.planJobTemplateID == "" {
		opts.planJobTemplateID = rootID
	}
	if opts.templates == nil {
		opts.templates = map[string]*jobtemplatepb.JobTemplate{
			rootID: {Id: rootID, Active: true, Name: "Aircon Service"},
		}
	}
	if opts.subActive == false && opts.billingKind == priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED {
		opts.subActive = true
		opts.billingKind = priceplanpb.BillingKind_BILLING_KIND_AD_HOC
	}
	if opts.billingAmount == 0 {
		opts.billingAmount = 2_500_00 // ₱2,500.00 per visit
	}
	return newInstFixture(t, opts)
}

// AdHoc-1: AD_HOC × TOTAL_PACKAGE, fresh sub, request 1 visit → 1 usage Job,
// ordinal=1, no BillingEvent.
func TestMaterializeInstanceJobs_AdHoc1_PoolFreshSpawnsOneVisit(t *testing.T) {
	f := adHocFixture(t, instFixtureOpts{
		subActive:           true,
		billingKind:         priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis:         priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		entitledOccurrences: 5,
		billingAmount:       25_000_00,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		UsageRequestDate: "2026-09-12",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.SkippedReason != "" {
		t.Fatalf("unexpected skip: %q", resp.SkippedReason)
	}
	if len(resp.SpawnedCycles) != 1 || len(resp.SpawnedCycles[0].Jobs) != 1 {
		t.Fatalf("expected 1 spawned usage; got %+v", resp.SpawnedCycles)
	}
	job := resp.SpawnedCycles[0].Jobs[0]
	if job.GetUsageOrdinal() != 1 {
		t.Errorf("usage_ordinal want 1, got %d", job.GetUsageOrdinal())
	}
	if job.GetUsageRequestDate() != "2026-09-12" {
		t.Errorf("usage_request_date want 2026-09-12, got %q", job.GetUsageRequestDate())
	}
	if got, want := job.GetCyclePeriodStart(), "2026-09-12#0001"; got != want {
		t.Errorf("composite cycle_period_start want %q, got %q", want, got)
	}
	if len(f.events.created) != 0 {
		t.Errorf("TOTAL_PACKAGE must NOT spawn BillingEvent; got %d", len(f.events.created))
	}
}

// AdHoc-2: AD_HOC × PER_OCCURRENCE, fresh sub → 1 usage Job + 1 BillingEvent
// (status=UNSPECIFIED, trigger=UNSPECIFIED, billable_amount = pricePlan.amount).
func TestMaterializeInstanceJobs_AdHoc2_PerCallSpawnsBillingEvent(t *testing.T) {
	f := adHocFixture(t, instFixtureOpts{
		subActive:     true,
		billingKind:   priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis:   priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE,
		billingAmount: 2_500_00,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		UsageRequestDate: "2026-09-12",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.SkippedReason != "" {
		t.Fatalf("unexpected skip: %q", resp.SkippedReason)
	}
	if len(f.events.created) != 1 {
		t.Fatalf("PER_OCCURRENCE must spawn 1 BillingEvent; got %d", len(f.events.created))
	}
	ev := f.events.created[0]
	if ev.GetSubscriptionId() != "sub-1" {
		t.Errorf("event.subscription_id want sub-1, got %q", ev.GetSubscriptionId())
	}
	if ev.GetJobId() != resp.SpawnedCycles[0].Jobs[0].GetId() {
		t.Errorf("event.job_id must point at the usage Job")
	}
	if ev.GetStatus() != billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_UNSPECIFIED {
		t.Errorf("event.status want UNSPECIFIED, got %v", ev.GetStatus())
	}
	if ev.GetTrigger() != billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_UNSPECIFIED {
		t.Errorf("event.trigger want UNSPECIFIED, got %v", ev.GetTrigger())
	}
	if ev.GetBillableAmount() != 2_500_00 {
		t.Errorf("event.billable_amount want 250000, got %d", ev.GetBillableAmount())
	}
}

// AdHoc-3: AD_HOC × TOTAL_PACKAGE, 5 visits used, request 6th → entitlement_exhausted.
func TestMaterializeInstanceJobs_AdHoc3_PoolEntitlementExhausted(t *testing.T) {
	// Pre-seed engagement + 5 prior usage Jobs.
	pre := []*jobpb.Job{
		mkEngagementShell("eng-1", "sub-1"),
	}
	for i := int32(1); i <= 5; i++ {
		pre = append(pre, mkUsageJob("usg-"+rune2s(i), "sub-1", "eng-1", i, "2026-08-0"+rune2s(i)))
	}
	f := adHocFixture(t, instFixtureOpts{
		subActive:           true,
		billingKind:         priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis:         priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		entitledOccurrences: 5,
		preExistingJobs:     pre,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		UsageRequestDate: "2026-09-12",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.SkippedReason != InstanceSkipReasonEntitlementExhausted {
		t.Fatalf("want entitlement_exhausted, got %q", resp.SkippedReason)
	}
	if len(resp.SpawnedCycles) != 0 {
		t.Errorf("blocked spawn must NOT add cycle entries; got %+v", resp.SpawnedCycles)
	}
}

// AdHoc-4: Subscription override beats PricePlan template (codex MAJ-1).
// Template entitled=3, override=10, used=5 → next spawn allowed.
func TestMaterializeInstanceJobs_AdHoc4_PoolUsesOverrideOverTemplate(t *testing.T) {
	pre := []*jobpb.Job{mkEngagementShell("eng-1", "sub-1")}
	for i := int32(1); i <= 5; i++ {
		pre = append(pre, mkUsageJob("usg-"+rune2s(i), "sub-1", "eng-1", i, "2026-08-0"+rune2s(i)))
	}
	f := adHocFixture(t, instFixtureOpts{
		subActive:                   true,
		billingKind:                 priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis:                 priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		entitledOccurrences:         3,  // template default
		entitledOccurrencesOverride: 10, // per-subscription extension
		preExistingJobs:             pre,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		UsageRequestDate: "2026-09-12",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.SkippedReason != "" {
		t.Fatalf("override should permit spawn; got skip %q", resp.SkippedReason)
	}
	if len(resp.SpawnedCycles) != 1 || resp.SpawnedCycles[0].Jobs[0].GetUsageOrdinal() != 6 {
		t.Errorf("expected ordinal=6 spawn under override; got %+v", resp.SpawnedCycles)
	}
}

// AdHoc-5: AD_HOC without Plan.job_template_id skips with ad_hoc_no_template.
// (Validator at PricePlan save should also block this combo per ad-hoc plan
// §6: pool_no_template + pay_per_call_no_template — defensive coverage here.)
func TestMaterializeInstanceJobs_AdHoc5_NoTemplateSkips(t *testing.T) {
	// Build directly without adHocFixture so we can leave job_template_id nil.
	f := newInstFixture(t, instFixtureOpts{
		subActive:           true,
		billingKind:         priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis:         priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		entitledOccurrences: 5,
		// planJobTemplateID intentionally empty — fixture leaves Plan.job_template_id nil.
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		UsageRequestDate: "2026-09-12",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.SkippedReason != InstanceSkipReasonAdHocNoTemplate {
		t.Errorf("want ad_hoc_no_template; got %q", resp.SkippedReason)
	}
	if len(resp.SpawnedCycles) != 0 {
		t.Errorf("no usage Job should be created when template is missing")
	}
}

// AdHoc-6: ordinal increments across consecutive spawns on the same date.
// 3 usages on the same date → ordinals 1, 2, 3; composite keys are distinct
// even though usage_request_date repeats.
func TestMaterializeInstanceJobs_AdHoc6_OrdinalIncrementsSameDay(t *testing.T) {
	f := adHocFixture(t, instFixtureOpts{
		subActive:     true,
		billingKind:   priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis:   priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE,
		billingAmount: 2_500_00,
	})
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		resp, err := f.uc.Execute(ctx, MaterializeInstanceJobsForSubscriptionRequest{
			SubscriptionId:   "sub-1",
			UsageRequestDate: "2026-09-12",
		})
		if err != nil {
			t.Fatalf("Execute #%d: %v", i, err)
		}
		if resp.SkippedReason != "" {
			t.Fatalf("Execute #%d skipped: %q", i, resp.SkippedReason)
		}
		got := resp.SpawnedCycles[0].Jobs[0]
		if int(got.GetUsageOrdinal()) != i {
			t.Errorf("call %d: usage_ordinal want %d, got %d", i, i, got.GetUsageOrdinal())
		}
	}
	// All 3 composite keys must be distinct so the partial unique index doesn't fire.
	keys := map[string]bool{}
	for _, j := range f.jobs.created {
		if j.GetParentJobId() == "" {
			continue
		}
		keys[j.GetCyclePeriodStart()] = true
	}
	if len(keys) != 3 {
		t.Errorf("expected 3 distinct composite keys, got %v", keys)
	}
	if len(f.events.created) != 3 {
		t.Errorf("expected 3 BillingEvents (one per PER_OCCURRENCE usage), got %d", len(f.events.created))
	}
}

// AdHoc-7: AD_HOC × PER_OCCURRENCE without BillingEvent repo errors before
// any Job is created.
func TestMaterializeInstanceJobs_AdHoc7_PerCallNoBillingEventRepoErrors(t *testing.T) {
	f := adHocFixture(t, instFixtureOpts{
		subActive:            true,
		billingKind:          priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis:          priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE,
		billingAmount:        2_500_00,
		omitBillingEventRepo: true,
	})
	_, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   "sub-1",
		UsageRequestDate: "2026-09-12",
	})
	if err == nil {
		t.Fatalf("expected error when BillingEvent repo is nil for PER_OCCURRENCE")
	}
	if !strings.Contains(err.Error(), "BillingEvent") {
		t.Errorf("expected BillingEvent error, got %v", err)
	}
	if len(f.jobs.created) != 0 {
		t.Errorf("repo-required check must run before any Job is created; got %d", len(f.jobs.created))
	}
}

// AdHoc-8: AD_HOC × invalid amount_basis (PER_CYCLE) → ad_hoc_invalid_basis skip.
func TestMaterializeInstanceJobs_AdHoc8_InvalidBasisSkips(t *testing.T) {
	f := adHocFixture(t, instFixtureOpts{
		subActive:   true,
		billingKind: priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
		amountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE, // illegal
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId: "sub-1",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.SkippedReason != InstanceSkipReasonAdHocInvalidBasis {
		t.Errorf("want ad_hoc_invalid_basis; got %q", resp.SkippedReason)
	}
}

// ---- AD_HOC fixture helpers ----

func mkEngagementShell(jobID, subID string) *jobpb.Job {
	originID := subID
	clientID := "client-1"
	return &jobpb.Job{
		Id:         jobID,
		OriginType: enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:   &originID,
		ClientId:   &clientID,
		Active:     true,
	}
}

func mkUsageJob(jobID, subID, parentID string, ordinal int32, requestDate string) *jobpb.Job {
	originID := subID
	parent := parentID
	clientID := "client-1"
	idx := ordinal
	rd := requestDate
	composite := requestDate + "#" + zeroPad4(ordinal)
	return &jobpb.Job{
		Id:                 jobID,
		OriginType:         enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:           &originID,
		ClientId:           &clientID,
		ParentJobId:        &parent,
		CycleIndex:         &idx,
		CyclePeriodStart:   &composite,
		UsageRequestDate:   &rd,
		UsageOrdinal:       &idx,
		Active:             true,
	}
}

func zeroPad4(n int32) string {
	s := ""
	for _, d := range []int32{1000, 100, 10, 1} {
		v := (n / d) % 10
		s += string(rune('0' + v))
	}
	return s
}

// rune2s converts a single-digit int32 to its ASCII representation. Test-only.
func rune2s(n int32) string { return string(rune('0' + (n % 10))) }

// ---- Compile-time assert: usage of common imports avoid "unused" errors. ----

var _ = commonpb.StringFilter{}
