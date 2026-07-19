package subscription

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"

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
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator

	// Optional. When set, the use case calls it for every spawned Job whose
	// billing_rule_type == MILESTONE per plan §3.7. Errors propagate and
	// roll back the entire spawn transaction.
	MaterializeBillingEventsForJob MaterializeBillingEventsForJobInvoker
}

// materializeJobsForSubscriptionInternalRequest is the internal input contract.
// Used inside the use case body; the public boundary uses *subscriptionpb.MaterializeJobsForSubscriptionRequest.
type materializeJobsForSubscriptionInternalRequest struct {
	SubscriptionId string
	SpawnJobs      bool
}

// materializeJobsForSubscriptionInternalResponse is the internal response.
// Used inside the use case body; the public boundary returns *subscriptionpb.MaterializeJobsForSubscriptionResponse.
type materializeJobsForSubscriptionInternalResponse struct {
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

// Execute is the DIRECT/manual proto-boundary entrypoint. It gates on
// subscription:update — the manual "Spawn Jobs" drawer and any direct/API
// caller reach the spawn ONLY through here, so this gate is preserved.
//
// The create-on-enrollment side effect does NOT flow through Execute: the
// CreateSubscriptionUseCase (already authorized by subscription:create) calls
// materializeCore directly via MaterializeJobsForSubscriptionInstantiator, so a
// least-privilege registrar role holding subscription:create — but not
// subscription:update — still spawns jobs under AUTHZ_ENFORCE (item #5, a).
func (uc *MaterializeJobsForSubscriptionUseCase) Execute(
	ctx context.Context, pbReq *subscriptionpb.MaterializeJobsForSubscriptionRequest,
) (*subscriptionpb.MaterializeJobsForSubscriptionResponse, error) {
	// Translate proto → internal at the boundary.
	req := materializeJobsForSubscriptionInternalRequest{}
	if pbReq != nil {
		req.SubscriptionId = pbReq.GetSubscriptionId()
		req.SpawnJobs = pbReq.GetSpawnJobs()
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Subscription,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	return uc.materializeCore(ctx, req)
}

// materializeCore drives the full spawn flow per plan §3 WITHOUT an
// authorization gate. The whole §3.3 → §3.7 chain runs in a single transaction.
//
// CHARTER — ungated internal core (item #5, option a). Callers MUST have an
// authorized ancestor operation that entitles this materialization, and the
// core operates SOLELY on the subscription named in req. The only permitted
// in-tree callers are:
//   - Execute above, which runs the subscription:update gate, and
//   - the create side effect (CreateSubscriptionUseCase, already authorized by
//     subscription:create) via the MaterializeJobsForSubscriptionInstantiator
//     adapter methods (InstantiateJobsFromPlan / InstantiateJobsFromPlanDetailed).
//
// NEVER wire this to a route, a proto handler, or the JobTemplateInstantiator
// port without an authorizing gate on the ancestor.
func (uc *MaterializeJobsForSubscriptionUseCase) materializeCore(
	ctx context.Context, req materializeJobsForSubscriptionInternalRequest,
) (*subscriptionpb.MaterializeJobsForSubscriptionResponse, error) {
	if req.SubscriptionId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
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
			ctx, uc.services.Translator,
			"subscription.errors.materialize_jobs_repositories_unavailable",
			"materialize_jobs_for_subscription is missing required repositories [DEFAULT]",
		))
	}

	// Plan §3.1 — operator override short-circuit before any DB read.
	if !req.SpawnJobs {
		skipReason := SkipReasonOperatorOptOut
		return &subscriptionpb.MaterializeJobsForSubscriptionResponse{Success: true, SkippedReason: &skipReason}, nil
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
		skipReason := SkipReasonNoTemplateFound
		return &subscriptionpb.MaterializeJobsForSubscriptionResponse{Success: true, SkippedReason: &skipReason}, nil
	}

	relations, err := uc.listChildRelations(ctx, rootTemplateID)
	if err != nil {
		return nil, err
	}

	// 2026-04-30 cyclic-subscription-jobs plan §4 — cyclic branch.
	//
	// For cyclic PricePlans (RECURRING, or CONTRACT with billing_cycle_value > 0),
	// this use case spawns ONLY the shell Job (no template, no
	// phases) plus any ONCE_AT_ENGAGEMENT_START children. The per-cycle
	// instance Jobs are spawned LATER by
	// MaterializeInstanceJobsForSubscription — triggered either by the
	// recognize-revenue piggyback (plan §5.2) or by an operator action.
	//
	// The non-cyclic path below remains UNCHANGED — phase4-subscription
	// regression specs (08-15) and the new C2 composition test are the canary.
	if IsCyclic(pricePlan) {
		internal, err := uc.executeCyclicShell(ctx, sub, pricePlan, plan, relations)
		if err != nil {
			return nil, err
		}
		return materializeJobsToProto(internal), nil
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

		// FIX-5 graph-wide parent pre-lock: enumerate EVERY job_template_phase across
		// all templates this graph will spawn (root + active relations) and take them
		// all FOR UPDATE in one global id order BEFORE the first job write, so the
		// entire graph serializes against any concurrent transition/spawn on a shared
		// parent (no-op on the mock/firestore providers).
		graphTemplateIDs := make([]string, 0, len(toSpawn))
		for _, entry := range toSpawn {
			graphTemplateIDs = append(graphTemplateIDs, entry.templateID)
		}
		if err := preLockSpawnGraph(txCtx, uc.spawnDeps(), graphTemplateIDs); err != nil {
			return err
		}

		for _, entry := range toSpawn {
			tpl, err := uc.readJobTemplate(txCtx, entry.templateID)
			if err != nil {
				return err
			}
			if !tpl.GetActive() {
				return errors.New(contextutil.GetTranslatedMessageWithContext(
					txCtx, uc.services.Translator,
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

	// FIX-5 (codex §5 HIGH): on the enforcing provider, template-backed spawn
	// requires a transaction — fail closed BEFORE the first job write so a miswired
	// transactor cannot autocommit a partial graph.
	if err := requireTxForTemplateSpawn(uc.repositories.JobPhase, uc.services.Transactor); err != nil {
		return nil, err
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, writeFn); err != nil {
			return nil, err
		}
	} else {
		if err := writeFn(ctx); err != nil {
			return nil, err
		}
	}

	return &subscriptionpb.MaterializeJobsForSubscriptionResponse{
		Success:     true,
		SpawnedJobs: spawnedJobs,
	}, nil
}

// materializeJobsToProto converts the internal response to proto.
func materializeJobsToProto(r *materializeJobsForSubscriptionInternalResponse) *subscriptionpb.MaterializeJobsForSubscriptionResponse {
	if r == nil {
		return &subscriptionpb.MaterializeJobsForSubscriptionResponse{Success: true}
	}
	resp := &subscriptionpb.MaterializeJobsForSubscriptionResponse{
		Success:     true,
		SpawnedJobs: r.SpawnedJobs,
	}
	if r.SkippedReason != "" {
		v := r.SkippedReason
		resp.SkippedReason = &v
	}
	return resp
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
			ctx, uc.services.Translator,
			"subscription.errors.not_found",
			"subscription not found [DEFAULT]",
		))
	}
	sub := resp.GetData()[0]
	if !sub.GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
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
			ctx, uc.services.Translator,
			"subscription.errors.price_plan_not_found",
			"price plan not found [DEFAULT]",
		))
	}
	resp, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
		Data: &priceplanpb.PricePlan{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
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
			ctx, uc.services.Translator,
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
			ctx, uc.services.Translator,
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
			ctx, uc.services.Translator,
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

// resolveInitialJobStatus maps a JobTemplate's initial_status (Q-GSE-9,
// job_template.proto field 50) to the lifecycle status a spawned Job takes.
//
// The field carries the canonical JobStatus enum name (e.g. "JOB_STATUS_ACTIVE").
// An empty/NULL value, or any value that does not resolve to a concrete
// (non-UNSPECIFIED) JobStatus, falls back to the caller-supplied default so
// every existing plan with NULL initial_status is byte-unchanged. A bare enum
// suffix ("ACTIVE") is accepted defensively.
func resolveInitialJobStatus(tpl *jobtemplatepb.JobTemplate, fallback enumspb.JobStatus) enumspb.JobStatus {
	if tpl == nil {
		return fallback
	}
	raw := strings.TrimSpace(tpl.GetInitialStatus())
	if raw == "" {
		return fallback
	}
	if v, ok := enumspb.JobStatus_value[raw]; ok && v != int32(enumspb.JobStatus_JOB_STATUS_UNSPECIFIED) {
		return enumspb.JobStatus(v)
	}
	up := strings.ToUpper(raw)
	if !strings.HasPrefix(up, "JOB_STATUS_") {
		up = "JOB_STATUS_" + up
	}
	if v, ok := enumspb.JobStatus_value[up]; ok && v != int32(enumspb.JobStatus_JOB_STATUS_UNSPECIFIED) {
		return enumspb.JobStatus(v)
	}
	return fallback
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
	if uc.services.IDGenerator != nil {
		jobID = uc.services.IDGenerator.GenerateID()
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
		Id:            jobID,
		Name:          jobName,
		JobTemplateId: &templateID,
		OriginType:    enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:      &originID,
		ClientId:      &clientID,
		// Q-GSE-9 (initial_status homed on job_template per the inversion rider):
		// a spawned Job adopts its source template's initial_status. A NULL/empty
		// or unrecognized value falls back to JOB_STATUS_PLANNED — the historical
		// default — so every existing plan (NULL initial_status) is byte-unchanged.
		// Grade-10 templates (initial_status=JOB_STATUS_ACTIVE) spawn ACTIVE jobs
		// so new-class jobs are immediately visible in Classes > Active. Applies to
		// root + child spawned jobs (both flow through this helper).
		Status:             resolveInitialJobStatus(tpl, enumspb.JobStatus_JOB_STATUS_PLANNED),
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
	// Denormalized job_category, copied from the workspace-owned template at
	// materialize time (job.proto:139 — copy-at-materialize; M7). NULL template
	// category → NULL job category.
	if tpl.JobCategoryId != nil && *tpl.JobCategoryId != "" {
		v := *tpl.JobCategoryId
		job.JobCategoryId = &v
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

// executeCyclicShell handles the cyclic branch (cyclic-subscription-
// jobs plan §4): spawn the shell Job (no template, no phases),
// then spawn any ONCE_AT_ENGAGEMENT_START child Jobs. NO cycle Jobs are
// spawned here — those come later via MaterializeInstanceJobsForSubscription.
//
// The whole sequence runs in a single transaction so Subscription.Create
// rolls back cleanly if any step fails. This preserves the atomic-rollback
// semantics of the non-cyclic path; the recognize-piggyback path
// (recognize_revenue_from_subscription.go) uses a non-fatal warning instead.
func (uc *MaterializeJobsForSubscriptionUseCase) executeCyclicShell(
	ctx context.Context,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	plan *planpb.Plan,
	relations []*jobtemplaterelationpb.JobTemplateRelation,
) (*materializeJobsForSubscriptionInternalResponse, error) {
	now := time.Now()
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)

	var spawnedJobs []*jobpb.Job

	writeFn := func(txCtx context.Context) error {
		spawnedJobs = spawnedJobs[:0]

		// FIX-5 / codex P3 §A5: graph-wide parent pre-lock BEFORE the first Job write
		// (the shell). The shell itself is template-less (no phases), but the
		// ONCE_AT_ENGAGEMENT_START children DO spawn template-backed phases; enumerate
		// EVERY such child template and take all their job_template_phase parents FOR
		// UPDATE in one global id order up front, so the whole cyclic-shell graph
		// serializes against any concurrent transition/spawn on a shared parent — the
		// cyclic-shell branch previously wrote its first Job with no prelock at all.
		// (no-op on the mock/firestore providers).
		graphTemplateIDs := make([]string, 0, len(relations))
		for _, rel := range relations {
			if !rel.GetActive() {
				continue
			}
			if rel.GetRelationType() != jobtemplaterelationpb.JobTemplateRelationType_JOB_TEMPLATE_RELATION_TYPE_ONCE_AT_ENGAGEMENT_START {
				continue
			}
			if cid := rel.GetChildTemplateId(); cid != "" {
				graphTemplateIDs = append(graphTemplateIDs, cid)
			}
		}
		if err := preLockSpawnGraph(txCtx, uc.spawnDeps(), graphTemplateIDs); err != nil {
			return err
		}

		// 1. Shell Job — no template, no phases. Status ACTIVE so the
		// shell stays open for life of subscription.
		shell, err := uc.spawnShell(txCtx, dc, dcs, sub, pricePlan)
		if err != nil {
			return err
		}
		spawnedJobs = append(spawnedJobs, shell)

		// 2. ONCE_AT_ENGAGEMENT_START children — spawn each as a child of the
		// shell Job (parent_job_id=shell.id, cycle_index=NULL). These
		// fire ONCE per engagement, not per cycle.
		for _, rel := range relations {
			if !rel.GetActive() {
				continue
			}
			if rel.GetRelationType() != jobtemplaterelationpb.JobTemplateRelationType_JOB_TEMPLATE_RELATION_TYPE_ONCE_AT_ENGAGEMENT_START {
				// SUB_TEMPLATE relations are NOT spawned in the cyclic branch.
				// In v1 they're treated as cycle-template extensions, which
				// belong on the cycle Job (out of scope for the engagement
				// shell). The cyclic plan §11.2 C3 documents this decision —
				// quietly skip rather than reject.
				continue
			}
			childID := rel.GetChildTemplateId()
			if childID == "" {
				continue
			}
			tpl, err := uc.readJobTemplate(txCtx, childID)
			if err != nil {
				return err
			}
			if !tpl.GetActive() {
				return errors.New(contextutil.GetTranslatedMessageWithContext(
					txCtx, uc.services.Translator,
					"subscription.errors.template_inactive",
					"job template is inactive [DEFAULT]",
				))
			}
			child, err := uc.spawnJob(txCtx, dc, dcs, tpl.GetName(), shell.GetId(), tpl, sub, pricePlan)
			if err != nil {
				return err
			}
			spawnedJobs = append(spawnedJobs, child)
			if err := uc.spawnPhasesAndTasks(txCtx, dc, dcs, child, tpl.GetId()); err != nil {
				return err
			}
		}
		return nil
	}

	// FIX-5 (codex §5 HIGH): on the enforcing provider, template-backed spawn
	// requires a transaction — fail closed BEFORE the first job write so a miswired
	// transactor cannot autocommit a partial graph.
	if err := requireTxForTemplateSpawn(uc.repositories.JobPhase, uc.services.Transactor); err != nil {
		return nil, err
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, writeFn); err != nil {
			return nil, err
		}
	} else {
		if err := writeFn(ctx); err != nil {
			return nil, err
		}
	}

	_ = plan // currently unused in cyclic shell creation; kept for symmetry
	return &materializeJobsForSubscriptionInternalResponse{SpawnedJobs: spawnedJobs}, nil
}

// spawnShell creates the cyclic shell Job: no
// job_template_id, no phases, parent_job_id=NULL, status=ACTIVE,
// billing_rule_type=NON_BILLABLE. cycle_* fields are NULL.
func (uc *MaterializeJobsForSubscriptionUseCase) spawnShell(
	ctx context.Context, dc int64, dcs string,
	sub *subscriptionpb.Subscription, pricePlan *priceplanpb.PricePlan,
) (*jobpb.Job, error) {
	jobID := ""
	if uc.services.IDGenerator != nil {
		jobID = uc.services.IDGenerator.GenerateID()
	} else {
		jobID = fmt.Sprintf("eng-%d", time.Now().UnixNano())
	}
	originID := sub.GetId()
	clientID := sub.GetClientId()
	name := sub.GetName()
	if name == "" {
		name = "(subscription shell)"
	}
	job := &jobpb.Job{
		Id:                 jobID,
		Name:               name,
		OriginType:         enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:           &originID,
		ClientId:           &clientID,
		Status:             enumspb.JobStatus_JOB_STATUS_ACTIVE,
		BillingRuleType:    enumspb.BillingRuleType_BILLING_RULE_TYPE_NON_BILLABLE,
		Active:             true,
		DateCreated:        &dc,
		DateCreatedString:  &dcs,
		DateModified:       &dc,
		DateModifiedString: &dcs,
	}
	if wsID := contextutil.ExtractWorkspaceIDFromContext(ctx); wsID != "" {
		v := wsID
		job.WorkspaceId = &v
	}
	_ = pricePlan
	resp, err := uc.repositories.Job.CreateJob(ctx, &jobpb.CreateJobRequest{Data: job})
	if err != nil {
		return nil, fmt.Errorf("create_shell_job: %w", err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0], nil
	}
	return job, nil
}

// spawnPhasesAndTasks delegates to the shared spawnPhasesAndTasksForJob seam
// (materialize_phase_spawn.go) so the W-SPAWN body is not duplicated between the
// base and instance materializers. New phases are born IN_PROGRESS/null-audit;
// the seam pre-locks template-phase parents and rejects hard-frozen targets on
// the postgres path.
func (uc *MaterializeJobsForSubscriptionUseCase) spawnPhasesAndTasks(
	ctx context.Context, dc int64, dcs string, job *jobpb.Job, templateID string,
) error {
	return spawnPhasesAndTasksForJob(ctx, uc.spawnDeps(), dc, dcs, job, templateID)
}

// spawnDeps returns the shared W-SPAWN dependency set (also used by the FIX-5
// graph-wide parent pre-lock).
func (uc *MaterializeJobsForSubscriptionUseCase) spawnDeps() phaseSpawnDeps {
	return phaseSpawnDeps{
		JobTemplatePhase: uc.repositories.JobTemplatePhase,
		JobTemplateTask:  uc.repositories.JobTemplateTask,
		JobPhase:         uc.repositories.JobPhase,
		JobTask:          uc.repositories.JobTask,
		IDGenerator:      uc.services.IDGenerator,
	}
}
