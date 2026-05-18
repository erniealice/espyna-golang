package subscription

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// File: materialize_instance_jobs_for_subscription.go
//
// Materializes per-cycle ("instance") Job rows for cyclic subscriptions, plus
// once-at-engagement-start onboarding child Jobs on the first call.
//
// The file name uses "instance" rather than "cycle" per cyclic-subscription-jobs
// plan §19.4 (forward-compat with the AD_HOC plan). For cyclic plans the
// "instance" IS a cycle Job; the AD_HOC plan extends the eligibility predicate
// without renaming the use case.
//
// Algorithm: cyclic-subscription-jobs plan §3 (cycle algorithm), §4 (composition
// with materialize-jobs), §5 (recognize-piggyback trigger contract).

// ---- Skip reason constants (plan §3.7 + ad-hoc plan §3.4) ----

const (
	InstanceSkipReasonNonCyclicPlan        = "non_cyclic_plan"
	InstanceSkipReasonNoTemplate           = "no_template"
	InstanceSkipReasonMilestoneUnsupported = "milestone_unsupported"
	InstanceSkipReasonNoPendingCycles      = "no_pending_cycles"
	// AD_HOC × TOTAL_PACKAGE: requested usage but used >= resolvedEntitlement.
	InstanceSkipReasonEntitlementExhausted = "entitlement_exhausted"
	// AD_HOC × TOTAL_PACKAGE without entitled_occurrences set on PricePlan
	// (and no per-subscription override) — defensive; the validator should
	// have blocked the PricePlan save.
	InstanceSkipReasonEntitlementRequired = "entitlement_required"
	// AD_HOC × {TOTAL_PACKAGE, PER_OCCURRENCE} without Plan.job_template_id —
	// usage Jobs need ops tracking. Validator should have blocked at PricePlan
	// save (see ad-hoc plan §6: pay_per_call_no_template + pool_no_template).
	InstanceSkipReasonAdHocNoTemplate = "ad_hoc_no_template"
	// AD_HOC × invalid amount_basis. Defensive — validator-only path.
	InstanceSkipReasonAdHocInvalidBasis = "ad_hoc_invalid_basis"
)

// MaxBackfillCycles caps a single backfill request (plan §15 risk). If a
// subscription has more missing cycles than this, the use case stops at the
// cap and surfaces the cap in `BackfillCappedAt`. The Operations tab drawer
// previews the count before submit so operators see the cap in advance.
const MaxBackfillCycles = 24

// MaterializeInstanceJobsForSubscriptionRepositories groups every repository
// the use case touches across subscription + operation domains.
type MaterializeInstanceJobsForSubscriptionRepositories struct {
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
	// AD_HOC × PER_OCCURRENCE: a paired BillingEvent is created at usage spawn.
	// Optional — when nil and PricePlan is AD_HOC × PER_OCCURRENCE, the use
	// case returns a wired-out error so the operator sees the misconfig early.
	BillingEvent billingeventpb.BillingEventDomainServiceServer
}

// MaterializeInstanceJobsForSubscriptionServices bundles the standard service
// dependencies. Mirrors MaterializeJobsForSubscriptionServices.
type MaterializeInstanceJobsForSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// materializeInstanceJobsInternalRequest is the internal input contract.
// Public boundary uses *subscriptionpb.MaterializeInstanceJobsForSubscriptionRequest.
type materializeInstanceJobsInternalRequest struct {
	SubscriptionId   string
	CyclePeriodStart string
	Backfill         bool

	// AD_HOC only: operator-supplied request date for the new usage Job
	// (ISO 8601 YYYY-MM-DD). Defaults to today (UTC) when empty. Ignored on
	// cyclic plans.
	UsageRequestDate string
}

// spawnedInstanceCycle is one cycle's spawn result (internal).
type spawnedInstanceCycle struct {
	CycleIndex       int32
	CyclePeriodStart string
	CyclePeriodEnd   string
	Jobs             []*jobpb.Job // 1 entry for visits_per_cycle=1, N for multi-visit
}

// materializeInstanceJobsInternalResponse is the internal response.
// Public boundary returns *subscriptionpb.MaterializeInstanceJobsForSubscriptionResponse.
type materializeInstanceJobsInternalResponse struct {
	ShellJob                  *jobpb.Job
	EngagementWasNewlyCreated bool
	SpawnedCycles             []spawnedInstanceCycle
	OnceAtStartJobs           []*jobpb.Job
	SkippedReason             string
	BackfillCappedAt          int32 // 0 when not capped; otherwise the cap that fired
}

// MaterializeInstanceJobsForSubscriptionUseCase is the cyclic Job spawn engine.
type MaterializeInstanceJobsForSubscriptionUseCase struct {
	repositories MaterializeInstanceJobsForSubscriptionRepositories
	services     MaterializeInstanceJobsForSubscriptionServices
}

// NewMaterializeInstanceJobsForSubscriptionUseCase wires the use case.
func NewMaterializeInstanceJobsForSubscriptionUseCase(
	repositories MaterializeInstanceJobsForSubscriptionRepositories,
	services MaterializeInstanceJobsForSubscriptionServices,
) *MaterializeInstanceJobsForSubscriptionUseCase {
	return &MaterializeInstanceJobsForSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// eligibleForInstanceSpawn reports whether the PricePlan's billing_kind
// participates in the two-tier shell+instance Job model. Cyclic kinds
// drive auto-spawn at cycle boundaries; AD_HOC kinds drive operator-requested
// usage spawns. See cyclic plan §19.3 + ad-hoc plan §3.1.
//
// NAMED FUNCTION: do not inline.
func eligibleForInstanceSpawn(pp *priceplanpb.PricePlan) bool {
	if pp == nil {
		return false
	}
	kind := pp.GetBillingKind()
	if kind == priceplanpb.BillingKind_BILLING_KIND_RECURRING {
		return true
	}
	if kind == priceplanpb.BillingKind_BILLING_KIND_CONTRACT && pp.GetBillingCycleValue() > 0 {
		return true
	}
	if kind == priceplanpb.BillingKind_BILLING_KIND_AD_HOC {
		return true
	}
	return false
}

// IsAdHoc returns true when the PricePlan uses the event-driven AD_HOC kind.
// Both AD_HOC × TOTAL_PACKAGE (prepaid pool) and AD_HOC × PER_OCCURRENCE
// (pay-per-call) flow through the same use case but diverge on entitlement
// gate and BillingEvent spawn — see executeAdHoc.
func IsAdHoc(pp *priceplanpb.PricePlan) bool {
	if pp == nil {
		return false
	}
	return pp.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_AD_HOC
}

// resolvedEntitlement returns the AD_HOC × TOTAL_PACKAGE pool size:
// Subscription.entitled_occurrences_override (when > 0) wins over
// PricePlan.entitled_occurrences. Codex MAJ-1: keeps the catalog template
// shared while letting "Extend pool" be a per-subscription operation.
func resolvedEntitlement(sub *subscriptionpb.Subscription, pp *priceplanpb.PricePlan) int32 {
	if v := sub.GetEntitledOccurrencesOverride(); v > 0 {
		return v
	}
	return pp.GetEntitledOccurrences()
}

// IsCyclic is the public mirror of eligibleForInstanceSpawn for callers that
// need to branch (recognize-revenue piggyback, materialize-jobs branch, view
// layer). Its body is identical so the AD_HOC plan only has to update the
// predicate in one place. (shell-Job predicate)
func IsCyclic(pp *priceplanpb.PricePlan) bool {
	return eligibleForInstanceSpawn(pp)
}

// Execute is the proto-boundary entry point. It translates the proto request to
// the internal request, delegates to executeInternal, and converts the result
// to a proto response (counts only — callers that need full Job records call
// executeInternal directly, e.g., tests and the composition-layer adapter).
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) Execute(
	ctx context.Context, pbReq *subscriptionpb.MaterializeInstanceJobsForSubscriptionRequest,
) (*subscriptionpb.MaterializeInstanceJobsForSubscriptionResponse, error) {
	req := materializeInstanceJobsInternalRequest{}
	if pbReq != nil {
		req.SubscriptionId = pbReq.GetSubscriptionId()
		req.CyclePeriodStart = pbReq.GetCyclePeriodStart()
		req.Backfill = pbReq.GetBackfill()
		req.UsageRequestDate = pbReq.GetUsageRequestDate()
	}
	internal, err := uc.executeInternal(ctx, req)
	if err != nil {
		return nil, err
	}
	if internal == nil {
		return &subscriptionpb.MaterializeInstanceJobsForSubscriptionResponse{Success: true}, nil
	}
	// Count spawned jobs across all cycles.
	var jobCount int32
	for _, c := range internal.SpawnedCycles {
		jobCount += int32(len(c.Jobs))
	}
	resp := &subscriptionpb.MaterializeInstanceJobsForSubscriptionResponse{
		Success:                   true,
		SpawnedCycleCount:         int32(len(internal.SpawnedCycles)),
		SpawnedJobCount:           jobCount,
		OnceAtStartJobCount:       int32(len(internal.OnceAtStartJobs)),
		EngagementWasNewlyCreated: internal.EngagementWasNewlyCreated,
		BackfillCappedAt:          internal.BackfillCappedAt,
	}
	if internal.SkippedReason != "" {
		v := internal.SkippedReason
		resp.SkippedReason = &v
	}
	return resp, nil
}

// executeInternal drives the full cycle-spawn flow per plan §3. The whole §3.2 → §3.5
// chain runs in a single transaction. Returns the rich internal response for callers
// (composition adapter, tests) that need cycle-level detail.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) executeInternal(
	ctx context.Context, req materializeInstanceJobsInternalRequest,
) (*materializeInstanceJobsInternalResponse, error) {
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
			"subscription.errors.materialize_instance_jobs_repositories_unavailable",
			"materialize_instance_jobs_for_subscription is missing required repositories [DEFAULT]",
		))
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

	// Plan §3.1 — eligibility gate.
	if !eligibleForInstanceSpawn(pricePlan) {
		return &materializeInstanceJobsInternalResponse{
			SkippedReason: InstanceSkipReasonNonCyclicPlan,
		}, nil
	}
	if pricePlan.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_MILESTONE {
		// Defensive — should never reach here because PricePlan-edit blocks
		// MILESTONE × cyclic. See plan §6 / §C.2.
		return &materializeInstanceJobsInternalResponse{
			SkippedReason: InstanceSkipReasonMilestoneUnsupported,
		}, nil
	}
	templateID := plan.GetJobTemplateId()
	if templateID == "" {
		// AD_HOC has its own skip reason since the validator check differs
		// (pool_no_template + pay_per_call_no_template — see ad-hoc plan §6).
		if IsAdHoc(pricePlan) {
			return &materializeInstanceJobsInternalResponse{
				SkippedReason: InstanceSkipReasonAdHocNoTemplate,
			}, nil
		}
		return &materializeInstanceJobsInternalResponse{
			SkippedReason: InstanceSkipReasonNoTemplate,
		}, nil
	}

	now := time.Now()
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)

	// AD_HOC dispatch — fully separate write path. The cyclic algorithm
	// below this branch is only invoked for RECURRING / CONTRACT-with-cycle.
	// See ad-hoc plan §3.2.
	if IsAdHoc(pricePlan) {
		return uc.executeAdHoc(ctx, dc, dcs, now, sub, pricePlan, plan, req)
	}

	// Compute cycle window list.
	cycleStarts, cappedAt, err := uc.computeCyclesToSpawn(ctx, sub, pricePlan, req, now)
	if err != nil {
		return nil, err
	}

	// In single-cycle mode (Backfill=false) with no pending cycles, surface
	// "no_pending_cycles" to the caller. The recognize-piggyback site treats
	// this as a no-op.
	if !req.Backfill && len(cycleStarts) == 0 {
		return &materializeInstanceJobsInternalResponse{
			SkippedReason: InstanceSkipReasonNoPendingCycles,
		}, nil
	}

	visitsPerCycle := plan.GetVisitsPerCycle()
	if visitsPerCycle < 1 {
		visitsPerCycle = 1
	}

	resp := &materializeInstanceJobsInternalResponse{
		BackfillCappedAt: cappedAt,
	}

	writeFn := func(txCtx context.Context) error {
		// Reset on retry/replay (transactional services may invoke twice).
		resp.SpawnedCycles = resp.SpawnedCycles[:0]
		resp.OnceAtStartJobs = resp.OnceAtStartJobs[:0]
		resp.EngagementWasNewlyCreated = false

		// Plan §3.2 — find or create shell Job (the "shell").
		shellJob, isNew, err := uc.findOrCreateShellJob(txCtx, dc, dcs, sub, pricePlan)
		if err != nil {
			return err
		}
		resp.ShellJob = shellJob
		resp.EngagementWasNewlyCreated = isNew

		// Plan §3.5 — spawn ONCE_AT_ENGAGEMENT_START children only on the very
		// first call (no cycle Jobs yet). Idempotent on re-run because the
		// existence check is based on counting cycle Jobs (cycle_index!=NULL)
		// — once any cycle exists, this branch is skipped.
		isFirstEverCall, err := uc.isFirstEverCall(txCtx, sub.GetId(), shellJob.GetId())
		if err != nil {
			return err
		}
		if isFirstEverCall {
			onceJobs, err := uc.spawnOnceAtShellStart(txCtx, dc, dcs, sub, pricePlan, plan, shellJob)
			if err != nil {
				return err
			}
			resp.OnceAtStartJobs = onceJobs
		}

		// Plan §3.4 — spawn cycle Jobs. Cycle index = current count + 1, which
		// is monotone within engagement.
		for _, billingCycleStart := range cycleStarts {
			billingCycleEnd, err := computeCycleEnd(billingCycleStart, pricePlan)
			if err != nil {
				return err
			}

			// Sub-cycle windows for multi-visit plans.
			subWindows := splitCycleWindow(billingCycleStart, billingCycleEnd, visitsPerCycle)

			cycleEntry := spawnedInstanceCycle{
				CyclePeriodStart: billingCycleStart,
				CyclePeriodEnd:   billingCycleEnd,
			}

			for _, win := range subWindows {
				// Cycle index = highest-existing + 1. Re-compute per visit so
				// multi-visit plans get monotonically increasing indices.
				nextIdx, err := uc.nextCycleIndex(txCtx, sub.GetId(), shellJob.GetId())
				if err != nil {
					return err
				}

				// Idempotency: skip if a cycle Job already exists for this
				// (origin_id, cycle_period_start). The DB-level partial unique
				// index is the canonical guard; this read-side check avoids
				// the wasted INSERT on the common single-thread case.
				existing, err := uc.findExistingCycleJob(txCtx, sub.GetId(), win.start)
				if err != nil {
					return err
				}
				if existing != nil {
					cycleEntry.Jobs = append(cycleEntry.Jobs, existing)
					if cycleEntry.CycleIndex == 0 {
						cycleEntry.CycleIndex = existing.GetCycleIndex()
					}
					continue
				}

				job, err := uc.spawnCycleJob(txCtx, dc, dcs, sub, pricePlan, plan, shellJob,
					nextIdx, win.start, win.end)
				if err != nil {
					return err
				}
				cycleEntry.Jobs = append(cycleEntry.Jobs, job)
				if cycleEntry.CycleIndex == 0 {
					cycleEntry.CycleIndex = nextIdx
				}

				if err := uc.spawnPhasesAndTasks(txCtx, dc, dcs, job, templateID); err != nil {
					return err
				}
			}

			resp.SpawnedCycles = append(resp.SpawnedCycles, cycleEntry)
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

	return resp, nil
}

// ---- helpers — read side ----

func (uc *MaterializeInstanceJobsForSubscriptionUseCase) readSubscription(
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

func (uc *MaterializeInstanceJobsForSubscriptionUseCase) readPricePlan(
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

func (uc *MaterializeInstanceJobsForSubscriptionUseCase) readPlan(
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

func (uc *MaterializeInstanceJobsForSubscriptionUseCase) readJobTemplate(
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

// listExistingJobsForOrigin returns every Job whose origin_id == sub.id. Used
// for shell-Job lookup, cycle-index counting, and idempotency. We do
// table-scan filtering in Go because the proto's ListJobs filter shape is the
// generic FilterRequest — applying a string-equals filter on origin_id keeps
// the call cheap for typical engagements (10-100 cycles).
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) listExistingJobsForOrigin(
	ctx context.Context, originID string,
) ([]*jobpb.Job, error) {
	resp, err := uc.repositories.Job.ListJobs(ctx, &jobpb.ListJobsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "origin_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    originID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list_jobs_for_origin (%s): %w", originID, err)
	}
	if resp == nil {
		return nil, nil
	}
	return resp.GetData(), nil
}

// findOrCreateShellJob returns the shell Job (parent_job_id IS
// NULL, origin = SUBSCRIPTION/sub.id), creating one if it doesn't yet exist
// (retroactive path for subscriptions created before this plan landed).
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) findOrCreateShellJob(
	ctx context.Context, dc int64, dcs string,
	sub *subscriptionpb.Subscription, pricePlan *priceplanpb.PricePlan,
) (*jobpb.Job, bool, error) {
	rows, err := uc.listExistingJobsForOrigin(ctx, sub.GetId())
	if err != nil {
		return nil, false, err
	}
	for _, j := range rows {
		if j.GetOriginType() != enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION {
			continue
		}
		if j.GetParentJobId() != "" {
			continue // child Job (cycle or once-at-start)
		}
		if j.GetCycleIndex() != 0 {
			// Defensive — shell Job never carries a cycle index.
			continue
		}
		return j, false, nil
	}

	// Shell Job missing — create one. Carries no template, no phases.
	jobID := ""
	if uc.services.IDService != nil {
		jobID = uc.services.IDService.GenerateID()
	} else {
		jobID = fmt.Sprintf("eng-%d", time.Now().UnixNano())
	}
	originID := sub.GetId()
	clientID := sub.GetClientId()
	job := &jobpb.Job{
		Id:                 jobID,
		Name:               shellJobName(sub),
		OriginType:         enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:           &originID,
		ClientId:           &clientID,
		Status:             enumspb.JobStatus_JOB_STATUS_ACTIVE, // shell stays open for life of subscription (plan §3.2 maps "IN_PROGRESS" → ACTIVE in this enum)
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
	_ = pricePlan // future: stamp currency on shell once Job carries one
	resp, err := uc.repositories.Job.CreateJob(ctx, &jobpb.CreateJobRequest{Data: job})
	if err != nil {
		return nil, false, fmt.Errorf("create_shell_job: %w", err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0], true, nil
	}
	return job, true, nil
}

// isFirstEverCall returns true when no cycle Jobs (cycle_index != NULL,
// parent_job_id == shell.id) exist yet. ONCE_AT_ENGAGEMENT_START spawns
// only on this path. Onboarding child Jobs (parent=shell, cycle_index=NULL)
// don't disqualify the first-call flag — they live alongside cycles, not within
// them.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) isFirstEverCall(
	ctx context.Context, originID, shellJobID string,
) (bool, error) {
	rows, err := uc.listExistingJobsForOrigin(ctx, originID)
	if err != nil {
		return false, err
	}
	for _, j := range rows {
		if j.GetParentJobId() != shellJobID {
			continue
		}
		if j.GetCycleIndex() == 0 {
			// Onboarding child — skip when deciding "first ever call".
			continue
		}
		return false, nil
	}
	return true, nil
}

// nextCycleIndex returns the cycle_index for the next cycle Job to spawn. Equal
// to highest-existing-cycle_index + 1. Monotone within engagement; gaps are
// allowed when subscription was paused mid-history.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) nextCycleIndex(
	ctx context.Context, originID, shellJobID string,
) (int32, error) {
	rows, err := uc.listExistingJobsForOrigin(ctx, originID)
	if err != nil {
		return 0, err
	}
	var maxIdx int32
	for _, j := range rows {
		if j.GetParentJobId() != shellJobID {
			continue
		}
		idx := j.GetCycleIndex()
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	return maxIdx + 1, nil
}

// findExistingCycleJob returns the cycle Job (if any) whose cycle_period_start
// matches `start`. Read-side mirror of the partial unique index from plan §3.6.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) findExistingCycleJob(
	ctx context.Context, originID, start string,
) (*jobpb.Job, error) {
	rows, err := uc.listExistingJobsForOrigin(ctx, originID)
	if err != nil {
		return nil, err
	}
	for _, j := range rows {
		if j.GetParentJobId() == "" {
			continue
		}
		if j.GetCyclePeriodStart() == start && start != "" {
			return j, nil
		}
	}
	return nil, nil
}

// ---- helpers — write side ----

// spawnCycleJob creates one cycle-instance Job (and, for multi-visit plans,
// one of N sub-cycle Jobs sharing a billing cycle).
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) spawnCycleJob(
	ctx context.Context, dc int64, dcs string,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	plan *planpb.Plan,
	shellJob *jobpb.Job,
	cycleIdx int32, periodStart, periodEnd string,
) (*jobpb.Job, error) {
	tpl, err := uc.readJobTemplate(ctx, plan.GetJobTemplateId())
	if err != nil {
		return nil, err
	}
	if !tpl.GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.template_inactive",
			"job template is inactive [DEFAULT]",
		))
	}

	jobID := ""
	if uc.services.IDService != nil {
		jobID = uc.services.IDService.GenerateID()
	} else {
		jobID = fmt.Sprintf("cyc-%d", time.Now().UnixNano())
	}
	templateID := tpl.GetId()
	originID := sub.GetId()
	clientID := sub.GetClientId()
	parentID := shellJob.GetId()

	// Cycle Jobs are operational only — billing fires from PricePlan cadence
	// via recognize-revenue, NOT from cycle-Job phase completion.
	billingRule := enumspb.BillingRuleType_BILLING_RULE_TYPE_NON_BILLABLE

	cycleIdxLocal := cycleIdx
	periodStartLocal := periodStart
	periodEndLocal := periodEnd

	job := &jobpb.Job{
		Id:                 jobID,
		Name:               cycleJobName(sub, cycleIdx),
		JobTemplateId:      &templateID,
		OriginType:         enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:           &originID,
		ClientId:           &clientID,
		Status:             enumspb.JobStatus_JOB_STATUS_PLANNED,
		BillingRuleType:    billingRule,
		Active:             true,
		ParentJobId:        &parentID,
		CycleIndex:         &cycleIdxLocal,
		CyclePeriodStart:   &periodStartLocal,
		CyclePeriodEnd:     &periodEndLocal,
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
	_ = pricePlan // currency lives on Revenue; cycle Job carries none today

	resp, err := uc.repositories.Job.CreateJob(ctx, &jobpb.CreateJobRequest{Data: job})
	if err != nil {
		return nil, fmt.Errorf("create_cycle_job (idx=%d, period_start=%s): %w", cycleIdx, periodStart, err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0], nil
	}
	return job, nil
}

// spawnOnceAtShellStart creates child Jobs from JobTemplateRelation rows
// whose relation_type=ONCE_AT_ENGAGEMENT_START. Carries cycle_index=NULL so
// the shell-level rollup separates onboarding from cycles.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) spawnOnceAtShellStart(
	ctx context.Context, dc int64, dcs string,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	plan *planpb.Plan,
	shellJob *jobpb.Job,
) ([]*jobpb.Job, error) {
	if uc.repositories.JobTemplateRelation == nil {
		return nil, nil
	}
	resp, err := uc.repositories.JobTemplateRelation.ListByParent(ctx,
		&jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest{
			ParentTemplateId: plan.GetJobTemplateId(),
		})
	if err != nil {
		return nil, fmt.Errorf("list_relations_for_shell_start: %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	rels := resp.GetJobTemplateRelations()
	sort.SliceStable(rels, func(i, j int) bool {
		return rels[i].GetSequenceOrder() < rels[j].GetSequenceOrder()
	})

	var spawned []*jobpb.Job
	for _, rel := range rels {
		if !rel.GetActive() {
			continue
		}
		if rel.GetRelationType() != jobtemplaterelationpb.JobTemplateRelationType_JOB_TEMPLATE_RELATION_TYPE_ONCE_AT_ENGAGEMENT_START {
			continue
		}
		childTplID := rel.GetChildTemplateId()
		if childTplID == "" {
			continue
		}
		tpl, err := uc.readJobTemplate(ctx, childTplID)
		if err != nil {
			return nil, err
		}
		if !tpl.GetActive() {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.TranslationService,
				"subscription.errors.template_inactive",
				"job template is inactive [DEFAULT]",
			))
		}

		jobID := ""
		if uc.services.IDService != nil {
			jobID = uc.services.IDService.GenerateID()
		} else {
			jobID = fmt.Sprintf("onb-%d", time.Now().UnixNano())
		}
		tplID := tpl.GetId()
		originID := sub.GetId()
		clientID := sub.GetClientId()
		parentID := shellJob.GetId()

		job := &jobpb.Job{
			Id:              jobID,
			Name:            tpl.GetName(),
			JobTemplateId:   &tplID,
			OriginType:      enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
			OriginId:        &originID,
			ClientId:        &clientID,
			Status:          enumspb.JobStatus_JOB_STATUS_PLANNED,
			BillingRuleType: enumspb.BillingRuleType_BILLING_RULE_TYPE_NON_BILLABLE,
			Active:          true,
			ParentJobId:     &parentID,
			// cycle_* fields intentionally NULL — onboarding fires once, not per cycle
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
		if tplID != "" {
			v := tplID
			job.JobTemplateRevisionId = &v
		}
		_ = pricePlan

		respCreate, err := uc.repositories.Job.CreateJob(ctx, &jobpb.CreateJobRequest{Data: job})
		if err != nil {
			return nil, fmt.Errorf("create_onboarding_job (template=%s): %w", tplID, err)
		}
		var created *jobpb.Job
		if respCreate != nil && len(respCreate.GetData()) > 0 {
			created = respCreate.GetData()[0]
		} else {
			created = job
		}

		if err := uc.spawnPhasesAndTasks(ctx, dc, dcs, created, tplID); err != nil {
			return nil, err
		}
		spawned = append(spawned, created)
	}
	return spawned, nil
}

// spawnPhasesAndTasks materialises JobPhase + JobTask rows from a JobTemplate
// (mirrors MaterializeJobsForSubscription.spawnPhasesAndTasks). Predecessor
// phase IDs are remapped from template-phase IDs to the freshly minted phase
// IDs.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) spawnPhasesAndTasks(
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

// ---- helpers — cycle math ----

// computeCyclesToSpawn returns the list of billing-cycle period_start dates
// to materialise. In single-cycle mode (Backfill=false): exactly one entry
// (or zero when the requested cycle is already present and req.CyclePeriodStart
// is empty). In backfill mode: every missing cycle from sub.date_time_start
// up to today, capped at MaxBackfillCycles.
//
// Returns the cycle starts + the cap that fired (0 when uncapped).
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) computeCyclesToSpawn(
	ctx context.Context,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	req materializeInstanceJobsInternalRequest,
	now time.Time,
) ([]string, int32, error) {
	if !req.Backfill {
		// Caller-supplied cycle_period_start wins. Otherwise compute the next
		// un-spawned cycle from sub.date_time_start.
		if req.CyclePeriodStart != "" {
			// Idempotent: if this cycle already exists, return empty list so
			// the caller surfaces "no_pending_cycles".
			existing, err := uc.findExistingCycleJob(ctx, sub.GetId(), req.CyclePeriodStart)
			if err != nil {
				return nil, 0, err
			}
			if existing != nil {
				return nil, 0, nil
			}
			return []string{req.CyclePeriodStart}, 0, nil
		}
		next, err := uc.computeNextUnspawnedCycle(ctx, sub, pricePlan, now)
		if err != nil {
			return nil, 0, err
		}
		if next == "" {
			return nil, 0, nil
		}
		return []string{next}, 0, nil
	}

	// Backfill mode — walk the cycle window.
	starts, err := uc.listAllCycleStarts(ctx, sub.GetId())
	if err != nil {
		return nil, 0, err
	}
	have := make(map[string]bool, len(starts))
	for _, s := range starts {
		have[s] = true
	}

	cursor, err := subscriptionStartDate(sub)
	if err != nil {
		return nil, 0, err
	}
	var out []string
	var cap_ int32
	today := now.UTC().Truncate(24 * time.Hour)
	for cursor.After(today) == false {
		date := cursor.Format("2006-01-02")
		if !have[date] {
			out = append(out, date)
			if int32(len(out)) >= MaxBackfillCycles {
				cap_ = MaxBackfillCycles
				break
			}
		}
		nextCursor, err := addCycle(cursor, pricePlan)
		if err != nil {
			return nil, 0, err
		}
		if !nextCursor.After(cursor) {
			// Defensive — cycle math returned non-advancing date; bail to avoid
			// infinite loop.
			break
		}
		cursor = nextCursor
	}
	return out, cap_, nil
}

// listAllCycleStarts returns the cycle_period_start dates of every existing
// cycle Job under this subscription.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) listAllCycleStarts(
	ctx context.Context, originID string,
) ([]string, error) {
	rows, err := uc.listExistingJobsForOrigin(ctx, originID)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, j := range rows {
		if j.GetParentJobId() == "" {
			continue
		}
		if s := j.GetCyclePeriodStart(); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// computeNextUnspawnedCycle returns the period_start of the next cycle that
// hasn't been materialised. Returns "" when every cycle up to today exists.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) computeNextUnspawnedCycle(
	ctx context.Context, sub *subscriptionpb.Subscription, pricePlan *priceplanpb.PricePlan, now time.Time,
) (string, error) {
	starts, err := uc.listAllCycleStarts(ctx, sub.GetId())
	if err != nil {
		return "", err
	}
	have := make(map[string]bool, len(starts))
	for _, s := range starts {
		have[s] = true
	}
	cursor, err := subscriptionStartDate(sub)
	if err != nil {
		return "", err
	}
	today := now.UTC().Truncate(24 * time.Hour)
	for cursor.After(today) == false {
		date := cursor.Format("2006-01-02")
		if !have[date] {
			return date, nil
		}
		next, err := addCycle(cursor, pricePlan)
		if err != nil {
			return "", err
		}
		if !next.After(cursor) {
			break
		}
		cursor = next
	}
	return "", nil
}

// subscriptionStartDate returns the UTC start-of-day of sub.date_time_start.
// Falls back to time.Now() if the field is nil (defensive — should never
// happen for a valid cyclic subscription).
func subscriptionStartDate(sub *subscriptionpb.Subscription) (time.Time, error) {
	if sub.GetDateTimeStart() != nil {
		return sub.GetDateTimeStart().AsTime().UTC().Truncate(24 * time.Hour), nil
	}
	return time.Now().UTC().Truncate(24 * time.Hour), nil
}

// addCycle advances `t` by one billing cycle as defined by the PricePlan's
// billing_cycle_value × billing_cycle_unit.
func addCycle(t time.Time, pricePlan *priceplanpb.PricePlan) (time.Time, error) {
	value := int(pricePlan.GetBillingCycleValue())
	if value <= 0 {
		value = 1
	}
	unit := strings.ToLower(strings.TrimSpace(pricePlan.GetBillingCycleUnit()))
	switch unit {
	case "day", "days", "":
		return t.AddDate(0, 0, value), nil
	case "week", "weeks":
		return t.AddDate(0, 0, 7*value), nil
	case "month", "months":
		return t.AddDate(0, value, 0), nil
	case "quarter", "quarters":
		return t.AddDate(0, 3*value, 0), nil
	case "year", "years":
		return t.AddDate(value, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported billing_cycle_unit %q", unit)
	}
}

// computeCycleEnd returns the inclusive end date of a billing cycle given its
// start (ISO 8601 YYYY-MM-DD) and the PricePlan's cycle cadence. End = start +
// cycle_length - 1 day.
func computeCycleEnd(start string, pricePlan *priceplanpb.PricePlan) (string, error) {
	t, err := time.Parse("2006-01-02", start)
	if err != nil {
		return "", fmt.Errorf("parse cycle_period_start %q: %w", start, err)
	}
	next, err := addCycle(t, pricePlan)
	if err != nil {
		return "", err
	}
	end := next.AddDate(0, 0, -1)
	return end.Format("2006-01-02"), nil
}

// subWindow describes one sub-cycle within a billing cycle (1-of-N for
// multi-visit plans).
type subWindow struct {
	start string
	end   string
}

// splitCycleWindow divides [billingStart, billingEnd] into N equal sub-windows
// (rounded to ISO date). The last sub-window absorbs the remainder so the full
// billing cycle is covered.
//
// For visitsPerCycle=1 returns one window equal to the full billing cycle.
func splitCycleWindow(billingStart, billingEnd string, visitsPerCycle int32) []subWindow {
	if visitsPerCycle <= 1 {
		return []subWindow{{start: billingStart, end: billingEnd}}
	}
	startT, err := time.Parse("2006-01-02", billingStart)
	if err != nil {
		return []subWindow{{start: billingStart, end: billingEnd}}
	}
	endT, err := time.Parse("2006-01-02", billingEnd)
	if err != nil {
		return []subWindow{{start: billingStart, end: billingEnd}}
	}
	totalDays := int(endT.Sub(startT).Hours()/24) + 1 // inclusive
	if totalDays < int(visitsPerCycle) {
		// Edge case — fewer days than visits; collapse to one window.
		return []subWindow{{start: billingStart, end: billingEnd}}
	}
	stride := totalDays / int(visitsPerCycle)
	windows := make([]subWindow, 0, visitsPerCycle)
	cursor := startT
	for k := int32(0); k < visitsPerCycle; k++ {
		var winEnd time.Time
		if k == visitsPerCycle-1 {
			winEnd = endT
		} else {
			winEnd = cursor.AddDate(0, 0, stride-1)
		}
		windows = append(windows, subWindow{
			start: cursor.Format("2006-01-02"),
			end:   winEnd.Format("2006-01-02"),
		})
		cursor = winEnd.AddDate(0, 0, 1)
	}
	return windows
}

// shellJobName returns "<sub.name> (subscription shell)" or "(subscription shell)" when
// the subscription has no name.
func shellJobName(sub *subscriptionpb.Subscription) string {
	name := strings.TrimSpace(sub.GetName())
	if name == "" {
		return "(subscription shell)"
	}
	return name + " (subscription shell)"
}

// cycleJobName returns "<sub.name> — Cycle <idx>" or "Cycle <idx>" when the
// subscription has no name.
func cycleJobName(sub *subscriptionpb.Subscription, cycleIdx int32) string {
	name := strings.TrimSpace(sub.GetName())
	if name == "" {
		return fmt.Sprintf("Cycle %d", cycleIdx)
	}
	return fmt.Sprintf("%s — Cycle %d", name, cycleIdx)
}

// usageJobName returns "<sub.name> — Usage <ordinal>" (or just "Usage <ordinal>"
// when the subscription has no name). Vertical-neutral phrasing — the lyngua
// layer overrides "Usage" to "Visit" / "Appointment" / "Service Call" / etc.
// per business tier.
func usageJobName(sub *subscriptionpb.Subscription, ordinal int32) string {
	name := strings.TrimSpace(sub.GetName())
	if name == "" {
		return fmt.Sprintf("Usage %d", ordinal)
	}
	return fmt.Sprintf("%s — Usage %d", name, ordinal)
}

// ---- AD_HOC algorithm — executeAdHoc ----------------------------------------
//
// AD_HOC plans spawn one usage Job per operator request. Two variants:
//   - TOTAL_PACKAGE  (prepaid pool):     gate on resolvedEntitlement; no BillingEvent
//   - PER_OCCURRENCE (pay-per-call):     no gate; spawn paired BillingEvent
//
// Both variants share the engagement Job + ONCE_AT_ENGAGEMENT_START onboarding
// path with cyclic plans. They diverge from cycle math only at the per-usage
// spawn step.
//
// Idempotency: cycle_period_start carries the composite key
// `YYYY-MM-DD#NNNN` (date + zero-padded ordinal). The cyclic plan's partial
// unique index `(origin_id, cycle_period_start) WHERE parent_job_id IS NOT
// NULL AND cycle_period_start IS NOT NULL` extends to AD_HOC for free —
// codex CRIT-4. The usage_request_date + usage_ordinal companion columns
// expose the parts for sort/query.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) executeAdHoc(
	ctx context.Context, dc int64, dcs string, now time.Time,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	plan *planpb.Plan,
	req materializeInstanceJobsInternalRequest,
) (*materializeInstanceJobsInternalResponse, error) {
	basis := pricePlan.GetAmountBasis()
	if basis != priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE &&
		basis != priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE {
		return &materializeInstanceJobsInternalResponse{
			SkippedReason: InstanceSkipReasonAdHocInvalidBasis,
		}, nil
	}
	if basis == priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE && uc.repositories.BillingEvent == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.materialize_instance_jobs_billing_event_repo_required",
			"BillingEvent repository is required for AD_HOC × PER_OCCURRENCE [DEFAULT]",
		))
	}

	// Resolve usage request date — operator-supplied wins, otherwise today UTC.
	requestDate := strings.TrimSpace(req.UsageRequestDate)
	if requestDate == "" {
		requestDate = now.UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", requestDate); err != nil {
		return nil, fmt.Errorf("parse usage_request_date %q: %w", requestDate, err)
	}

	resp := &materializeInstanceJobsInternalResponse{}

	writeFn := func(txCtx context.Context) error {
		// Reset on retry.
		resp.SpawnedCycles = resp.SpawnedCycles[:0]
		resp.OnceAtStartJobs = resp.OnceAtStartJobs[:0]
		resp.EngagementWasNewlyCreated = false
		resp.SkippedReason = ""

		shellJob, isNew, err := uc.findOrCreateShellJob(txCtx, dc, dcs, sub, pricePlan)
		if err != nil {
			return err
		}
		resp.ShellJob = shellJob
		resp.EngagementWasNewlyCreated = isNew

		// First-call onboarding (mirrors cyclic path). Onboarding fires once
		// per engagement regardless of kind.
		isFirstEverCall, err := uc.isFirstEverCallAdHoc(txCtx, sub.GetId(), shellJob.GetId())
		if err != nil {
			return err
		}
		if isFirstEverCall {
			onceJobs, err := uc.spawnOnceAtShellStart(txCtx, dc, dcs, sub, pricePlan, plan, shellJob)
			if err != nil {
				return err
			}
			resp.OnceAtStartJobs = onceJobs
		}

		// Count existing usage Jobs under this engagement. Equal to
		// "redeemed_count" per ad-hoc plan §2.4.
		used, err := uc.countUsageJobs(txCtx, sub.GetId(), shellJob.GetId())
		if err != nil {
			return err
		}

		// TOTAL_PACKAGE entitlement gate.
		if basis == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
			entitled := resolvedEntitlement(sub, pricePlan)
			if entitled <= 0 {
				resp.SkippedReason = InstanceSkipReasonEntitlementRequired
				return nil
			}
			if used >= entitled {
				resp.SkippedReason = InstanceSkipReasonEntitlementExhausted
				return nil
			}
		}

		ordinal := used + 1
		visitKey := fmt.Sprintf("%s#%04d", requestDate, ordinal)

		// Idempotency — partial unique index would catch a same-key INSERT.
		// Read-side check sidesteps the wasted INSERT in the common path.
		if existing, err := uc.findExistingCycleJob(txCtx, sub.GetId(), visitKey); err != nil {
			return err
		} else if existing != nil {
			resp.SpawnedCycles = append(resp.SpawnedCycles, spawnedInstanceCycle{
				CycleIndex:       existing.GetCycleIndex(),
				CyclePeriodStart: existing.GetCyclePeriodStart(),
				Jobs:             []*jobpb.Job{existing},
			})
			return nil
		}

		usageJob, err := uc.spawnUsageJob(txCtx, dc, dcs, sub, pricePlan, plan, shellJob,
			ordinal, requestDate, visitKey, basis)
		if err != nil {
			return err
		}
		if err := uc.spawnPhasesAndTasks(txCtx, dc, dcs, usageJob, plan.GetJobTemplateId()); err != nil {
			return err
		}

		// PER_OCCURRENCE: paired BillingEvent for per-visit billing audit.
		if basis == priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE {
			if err := uc.spawnAdHocBillingEvent(txCtx, dc, sub, pricePlan, usageJob); err != nil {
				return err
			}
		}

		resp.SpawnedCycles = append(resp.SpawnedCycles, spawnedInstanceCycle{
			CycleIndex:       ordinal,
			CyclePeriodStart: visitKey,
			Jobs:             []*jobpb.Job{usageJob},
		})
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
	return resp, nil
}

// countUsageJobs returns the number of usage Jobs (parent_job_id == shellJob,
// usage_ordinal != NULL) under this subscription. Onboarding children are
// excluded because they have no usage_ordinal. Equivalent to ad-hoc plan
// §2.4's `COUNT(*)` query.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) countUsageJobs(
	ctx context.Context, originID, shellJobID string,
) (int32, error) {
	rows, err := uc.listExistingJobsForOrigin(ctx, originID)
	if err != nil {
		return 0, err
	}
	var count int32
	for _, j := range rows {
		if j.GetParentJobId() != shellJobID {
			continue
		}
		if j.GetUsageOrdinal() == 0 {
			continue
		}
		count++
	}
	return count, nil
}

// isFirstEverCallAdHoc mirrors isFirstEverCall but discriminates on
// usage_ordinal rather than cycle_index. AD_HOC + cyclic engagements never
// coexist (validator blocks AD_HOC × cycle_value > 0), so the two predicates
// don't interact.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) isFirstEverCallAdHoc(
	ctx context.Context, originID, shellJobID string,
) (bool, error) {
	rows, err := uc.listExistingJobsForOrigin(ctx, originID)
	if err != nil {
		return false, err
	}
	for _, j := range rows {
		if j.GetParentJobId() != shellJobID {
			continue
		}
		if j.GetUsageOrdinal() == 0 {
			continue
		}
		return false, nil
	}
	return true, nil
}

// spawnUsageJob writes one AD_HOC usage Job. cycle_period_start carries the
// composite uniqueness key `YYYY-MM-DD#NNNN`; the date and ordinal also live
// in usage_request_date + usage_ordinal for sort/query convenience.
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) spawnUsageJob(
	ctx context.Context, dc int64, dcs string,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	plan *planpb.Plan,
	shellJob *jobpb.Job,
	ordinal int32, requestDate, visitKey string,
	basis priceplanpb.AmountBasis,
) (*jobpb.Job, error) {
	tpl, err := uc.readJobTemplate(ctx, plan.GetJobTemplateId())
	if err != nil {
		return nil, err
	}
	if !tpl.GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.template_inactive",
			"job template is inactive [DEFAULT]",
		))
	}

	jobID := ""
	if uc.services.IDService != nil {
		jobID = uc.services.IDService.GenerateID()
	} else {
		jobID = fmt.Sprintf("usg-%d", time.Now().UnixNano())
	}
	templateID := tpl.GetId()
	originID := sub.GetId()
	clientID := sub.GetClientId()
	parentID := shellJob.GetId()

	// AD_HOC usage Jobs carry NON_BILLABLE because the billing trigger lives
	// on the paired BillingEvent (for PER_OCCURRENCE) or on the pool invoice
	// fired at Subscription.Create (for TOTAL_PACKAGE). Job-level rule type
	// is operational metadata only.
	billingRule := enumspb.BillingRuleType_BILLING_RULE_TYPE_NON_BILLABLE
	_ = basis

	ordinalLocal := ordinal
	visitKeyLocal := visitKey
	_ = requestDate // captured in composite cycle_period_start; usage_request_date
	// DB column is DATE which round-trips back as epoch ms via dbOps,
	// breaking the proto string field on read. Companion column kept in the
	// schema for direct SQL queries (DB→view-layer joins, dashboards) but
	// not persisted via dbOps until the proto type is reconciled (v1.5
	// follow-up). usage_ordinal (INTEGER) round-trips cleanly so that one
	// stays wired.

	job := &jobpb.Job{
		Id:              jobID,
		Name:            usageJobName(sub, ordinal),
		JobTemplateId:   &templateID,
		OriginType:      enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION,
		OriginId:        &originID,
		ClientId:        &clientID,
		Status:          enumspb.JobStatus_JOB_STATUS_PLANNED,
		BillingRuleType: billingRule,
		Active:          true,
		ParentJobId:     &parentID,
		// AD_HOC reuses cycle_index as the per-engagement ordinal and
		// cycle_period_start as the composite uniqueness key
		// (`YYYY-MM-DD#NNNN`). The view layer parses back to date+ordinal
		// via the composite + usage_ordinal field.
		CycleIndex:         &ordinalLocal,
		CyclePeriodStart:   &visitKeyLocal,
		UsageOrdinal:       &ordinalLocal,
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
	_ = pricePlan

	respCreate, err := uc.repositories.Job.CreateJob(ctx, &jobpb.CreateJobRequest{Data: job})
	if err != nil {
		return nil, fmt.Errorf("create_usage_job (ordinal=%d, key=%s): %w", ordinal, visitKey, err)
	}
	if respCreate != nil && len(respCreate.GetData()) > 0 {
		return respCreate.GetData()[0], nil
	}
	return job, nil
}

// spawnAdHocBillingEvent creates a BillingEvent paired to a PER_OCCURRENCE
// usage Job. status=UNSPECIFIED; trigger=UNSPECIFIED; flips to READY +
// VISIT_COMPLETED when the visit's last phase completes (Phase C hook).
func (uc *MaterializeInstanceJobsForSubscriptionUseCase) spawnAdHocBillingEvent(
	ctx context.Context, dc int64,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	usageJob *jobpb.Job,
) error {
	eventID := ""
	if uc.services.IDService != nil {
		eventID = uc.services.IDService.GenerateID()
	} else {
		eventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	jobID := usageJob.GetId()
	dcLocal := dc
	dcsLocal := time.UnixMilli(dc).UTC().Format(time.RFC3339)

	event := &billingeventpb.BillingEvent{
		Id:                 eventID,
		SubscriptionId:     sub.GetId(),
		JobId:              &jobID,
		BillableAmount:     pricePlan.GetBillingAmount(),
		BillingCurrency:    pricePlan.GetBillingCurrency(),
		Status:             billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_UNSPECIFIED,
		Trigger:            billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_UNSPECIFIED,
		Active:             true,
		DateCreated:        &dcLocal,
		DateCreatedString:  &dcsLocal,
		DateModified:       &dcLocal,
		DateModifiedString: &dcsLocal,
	}
	if _, err := uc.repositories.BillingEvent.CreateBillingEvent(ctx,
		&billingeventpb.CreateBillingEventRequest{Data: event}); err != nil {
		return fmt.Errorf("create_ad_hoc_billing_event (job=%s): %w", jobID, err)
	}
	return nil
}
