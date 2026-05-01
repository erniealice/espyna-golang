package consumer

import (
	"context"

	subscription "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/subscription"
)

// MaterializeInstanceJobsForSubscriptionRequest is the public request shape
// for the cyclic-instance-Job spawn use case (cyclic-subscription-jobs plan §3).
//
// CyclePeriodStart is optional — when empty, the use case spawns the next
// un-spawned cycle from sub.date_time_start. Backfill, when true, materialises
// every missing cycle from sub.date_time_start up to today (capped at 24).
//
// UsageRequestDate is the AD_HOC equivalent: when the subscription's PricePlan
// is BILLING_KIND_AD_HOC, the use case spawns a single usage Job dated to
// UsageRequestDate (defaults to today UTC when empty). Cycle fields are
// ignored on AD_HOC plans.
type MaterializeInstanceJobsForSubscriptionRequest struct {
	SubscriptionID   string
	CyclePeriodStart string
	Backfill         bool
	UsageRequestDate string
}

// MaterializeInstanceJobsForSubscriptionResponse is the public response shape.
// Counts + skip reason + cap signal are surfaced; the full Job proto records
// are intentionally omitted from the public surface (callers fetch by origin
// if they need the records — same contract as MaterializeJobsForSubscription).
type MaterializeInstanceJobsForSubscriptionResponse struct {
	// SpawnedCycleCount is the number of cycle accordions materialised. For
	// multi-visit plans, one cycle accordion contains N sub-cycle Jobs.
	SpawnedCycleCount int

	// SpawnedJobCount is the total cycle Job rows actually inserted (sum of
	// jobs across all SpawnedCycles). For visits_per_cycle=1 this equals
	// SpawnedCycleCount; for multi-visit it's higher.
	SpawnedJobCount int

	// OnceAtStartJobCount is how many ONCE_AT_ENGAGEMENT_START child Jobs
	// fired on this call (only non-zero on the first-ever call when the
	// engagement was just spawned and the Plan has onboarding relations).
	OnceAtStartJobCount int

	// EngagementWasNewlyCreated reports whether this call created the
	// engagement-shell Job (retroactive path for subscriptions created
	// before this plan landed) versus reusing an existing shell.
	EngagementWasNewlyCreated bool

	// SkippedReason is non-empty when the use case skipped without spawning
	// (eligibility gate). One of: "non_cyclic_plan", "no_template",
	// "milestone_unsupported", "no_pending_cycles".
	SkippedReason string

	// BackfillCappedAt > 0 when the backfill window exceeded the cap
	// (cyclic-subscription-jobs plan §15 — set to 24 in v1). Operators see
	// this in the drawer preview and can re-submit for the remainder.
	BackfillCappedAt int32
}

// Skip-reason constants exposed for consumer-side switching.
const (
	InstanceSpawnSkipNonCyclicPlan        = subscription.InstanceSkipReasonNonCyclicPlan
	InstanceSpawnSkipNoTemplate           = subscription.InstanceSkipReasonNoTemplate
	InstanceSpawnSkipMilestoneUnsupported = subscription.InstanceSkipReasonMilestoneUnsupported
	InstanceSpawnSkipNoPendingCycles      = subscription.InstanceSkipReasonNoPendingCycles
)

// MaterializeInstanceJobsForSubscription invokes the cyclic-instance-Job
// spawn use case (cyclic-subscription-jobs plan §3). Returns (nil, nil) when
// the use case is not wired (caller must guard) — typically because the
// operation domain is unavailable in the current composition.
//
// The recognize-revenue piggyback (plan §5.2) calls this internally inside
// espyna; the consumer surface is for centymo's Operations tab "Spawn this
// cycle now" / "Backfill missing cycles" handlers.
func MaterializeInstanceJobsForSubscription(
	useCases *UseCases,
	ctx context.Context,
	req *MaterializeInstanceJobsForSubscriptionRequest,
) (*MaterializeInstanceJobsForSubscriptionResponse, error) {
	if useCases == nil || useCases.Subscription == nil {
		return nil, nil
	}
	uc := useCases.Subscription.MaterializeInstanceJobsForSubscription
	if uc == nil {
		return nil, nil
	}
	if req == nil {
		return nil, nil
	}
	espResp, err := uc.Execute(ctx, subscription.MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId:   req.SubscriptionID,
		CyclePeriodStart: req.CyclePeriodStart,
		Backfill:         req.Backfill,
		UsageRequestDate: req.UsageRequestDate,
	})
	if err != nil {
		return nil, err
	}
	if espResp == nil {
		return &MaterializeInstanceJobsForSubscriptionResponse{}, nil
	}
	jobCount := 0
	for _, c := range espResp.SpawnedCycles {
		jobCount += len(c.Jobs)
	}
	return &MaterializeInstanceJobsForSubscriptionResponse{
		SpawnedCycleCount:         len(espResp.SpawnedCycles),
		SpawnedJobCount:           jobCount,
		OnceAtStartJobCount:       len(espResp.OnceAtStartJobs),
		EngagementWasNewlyCreated: espResp.EngagementWasNewlyCreated,
		SkippedReason:             espResp.SkippedReason,
		BackfillCappedAt:          espResp.BackfillCappedAt,
	}, nil
}
