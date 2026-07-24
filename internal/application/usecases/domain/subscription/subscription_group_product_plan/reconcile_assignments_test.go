package subscription_group_product_plan

// DERIVE reconcile semantics (plan.md §2.5, espyna.md §3): exactly-one-
// claimant stamps assigned_to; 0/>1 claimants leave the task untouched and
// return a warning row; RECORDED (task_outcome exists) cells are never
// touched; re-running is idempotent (no redundant write on an already-correct
// stamp). In-package mocks — every List* method ignores its filter argument
// and returns the fixture's full slice; the use case's own in-memory
// re-filtering (mirrors the sgpps guard idiom) does the real scoping, so the
// mocks stay trivial.

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	sgmemberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	sgppspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// ----- mocks ------------------------------------------------------------------

type reconcileClassRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	class *pb.SubscriptionGroupProductPlan
}

func (m *reconcileClassRepo) ReadSubscriptionGroupProductPlan(_ context.Context, _ *pb.ReadSubscriptionGroupProductPlanRequest) (*pb.ReadSubscriptionGroupProductPlanResponse, error) {
	if m.class == nil {
		return &pb.ReadSubscriptionGroupProductPlanResponse{Success: true}, nil
	}
	return &pb.ReadSubscriptionGroupProductPlanResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlan{m.class}}, nil
}

type reconcileAssignmentRepo struct {
	sgppspb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	rows []*sgppspb.SubscriptionGroupProductPlanStaff
}

func (m *reconcileAssignmentRepo) ListSubscriptionGroupProductPlanStaffs(_ context.Context, _ *sgppspb.ListSubscriptionGroupProductPlanStaffsRequest) (*sgppspb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	return &sgppspb.ListSubscriptionGroupProductPlanStaffsResponse{Success: true, Data: m.rows}, nil
}

type reconcileMemberRepo struct {
	sgmemberpb.UnimplementedSubscriptionGroupMemberDomainServiceServer
	rows []*sgmemberpb.SubscriptionGroupMember
}

func (m *reconcileMemberRepo) ListSubscriptionGroupMembers(_ context.Context, _ *sgmemberpb.ListSubscriptionGroupMembersRequest) (*sgmemberpb.ListSubscriptionGroupMembersResponse, error) {
	return &sgmemberpb.ListSubscriptionGroupMembersResponse{Success: true, Data: m.rows}, nil
}

type reconcilePlanRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	plan *productplanpb.ProductPlan
}

func (m *reconcilePlanRepo) ReadProductPlan(_ context.Context, _ *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	if m.plan == nil {
		return &productplanpb.ReadProductPlanResponse{Success: true}, nil
	}
	return &productplanpb.ReadProductPlanResponse{Success: true, Data: []*productplanpb.ProductPlan{m.plan}}, nil
}

type reconcilePhaseRepo struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	rows []*jobtemplatephasepb.JobTemplatePhase
}

func (m *reconcilePhaseRepo) ListJobTemplatePhases(_ context.Context, _ *jobtemplatephasepb.ListJobTemplatePhasesRequest) (*jobtemplatephasepb.ListJobTemplatePhasesResponse, error) {
	return &jobtemplatephasepb.ListJobTemplatePhasesResponse{Success: true, Data: m.rows}, nil
}

type reconcileJobRepo struct {
	jobpb.UnimplementedJobDomainServiceServer
	rows []*jobpb.Job
}

func (m *reconcileJobRepo) ListJobs(_ context.Context, _ *jobpb.ListJobsRequest) (*jobpb.ListJobsResponse, error) {
	return &jobpb.ListJobsResponse{Success: true, Data: m.rows}, nil
}

type reconcileJobPhaseRepo struct {
	jobphasepb.UnimplementedJobPhaseDomainServiceServer
	rows []*jobphasepb.JobPhase
}

func (m *reconcileJobPhaseRepo) ListJobPhases(_ context.Context, _ *jobphasepb.ListJobPhasesRequest) (*jobphasepb.ListJobPhasesResponse, error) {
	return &jobphasepb.ListJobPhasesResponse{Success: true, Data: m.rows}, nil
}

type reconcileJobTaskRepo struct {
	jobtaskpb.UnimplementedJobTaskDomainServiceServer
	rows        []*jobtaskpb.JobTask
	updateCalls int
}

func (m *reconcileJobTaskRepo) ListJobTasks(_ context.Context, _ *jobtaskpb.ListJobTasksRequest) (*jobtaskpb.ListJobTasksResponse, error) {
	return &jobtaskpb.ListJobTasksResponse{Success: true, Data: m.rows}, nil
}

func (m *reconcileJobTaskRepo) UpdateJobTask(_ context.Context, req *jobtaskpb.UpdateJobTaskRequest) (*jobtaskpb.UpdateJobTaskResponse, error) {
	m.updateCalls++
	for _, t := range m.rows {
		if t.GetId() == req.GetData().GetId() {
			t.AssignedTo = req.GetData().AssignedTo
		}
	}
	return &jobtaskpb.UpdateJobTaskResponse{Success: true, Data: []*jobtaskpb.JobTask{req.GetData()}}, nil
}

type reconcileTaskOutcomeRepo struct {
	taskoutcomepb.UnimplementedTaskOutcomeDomainServiceServer
	byJobTaskID map[string][]*taskoutcomepb.TaskOutcome
}

func (m *reconcileTaskOutcomeRepo) ListTaskOutcomes(_ context.Context, req *taskoutcomepb.ListTaskOutcomesRequest) (*taskoutcomepb.ListTaskOutcomesResponse, error) {
	// The real adapter filters server-side; this mock does it directly (no
	// re-filter is needed downstream since it's exact) by reading the Field
	// value straight off the request — mirrors how the use case builds it via
	// filterEq("job_task_id", ...).
	var jobTaskID string
	if req.GetFilters() != nil {
		for _, f := range req.GetFilters().GetFilters() {
			if f.GetField() == "job_task_id" {
				jobTaskID = f.GetStringFilter().GetValue()
			}
		}
	}
	return &taskoutcomepb.ListTaskOutcomesResponse{Success: true, Data: m.byJobTaskID[jobTaskID]}, nil
}

// ----- fixture -----------------------------------------------------------------

type reconcileFixture struct {
	classRepo      *reconcileClassRepo
	assignmentRepo *reconcileAssignmentRepo
	memberRepo     *reconcileMemberRepo
	planRepo       *reconcilePlanRepo
	phaseRepo      *reconcilePhaseRepo
	jobRepo        *reconcileJobRepo
	jobPhaseRepo   *reconcileJobPhaseRepo
	jobTaskRepo    *reconcileJobTaskRepo
	outcomeRepo    *reconcileTaskOutcomeRepo
}

func newReconcileFixture() *reconcileFixture {
	return &reconcileFixture{
		classRepo:      &reconcileClassRepo{},
		assignmentRepo: &reconcileAssignmentRepo{},
		memberRepo:     &reconcileMemberRepo{},
		planRepo:       &reconcilePlanRepo{},
		phaseRepo:      &reconcilePhaseRepo{},
		jobRepo:        &reconcileJobRepo{},
		jobPhaseRepo:   &reconcileJobPhaseRepo{},
		jobTaskRepo:    &reconcileJobTaskRepo{},
		outcomeRepo:    &reconcileTaskOutcomeRepo{byJobTaskID: map[string][]*taskoutcomepb.TaskOutcome{}},
	}
}

func (f *reconcileFixture) uc() *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase {
	return NewReconcileSubscriptionGroupProductPlanAssignmentsUseCase(
		ReconcileSubscriptionGroupProductPlanAssignmentsRepositories{
			SubscriptionGroupProductPlan:      f.classRepo,
			SubscriptionGroupProductPlanStaff: f.assignmentRepo,
			SubscriptionGroupMember:           f.memberRepo,
			ProductPlan:                       f.planRepo,
			JobTemplatePhase:                  f.phaseRepo,
			Job:                               f.jobRepo,
			JobPhase:                          f.jobPhaseRepo,
			JobTask:                           f.jobTaskRepo,
			TaskOutcome:                       f.outcomeRepo,
		},
		ReconcileSubscriptionGroupProductPlanAssignmentsServices{
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

// baseScenario wires ONE class ("class-1", section "sg-1", template "tmpl-1",
// umbrella offering — no product_variant_id), ONE roster member ("client-1"),
// ONE job ("job-1") with ONE phase ("jobphase-1" -> template phase
// "tphase-1") and ONE task ("task-1"). Callers layer assignments / task
// state on top.
func (f *reconcileFixture) baseScenario() {
	f.classRepo.class = &pb.SubscriptionGroupProductPlan{
		Id: "class-1", SubscriptionGroupId: "sg-1", ProductPlanId: "pp-1", JobTemplateId: "tmpl-1", Active: true,
	}
	f.planRepo.plan = &productplanpb.ProductPlan{Id: "pp-1"} // umbrella: no ProductVariantId
	f.phaseRepo.rows = []*jobtemplatephasepb.JobTemplatePhase{
		{Id: "tphase-1", JobTemplateId: "tmpl-1", Active: true},
	}
	f.memberRepo.rows = []*sgmemberpb.SubscriptionGroupMember{
		{Id: "member-1", SubscriptionGroupId: "sg-1", ClientId: "client-1", Active: true},
	}
	f.jobRepo.rows = []*jobpb.Job{
		{Id: "job-1", ClientId: strptr("client-1"), JobTemplateId: strptr("tmpl-1"), Active: true},
	}
	f.jobPhaseRepo.rows = []*jobphasepb.JobPhase{
		{Id: "jobphase-1", JobId: "job-1", TemplatePhaseId: strptr("tphase-1"), Active: true},
	}
	f.jobTaskRepo.rows = []*jobtaskpb.JobTask{
		{Id: "task-1", JobPhaseId: "jobphase-1", Active: true},
	}
}

func strptr(s string) *string { return &s }

func assignment(id, classID, staffID, phaseID string) *sgppspb.SubscriptionGroupProductPlanStaff {
	a := &sgppspb.SubscriptionGroupProductPlanStaff{
		Id: id, Active: true, StaffId: staffID,
		SubscriptionGroupProductPlanId: &classID,
	}
	if phaseID != "" {
		a.JobTemplatePhaseId = &phaseID
	}
	return a
}

// ----- tests -------------------------------------------------------------------

func TestReconcile_ExactlyOneClaimant_Stamps(t *testing.T) {
	f := newReconcileFixture()
	f.baseScenario()
	f.assignmentRepo.rows = []*sgppspb.SubscriptionGroupProductPlanStaff{
		assignment("a-1", "class-1", "staff-1", ""), // phase-NULL, umbrella class -> covers tphase-1
	}

	resp, err := f.uc().Execute(context.Background(), &ReconcileSubscriptionGroupProductPlanAssignmentsRequest{SubscriptionGroupProductPlanID: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Stamped) != 1 {
		t.Fatalf("expected 1 stamp, got %d (%+v)", len(resp.Stamped), resp.Stamped)
	}
	if resp.Stamped[0].StaffID != "staff-1" || resp.Stamped[0].JobTaskID != "task-1" {
		t.Errorf("unexpected stamp: %+v", resp.Stamped[0])
	}
	if len(resp.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(resp.Warnings))
	}
	if f.jobTaskRepo.updateCalls != 1 {
		t.Errorf("expected 1 UpdateJobTask call, got %d", f.jobTaskRepo.updateCalls)
	}
	if f.jobTaskRepo.rows[0].GetAssignedTo() != "staff-1" {
		t.Errorf("job_task.assigned_to = %q, want staff-1", f.jobTaskRepo.rows[0].GetAssignedTo())
	}
}

func TestReconcile_ZeroClaimants_WarnsAndLeavesEmpty(t *testing.T) {
	f := newReconcileFixture()
	f.baseScenario()
	// No assignments at all — the class is legally "unstaffed" (plan.md §1.2 #5).

	resp, err := f.uc().Execute(context.Background(), &ReconcileSubscriptionGroupProductPlanAssignmentsRequest{SubscriptionGroupProductPlanID: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Stamped) != 0 {
		t.Fatalf("expected 0 stamps, got %d", len(resp.Stamped))
	}
	if len(resp.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(resp.Warnings))
	}
	if len(resp.Warnings[0].ClaimantStaffIDs) != 0 {
		t.Errorf("expected 0 claimants in the warning, got %v", resp.Warnings[0].ClaimantStaffIDs)
	}
	if f.jobTaskRepo.updateCalls != 0 {
		t.Errorf("expected 0 UpdateJobTask calls on an ambiguous cell, got %d", f.jobTaskRepo.updateCalls)
	}
	if f.jobTaskRepo.rows[0].GetAssignedTo() != "" {
		t.Errorf("job_task.assigned_to must stay empty, got %q", f.jobTaskRepo.rows[0].GetAssignedTo())
	}
}

// TestReconcile_CoTaught_WarnsAndLeavesUnstamped is the >1-claimant branch —
// two DIFFERENT staff both cover the same phase (co-teaching), which
// single-writer assigned_to cannot express (plan.md §1.3 rationale for the
// M5-E flip). DERIVE must not pick one arbitrarily.
func TestReconcile_CoTaught_WarnsAndLeavesUnstamped(t *testing.T) {
	f := newReconcileFixture()
	f.baseScenario()
	f.assignmentRepo.rows = []*sgppspb.SubscriptionGroupProductPlanStaff{
		assignment("a-1", "class-1", "staff-1", ""),
		assignment("a-2", "class-1", "staff-2", ""),
	}

	resp, err := f.uc().Execute(context.Background(), &ReconcileSubscriptionGroupProductPlanAssignmentsRequest{SubscriptionGroupProductPlanID: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Stamped) != 0 {
		t.Fatalf("expected 0 stamps for a co-taught cell, got %d", len(resp.Stamped))
	}
	if len(resp.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(resp.Warnings))
	}
	if len(resp.Warnings[0].ClaimantStaffIDs) != 2 {
		t.Errorf("expected 2 co-teaching claimants, got %v", resp.Warnings[0].ClaimantStaffIDs)
	}
	if f.jobTaskRepo.updateCalls != 0 {
		t.Errorf("expected 0 UpdateJobTask calls, got %d", f.jobTaskRepo.updateCalls)
	}
}

// TestReconcile_RecordedCell_NeverTouched: a task with an active task_outcome
// is skipped entirely, even though it has an unambiguous single covering
// assignment — "recorded cells never touched" (espyna.md §3).
func TestReconcile_RecordedCell_NeverTouched(t *testing.T) {
	f := newReconcileFixture()
	f.baseScenario()
	f.assignmentRepo.rows = []*sgppspb.SubscriptionGroupProductPlanStaff{
		assignment("a-1", "class-1", "staff-1", ""),
	}
	f.outcomeRepo.byJobTaskID["task-1"] = []*taskoutcomepb.TaskOutcome{
		{Id: "outcome-1", JobTaskId: "task-1", RecordedBy: "staff-1"},
	}

	resp, err := f.uc().Execute(context.Background(), &ReconcileSubscriptionGroupProductPlanAssignmentsRequest{SubscriptionGroupProductPlanID: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SkippedRecorded != 1 {
		t.Fatalf("expected 1 skipped-recorded, got %d", resp.SkippedRecorded)
	}
	if len(resp.Stamped) != 0 || len(resp.Warnings) != 0 {
		t.Fatalf("a RECORDED cell must produce neither a stamp nor a warning: stamped=%d warnings=%d", len(resp.Stamped), len(resp.Warnings))
	}
	if f.jobTaskRepo.updateCalls != 0 {
		t.Errorf("expected 0 UpdateJobTask calls on a recorded cell, got %d", f.jobTaskRepo.updateCalls)
	}
}

// TestReconcile_Idempotent_SecondRunIsNoop re-runs the exactly-one-claimant
// scenario a second time and asserts NO further UpdateJobTask call — the
// stamp is already correct.
func TestReconcile_Idempotent_SecondRunIsNoop(t *testing.T) {
	f := newReconcileFixture()
	f.baseScenario()
	f.assignmentRepo.rows = []*sgppspb.SubscriptionGroupProductPlanStaff{
		assignment("a-1", "class-1", "staff-1", ""),
	}
	uc := f.uc()
	ctx := context.Background()
	req := &ReconcileSubscriptionGroupProductPlanAssignmentsRequest{SubscriptionGroupProductPlanID: "class-1"}

	first, err := uc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error on first run: %v", err)
	}
	if len(first.Stamped) != 1 || f.jobTaskRepo.updateCalls != 1 {
		t.Fatalf("first run: expected 1 stamp / 1 update call, got stamped=%d updateCalls=%d", len(first.Stamped), f.jobTaskRepo.updateCalls)
	}

	second, err := uc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error on second run: %v", err)
	}
	if len(second.Stamped) != 0 {
		t.Errorf("second run: expected 0 NEW stamps, got %d", len(second.Stamped))
	}
	if second.AlreadyStamped != 1 {
		t.Errorf("second run: expected AlreadyStamped=1, got %d", second.AlreadyStamped)
	}
	if f.jobTaskRepo.updateCalls != 1 {
		t.Errorf("second run must NOT call UpdateJobTask again — idempotent, got %d total calls", f.jobTaskRepo.updateCalls)
	}
}

// TestReconcile_PhaseScopedAssignment_CoversOnlyItsOwnPhase is the Palladium
// proof (plan.md §1.3): two phase-scoped assignments on ONE umbrella class,
// each covering only its own semester's tasks.
func TestReconcile_PhaseScopedAssignment_CoversOnlyItsOwnPhase(t *testing.T) {
	f := newReconcileFixture()
	f.baseScenario()
	// A second phase + task the base scenario doesn't define.
	f.phaseRepo.rows = append(f.phaseRepo.rows, &jobtemplatephasepb.JobTemplatePhase{Id: "tphase-2", JobTemplateId: "tmpl-1", Active: true})
	f.jobPhaseRepo.rows = append(f.jobPhaseRepo.rows, &jobphasepb.JobPhase{Id: "jobphase-2", JobId: "job-1", TemplatePhaseId: strptr("tphase-2"), Active: true})
	f.jobTaskRepo.rows = append(f.jobTaskRepo.rows, &jobtaskpb.JobTask{Id: "task-2", JobPhaseId: "jobphase-2", Active: true})

	f.assignmentRepo.rows = []*sgppspb.SubscriptionGroupProductPlanStaff{
		assignment("a-sem1", "class-1", "staff-sem1", "tphase-1"),
		assignment("a-sem2", "class-1", "staff-sem2", "tphase-2"),
	}

	resp, err := f.uc().Execute(context.Background(), &ReconcileSubscriptionGroupProductPlanAssignmentsRequest{SubscriptionGroupProductPlanID: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Stamped) != 2 {
		t.Fatalf("expected 2 stamps (one per phase-scoped task), got %d (%+v)", len(resp.Stamped), resp.Stamped)
	}
	got := map[string]string{}
	for _, s := range resp.Stamped {
		got[s.JobTaskID] = s.StaffID
	}
	if got["task-1"] != "staff-sem1" {
		t.Errorf("task-1 stamped %q, want staff-sem1", got["task-1"])
	}
	if got["task-2"] != "staff-sem2" {
		t.Errorf("task-2 stamped %q, want staff-sem2", got["task-2"])
	}
}

// TestReconcile_StrandVariant_PhaseNullCoversOnlyMatchingVariant is the
// plan.md §2.5 strand rule: on a strand class (product_plan.product_variant_id
// set), a phase-NULL ("all phases") assignment covers ONLY the phases whose
// output_product_variant_id matches the class's own strand — never the
// sibling strand's phase (Cabornay-on-Music must not edit Visual-Arts cells).
func TestReconcile_StrandVariant_PhaseNullCoversOnlyMatchingVariant(t *testing.T) {
	f := newReconcileFixture()
	f.baseScenario()
	musicVariant := "variant-music"
	f.planRepo.plan = &productplanpb.ProductPlan{Id: "pp-1", ProductVariantId: &musicVariant} // strand class
	visualArtsVariant := "variant-visual-arts"
	f.phaseRepo.rows = []*jobtemplatephasepb.JobTemplatePhase{
		{Id: "tphase-1", JobTemplateId: "tmpl-1", Active: true, OutputProductVariantId: &musicVariant},      // matches the class's strand
		{Id: "tphase-2", JobTemplateId: "tmpl-1", Active: true, OutputProductVariantId: &visualArtsVariant}, // sibling strand
	}
	f.jobPhaseRepo.rows = append(f.jobPhaseRepo.rows, &jobphasepb.JobPhase{Id: "jobphase-2", JobId: "job-1", TemplatePhaseId: strptr("tphase-2"), Active: true})
	f.jobTaskRepo.rows = append(f.jobTaskRepo.rows, &jobtaskpb.JobTask{Id: "task-2", JobPhaseId: "jobphase-2", Active: true})

	f.assignmentRepo.rows = []*sgppspb.SubscriptionGroupProductPlanStaff{
		assignment("a-1", "class-1", "staff-music", ""), // phase-NULL — "all phases" of THIS class's grain
	}

	resp, err := f.uc().Execute(context.Background(), &ReconcileSubscriptionGroupProductPlanAssignmentsRequest{SubscriptionGroupProductPlanID: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Stamped) != 1 {
		t.Fatalf("expected exactly 1 stamp (the matching-strand task only), got %d (%+v)", len(resp.Stamped), resp.Stamped)
	}
	if resp.Stamped[0].JobTaskID != "task-1" {
		t.Errorf("expected task-1 (music strand) stamped, got %q", resp.Stamped[0].JobTaskID)
	}
	// task-2 (visual-arts strand) must be left as a 0-claimant warning.
	if len(resp.Warnings) != 1 || resp.Warnings[0].JobTaskID != "task-2" {
		t.Fatalf("expected exactly 1 warning for task-2 (uncovered sibling strand), got %+v", resp.Warnings)
	}
}
