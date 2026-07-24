package subscription_group_product_plan

import (
	"context"
	"errors"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
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

// ReconcileSubscriptionGroupProductPlanAssignmentsRepositories carries every
// repo the DERIVE algorithm reads (plan.md §2.5, espyna.md §3). Cross-domain
// reads into operation/ are unavoidable here — the same shape as
// MaterializeJobsForSubscriptionRepositories (subscription/subscription).
// TaskOutcome/JobTask are also written (never created — read-then-conditional-
// UpdateJobTask only; task_outcome is read-only, "recorded cells never
// touched").
type ReconcileSubscriptionGroupProductPlanAssignmentsRepositories struct {
	SubscriptionGroupProductPlan      pb.SubscriptionGroupProductPlanDomainServiceServer
	SubscriptionGroupProductPlanStaff sgppspb.SubscriptionGroupProductPlanStaffDomainServiceServer
	SubscriptionGroupMember           sgmemberpb.SubscriptionGroupMemberDomainServiceServer
	ProductPlan                       productplanpb.ProductPlanDomainServiceServer
	JobTemplatePhase                  jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	Job                               jobpb.JobDomainServiceServer
	JobPhase                          jobphasepb.JobPhaseDomainServiceServer
	JobTask                           jobtaskpb.JobTaskDomainServiceServer
	TaskOutcome                       taskoutcomepb.TaskOutcomeDomainServiceServer
}

type ReconcileSubscriptionGroupProductPlanAssignmentsServices struct {
	Translator ports.Translator
	// ActionGatekeeper: DERIVE is operational/CLI control, not an end-user grant
	// (plan.md §5) — no new subscription_group_product_plan:derive permission is
	// introduced; the CLI/M5 invoker runs with a disabled/NoOp Authorizer
	// (Check() short-circuits when !authorizer.IsEnabled(), the same seam every
	// other CLI in this codebase uses), so gating on the existing
	// subscription_group_product_plan:update code stays fail-closed for any
	// future end-user surface without requiring a seed grant today.
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReconcileSubscriptionGroupProductPlanAssignmentsRequest scopes one DERIVE run
// to a single class row — the CLI/M5 caller loops classes (idempotent per call).
type ReconcileSubscriptionGroupProductPlanAssignmentsRequest struct {
	SubscriptionGroupProductPlanID string
}

// ReconcileStamp is one job_task the run wrote assigned_to onto (exactly one
// covering assignment resolved).
type ReconcileStamp struct {
	JobTaskID       string
	ClientID        string
	TemplatePhaseID string
	StaffID         string
}

// ReconcileWarning is one EMPTY, unrecorded job_task the run left untouched
// because coverage was ambiguous (0 or >1 covering assignments).
type ReconcileWarning struct {
	JobTaskID        string
	ClientID         string
	TemplatePhaseID  string
	ClaimantStaffIDs []string // len 0 = uncovered; len >1 = co-taught
}

// ReconcileSubscriptionGroupProductPlanAssignmentsResponse is the manifest
// payload for one class's DERIVE run.
type ReconcileSubscriptionGroupProductPlanAssignmentsResponse struct {
	SubscriptionGroupProductPlanID string
	Stamped                        []ReconcileStamp
	AlreadyStamped                 int // exactly-one-claimant cells already correct — idempotent no-op
	Warnings                       []ReconcileWarning
	SkippedRecorded                int // job_task already has an active task_outcome — never touched
	ProcessedMembers               int
	ProcessedJobs                  int
}

type ReconcileSubscriptionGroupProductPlanAssignmentsUseCase struct {
	repositories ReconcileSubscriptionGroupProductPlanAssignmentsRepositories
	services     ReconcileSubscriptionGroupProductPlanAssignmentsServices
}

func NewReconcileSubscriptionGroupProductPlanAssignmentsUseCase(r ReconcileSubscriptionGroupProductPlanAssignmentsRepositories, s ReconcileSubscriptionGroupProductPlanAssignmentsServices) *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase {
	return &ReconcileSubscriptionGroupProductPlanAssignmentsUseCase{repositories: r, services: s}
}

// Execute implements the DERIVE coverage rule (plan.md §2.5, espyna.md §3):
// for every roster member's EMPTY (no task_outcome) job_task under phase P,
// exactly one covering assignment stamps job_task.assigned_to; 0 or >1 leaves
// the task untouched and returns a warning row. Idempotent: re-running never
// re-writes an already-correct stamp and never touches a RECORDED task.
func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) Execute(ctx context.Context, req *ReconcileSubscriptionGroupProductPlanAssignmentsRequest) (*ReconcileSubscriptionGroupProductPlanAssignmentsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlan, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.SubscriptionGroupProductPlanID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}

	class, err := uc.readClass(ctx, req.SubscriptionGroupProductPlanID)
	if err != nil {
		return nil, err
	}

	classVariantID, err := uc.classProductVariantID(ctx, class.GetProductPlanId())
	if err != nil {
		return nil, err
	}

	phasesByID, err := uc.listTemplatePhases(ctx, class.GetJobTemplateId())
	if err != nil {
		return nil, err
	}

	assignments, err := uc.listActiveAssignments(ctx, class.GetId())
	if err != nil {
		return nil, err
	}
	phaseClaimants := buildPhaseClaimants(assignments, classVariantID, phasesByID)

	members, err := uc.listActiveMembers(ctx, class.GetSubscriptionGroupId())
	if err != nil {
		return nil, err
	}

	resp := &ReconcileSubscriptionGroupProductPlanAssignmentsResponse{
		SubscriptionGroupProductPlanID: class.GetId(),
		ProcessedMembers:               len(members),
	}

	for _, member := range members {
		jobs, err := uc.listStudentSubjectJobs(ctx, member.GetClientId(), class.GetJobTemplateId())
		if err != nil {
			return nil, err
		}
		for _, job := range jobs {
			resp.ProcessedJobs++
			if err := uc.reconcileJob(ctx, job, member.GetClientId(), phaseClaimants, resp); err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) reconcileJob(ctx context.Context, job *jobpb.Job, clientID string, phaseClaimants map[string]map[string]bool, resp *ReconcileSubscriptionGroupProductPlanAssignmentsResponse) error {
	phasesResp, err := uc.repositories.JobPhase.ListJobPhases(ctx, &jobphasepb.ListJobPhasesRequest{
		Filters: filterEq("job_id", job.GetId()),
	})
	if err != nil {
		return err
	}
	for _, phase := range phasesResp.GetData() {
		if phase == nil || !phase.GetActive() || phase.GetJobId() != job.GetId() {
			continue
		}
		templatePhaseID := phase.GetTemplatePhaseId()
		if templatePhaseID == "" {
			continue // unanchored phase — not attributable to any class phase
		}

		tasksResp, err := uc.repositories.JobTask.ListJobTasks(ctx, &jobtaskpb.ListJobTasksRequest{
			Filters: filterEq("job_phase_id", phase.GetId()),
		})
		if err != nil {
			return err
		}
		for _, task := range tasksResp.GetData() {
			if task == nil || !task.GetActive() || task.GetJobPhaseId() != phase.GetId() {
				continue
			}
			if err := uc.reconcileTask(ctx, task, clientID, templatePhaseID, phaseClaimants, resp); err != nil {
				return err
			}
		}
	}
	return nil
}

func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) reconcileTask(ctx context.Context, task *jobtaskpb.JobTask, clientID, templatePhaseID string, phaseClaimants map[string]map[string]bool, resp *ReconcileSubscriptionGroupProductPlanAssignmentsResponse) error {
	outcomesResp, err := uc.repositories.TaskOutcome.ListTaskOutcomes(ctx, &taskoutcomepb.ListTaskOutcomesRequest{
		Filters: filterEq("job_task_id", task.GetId()),
	})
	if err != nil {
		return err
	}
	for _, o := range outcomesResp.GetData() {
		if o != nil && o.GetJobTaskId() == task.GetId() {
			resp.SkippedRecorded++
			return nil // RECORDED — never touched, regardless of stamp state
		}
	}

	claimants := phaseClaimants[templatePhaseID]
	staffIDs := make([]string, 0, len(claimants))
	for id := range claimants {
		staffIDs = append(staffIDs, id)
	}
	sort.Strings(staffIDs)

	if len(staffIDs) != 1 {
		resp.Warnings = append(resp.Warnings, ReconcileWarning{
			JobTaskID:        task.GetId(),
			ClientID:         clientID,
			TemplatePhaseID:  templatePhaseID,
			ClaimantStaffIDs: staffIDs,
		})
		return nil
	}

	target := staffIDs[0]
	if task.GetAssignedTo() == target {
		resp.AlreadyStamped++
		return nil // idempotent — already correct, no write
	}

	if _, err := uc.repositories.JobTask.UpdateJobTask(ctx, &jobtaskpb.UpdateJobTaskRequest{
		Data: &jobtaskpb.JobTask{Id: task.GetId(), AssignedTo: &target},
	}); err != nil {
		return err
	}
	resp.Stamped = append(resp.Stamped, ReconcileStamp{
		JobTaskID:       task.GetId(),
		ClientID:        clientID,
		TemplatePhaseID: templatePhaseID,
		StaffID:         target,
	})
	return nil
}

func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) readClass(ctx context.Context, id string) (*pb.SubscriptionGroupProductPlan, error) {
	resp, err := uc.repositories.SubscriptionGroupProductPlan.ReadSubscriptionGroupProductPlan(ctx, &pb.ReadSubscriptionGroupProductPlanRequest{
		Data: &pb.SubscriptionGroupProductPlan{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.errors.not_found", "class not found [DEFAULT]"))
	}
	class := resp.GetData()[0]
	if !class.GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.errors.not_found", "class not found [DEFAULT]"))
	}
	return class, nil
}

func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) classProductVariantID(ctx context.Context, productPlanID string) (string, error) {
	if uc.repositories.ProductPlan == nil || productPlanID == "" {
		return "", nil
	}
	resp, err := uc.repositories.ProductPlan.ReadProductPlan(ctx, &productplanpb.ReadProductPlanRequest{
		Data: &productplanpb.ProductPlan{Id: productPlanID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return "", nil
	}
	return resp.GetData()[0].GetProductVariantId(), nil
}

func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) listTemplatePhases(ctx context.Context, jobTemplateID string) (map[string]*jobtemplatephasepb.JobTemplatePhase, error) {
	out := map[string]*jobtemplatephasepb.JobTemplatePhase{}
	if uc.repositories.JobTemplatePhase == nil || jobTemplateID == "" {
		return out, nil
	}
	resp, err := uc.repositories.JobTemplatePhase.ListJobTemplatePhases(ctx, &jobtemplatephasepb.ListJobTemplatePhasesRequest{
		Filters: filterEq("job_template_id", jobTemplateID),
	})
	if err != nil {
		return nil, err
	}
	for _, p := range resp.GetData() {
		if p == nil || !p.GetActive() || p.GetJobTemplateId() != jobTemplateID {
			continue
		}
		out[p.GetId()] = p
	}
	return out, nil
}

func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) listActiveAssignments(ctx context.Context, classID string) ([]*sgppspb.SubscriptionGroupProductPlanStaff, error) {
	resp, err := uc.repositories.SubscriptionGroupProductPlanStaff.ListSubscriptionGroupProductPlanStaffs(ctx, &sgppspb.ListSubscriptionGroupProductPlanStaffsRequest{
		Filters: filterEq("subscription_group_product_plan_id", classID),
	})
	if err != nil {
		return nil, err
	}
	var out []*sgppspb.SubscriptionGroupProductPlanStaff
	for _, a := range resp.GetData() {
		if a == nil || !a.GetActive() {
			continue
		}
		if a.GetSubscriptionGroupProductPlanId() != classID {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) listActiveMembers(ctx context.Context, subscriptionGroupID string) ([]*sgmemberpb.SubscriptionGroupMember, error) {
	resp, err := uc.repositories.SubscriptionGroupMember.ListSubscriptionGroupMembers(ctx, &sgmemberpb.ListSubscriptionGroupMembersRequest{
		Filters: filterEq("subscription_group_id", subscriptionGroupID),
	})
	if err != nil {
		return nil, err
	}
	var out []*sgmemberpb.SubscriptionGroupMember
	for _, m := range resp.GetData() {
		if m == nil || !m.GetActive() {
			continue
		}
		if m.GetSubscriptionGroupId() != subscriptionGroupID {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// listStudentSubjectJobs finds the student's job(s) for this class's subject
// (job.client_id == studentID AND job.job_template_id == the class's curriculum
// anchor) — the W-SPAWN "one job PER STUDENT per subject" contract
// (docs/wiki business-logic-education-vertical.md). Zero results means the job
// has not been spawned yet (skip, not an error); MaterializeJobsForSubscription
// is expected to spawn exactly one, but this defensively iterates every match.
func (uc *ReconcileSubscriptionGroupProductPlanAssignmentsUseCase) listStudentSubjectJobs(ctx context.Context, studentClientID, jobTemplateID string) ([]*jobpb.Job, error) {
	if studentClientID == "" || jobTemplateID == "" {
		return nil, nil
	}
	resp, err := uc.repositories.Job.ListJobs(ctx, &jobpb.ListJobsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field:      "client_id",
					FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: studentClientID, Operator: commonpb.StringOperator_STRING_EQUALS}},
				},
				{
					Field:      "job_template_id",
					FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: jobTemplateID, Operator: commonpb.StringOperator_STRING_EQUALS}},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	var out []*jobpb.Job
	for _, j := range resp.GetData() {
		if j == nil || !j.GetActive() {
			continue
		}
		if j.GetClientId() != studentClientID || j.GetJobTemplateId() != jobTemplateID {
			continue
		}
		out = append(out, j)
	}
	return out, nil
}

// buildPhaseClaimants computes, for every class template phase, the set of
// distinct staff IDs whose active assignment covers it (plan.md §2.5):
//
//	covered_phases(A) = A.job_template_phase_id set   -> { that phase }
//	                     else if class.pp variant set  -> { P : P.opv == variant }
//	                     else                           -> phases(class.job_template_id)
//
// staffID is read from the dual-write legacy field (SubscriptionGroupProductPlanStaff.StaffId,
// f10) — kept in sync with the v2 product_plan_staff_id FK by the create/update
// use cases (espyna.md §2), so it is authoritative without a second join.
func buildPhaseClaimants(assignments []*sgppspb.SubscriptionGroupProductPlanStaff, classVariantID string, phasesByID map[string]*jobtemplatephasepb.JobTemplatePhase) map[string]map[string]bool {
	claimants := map[string]map[string]bool{}
	claim := func(phaseID, staffID string) {
		if phaseID == "" || staffID == "" {
			return
		}
		set, ok := claimants[phaseID]
		if !ok {
			set = map[string]bool{}
			claimants[phaseID] = set
		}
		set[staffID] = true
	}

	for _, a := range assignments {
		staffID := a.GetStaffId()
		if staffID == "" {
			continue
		}
		if phaseID := a.GetJobTemplatePhaseId(); phaseID != "" {
			claim(phaseID, staffID)
			continue
		}
		// phase-NULL — "all phases" of the class's covered grain.
		for phaseID, phase := range phasesByID {
			if classVariantID != "" && phase.GetOutputProductVariantId() != classVariantID {
				continue
			}
			claim(phaseID, staffID)
		}
	}
	return claimants
}

// filterEq builds a single string-equals FilterRequest — the List default
// already scopes to active=true (adapter/core/operations.go), so no explicit
// active filter is added here; callers additionally re-check GetActive() in
// memory (defense-in-depth, matches the sgpps guard idiom).
func filterEq(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{{
			Field:      field,
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: value, Operator: commonpb.StringOperator_STRING_EQUALS}},
		}},
	}
}
