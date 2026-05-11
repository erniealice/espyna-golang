package job

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ---- Stub repos ----

type stubMBEJobRepo struct {
	jobpb.UnimplementedJobDomainServiceServer
	rows map[string]*jobpb.Job
}

func (r *stubMBEJobRepo) ReadJob(_ context.Context, req *jobpb.ReadJobRequest) (*jobpb.ReadJobResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New("nil")
	}
	j, ok := r.rows[req.Data.Id]
	if !ok {
		return &jobpb.ReadJobResponse{}, errors.New("not found")
	}
	return &jobpb.ReadJobResponse{Data: []*jobpb.Job{j}, Success: true}, nil
}

type stubMBEJobTemplatePhaseRepo struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	byTemplate map[string][]*jobtemplatephasepb.JobTemplatePhase
}

func (r *stubMBEJobTemplatePhaseRepo) ListByJobTemplate(_ context.Context, req *jobtemplatephasepb.ListByJobTemplateRequest) (*jobtemplatephasepb.ListByJobTemplateResponse, error) {
	return &jobtemplatephasepb.ListByJobTemplateResponse{
		JobTemplatePhases: r.byTemplate[req.JobTemplateId],
		Success:           true,
	}, nil
}

type stubMBEJobPhaseRepo struct {
	jobphasepb.UnimplementedJobPhaseDomainServiceServer
	byJob map[string][]*jobphasepb.JobPhase
}

func (r *stubMBEJobPhaseRepo) ListByJob(_ context.Context, req *jobphasepb.ListJobPhasesByJobRequest) (*jobphasepb.ListJobPhasesByJobResponse, error) {
	return &jobphasepb.ListJobPhasesByJobResponse{
		JobPhases: r.byJob[req.JobId],
		Success:   true,
	}, nil
}

type stubMBEBillingEventRepo struct {
	billingeventpb.UnimplementedBillingEventDomainServiceServer
	created        []*billingeventpb.BillingEvent
	bySubscription map[string][]*billingeventpb.BillingEvent
	failOnCreate   bool
}

func (r *stubMBEBillingEventRepo) CreateBillingEvent(_ context.Context, req *billingeventpb.CreateBillingEventRequest) (*billingeventpb.CreateBillingEventResponse, error) {
	if r.failOnCreate {
		return nil, errors.New("simulated CreateBillingEvent failure")
	}
	r.created = append(r.created, req.Data)
	return &billingeventpb.CreateBillingEventResponse{
		Data:    []*billingeventpb.BillingEvent{req.Data},
		Success: true,
	}, nil
}

func (r *stubMBEBillingEventRepo) ListBySubscription(_ context.Context, req *billingeventpb.ListBillingEventsBySubscriptionRequest) (*billingeventpb.ListBillingEventsBySubscriptionResponse, error) {
	return &billingeventpb.ListBillingEventsBySubscriptionResponse{
		BillingEvents: r.bySubscription[req.SubscriptionId],
		Success:       true,
	}, nil
}

type stubMBESubscriptionRepo struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	rows map[string]*subscriptionpb.Subscription
}

func (r *stubMBESubscriptionRepo) ReadSubscription(_ context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New("nil")
	}
	s, ok := r.rows[req.Data.Id]
	if !ok {
		return &subscriptionpb.ReadSubscriptionResponse{}, errors.New("not found")
	}
	return &subscriptionpb.ReadSubscriptionResponse{Data: []*subscriptionpb.Subscription{s}, Success: true}, nil
}

type stubMBEPricePlanRepo struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	rows map[string]*priceplanpb.PricePlan
}

func (r *stubMBEPricePlanRepo) ReadPricePlan(_ context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	pp, ok := r.rows[req.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &priceplanpb.ReadPricePlanResponse{Data: []*priceplanpb.PricePlan{pp}, Success: true}, nil
}

type stubMBEProductPricePlanRepo struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	rows []*productpriceplanpb.ProductPricePlan
}

func (r *stubMBEProductPricePlanRepo) ListProductPricePlans(_ context.Context, _ *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	return &productpriceplanpb.ListProductPricePlansResponse{
		Data:    r.rows,
		Success: true,
	}, nil
}

// ---- Fixture builder ----

type mbeFixture struct {
	uc      *MaterializeBillingEventsForJobUseCase
	billing *stubMBEBillingEventRepo
}

type mbeOpts struct {
	job              *jobpb.Job
	templatePhases   []*jobtemplatephasepb.JobTemplatePhase
	jobPhases        []*jobphasepb.JobPhase
	subscription     *subscriptionpb.Subscription
	pricePlan        *priceplanpb.PricePlan
	productPricePlan []*productpriceplanpb.ProductPricePlan
	existingEvents   []*billingeventpb.BillingEvent
	failOnCreate     bool

	// Toggle nil-repo paths.
	noSubscriptionRepo bool
	noPricePlanRepo    bool
	noPPPRepo          bool
}

func newMBEFixture(t *testing.T, opts mbeOpts) *mbeFixture {
	t.Helper()
	jobs := &stubMBEJobRepo{rows: map[string]*jobpb.Job{}}
	if opts.job != nil {
		jobs.rows[opts.job.GetId()] = opts.job
	}
	tplPhases := &stubMBEJobTemplatePhaseRepo{byTemplate: map[string][]*jobtemplatephasepb.JobTemplatePhase{}}
	if opts.job != nil && opts.job.JobTemplateId != nil {
		tplPhases.byTemplate[opts.job.GetJobTemplateId()] = opts.templatePhases
	}
	jobPhases := &stubMBEJobPhaseRepo{byJob: map[string][]*jobphasepb.JobPhase{}}
	if opts.job != nil {
		jobPhases.byJob[opts.job.GetId()] = opts.jobPhases
	}
	billing := &stubMBEBillingEventRepo{
		bySubscription: map[string][]*billingeventpb.BillingEvent{},
		failOnCreate:   opts.failOnCreate,
	}
	if len(opts.existingEvents) > 0 && opts.subscription != nil {
		billing.bySubscription[opts.subscription.GetId()] = opts.existingEvents
	}

	repos := MaterializeBillingEventsForJobRepositories{
		Job:              jobs,
		JobTemplatePhase: tplPhases,
		JobPhase:         jobPhases,
		BillingEvent:     billing,
	}
	if !opts.noSubscriptionRepo && opts.subscription != nil {
		repos.Subscription = &stubMBESubscriptionRepo{
			rows: map[string]*subscriptionpb.Subscription{opts.subscription.GetId(): opts.subscription},
		}
	}
	if !opts.noPricePlanRepo && opts.pricePlan != nil {
		repos.PricePlan = &stubMBEPricePlanRepo{
			rows: map[string]*priceplanpb.PricePlan{opts.pricePlan.GetId(): opts.pricePlan},
		}
	}
	if !opts.noPPPRepo {
		repos.ProductPricePlan = &stubMBEProductPricePlanRepo{rows: opts.productPricePlan}
	}

	uc := NewMaterializeBillingEventsForJobUseCase(repos, MaterializeBillingEventsForJobServices{
		AuthorizationService: ports.NewNoOpAuthorizationService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	})
	return &mbeFixture{uc: uc, billing: billing}
}

// ---- Helpers ----

func ptrStr(s string) *string { return &s }

func milestoneJob(id, templateID, subID string) *jobpb.Job {
	originID := subID
	tplLocal := templateID
	return &jobpb.Job{
		Id:              id,
		JobTemplateId:   &tplLocal,
		OriginType:      enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:        &originID,
		BillingRuleType: enumspb.BillingRuleType_BILLING_RULE_TYPE_MILESTONE,
		Active:          true,
	}
}

func tplPhase(id, templateID, name string, order int32) *jobtemplatephasepb.JobTemplatePhase {
	return &jobtemplatephasepb.JobTemplatePhase{
		Id: id, JobTemplateId: templateID, Name: name, PhaseOrder: order, Active: true,
	}
}

func withTriggers(p *jobtemplatephasepb.JobTemplatePhase, triggers bool) *jobtemplatephasepb.JobTemplatePhase {
	v := triggers
	p.TriggersBilling = &v
	return p
}

func withPercent(p *jobtemplatephasepb.JobTemplatePhase, bps int32) *jobtemplatephasepb.JobTemplatePhase {
	v := bps
	p.BillingPercentBps = &v
	return p
}

func withFixed(p *jobtemplatephasepb.JobTemplatePhase, amount int64, currency string) *jobtemplatephasepb.JobTemplatePhase {
	a := amount
	c := currency
	p.BillingAmount = &a
	p.BillingCurrency = &c
	return p
}

// ---- Tests ----

// Bug 1 fix — percent precedence is honored when PricePlan is resolvable.
//
// JobTemplatePhase.billing_percent_bps=3000 (30%) + PricePlan.billing_amount=50000000 (₱500,000)
// → BillingEvent.billable_amount=15000000 (₱150,000).
func TestMBE_PercentPrecedence_HonoredViaPricePlan(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withPercent(withTriggers(tplPhase("p1", "tpl-1", "Design", 1), true), 3000),
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 50_000_000, BillingCurrency: "PHP", Active: true}

	f := newMBEFixture(t, mbeOpts{
		job:            job,
		templatePhases: phases,
		subscription:   sub,
		pricePlan:      pp,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Events); got != 1 {
		t.Fatalf("want 1 event, got %d", got)
	}
	if got := resp.Events[0].GetBillableAmount(); got != 15_000_000 {
		t.Errorf("want billable_amount=15000000 (30%% of 50000000), got %d", got)
	}
	if got := resp.Events[0].GetBillingCurrency(); got != "PHP" {
		t.Errorf("want currency=PHP (inherited from PricePlan), got %q", got)
	}
}

// Fixed billing_amount wins over percent when both are set on a single phase
// is rejected per plan §2.3. But when billing_amount is set ALONE, it wins
// over the PricePlan percent path.
func TestMBE_FixedAmountWinsOverPercent(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withFixed(withTriggers(tplPhase("p1", "tpl-1", "Setup", 1), true), 10_000_000, "USD"),
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 99_999_999, BillingCurrency: "PHP", Active: true}

	f := newMBEFixture(t, mbeOpts{
		job: job, templatePhases: phases, subscription: sub, pricePlan: pp,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Events); got != 1 {
		t.Fatalf("want 1 event, got %d", got)
	}
	if got := resp.Events[0].GetBillableAmount(); got != 10_000_000 {
		t.Errorf("want billable_amount=10000000 (fixed), got %d", got)
	}
	if got := resp.Events[0].GetBillingCurrency(); got != "USD" {
		t.Errorf("want currency=USD (fixed-amount carries its own), got %q", got)
	}
}

// PPP-sum branch — when neither fixed nor percent is set, sum the
// ProductPricePlan rows tagged with the template_phase_id.
func TestMBE_PPPSum_WhenNoFixedNorPercent(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withTriggers(tplPhase("p1", "tpl-1", "Design", 1), true),
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 0, BillingCurrency: "PHP", Active: true}
	tplPhaseID := "p1"
	ppps := []*productpriceplanpb.ProductPricePlan{
		{Id: "ppp-1", PricePlanId: "pp-1", BillingAmount: 6_000_000, JobTemplatePhaseId: &tplPhaseID},
		{Id: "ppp-2", PricePlanId: "pp-1", BillingAmount: 4_000_000, JobTemplatePhaseId: &tplPhaseID},
		// Different plan; must NOT contribute even when JobTemplatePhase id matches.
		{Id: "ppp-other", PricePlanId: "other-pp", BillingAmount: 99_999_999, JobTemplatePhaseId: &tplPhaseID},
	}

	f := newMBEFixture(t, mbeOpts{
		job: job, templatePhases: phases, subscription: sub, pricePlan: pp, productPricePlan: ppps,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Events); got != 1 {
		t.Fatalf("want 1 event, got %d", got)
	}
	if got := resp.Events[0].GetBillableAmount(); got != 10_000_000 {
		t.Errorf("want billable_amount=10000000 (sum of own-plan PPPs), got %d", got)
	}
}

// Persistence sanity — the use case actually writes BillingEvent rows via
// CreateBillingEvent and surfaces an error when the write fails. This is the
// regression guard for Phase G E2E spec 13.2 (jobs persist but events don't).
func TestMBE_PersistsEvents_WhenCreateSucceeds(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withPercent(withTriggers(tplPhase("p1", "tpl-1", "Design", 1), true), 3000),
		withPercent(withTriggers(tplPhase("p2", "tpl-1", "Build", 2), true), 5000),
		withPercent(withTriggers(tplPhase("p3", "tpl-1", "Commission", 3), true), 2000),
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 50_000_000, BillingCurrency: "PHP", Active: true}

	f := newMBEFixture(t, mbeOpts{job: job, templatePhases: phases, subscription: sub, pricePlan: pp})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(f.billing.created); got != 3 {
		t.Errorf("want 3 BillingEvent rows persisted, got %d", got)
	}
	if got := len(resp.Events); got != 3 {
		t.Errorf("want 3 events in response, got %d", got)
	}
	// Each event must have a non-empty subscription_id and a job_id back-pointer.
	for i, ev := range resp.Events {
		if ev.GetSubscriptionId() != "sub-1" {
			t.Errorf("event[%d] subscription_id=%q want sub-1", i, ev.GetSubscriptionId())
		}
		if ev.GetJobId() != "job-1" {
			t.Errorf("event[%d] job_id=%q want job-1", i, ev.GetJobId())
		}
		if ev.GetBillingCurrency() != "PHP" {
			t.Errorf("event[%d] currency=%q want PHP", i, ev.GetBillingCurrency())
		}
	}
}

// Persistence: when CreateBillingEvent fails, the use case surfaces the error
// (no silent swallow). Guards against the symptom in Bug 2.
func TestMBE_SurfacesCreateBillingEventError(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withPercent(withTriggers(tplPhase("p1", "tpl-1", "Design", 1), true), 3000),
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 50_000_000, BillingCurrency: "PHP", Active: true}

	f := newMBEFixture(t, mbeOpts{
		job: job, templatePhases: phases, subscription: sub, pricePlan: pp, failOnCreate: true,
	})
	if _, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"}); err == nil {
		t.Fatalf("want error from failed CreateBillingEvent, got nil")
	}
}

// Phases with triggers_billing=false are skipped — no event row created.
func TestMBE_SkipsNonTriggeringPhases(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withTriggers(tplPhase("p1", "tpl-1", "Internal", 1), false), // no event
		withPercent(withTriggers(tplPhase("p2", "tpl-1", "Bills", 2), true), 10000),
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 30_000_000, BillingCurrency: "PHP"}

	f := newMBEFixture(t, mbeOpts{job: job, templatePhases: phases, subscription: sub, pricePlan: pp})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Events); got != 1 {
		t.Errorf("want 1 event (only triggering phase), got %d", got)
	}
}

// Idempotency — re-running the use case after events already exist is a no-op
// for the existing phases. Guards against double-write on retry.
func TestMBE_Idempotent_SkipsExistingPhases(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withPercent(withTriggers(tplPhase("p1", "tpl-1", "Design", 1), true), 3000),
		withPercent(withTriggers(tplPhase("p2", "tpl-1", "Build", 2), true), 7000),
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 50_000_000, BillingCurrency: "PHP"}

	jobIDLocal := "job-1"
	tplPhaseIDLocal := "p1"
	existing := []*billingeventpb.BillingEvent{
		{Id: "ev-existing", SubscriptionId: "sub-1", JobId: &jobIDLocal, JobTemplatePhaseId: &tplPhaseIDLocal, Active: true},
	}

	f := newMBEFixture(t, mbeOpts{
		job: job, templatePhases: phases, subscription: sub, pricePlan: pp, existingEvents: existing,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Events); got != 1 {
		t.Errorf("want 1 new event (p1 skipped, p2 created), got %d", got)
	}
	if got := len(f.billing.created); got != 1 {
		t.Errorf("want 1 row written, got %d", got)
	}
}

// JobPhase mapping — when job_phase rows exist, BillingEvent.job_phase_id
// is back-filled from the matching template_phase_id.
func TestMBE_BackfillsJobPhaseId(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withPercent(withTriggers(tplPhase("tpl-p1", "tpl-1", "Design", 1), true), 10000),
	}
	jobPhases := []*jobphasepb.JobPhase{
		{Id: "jp-1", JobId: "job-1", TemplatePhaseId: ptrStr("tpl-p1"), Active: true},
	}
	sub := &subscriptionpb.Subscription{Id: "sub-1", PricePlanId: "pp-1", Active: true}
	pp := &priceplanpb.PricePlan{Id: "pp-1", BillingAmount: 1_000_000, BillingCurrency: "PHP"}

	f := newMBEFixture(t, mbeOpts{
		job: job, templatePhases: phases, jobPhases: jobPhases, subscription: sub, pricePlan: pp,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Events); got != 1 {
		t.Fatalf("want 1 event, got %d", got)
	}
	if resp.Events[0].JobPhaseId == nil || *resp.Events[0].JobPhaseId != "jp-1" {
		t.Errorf("want job_phase_id=jp-1, got %v", resp.Events[0].JobPhaseId)
	}
}

// Non-milestone Job — use case rejects clearly. Guards against accidental
// invocation outside the milestone branch.
func TestMBE_RejectsNonMilestoneJob(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	job.BillingRuleType = enumspb.BillingRuleType_BILLING_RULE_TYPE_UNSPECIFIED
	f := newMBEFixture(t, mbeOpts{job: job})
	if _, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{JobID: "job-1"}); err == nil {
		t.Fatalf("want error for non-milestone job, got nil")
	}
}

// Subscription resolution falls through gracefully when Subscription/PricePlan
// repos are nil — percent path is skipped (returns 0), PPP-sum still works.
// Documents the degraded contract for environments where the cross-domain
// repos didn't load.
func TestMBE_DegradesGracefully_WhenSubRepoNil(t *testing.T) {
	job := milestoneJob("job-1", "tpl-1", "sub-1")
	tplPhaseID := "p1"
	phases := []*jobtemplatephasepb.JobTemplatePhase{
		withPercent(withTriggers(tplPhase("p1", "tpl-1", "Design", 1), true), 3000),
	}
	// PPP-sum fallback contributes — even without the Subscription read,
	// the unfiltered PPP iteration matches by job_template_phase_id.
	ppps := []*productpriceplanpb.ProductPricePlan{
		{Id: "ppp-1", PricePlanId: "pp-1", BillingAmount: 7_500_000, JobTemplatePhaseId: &tplPhaseID},
	}
	f := newMBEFixture(t, mbeOpts{
		job:                job,
		templatePhases:     phases,
		productPricePlan:   ppps,
		noSubscriptionRepo: true,
		noPricePlanRepo:    true,
	})
	resp, err := f.uc.Execute(context.Background(), MaterializeBillingEventsForJobRequest{
		JobID: "job-1", SubscriptionID: "sub-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Events); got != 1 {
		t.Fatalf("want 1 event (PPP-sum fallback), got %d", got)
	}
	if got := resp.Events[0].GetBillableAmount(); got != 7_500_000 {
		t.Errorf("want billable_amount=7500000 from PPP-sum fallback, got %d", got)
	}
}
