package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	planjobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/plan_job_template"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ---- Stub repos (handcrafted; mock_db build tag isolates this from prod) ----

type stubSubscriptionRepo struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	rows map[string]*subscriptionpb.Subscription
}

func (r *stubSubscriptionRepo) ReadSubscription(_ context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if req.Data == nil {
		return nil, errors.New("nil")
	}
	sub, ok := r.rows[req.Data.Id]
	if !ok {
		return &subscriptionpb.ReadSubscriptionResponse{}, errors.New("not found")
	}
	return &subscriptionpb.ReadSubscriptionResponse{Data: []*subscriptionpb.Subscription{sub}}, nil
}

type stubPricePlanRepo struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	rows map[string]*priceplanpb.PricePlan
}

func (r *stubPricePlanRepo) ReadPricePlan(_ context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	pp, ok := r.rows[req.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &priceplanpb.ReadPricePlanResponse{Data: []*priceplanpb.PricePlan{pp}}, nil
}

type stubPriceScheduleRepo struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	rows map[string]*priceschedulepb.PriceSchedule
}

func (r *stubPriceScheduleRepo) ReadPriceSchedule(_ context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if req.GetData() == nil {
		return nil, errors.New("nil")
	}
	s, ok := r.rows[req.GetData().GetId()]
	if !ok {
		return nil, errors.New("not found")
	}
	return &priceschedulepb.ReadPriceScheduleResponse{Data: []*priceschedulepb.PriceSchedule{s}}, nil
}

type stubPlanRepo struct {
	planpb.UnimplementedPlanDomainServiceServer
	rows map[string]*planpb.Plan
}

func (r *stubPlanRepo) ReadPlan(_ context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	id := ""
	if req.Data != nil && req.Data.Id != nil {
		id = *req.Data.Id
	}
	p, ok := r.rows[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &planpb.ReadPlanResponse{Data: []*planpb.Plan{p}}, nil
}

type stubJobTemplateRepo struct {
	jobtemplatepb.UnimplementedJobTemplateDomainServiceServer
	rows map[string]*jobtemplatepb.JobTemplate
}

func (r *stubJobTemplateRepo) ReadJobTemplate(_ context.Context, req *jobtemplatepb.ReadJobTemplateRequest) (*jobtemplatepb.ReadJobTemplateResponse, error) {
	tpl, ok := r.rows[req.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &jobtemplatepb.ReadJobTemplateResponse{Data: []*jobtemplatepb.JobTemplate{tpl}}, nil
}

type stubJobTemplatePhaseRepo struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	byTemplate map[string][]*jobtemplatephasepb.JobTemplatePhase
}

func (r *stubJobTemplatePhaseRepo) ListByJobTemplate(_ context.Context, req *jobtemplatephasepb.ListByJobTemplateRequest) (*jobtemplatephasepb.ListByJobTemplateResponse, error) {
	return &jobtemplatephasepb.ListByJobTemplateResponse{
		JobTemplatePhases: r.byTemplate[req.JobTemplateId],
		Success:           true,
	}, nil
}

type stubJobTemplateTaskRepo struct {
	jobtemplatetaskpb.UnimplementedJobTemplateTaskDomainServiceServer
	byPhase map[string][]*jobtemplatetaskpb.JobTemplateTask
}

func (r *stubJobTemplateTaskRepo) ListByPhase(_ context.Context, req *jobtemplatetaskpb.ListJobTemplateTasksByPhaseRequest) (*jobtemplatetaskpb.ListJobTemplateTasksByPhaseResponse, error) {
	return &jobtemplatetaskpb.ListJobTemplateTasksByPhaseResponse{
		JobTemplateTasks: r.byPhase[req.JobTemplatePhaseId],
		Success:          true,
	}, nil
}

type stubJobTemplateRelationRepo struct {
	jobtemplaterelationpb.UnimplementedJobTemplateRelationDomainServiceServer
	byParent map[string][]*jobtemplaterelationpb.JobTemplateRelation
}

type stubPlanJobTemplateRepo struct {
	planjobtemplatepb.UnimplementedPlanJobTemplateDomainServiceServer
	byPlan map[string][]*planjobtemplatepb.PlanJobTemplate
}

func (r *stubPlanJobTemplateRepo) ListPlanJobTemplatesByPlan(_ context.Context, req *planjobtemplatepb.ListPlanJobTemplatesByPlanRequest) (*planjobtemplatepb.ListPlanJobTemplatesByPlanResponse, error) {
	return &planjobtemplatepb.ListPlanJobTemplatesByPlanResponse{Success: true, PlanJobTemplates: r.byPlan[req.GetPlanId()]}, nil
}

func (r *stubJobTemplateRelationRepo) ListByParent(_ context.Context, req *jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest) (*jobtemplaterelationpb.ListJobTemplateRelationsByParentResponse, error) {
	return &jobtemplaterelationpb.ListJobTemplateRelationsByParentResponse{
		JobTemplateRelations: r.byParent[req.ParentTemplateId],
		Success:              true,
	}, nil
}

type stubJobRepo struct {
	jobpb.UnimplementedJobDomainServiceServer
	created    []*jobpb.Job
	failOnRoot bool
	failOnIdx  int
}

func (r *stubJobRepo) CreateJob(_ context.Context, req *jobpb.CreateJobRequest) (*jobpb.CreateJobResponse, error) {
	// failOnIdx=N → fail on the Nth invocation (1-indexed). Caller sets it
	// to model "Nth Job creation explodes" e.g., failOnIdx=2 → root succeeds,
	// child fails.
	if r.failOnIdx > 0 && len(r.created)+1 == r.failOnIdx {
		return nil, errors.New("simulated CreateJob failure")
	}
	r.created = append(r.created, req.Data)
	return &jobpb.CreateJobResponse{Data: []*jobpb.Job{req.Data}, Success: true}, nil
}

type stubJobPhaseRepo struct {
	jobphasepb.UnimplementedJobPhaseDomainServiceServer
	created []*jobphasepb.JobPhase
}

func (r *stubJobPhaseRepo) CreateJobPhase(_ context.Context, req *jobphasepb.CreateJobPhaseRequest) (*jobphasepb.CreateJobPhaseResponse, error) {
	r.created = append(r.created, req.Data)
	return &jobphasepb.CreateJobPhaseResponse{Data: []*jobphasepb.JobPhase{req.Data}, Success: true}, nil
}

type stubJobTaskRepo struct {
	jobtaskpb.UnimplementedJobTaskDomainServiceServer
	created []*jobtaskpb.JobTask
}

func (r *stubJobTaskRepo) CreateJobTask(_ context.Context, req *jobtaskpb.CreateJobTaskRequest) (*jobtaskpb.CreateJobTaskResponse, error) {
	r.created = append(r.created, req.Data)
	return &jobtaskpb.CreateJobTaskResponse{Data: []*jobtaskpb.JobTask{req.Data}, Success: true}, nil
}

// stubMaterializeBillingEventsForJob captures invocations.
type stubMaterializeBillingEventsForJob struct {
	calls []string // jobID
}

func (s *stubMaterializeBillingEventsForJob) Execute(_ context.Context, jobID, _ string) error {
	s.calls = append(s.calls, jobID)
	return nil
}

// ---- Inline tx stub (avoids the mock_db build-tag chain whose
// transitive deps include leapfor.xyz/copya/golang and aren't in go.work
// for this branch). NoOp{Auth,Translation,ID} services come from ports. ----

type stubTxService struct{}

func (stubTxService) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (stubTxService) SupportsTransactions() bool                 { return true }
func (stubTxService) IsTransactionActive(_ context.Context) bool { return false }

// ---- Fixture builder ----

type fixture struct {
	uc    *MaterializeJobsForSubscriptionUseCase
	jobs  *stubJobRepo
	phs   *stubJobPhaseRepo
	tasks *stubJobTaskRepo
	mbe   *stubMaterializeBillingEventsForJob
}

type fixtureOpts struct {
	planJobTemplateID string
	billingKind       priceplanpb.BillingKind
	relations         []*jobtemplaterelationpb.JobTemplateRelation
	templates         map[string]*jobtemplatepb.JobTemplate
	phasesByTpl       map[string][]*jobtemplatephasepb.JobTemplatePhase
	tasksByPhase      map[string][]*jobtemplatetaskpb.JobTemplateTask
	failOnNthCreate   int
	withMBE           bool
	withRelationRepo  bool
	composition       []*planjobtemplatepb.PlanJobTemplate
	// Closed-AY spawn guard (red-team HIGH #3). When priceScheduleID is set, the
	// price_plan points at it and a matching price_schedule row is wired.
	// scheduleInactive / scheduleEnded model the two blocking window states.
	priceScheduleID  string
	scheduleInactive bool
	scheduleEnded    bool
}

func newFixture(t *testing.T, opts fixtureOpts) *fixture {
	t.Helper()

	subID := "sub-1"
	planID := "plan-1"
	pricePlanID := "pp-1"

	subRepo := &stubSubscriptionRepo{rows: map[string]*subscriptionpb.Subscription{
		subID: {Id: subID, Active: true, ClientId: "client-1", PricePlanId: pricePlanID, Name: "TestSub"},
	}}
	pp := &priceplanpb.PricePlan{Id: pricePlanID, Active: true, PlanId: planID, BillingKind: opts.billingKind}
	var schedRepo priceschedulepb.PriceScheduleDomainServiceServer
	if opts.priceScheduleID != "" {
		sid := opts.priceScheduleID
		pp.PriceScheduleId = &sid
		sched := &priceschedulepb.PriceSchedule{Id: sid, Active: !opts.scheduleInactive}
		if opts.scheduleEnded {
			sched.DateTimeEnd = timestamppb.New(time.Now().UTC().Add(-24 * time.Hour))
		}
		schedRepo = &stubPriceScheduleRepo{rows: map[string]*priceschedulepb.PriceSchedule{sid: sched}}
	}
	ppRepo := &stubPricePlanRepo{rows: map[string]*priceplanpb.PricePlan{pricePlanID: pp}}
	plan := &planpb.Plan{}
	if opts.planJobTemplateID != "" {
		v := opts.planJobTemplateID
		plan.JobTemplateId = &v
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

	var relRepo jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	if opts.withRelationRepo || len(opts.relations) > 0 {
		byParent := map[string][]*jobtemplaterelationpb.JobTemplateRelation{}
		for _, rel := range opts.relations {
			byParent[rel.GetParentTemplateId()] = append(byParent[rel.GetParentTemplateId()], rel)
		}
		relRepo = &stubJobTemplateRelationRepo{byParent: byParent}
	}
	var compositionRepo planjobtemplatepb.PlanJobTemplateDomainServiceServer
	if len(opts.composition) > 0 {
		compositionRepo = &stubPlanJobTemplateRepo{byPlan: map[string][]*planjobtemplatepb.PlanJobTemplate{planID: opts.composition}}
	}

	jobRepo := &stubJobRepo{failOnIdx: opts.failOnNthCreate}
	jobPhaseRepo := &stubJobPhaseRepo{}
	jobTaskRepo := &stubJobTaskRepo{}

	var mbe MaterializeBillingEventsForJobInvoker
	mbeStub := &stubMaterializeBillingEventsForJob{}
	if opts.withMBE {
		mbe = mbeStub
	}

	uc := NewMaterializeJobsForSubscriptionUseCase(
		MaterializeJobsForSubscriptionRepositories{
			Subscription:        subRepo,
			PricePlan:           ppRepo,
			PriceSchedule:       schedRepo,
			Plan:                planRepo,
			JobTemplate:         tplRepo,
			JobTemplatePhase:    phaseRepo,
			JobTemplateTask:     taskRepo,
			JobTemplateRelation: relRepo,
			PlanJobTemplate:     compositionRepo,
			Job:                 jobRepo,
			JobPhase:            jobPhaseRepo,
			JobTask:             jobTaskRepo,
		},
		MaterializeJobsForSubscriptionServices{
			Authorizer:                     ports.NewNoOpAuthorizer(),
			ActionGatekeeper:               actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
			Transactor:                     stubTxService{},
			Translator:                     ports.NewNoOpTranslator(),
			IDGenerator:                    ports.NewNoOpIDGenerator(),
			MaterializeBillingEventsForJob: mbe,
		},
	)
	return &fixture{uc: uc, jobs: jobRepo, phs: jobPhaseRepo, tasks: jobTaskRepo, mbe: mbeStub}
}

// ---- Test helpers ----

func makePhase(id, tplID, name string, order int32, predecessor string) *jobtemplatephasepb.JobTemplatePhase {
	p := &jobtemplatephasepb.JobTemplatePhase{Id: id, JobTemplateId: tplID, Name: name, PhaseOrder: order, Active: true}
	if predecessor != "" {
		v := predecessor
		p.PredecessorTemplatePhaseId = &v
	}
	return p
}

func makeTask(id, phaseID, name string, step int32) *jobtemplatetaskpb.JobTemplateTask {
	return &jobtemplatetaskpb.JobTemplateTask{Id: id, JobTemplatePhaseId: phaseID, Name: name, StepOrder: step, Active: true}
}

func makeTemplate(id, name string, active bool) *jobtemplatepb.JobTemplate {
	return &jobtemplatepb.JobTemplate{Id: id, Name: name, Active: active}
}

func makeRelation(parent, child string, seq int32) *jobtemplaterelationpb.JobTemplateRelation {
	return &jobtemplaterelationpb.JobTemplateRelation{
		Id: parent + "->" + child, ParentTemplateId: parent, ChildTemplateId: child,
		SequenceOrder: seq, Active: true,
	}
}

// ---- Tests (covers plan §7.1 cases 1-13) ----

func TestMaterializeJobs_Case1_RootOnlyNoRelations(t *testing.T) {
	rootID := "tpl-root"
	tplPhases := []*jobtemplatephasepb.JobTemplatePhase{
		makePhase("p1", rootID, "Design", 1, ""),
		makePhase("p2", rootID, "Build", 2, "p1"),
	}
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Root", true)},
		phasesByTpl:       map[string][]*jobtemplatephasepb.JobTemplatePhase{rootID: tplPhases},
		tasksByPhase: map[string][]*jobtemplatetaskpb.JobTemplateTask{
			"p1": {makeTask("t1", "p1", "Survey", 1), makeTask("t2", "p1", "Sketch", 2)},
		},
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.SpawnedJobs); got != 1 {
		t.Errorf("want 1 spawned job, got %d", got)
	}
	if got := len(f.phs.created); got != 2 {
		t.Errorf("want 2 phases, got %d", got)
	}
	if got := len(f.tasks.created); got != 2 {
		t.Errorf("want 2 tasks, got %d", got)
	}
}

func TestMaterializeJobs_Case2_OneChildRelation(t *testing.T) {
	rootID, childID := "tpl-root", "tpl-child"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID:  makeTemplate(rootID, "Root", true),
			childID: makeTemplate(childID, "Child", true),
		},
		phasesByTpl: map[string][]*jobtemplatephasepb.JobTemplatePhase{
			rootID:  {makePhase("rp1", rootID, "Root1", 1, "")},
			childID: {makePhase("cp1", childID, "Child1", 1, "")},
		},
		relations: []*jobtemplaterelationpb.JobTemplateRelation{
			makeRelation(rootID, childID, 1),
		},
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.SpawnedJobs); got != 2 {
		t.Errorf("want 2 jobs (root+child), got %d", got)
	}
	// Root has nil parent_job_id; child points at root.
	if resp.SpawnedJobs[0].ParentJobId != nil {
		t.Errorf("root should have no parent")
	}
	if resp.SpawnedJobs[1].ParentJobId == nil || *resp.SpawnedJobs[1].ParentJobId != resp.SpawnedJobs[0].Id {
		t.Errorf("child should point at root")
	}
}

func TestMaterializeJobs_Case3_PlanJobTemplateIdNull(t *testing.T) {
	f := newFixture(t, fixtureOpts{planJobTemplateID: ""})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSkippedReason() != SkipReasonNoTemplateFound {
		t.Errorf("want %q, got %q", SkipReasonNoTemplateFound, resp.GetSkippedReason())
	}
	if len(resp.SpawnedJobs) != 0 {
		t.Errorf("want 0 jobs, got %d", len(resp.SpawnedJobs))
	}
}

func TestMaterializeJobs_Case4_TwoChildrenOrderedBySequence(t *testing.T) {
	rootID, c1, c2 := "tpl-root", "tpl-c1", "tpl-c2"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID: makeTemplate(rootID, "Root", true),
			c1:     makeTemplate(c1, "C1", true),
			c2:     makeTemplate(c2, "C2", true),
		},
		// Relations passed out-of-order; sort step inside use case must restore.
		relations: []*jobtemplaterelationpb.JobTemplateRelation{
			makeRelation(rootID, c2, 2),
			makeRelation(rootID, c1, 1),
		},
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.SpawnedJobs); got != 3 {
		t.Fatalf("want 3 jobs, got %d", got)
	}
	if *resp.SpawnedJobs[1].JobTemplateId != c1 {
		t.Errorf("want first child = c1, got %v", resp.SpawnedJobs[1].JobTemplateId)
	}
	if *resp.SpawnedJobs[2].JobTemplateId != c2 {
		t.Errorf("want second child = c2, got %v", resp.SpawnedJobs[2].JobTemplateId)
	}
}

func TestMaterializeJobs_Case5_MilestoneTriggersBillingEvents(t *testing.T) {
	rootID := "tpl-milestone"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_MILESTONE,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "M", true)},
		withMBE:           true,
	})
	_, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(f.mbe.calls); got != 1 {
		t.Errorf("want 1 MaterializeBillingEvents call, got %d", got)
	}
}

func TestMaterializeJobs_Case6_RecurringNoBillingEvents(t *testing.T) {
	rootID := "tpl-recurring"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		billingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "R", true)},
		withMBE:           true,
	})
	_, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(f.mbe.calls); got != 0 {
		t.Errorf("RECURRING should not invoke MBE, got %d calls", got)
	}
}

func TestMaterializeJobs_Case7_SpawnJobsFalseOptOut(t *testing.T) {
	f := newFixture(t, fixtureOpts{planJobTemplateID: "tpl-x"})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSkippedReason() != SkipReasonOperatorOptOut {
		t.Errorf("want %q, got %q", SkipReasonOperatorOptOut, resp.GetSkippedReason())
	}
}

func TestMaterializeJobs_Case8_TemplateRevisionRecorded(t *testing.T) {
	rootID := "tpl-rev"
	rev := int32(7)
	tpl := makeTemplate(rootID, "Revisioned", true)
	tpl.Revision = &rev
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: tpl},
	})
	resp, _ := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if len(resp.SpawnedJobs) != 1 {
		t.Fatalf("want 1 job")
	}
	if resp.SpawnedJobs[0].JobTemplateRevisionSnapshot == nil || *resp.SpawnedJobs[0].JobTemplateRevisionSnapshot != rev {
		t.Errorf("want revision_snapshot=%d, got %v", rev, resp.SpawnedJobs[0].JobTemplateRevisionSnapshot)
	}
}

func TestMaterializeJobs_Case9_ChildJobFailRollbackPath(t *testing.T) {
	rootID, childID := "tpl-root", "tpl-child"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID:  makeTemplate(rootID, "Root", true),
			childID: makeTemplate(childID, "Child", true),
		},
		relations:       []*jobtemplaterelationpb.JobTemplateRelation{makeRelation(rootID, childID, 1)},
		failOnNthCreate: 2, // root succeeds (idx 0 → len becomes 1), child fails (idx 1 → len already 1, hits failOnIdx 2)
	})
	_, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err == nil {
		t.Fatalf("expected error from child Job spawn failure")
	}
}

func TestMaterializeJobs_Case10_InactiveChildTemplate(t *testing.T) {
	rootID, childID := "tpl-root", "tpl-inactive"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID:  makeTemplate(rootID, "Root", true),
			childID: makeTemplate(childID, "Inactive", false), // <-- inactive
		},
		relations: []*jobtemplaterelationpb.JobTemplateRelation{makeRelation(rootID, childID, 1)},
	})
	_, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err == nil {
		t.Fatalf("expected error from inactive child template")
	}
}

func TestMaterializeJobs_Case11_TemplateZeroPhases(t *testing.T) {
	rootID := "tpl-empty"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Empty", true)},
		// no phases, no tasks
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.SpawnedJobs) != 1 {
		t.Errorf("want 1 job, got %d", len(resp.SpawnedJobs))
	}
	if len(f.phs.created) != 0 {
		t.Errorf("want 0 phases, got %d", len(f.phs.created))
	}
}

func TestMaterializeJobs_Case12_PhaseZeroTasks(t *testing.T) {
	rootID := "tpl-phase-only"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "P", true)},
		phasesByTpl: map[string][]*jobtemplatephasepb.JobTemplatePhase{
			rootID: {makePhase("p1", rootID, "Solo", 1, "")},
		},
		// no tasks
	})
	_, _ = f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if got := len(f.phs.created); got != 1 {
		t.Errorf("want 1 phase, got %d", got)
	}
	if got := len(f.tasks.created); got != 0 {
		t.Errorf("want 0 tasks, got %d", got)
	}
}

func TestMaterializeJobs_Case13_PredecessorPhaseRemap(t *testing.T) {
	rootID := "tpl-precede"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "T", true)},
		phasesByTpl: map[string][]*jobtemplatephasepb.JobTemplatePhase{
			rootID: {
				makePhase("tpl-p1", rootID, "Phase1", 1, ""),
				makePhase("tpl-p2", rootID, "Phase2", 2, "tpl-p1"),
			},
		},
	})
	_, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := len(f.phs.created); got != 2 {
		t.Fatalf("want 2 phases, got %d", got)
	}
	first, second := f.phs.created[0], f.phs.created[1]
	if second.PredecessorPhaseId == nil {
		t.Fatalf("phase 2 must have predecessor remapped")
	}
	if *second.PredecessorPhaseId != first.Id {
		t.Errorf("predecessor should point at spawned phase ID %q; got %q", first.Id, *second.PredecessorPhaseId)
	}
	// Sanity: the predecessor must NOT still equal the template-phase id.
	if *second.PredecessorPhaseId == "tpl-p1" {
		t.Errorf("predecessor was not remapped (still equals template_phase_id)")
	}
}

// Q-GSE-9: a template whose initial_status=JOB_STATUS_ACTIVE spawns ACTIVE jobs
// (root + child both honor the template's initial_status).
func TestMaterializeJobs_InitialStatusActive(t *testing.T) {
	rootID, childID := "tpl-root", "tpl-child"
	active := "JOB_STATUS_ACTIVE"
	rootTpl := makeTemplate(rootID, "Root", true)
	rootTpl.InitialStatus = &active
	childTpl := makeTemplate(childID, "Child", true)
	childTpl.InitialStatus = &active
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates: map[string]*jobtemplatepb.JobTemplate{
			rootID:  rootTpl,
			childID: childTpl,
		},
		relations: []*jobtemplaterelationpb.JobTemplateRelation{makeRelation(rootID, childID, 1)},
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.SpawnedJobs); got != 2 {
		t.Fatalf("want 2 jobs, got %d", got)
	}
	for i, j := range resp.SpawnedJobs {
		if j.Status != enumspb.JobStatus_JOB_STATUS_ACTIVE {
			t.Errorf("job[%d] status want ACTIVE, got %v", i, j.Status)
		}
	}
}

// Q-GSE-9: a template with NULL initial_status falls back to PLANNED — the
// historical default, so every existing plan is byte-unchanged.
func TestMaterializeJobs_InitialStatusNullFallsBackToPlanned(t *testing.T) {
	rootID := "tpl-root"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Root", true)}, // no initial_status
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.SpawnedJobs); got != 1 {
		t.Fatalf("want 1 job, got %d", got)
	}
	if resp.SpawnedJobs[0].Status != enumspb.JobStatus_JOB_STATUS_PLANNED {
		t.Errorf("NULL initial_status must fall back to PLANNED, got %v", resp.SpawnedJobs[0].Status)
	}
}

// Q-GSE-9: resolveInitialJobStatus unit — bare suffix + unknown handling.
func TestResolveInitialJobStatus(t *testing.T) {
	fallback := enumspb.JobStatus_JOB_STATUS_PLANNED
	cases := []struct {
		raw  string
		want enumspb.JobStatus
	}{
		{"", fallback},
		{"JOB_STATUS_ACTIVE", enumspb.JobStatus_JOB_STATUS_ACTIVE},
		{"ACTIVE", enumspb.JobStatus_JOB_STATUS_ACTIVE},
		{"job_status_active", enumspb.JobStatus_JOB_STATUS_ACTIVE},
		{"JOB_STATUS_UNSPECIFIED", fallback}, // UNSPECIFIED must not override the default
		{"NOT_A_STATUS", fallback},
	}
	for _, c := range cases {
		tpl := &jobtemplatepb.JobTemplate{}
		if c.raw != "" {
			v := c.raw
			tpl.InitialStatus = &v
		}
		if got := resolveInitialJobStatus(tpl, fallback); got != c.want {
			t.Errorf("resolveInitialJobStatus(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
	if got := resolveInitialJobStatus(nil, fallback); got != fallback {
		t.Errorf("nil template must fall back, got %v", got)
	}
}

// Sanity: spawned root Job carries origin_type=SUBSCRIPTION + origin_id=sub.id.
func TestMaterializeJobs_OriginFieldsSet(t *testing.T) {
	rootID := "tpl-x"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Root", true)},
	})
	resp, _ := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if len(resp.SpawnedJobs) != 1 {
		t.Fatalf("want 1 job")
	}
	j := resp.SpawnedJobs[0]
	if j.OriginType != enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION {
		t.Errorf("origin_type want SUBSCRIPTION, got %v", j.OriginType)
	}
	if j.OriginId == nil || *j.OriginId != "sub-1" {
		t.Errorf("origin_id want sub-1, got %v", j.OriginId)
	}
}

// Red-team HIGH #3: a scheduled price_plan whose price_schedule window has ended
// (date_time_end in the past) must skip the spawn with SkipReasonScheduleClosed,
// spawning no jobs, on the best-effort (direct) path.
func TestMaterializeJobs_ScheduleClosed_Skips(t *testing.T) {
	rootID := "tpl-root"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Root", true)},
		priceScheduleID:   "sched-closed",
		scheduleEnded:     true,
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSkippedReason() != SkipReasonScheduleClosed {
		t.Errorf("want skip reason %q, got %q", SkipReasonScheduleClosed, resp.GetSkippedReason())
	}
	if len(resp.SpawnedJobs) != 0 {
		t.Errorf("closed-AY schedule must spawn no jobs, got %d", len(resp.SpawnedJobs))
	}
	if len(f.jobs.created) != 0 {
		t.Errorf("no Job rows may be written into a closed AY, got %d", len(f.jobs.created))
	}
}

// An inactive price_schedule also blocks the spawn.
func TestMaterializeJobs_ScheduleInactive_Skips(t *testing.T) {
	rootID := "tpl-root"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Root", true)},
		priceScheduleID:   "sched-inactive",
		scheduleInactive:  true,
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSkippedReason() != SkipReasonScheduleClosed {
		t.Errorf("want skip reason %q, got %q", SkipReasonScheduleClosed, resp.GetSkippedReason())
	}
	if len(resp.SpawnedJobs) != 0 {
		t.Errorf("inactive schedule must spawn no jobs, got %d", len(resp.SpawnedJobs))
	}
}

// An active, open-ended (or future-ending) schedule spawns normally — the guard
// must not block a live AY.
func TestMaterializeJobs_ScheduleOpen_Spawns(t *testing.T) {
	rootID := "tpl-root"
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Root", true)},
		priceScheduleID:   "sched-open",
		// active, no end date -> open window
	})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSkippedReason() != "" {
		t.Errorf("open schedule must not skip, got reason %q", resp.GetSkippedReason())
	}
	if len(resp.SpawnedJobs) != 1 {
		t.Errorf("open schedule must spawn the root job, got %d", len(resp.SpawnedJobs))
	}
}

// The spawned job_phase must inherit scoring_scheme_id from its template phase
// (education grading pipeline enumerates job_phase by scoring_scheme_id; a NULL
// scheme makes the phase invisible to grade-compute / report cards). A template
// phase with no scheme stays NULL (unchanged for non-graded verticals).
func TestMaterializeJobs_ScoringSchemePropagated(t *testing.T) {
	rootID := "tpl-root"
	scheme := "seed-ibmyp-scheme"
	graded := makePhase("p1", rootID, "Semester 1", 1, "")
	graded.ScoringSchemeId = &scheme
	ungraded := makePhase("p2", rootID, "Semester 2", 2, "")
	f := newFixture(t, fixtureOpts{
		planJobTemplateID: rootID,
		templates:         map[string]*jobtemplatepb.JobTemplate{rootID: makeTemplate(rootID, "Root", true)},
		phasesByTpl:       map[string][]*jobtemplatephasepb.JobTemplatePhase{rootID: {graded, ungraded}},
	})
	if _, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var gotGraded, gotUngraded *jobphasepb.JobPhase
	for _, ph := range f.phs.created {
		switch ph.GetName() {
		case "Semester 1":
			gotGraded = ph
		case "Semester 2":
			gotUngraded = ph
		}
	}
	if gotGraded == nil || gotUngraded == nil {
		t.Fatalf("expected both spawned phases; graded=%v ungraded=%v", gotGraded, gotUngraded)
	}
	if gotGraded.GetScoringSchemeId() != scheme {
		t.Errorf("graded phase: want scoring_scheme_id %q, got %q", scheme, gotGraded.GetScoringSchemeId())
	}
	if gotUngraded.ScoringSchemeId != nil {
		t.Errorf("ungraded phase: want nil scoring_scheme_id, got %q", gotUngraded.GetScoringSchemeId())
	}
}
