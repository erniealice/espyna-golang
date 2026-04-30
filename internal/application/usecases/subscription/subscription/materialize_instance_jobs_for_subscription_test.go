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
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

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
}

type instFixtureOpts struct {
	subActive            bool
	subDateTimeStart     time.Time
	subName              string
	billingKind          priceplanpb.BillingKind
	billingCycleValue    int32
	billingCycleUnit     string
	visitsPerCycle       int32
	planJobTemplateID    string
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

	subRepo := &stubSubscriptionRepo{rows: map[string]*subscriptionpb.Subscription{
		subID: {
			Id:            subID,
			Active:        subActive,
			ClientId:      "client-1",
			PricePlanId:   pricePlanID,
			Name:          subName,
			DateTimeStart: timestamppb.New(opts.subDateTimeStart),
		},
	}}

	cycleVal := opts.billingCycleValue
	cycleUnit := opts.billingCycleUnit
	pp := &priceplanpb.PricePlan{
		Id:              pricePlanID,
		Active:          true,
		PlanId:          planID,
		BillingKind:     opts.billingKind,
		BillingCurrency: "PHP",
	}
	if cycleVal > 0 {
		v := cycleVal
		pp.BillingCycleValue = &v
	}
	if cycleUnit != "" {
		pp.BillingCycleUnit = &cycleUnit
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

	uc := NewMaterializeInstanceJobsForSubscriptionUseCase(
		MaterializeInstanceJobsForSubscriptionRepositories{
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
		},
		MaterializeInstanceJobsForSubscriptionServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TransactionService:   stubTxService{},
			TranslationService:   ports.NewNoOpTranslationService(),
			IDService:            ports.NewNoOpIDService(),
		},
	)
	return &instFixture{uc: uc, jobs: jobRepo, phases: jobPhaseRepo, tasks: jobTaskRepo}
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

// ---- Compile-time assert: usage of common imports avoid "unused" errors. ----

var _ = commonpb.StringFilter{}
