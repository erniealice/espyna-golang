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
// is used; the other arguments are intentionally ignored — the new use
// case resolves them itself via cross-domain reads.
func (a *MaterializeJobsForSubscriptionInstantiator) InstantiateJobsFromPlan(
	ctx context.Context, _, _, subscriptionID, _ string,
) error {
	if a == nil || a.UseCase == nil {
		return nil
	}
	if subscriptionID == "" {
		return fmt.Errorf("instantiate_jobs: subscription_id required")
	}
	_, err := a.UseCase.Execute(ctx, MaterializeJobsForSubscriptionRequest{
		SubscriptionId: subscriptionID,
		SpawnJobs:      true,
	})
	return err
}
