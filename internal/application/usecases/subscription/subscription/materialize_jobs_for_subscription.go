package subscription

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// MaterializeBillingEventsForJobInvoker is the narrow contract for the
// milestone-billing composition hook (plan §3.7). Provided by
// operation/job.MaterializeBillingEventsForJobUseCase.
//
// The interface is declared here (not imported) to avoid an espyna-internal
// cycle between subscription/subscription and operation/job. The composition
// layer wires the concrete use case as a closure via the
// MaterializeBillingEventsForJob field.
type MaterializeBillingEventsForJobInvoker interface {
	Execute(ctx context.Context, jobID, subscriptionID string) error
}

// MaterializeJobsForSubscriptionRepositories groups every repository the
// use case touches across subscription + operation domains. Cross-domain
// reads are unavoidable here per plan §6 — the spawn algorithm needs Plan
// (subscription domain), JobTemplate / JobTemplatePhase / JobTemplateTask /
// JobTemplateRelation (operation domain), and writes Job / JobPhase /
// JobTask in the operation domain.
type MaterializeJobsForSubscriptionRepositories struct {
	Subscription        subscriptionpb.SubscriptionDomainServiceServer
	PricePlan           priceplanpb.PricePlanDomainServiceServer
	Plan                planpb.PlanDomainServiceServer
	JobTemplate         jobtemplatepb.JobTemplateDomainServiceServer
	JobTemplatePhase    jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask     jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	Job                 jobpb.JobDomainServiceServer
	JobPhase            jobphasepb.JobPhaseDomainServiceServer
	JobTask             jobtaskpb.JobTaskDomainServiceServer
}

// MaterializeJobsForSubscriptionServices mirrors the standard service struct
// pattern used by every other use case in this package.
type MaterializeJobsForSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService

	// Optional. When set, the use case calls it for every spawned Job whose
	// billing_rule_type == MILESTONE per plan §3.7. Errors propagate and
	// roll back the entire spawn transaction.
	MaterializeBillingEventsForJob MaterializeBillingEventsForJobInvoker
}

// MaterializeJobsForSubscriptionRequest is the input contract.
type MaterializeJobsForSubscriptionRequest struct {
	SubscriptionId string
	// SpawnJobs is the operator-facing override (plan §3.1). When false the
	// use case returns immediately with skipped_reason="operator_opt_out".
	// When true, the algorithm falls through to template resolution.
	SpawnJobs bool
}

// MaterializeJobsForSubscriptionResponse echoes back the spawned root +
// child Jobs and a skip reason when no spawn was attempted.
type MaterializeJobsForSubscriptionResponse struct {
	SpawnedJobs   []*jobpb.Job
	SkippedReason string
	Warning       string
}

// Skip reason constants (plan §3.1).
const (
	SkipReasonNoTemplateFound = "no_template_found"
	SkipReasonOperatorOptOut  = "operator_opt_out"
)

// MaterializeJobsForSubscriptionUseCase spawns Job / JobPhase / JobTask rows
// from the JobTemplate referenced by the subscription's Plan. Composes with
// the milestone-billing MaterializeBillingEventsForJob via the invoker
// interface. See plan.md §3 for the full algorithm.
type MaterializeJobsForSubscriptionUseCase struct {
	repositories MaterializeJobsForSubscriptionRepositories
	services     MaterializeJobsForSubscriptionServices
}

// NewMaterializeJobsForSubscriptionUseCase wires the use case.
func NewMaterializeJobsForSubscriptionUseCase(
	repositories MaterializeJobsForSubscriptionRepositories,
	services MaterializeJobsForSubscriptionServices,
) *MaterializeJobsForSubscriptionUseCase {
	return &MaterializeJobsForSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute drives the full spawn flow per plan §3. The whole §3.3 → §3.7
// chain runs in a single transaction.
func (uc *MaterializeJobsForSubscriptionUseCase) Execute(
	ctx context.Context, req MaterializeJobsForSubscriptionRequest,
) (*MaterializeJobsForSubscriptionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscription, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if req.SubscriptionId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.validation.id_required",
			"subscription ID is required [DEFAULT]",
		))
	}
	if uc.repositories.Subscription == nil ||
		uc.repositories.PricePlan == nil ||
		uc.repositories.Plan == nil ||
		uc.repositories.JobTemplate == nil ||
		uc.repositories.JobTemplatePhase == nil ||
		uc.repositories.JobTemplateTask == nil ||
		uc.repositories.Job == nil ||
		uc.repositories.JobPhase == nil ||
		uc.repositories.JobTask == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.materialize_jobs_repositories_unavailable",
			"materialize_jobs_for_subscription is missing required repositories [DEFAULT]",
		))
	}

	// Plan §3.1 — operator override short-circuit before any DB read.
	if !req.SpawnJobs {
		return &MaterializeJobsForSubscriptionResponse{SkippedReason: SkipReasonOperatorOptOut}, nil
	}

	sub, err := uc.readSubscription(ctx, req.SubscriptionId)
	if err != nil {
		return nil, err
	}
	pricePlan, err := uc.readPricePlan(ctx, sub.GetPricePlanId())
	if err != nil {
		return nil, err
	}
	plan, err := uc.readPlan(ctx, pricePlan.GetPlanId())
	if err != nil {
		return nil, err
	}

	rootTemplateID := plan.GetJobTemplateId()
	if rootTemplateID == "" {
		return &MaterializeJobsForSubscriptionResponse{SkippedReason: SkipReasonNoTemplateFound}, nil
	}

	relations, err := uc.listChildRelations(ctx, rootTemplateID)
	if err != nil {
		return nil, err
	}

	type spawnEntry struct {
		templateID string
		isRoot     bool
	}
	toSpawn := make([]spawnEntry, 0, 1+len(relations))
	toSpawn = append(toSpawn, spawnEntry{templateID: rootTemplateID, isRoot: true})
	for _, rel := range relations {
		if !rel.GetActive() {
			continue
		}
		childID := rel.GetChildTemplateId()
		if childID == "" || childID == rootTemplateID {
			continue
		}
		toSpawn = append(toSpawn, spawnEntry{templateID: childID, isRoot: false})
	}

	now := time.Now()
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)

	var (
		rootJob     *jobpb.Job
		spawnedJobs []*jobpb.Job
	)

	writeFn := func(txCtx context.Context) error {
		spawnedJobs = spawnedJobs[:0]
		rootJob = nil

		for _, entry := range toSpawn {
			tpl, err := uc.readJobTemplate(txCtx, entry.templateID)
			if err != nil {
				return err
			}
			if !tpl.GetActive() {
				return errors.New(contextutil.GetTranslatedMessageWithContext(
					txCtx, uc.services.TranslationService,
					"subscription.errors.template_inactive",
					"job template is inactive [DEFAULT]",
				))
			}

			parentID := ""
			if !entry.isRoot && rootJob != nil {
				parentID = rootJob.GetId()
			}

			jobName := tpl.GetName()
			if entry.isRoot && sub.GetName() != "" {
				jobName = sub.GetName()
			}

			job, err := uc.spawnJob(txCtx, dc, dcs, jobName, parentID, tpl, sub, pricePlan)
			if err != nil {
				return err
			}
			if entry.isRoot {
				rootJob = job
			}
			spawnedJobs = append(spawnedJobs, job)

			if err := uc.spawnPhasesAndTasks(txCtx, dc, dcs, job, tpl.GetId()); err != nil {
				return err
			}
		}

		// Plan §3.7 — milestone composition.
		if uc.services.MaterializeBillingEventsForJob != nil {
			for _, job := range spawnedJobs {
				if job.GetBillingRuleType() != enumspb.BillingRuleType_BILLING_RULE_TYPE_MILESTONE {
					continue
				}
				if err := uc.services.MaterializeBillingEventsForJob.Execute(
					txCtx, job.GetId(), sub.GetId(),
				); err != nil {
					return fmt.Errorf("materialize_billing_events_for_job: %w", err)
				}
			}
		}
		return nil
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		if err := uc.services.TransactionService.ExecuteInTransaction(ctx, writeFn); err != nil {
			return nil, err
		}
	} else {
		if err := writeFn(ctx); err != nil {
			return nil, err
		}
	}

	return &MaterializeJobsForSubscriptionResponse{
		SpawnedJobs: spawnedJobs,
	}, nil
}

// ---- helpers ----

func (uc *MaterializeJobsForSubscriptionUseCase) readSubscription(
	ctx context.Context, id string,
) (*subscriptionpb.Subscription, error) {
	resp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.not_found",
			"subscription not found [DEFAULT]",
		))
	}
	sub := resp.GetData()[0]
	if !sub.GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.subscription_inactive",
			"subscription is inactive [DEFAULT]",
		))
	}
	return sub, nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) readPricePlan(
	ctx context.Context, id string,
) (*priceplanpb.PricePlan, error) {
	if id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.price_plan_not_found",
			"price plan not found [DEFAULT]",
		))
	}
	resp, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
		Data: &priceplanpb.PricePlan{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.price_plan_not_found",
			"price plan not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) readPlan(
	ctx context.Context, id string,
) (*planpb.Plan, error) {
	if id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.plan_not_found",
			"plan not found [DEFAULT]",
		))
	}
	idLocal := id
	resp, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
		Data: &planpb.Plan{Id: &idLocal},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.plan_not_found",
			"plan not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) readJobTemplate(
	ctx context.Context, id string,
) (*jobtemplatepb.JobTemplate, error) {
	resp, err := uc.repositories.JobTemplate.ReadJobTemplate(ctx, &jobtemplatepb.ReadJobTemplateRequest{
		Data: &jobtemplatepb.JobTemplate{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.template_not_found",
			"job template not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) listChildRelations(
	ctx context.Context, rootTemplateID string,
) ([]*jobtemplaterelationpb.JobTemplateRelation, error) {
	if uc.repositories.JobTemplateRelation == nil {
		return nil, nil
	}
	resp, err := uc.repositories.JobTemplateRelation.ListByParent(ctx,
		&jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest{
			ParentTemplateId: rootTemplateID,
		})
	if err != nil {
		return nil, fmt.Errorf("list_job_template_relations_by_parent: %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	rels := resp.GetJobTemplateRelations()
	sort.SliceStable(rels, func(i, j int) bool {
		return rels[i].GetSequenceOrder() < rels[j].GetSequenceOrder()
	})
	return rels, nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) spawnJob(
	ctx context.Context,
	dc int64, dcs string,
	jobName string,
	parentJobID string,
	tpl *jobtemplatepb.JobTemplate,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
) (*jobpb.Job, error) {
	jobID := ""
	if uc.services.IDService != nil {
		jobID = uc.services.IDService.GenerateID()
	} else {
		jobID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	templateID := tpl.GetId()
	originID := sub.GetId()
	clientID := sub.GetClientId()

	billingRule := enumspb.BillingRuleType_BILLING_RULE_TYPE_UNSPECIFIED
	if pricePlan != nil && pricePlan.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_MILESTONE {
		billingRule = enumspb.BillingRuleType_BILLING_RULE_TYPE_MILESTONE
	} else if tpl.DefaultBillingRuleType != nil {
		billingRule = *tpl.DefaultBillingRuleType
	}

	job := &jobpb.Job{
		Id:                 jobID,
		Name:               jobName,
		JobTemplateId:      &templateID,
		OriginType:         enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:           &originID,
		ClientId:           &clientID,
		Status:             enumspb.JobStatus_JOB_STATUS_PLANNED,
		BillingRuleType:    billingRule,
		Active:             true,
		DateCreated:        &dc,
		DateCreatedString:  &dcs,
		DateModified:       &dc,
		DateModifiedString: &dcs,
	}

	if tpl.DefaultFulfillmentType != nil {
		job.FulfillmentType = *tpl.DefaultFulfillmentType
	}
	if tpl.DefaultCostFlowType != nil {
		job.CostFlowType = *tpl.DefaultCostFlowType
	}
	if tpl.OutputProductId != nil && *tpl.OutputProductId != "" {
		v := *tpl.OutputProductId
		job.OutputProductId = &v
	}
	if tpl.OutputProductVariantId != nil && *tpl.OutputProductVariantId != "" {
		v := *tpl.OutputProductVariantId
		job.OutputProductVariantId = &v
	}
	if tpl.WorkspaceId != nil && *tpl.WorkspaceId != "" {
		v := *tpl.WorkspaceId
		job.WorkspaceId = &v
	} else if wsID := contextutil.ExtractWorkspaceIDFromContext(ctx); wsID != "" {
		v := wsID
		job.WorkspaceId = &v
	}
	if tpl.Revision != nil {
		v := *tpl.Revision
		job.JobTemplateRevisionSnapshot = &v
	}
	if templateID != "" {
		v := templateID
		job.JobTemplateRevisionId = &v
	}
	if parentJobID != "" {
		v := parentJobID
		job.ParentJobId = &v
	}

	resp, err := uc.repositories.Job.CreateJob(ctx, &jobpb.CreateJobRequest{Data: job})
	if err != nil {
		return nil, fmt.Errorf("create_job (template=%s): %w", templateID, err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0], nil
	}
	return job, nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) spawnPhasesAndTasks(
	ctx context.Context, dc int64, dcs string, job *jobpb.Job, templateID string,
) error {
	phaseResp, err := uc.repositories.JobTemplatePhase.ListByJobTemplate(ctx,
		&jobtemplatephasepb.ListByJobTemplateRequest{JobTemplateId: templateID})
	if err != nil {
		return fmt.Errorf("list_template_phases (template=%s): %w", templateID, err)
	}
	tplPhases := []*jobtemplatephasepb.JobTemplatePhase{}
	if phaseResp != nil {
		tplPhases = phaseResp.GetJobTemplatePhases()
	}
	sort.SliceStable(tplPhases, func(i, j int) bool {
		return tplPhases[i].GetPhaseOrder() < tplPhases[j].GetPhaseOrder()
	})

	phaseIDMap := make(map[string]string, len(tplPhases))

	for _, tp := range tplPhases {
		var phaseID string
		if uc.services.IDService != nil {
			phaseID = uc.services.IDService.GenerateID()
		} else {
			phaseID = fmt.Sprintf("phase-%d", time.Now().UnixNano())
		}
		tplPhaseID := tp.GetId()
		phase := &jobphasepb.JobPhase{
			Id:                 phaseID,
			JobId:              job.GetId(),
			Name:               tp.GetName(),
			PhaseOrder:         tp.GetPhaseOrder(),
			Status:             jobphasepb.PhaseStatus_PHASE_STATUS_PENDING,
			Active:             true,
			TemplatePhaseId:    &tplPhaseID,
			DateCreated:        &dc,
			DateCreatedString:  &dcs,
			DateModified:       &dc,
			DateModifiedString: &dcs,
		}
		if tp.PredecessorTemplatePhaseId != nil && *tp.PredecessorTemplatePhaseId != "" {
			if mapped, ok := phaseIDMap[*tp.PredecessorTemplatePhaseId]; ok {
				v := mapped
				phase.PredecessorPhaseId = &v
			}
		}
		if _, err := uc.repositories.JobPhase.CreateJobPhase(ctx,
			&jobphasepb.CreateJobPhaseRequest{Data: phase}); err != nil {
			return fmt.Errorf("create_job_phase (template_phase=%s): %w", tplPhaseID, err)
		}
		phaseIDMap[tplPhaseID] = phaseID

		taskResp, err := uc.repositories.JobTemplateTask.ListByPhase(ctx,
			&jobtemplatetaskpb.ListJobTemplateTasksByPhaseRequest{JobTemplatePhaseId: tplPhaseID})
		if err != nil {
			return fmt.Errorf("list_template_tasks (phase=%s): %w", tplPhaseID, err)
		}
		tplTasks := []*jobtemplatetaskpb.JobTemplateTask{}
		if taskResp != nil {
			tplTasks = taskResp.GetJobTemplateTasks()
		}
		sort.SliceStable(tplTasks, func(i, j int) bool {
			return tplTasks[i].GetStepOrder() < tplTasks[j].GetStepOrder()
		})
		for _, tt := range tplTasks {
			var taskID string
			if uc.services.IDService != nil {
				taskID = uc.services.IDService.GenerateID()
			} else {
				taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
			}
			tplTaskID := tt.GetId()
			task := &jobtaskpb.JobTask{
				Id:                 taskID,
				JobPhaseId:         phaseID,
				Name:               tt.GetName(),
				StepOrder:          tt.GetStepOrder(),
				Status:             jobtaskpb.TaskStatus_TASK_STATUS_PENDING,
				IsAdHoc:            false,
				Active:             true,
				TemplateTaskId:     &tplTaskID,
				DateCreated:        &dc,
				DateCreatedString:  &dcs,
				DateModified:       &dc,
				DateModifiedString: &dcs,
			}
			if _, err := uc.repositories.JobTask.CreateJobTask(ctx,
				&jobtaskpb.CreateJobTaskRequest{Data: task}); err != nil {
				return fmt.Errorf("create_job_task (template_task=%s): %w", tplTaskID, err)
			}
		}
	}
	return nil
}
