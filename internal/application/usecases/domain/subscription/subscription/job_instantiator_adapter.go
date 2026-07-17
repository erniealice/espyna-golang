package subscription

import (
	"context"
	"fmt"
)

// MaterializeJobsForSubscriptionInstantiator adapts
// MaterializeJobsForSubscriptionUseCase to the legacy JobTemplateInstantiator
// port consumed by CreateSubscriptionUseCase (plan §6 hook).
//
// The legacy InstantiateJobsFromPlan(ctx, planID, clientID, subscriptionID,
// workspaceID) signature carries four IDs, three of which are now redundant:
// the new use case resolves Plan via Subscription.PricePlan.Plan and reads
// the workspace from context. We keep the shape so the
// CreateSubscriptionUseCase hook does not need to change, while the body
// delegates to the canonical MaterializeJobsForSubscriptionUseCase.
type MaterializeJobsForSubscriptionInstantiator struct {
	UseCase *MaterializeJobsForSubscriptionUseCase
}

// InstantiateJobsFromPlan implements the legacy port. Only subscriptionID
// and spawnJobs are used; the other arguments are intentionally ignored —
// the new use case resolves them itself via cross-domain reads.
//
// 2026-04-29 auto-spawn-jobs-from-subscription plan §5.1 — spawnJobs is
// forwarded straight through. When false the underlying use case skips the
// spawn with SkipReasonOperatorOptOut so the operator's toggle decision is
// honored.
//
// Item #5 (option a): this is a CREATE side effect (CreateSubscriptionUseCase
// is already authorized by subscription:create), so it delegates to the ungated
// materializeCore rather than the subscription:update-gated Execute.
func (a *MaterializeJobsForSubscriptionInstantiator) InstantiateJobsFromPlan(
	ctx context.Context, _, _, subscriptionID, _ string, spawnJobs bool,
) error {
	if a == nil || a.UseCase == nil {
		return nil
	}
	if subscriptionID == "" {
		return fmt.Errorf("instantiate_jobs: subscription_id required")
	}
	_, err := a.UseCase.materializeCore(ctx, materializeJobsForSubscriptionInternalRequest{
		SubscriptionId: subscriptionID,
		SpawnJobs:      spawnJobs,
	})
	return err
}
