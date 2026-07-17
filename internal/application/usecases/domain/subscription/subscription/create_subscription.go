package subscription

import (
	"context"
	"errors"
	"log"
	"time"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

type CreateSubscriptionRepositories struct {
	Subscription  subscriptionpb.SubscriptionDomainServiceServer
	Client        clientpb.ClientDomainServiceServer
	PricePlan     priceplanpb.PricePlanDomainServiceServer
	Plan          planpb.PlanDomainServiceServer
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer
}

type CreateSubscriptionServices struct {
	Authorizer              ports.Authorizer
	Transactor              ports.Transactor
	Translator              ports.Translator
	ActionGatekeeper        *actiongate.ActionGatekeeper
	IDGenerator             ports.IDGenerator
	JobTemplateInstantiator JobTemplateInstantiator
	// CodeFormat is the SUBSCRIPTION_CODE_FORMAT template threaded from the
	// composition root (.env). Empty or "auto" -> DefaultCodeFormat. The domain
	// layer never reads env directly; this value is injected.
	CodeFormat string
}

// CreateSubscriptionUseCase handles the business logic for creating subscriptions
type CreateSubscriptionUseCase struct {
	repositories CreateSubscriptionRepositories
	services     CreateSubscriptionServices
}

// NewCreateSubscriptionUseCase creates a new CreateSubscriptionUseCase
func NewCreateSubscriptionUseCase(
	repositories CreateSubscriptionRepositories,
	services CreateSubscriptionServices,
) *CreateSubscriptionUseCase {
	return &CreateSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create subscription operation
func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Subscription,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.data_required", "[ERR-DEFAULT] Subscription data is required"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation — also returns the PricePlan so we can read plan_id later.
	pricePlan, err := uc.validateEntityReferences(ctx, req.Data)
	if err != nil {
		return nil, err
	}

	// Auto-generate subscription.code when the caller did not supply one.
	// Never overwrite a caller-supplied code, and never block creation on a
	// generation lookup failure (best-effort; blank tokens for the rest).
	if req.Data != nil && req.Data.GetCode() == "" {
		generated := uc.generateSubscriptionCode(ctx, req.Data, pricePlan)
		req.Data.Code = &generated
	}

	// Business enrichment
	enrichedSubscription := uc.applyBusinessLogic(req.Data)

	// 2026-04-29 auto-spawn-jobs-from-subscription plan §5.1 — the operator's
	// "Spawn Jobs on Create" toggle is propagated from the centymo view layer
	// via context (see espyna shared/context/spawn_jobs.go). When unset, fall
	// back to true to preserve the legacy default-on behavior for any callers
	// that did not adopt the toggle yet.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	spawnJobs := true
	if override, set := contextutil.ExtractSpawnJobsOverride(ctx); set {
		spawnJobs = override
	}

	// Q-GSE-8 (require_spawn_success + owner nuance): the spawn is only REQUIRED
	// when the plan graph declares a root template (plan.job_template_id set) AND
	// the operator did not opt out. NOT all verticals/subscriptions materialize
	// jobs — a plan with no root template is a CLEAN SKIP, never an error, even
	// under require_spawn_success.
	//
	// STRICT path is FAIL-CLOSED: when require_spawn_success is set we may only
	// clean-skip on a SUCCESSFUL plan read that proves job_template_id is empty.
	// A nil instantiator, a missing Plan repository, or a plan read error cannot
	// silently downgrade to the legacy best-effort path (which returns SUCCESS
	// with zero jobs); each fails closed with a translated error. When
	// require_spawn_success is unset, none of this runs and the legacy path below
	// executes byte-for-byte.
	requireSpawn := req.GetRequireSpawnSuccess()
	if requireSpawn && spawnJobs {
		// Fail closed: a required spawn with no instantiator wired cannot be
		// verified or performed.
		if uc.services.JobTemplateInstantiator == nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"subscription.errors.spawn_required_no_instantiator",
				"[ERR-DEFAULT] Job spawn was required but no instantiator is configured",
			))
		}
		declared, err := uc.planDeclaresRootTemplate(ctx, pricePlan)
		if err != nil {
			// Could not verify whether the plan declares a root template (nil
			// price plan / missing Plan repo / read error / plan not found) ->
			// fail closed, never downgrade to best-effort.
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"subscription.errors.spawn_required_unverifiable",
				"[ERR-DEFAULT] Job spawn was required but the plan's spawn requirement could not be verified",
			))
		}
		if declared {
			// A root template is declared: the spawn is REQUIRED and must be
			// atomic with the insert. Without a usable Transactor the strict path
			// would INSERT the subscription and only then detect an empty/failed
			// spawn (non-atomic, unsafe to retry). Reject BEFORE creating anything.
			if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
					ctx, uc.services.Translator,
					"subscription.errors.spawn_required_no_transaction",
					"[ERR-DEFAULT] Job spawn was required but no transaction is available to create the subscription atomically",
				))
			}
			// Strict path: create + spawn are bound so an empty/failed spawn rolls
			// the subscription back. Every other caller keeps the legacy
			// best-effort post-commit behavior below, byte-for-byte.
			return uc.executeWithRequiredSpawn(ctx, req, enrichedSubscription, pricePlan, wsID, spawnJobs)
		}
		// declared == false with no error: the read succeeded and proved the plan
		// declares no root template -> CLEAN SKIP. Fall through to the legacy path.
	}

	// Legacy path — byte-unchanged behavior. Use transaction service if available.
	var resp *subscriptionpb.CreateSubscriptionResponse
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		resp, err = uc.executeWithTransaction(ctx, req, enrichedSubscription)
	} else {
		// Fallback to non-transactional execution
		resp, err = uc.executeCore(ctx, req, enrichedSubscription)
	}
	if err != nil {
		return nil, err
	}

	// After successful creation, instantiate jobs from the plan (best-effort,
	// non-blocking). A spawn failure is logged, never fatal, exactly as before.
	// spawned_job_ids / spawn_skip_reason are populated additively from the
	// materialize result on success (Q-GSE-8); callers that ignore the fields are
	// unaffected.
	if uc.services.JobTemplateInstantiator != nil && pricePlan != nil {
		outcome, jiErr := uc.instantiateJobs(
			ctx, pricePlan.PlanId, enrichedSubscription.ClientId, enrichedSubscription.Id, wsID, spawnJobs,
		)
		if jiErr != nil {
			log.Printf("Warning: job instantiation failed for subscription %s: %v", enrichedSubscription.Id, jiErr)
			// Do not fail subscription creation — log and continue.
		} else if resp != nil {
			applySpawnOutcome(resp, outcome)
		}
	}

	return resp, nil
}

// executeWithRequiredSpawn is the Q-GSE-8 strict path: the plan declares a root
// template and the caller set require_spawn_success, so an empty (no jobs
// spawned) or failed materialize is a hard failure. The caller GUARANTEES a
// usable Transactor before invoking this (see Execute), so create + spawn always
// run in ONE transaction and an empty/failed spawn rolls the subscription back.
// There is no non-atomic fallback: a strict spawn without a usable Transactor is
// rejected before any insert.
func (uc *CreateSubscriptionUseCase) executeWithRequiredSpawn(
	ctx context.Context,
	req *subscriptionpb.CreateSubscriptionRequest,
	enrichedSubscription *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	wsID string,
	spawnJobs bool,
) (*subscriptionpb.CreateSubscriptionResponse, error) {
	createAndSpawn := func(txCtx context.Context) (*subscriptionpb.CreateSubscriptionResponse, error) {
		resp, err := uc.executeCore(txCtx, req, enrichedSubscription)
		if err != nil {
			return nil, err
		}
		outcome, spawnErr := uc.instantiateJobs(
			txCtx, pricePlan.PlanId, enrichedSubscription.ClientId, enrichedSubscription.Id, wsID, spawnJobs,
		)
		if spawnErr != nil {
			return nil, spawnErr
		}
		if len(outcome.SpawnedJobIDs) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				txCtx, uc.services.Translator,
				"subscription.errors.spawn_required_but_empty",
				"[ERR-DEFAULT] Job spawn was required but produced no jobs",
			))
		}
		if resp != nil {
			applySpawnOutcome(resp, outcome)
		}
		return resp, nil
	}

	// A usable Transactor is guaranteed by the caller — create + spawn commit or
	// roll back together.
	var result *subscriptionpb.CreateSubscriptionResponse
	if err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := createAndSpawn(txCtx)
		if err != nil {
			return err
		}
		result = res
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// planDeclaresRootTemplate reports whether the subscription's resolved Plan
// declares a root job template (plan.job_template_id set) — the Q-GSE-8 nuance
// that distinguishes a materialize-bearing plan from a clean-skip plan.
//
// TRI-STATE (fail-closed contract for the strict require_spawn_success path):
//   - (true, nil):  a SUCCESSFUL read proved the plan declares a root template
//     -> spawn is required.
//   - (false, nil): a SUCCESSFUL read proved the plan declares NO root template
//     (job_template_id empty), or the price plan carries no plan_id at all ->
//     clean skip.
//   - (_, err):     the requirement could NOT be verified — a nil price plan, a
//     missing Plan repository, a read error, or a plan that could not be found.
//     Strict-mode callers MUST fail closed and never downgrade to best-effort.
func (uc *CreateSubscriptionUseCase) planDeclaresRootTemplate(ctx context.Context, pricePlan *priceplanpb.PricePlan) (bool, error) {
	if pricePlan == nil {
		return false, errors.New("planDeclaresRootTemplate: nil price plan")
	}
	if uc.repositories.Plan == nil {
		return false, errors.New("planDeclaresRootTemplate: nil Plan repository")
	}
	planID := pricePlan.GetPlanId()
	if planID == "" {
		// No plan reference -> no root template can be declared. This is a genuine
		// determination (not a lookup failure), so it is a clean skip.
		return false, nil
	}
	resp, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
		Data: &planpb.Plan{Id: &planID},
	})
	if err != nil {
		return false, err
	}
	if resp == nil || len(resp.GetData()) == 0 {
		// The plan was referenced but could not be read/found — this does NOT
		// prove job_template_id is empty, so it must not clean-skip.
		return false, errors.New("planDeclaresRootTemplate: plan not found")
	}
	return resp.GetData()[0].GetJobTemplateId() != "", nil
}

// jobSpawnOutcome is the richer materialize result that Q-GSE-8 needs: the
// spawned Job IDs and the clean-skip reason (if any). The base
// JobTemplateInstantiator port returns only error; detailedJobTemplateInstantiator
// (implemented by the canonical adapter, see InstantiateJobsFromPlanDetailed
// below) surfaces the full result.
type jobSpawnOutcome struct {
	SpawnedJobIDs []string
	SkipReason    string
}

// detailedJobTemplateInstantiator is the optional richer extension of
// JobTemplateInstantiator. When the injected instantiator implements it, the
// create path can read the spawned Job IDs + skip reason; otherwise it falls
// back to the error-only base port (spawned_job_ids simply stays empty).
type detailedJobTemplateInstantiator interface {
	InstantiateJobsFromPlanDetailed(ctx context.Context, planID, clientID, subscriptionID, workspaceID string, spawnJobs bool) (jobSpawnOutcome, error)
}

// instantiateJobs invokes the configured instantiator, preferring the detailed
// variant so spawned_job_ids / spawn_skip_reason can be populated. Falls back to
// the error-only base port when the detailed extension is unavailable.
func (uc *CreateSubscriptionUseCase) instantiateJobs(
	ctx context.Context, planID, clientID, subscriptionID, workspaceID string, spawnJobs bool,
) (jobSpawnOutcome, error) {
	inst := uc.services.JobTemplateInstantiator
	if inst == nil {
		return jobSpawnOutcome{}, nil
	}
	if detailed, ok := inst.(detailedJobTemplateInstantiator); ok {
		return detailed.InstantiateJobsFromPlanDetailed(ctx, planID, clientID, subscriptionID, workspaceID, spawnJobs)
	}
	return jobSpawnOutcome{}, inst.InstantiateJobsFromPlan(ctx, planID, clientID, subscriptionID, workspaceID, spawnJobs)
}

// applySpawnOutcome copies the materialize result onto the create response
// (additive — Q-GSE-8).
func applySpawnOutcome(resp *subscriptionpb.CreateSubscriptionResponse, outcome jobSpawnOutcome) {
	if resp == nil {
		return
	}
	resp.SpawnedJobIds = outcome.SpawnedJobIDs
	if outcome.SkipReason != "" {
		sr := outcome.SkipReason
		resp.SpawnSkipReason = &sr
	}
}

// InstantiateJobsFromPlanDetailed is the Q-GSE-8 richer variant of the legacy
// port on the canonical adapter. It delegates to the ungated
// MaterializeJobsForSubscriptionUseCase.materializeCore (item #5, a) — the
// create side effect is already authorized by subscription:create at the
// CreateSubscriptionUseCase front door, so the intrinsic materialize must NOT
// re-gate on subscription:update. It additionally returns the spawned Job IDs +
// clean-skip reason. Defined here (not in job_instantiator_adapter.go) so this
// Q-GSE-8 wiring is self-contained; Go permits methods on a same-package type
// across files.
func (a *MaterializeJobsForSubscriptionInstantiator) InstantiateJobsFromPlanDetailed(
	ctx context.Context, _, _, subscriptionID, _ string, spawnJobs bool,
) (jobSpawnOutcome, error) {
	if a == nil || a.UseCase == nil {
		return jobSpawnOutcome{}, nil
	}
	if subscriptionID == "" {
		return jobSpawnOutcome{}, errors.New("instantiate_jobs: subscription_id required")
	}
	resp, err := a.UseCase.materializeCore(ctx, materializeJobsForSubscriptionInternalRequest{
		SubscriptionId: subscriptionID,
		SpawnJobs:      spawnJobs,
	})
	if err != nil {
		return jobSpawnOutcome{}, err
	}
	out := jobSpawnOutcome{}
	if resp != nil {
		for _, j := range resp.GetSpawnedJobs() {
			out.SpawnedJobIDs = append(out.SpawnedJobIDs, j.GetId())
		}
		out.SkipReason = resp.GetSkippedReason()
	}
	return out, nil
}

// executeWithTransaction executes subscription creation within a transaction
func (uc *CreateSubscriptionUseCase) executeWithTransaction(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest, enrichedSubscription *subscriptionpb.Subscription) (*subscriptionpb.CreateSubscriptionResponse, error) {
	var result *subscriptionpb.CreateSubscriptionResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req, enrichedSubscription)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for creating a subscription
func (uc *CreateSubscriptionUseCase) executeCore(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest, enrichedSubscription *subscriptionpb.Subscription) (*subscriptionpb.CreateSubscriptionResponse, error) {
	resp, err := uc.repositories.Subscription.CreateSubscription(ctx, &subscriptionpb.CreateSubscriptionRequest{
		Data: enrichedSubscription,
	})
	if err != nil {
		log.Printf("CreateSubscription DB error: %v", err)
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.errors.creation_failed", "[ERR-DEFAULT] Subscription creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched subscription
func (uc *CreateSubscriptionUseCase) applyBusinessLogic(subscription *subscriptionpb.Subscription) *subscriptionpb.Subscription {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if subscription.Id == "" {
		subscription.Id = uc.services.IDGenerator.GenerateID()
	}

	// Business logic: Set active status for new subscriptions
	subscription.Active = true

	// Business logic: Set creation audit fields
	subscription.DateCreated = &[]int64{now.UnixMilli()}[0]
	subscription.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	subscription.DateModified = &[]int64{now.UnixMilli()}[0]
	subscription.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return subscription
}

// validateBusinessRules enforces business constraints
func (uc *CreateSubscriptionUseCase) validateBusinessRules(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Business rule: Required data validation
	if subscription == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.data_required", "[ERR-DEFAULT] Subscription data is required"))
	}
	if subscription.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.name_required", "[ERR-DEFAULT] Subscription name is required"))
	}
	if subscription.PricePlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.price_plan_id_required", "[ERR-DEFAULT] Price plan ID is required"))
	}
	if subscription.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.client_id_required", "[ERR-DEFAULT] Client ID is required"))
	}

	// Business rule: Name length constraints
	if len(subscription.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.name_too_short", "[ERR-DEFAULT] Subscription name is too short"))
	}

	if len(subscription.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.name_too_long", "[ERR-DEFAULT] Subscription name is too long"))
	}

	// Business rule: PricePlan ID format validation
	if len(subscription.PricePlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.price_plan_id_too_short", "[ERR-DEFAULT] Price plan ID is too short"))
	}

	// Business rule: Client ID format validation
	if len(subscription.ClientId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.validation.client_id_too_short", "[ERR-DEFAULT] Client ID is too short"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist.
// It returns the resolved PricePlan so the caller can access plan_id after validation.
//
// Plan §3.3 — when the chosen PricePlan is client-scoped (client_id != ""), it
// must match the subscription's client_id. Master PricePlans (client_id == "")
// remain attachable for any client.
func (uc *CreateSubscriptionUseCase) validateEntityReferences(ctx context.Context, subscription *subscriptionpb.Subscription) (*priceplanpb.PricePlan, error) {
	if subscription == nil {
		return nil, nil // Should be caught by validateBusinessRules
	}

	var resolvedPricePlan *priceplanpb.PricePlan

	// Validate PricePlan entity reference
	if subscription.PricePlanId != "" {
		pricePlan, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
			Data: &priceplanpb.PricePlan{Id: subscription.PricePlanId},
		})
		if err != nil || pricePlan == nil || pricePlan.Data == nil || len(pricePlan.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.errors.price_plan_not_found", "[ERR-DEFAULT] Price plan not found"))
		}
		if !pricePlan.Data[0].Active {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.errors.price_plan_not_active", "[ERR-DEFAULT] Price plan is not active"))
		}
		resolvedPricePlan = pricePlan.Data[0]

		// §3.3 — client-scope mismatch hard reject.
		if ppClientID := resolvedPricePlan.GetClientId(); ppClientID != "" && ppClientID != subscription.ClientId {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"subscription.errors.planClientMismatch",
				"This package belongs to a different client and cannot be attached here. [DEFAULT]",
			))
		}
	}

	// Validate Client entity reference
	if subscription.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: subscription.ClientId},
		})
		if err != nil || client == nil || client.Data == nil || len(client.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.errors.client_not_found", "[ERR-DEFAULT] Client not found"))
		}
		if !client.Data[0].Active {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.errors.client_not_active", "[ERR-DEFAULT] Client is not active"))
		}
	}

	return resolvedPricePlan, nil
}

// generateSubscriptionCode resolves the fixed token set from the create
// request's client + price_plan (plan_id, price_schedule_id) and renders the
// configured SUBSCRIPTION_CODE_FORMAT template. It is fully best-effort: any
// lookup error contributes empty tokens rather than aborting creation. The
// caller only invokes this when req.Data.Code == "".
func (uc *CreateSubscriptionUseCase) generateSubscriptionCode(ctx context.Context, subscription *subscriptionpb.Subscription, pricePlan *priceplanpb.PricePlan) string {
	var client *clientpb.Client
	var plan *planpb.Plan
	var schedule *priceschedulepb.PriceSchedule

	// Student names come from the client's own first_name/last_name.
	if subscription.ClientId != "" && uc.repositories.Client != nil {
		if resp, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: subscription.ClientId},
		}); err == nil && resp != nil && len(resp.Data) > 0 {
			client = resp.Data[0]
		}
	}

	if pricePlan != nil {
		// {grade} <- price_plan.plan_id -> plan.name
		if planID := pricePlan.GetPlanId(); planID != "" && uc.repositories.Plan != nil {
			if resp, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
				Data: &planpb.Plan{Id: &planID},
			}); err == nil && resp != nil && len(resp.Data) > 0 {
				plan = resp.Data[0]
			}
		}
		// {price_schedule} <- price_plan.price_schedule_id -> price_schedule.name.
		// price_schedule_id is optional (master/non-scheduled plans) -> empty token.
		if scheduleID := pricePlan.GetPriceScheduleId(); scheduleID != "" && uc.repositories.PriceSchedule != nil {
			if resp, err := uc.repositories.PriceSchedule.ReadPriceSchedule(ctx, &priceschedulepb.ReadPriceScheduleRequest{
				Data: &priceschedulepb.PriceSchedule{Id: scheduleID},
			}); err == nil && resp != nil && len(resp.Data) > 0 {
				schedule = resp.Data[0]
			}
		}
	}

	tokens := ResolveCodeTokens(client, plan, schedule)
	return FormatCode(uc.services.CodeFormat, tokens)
}
