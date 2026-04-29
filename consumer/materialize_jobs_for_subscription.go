package consumer

import (
	"context"

	subscription "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/subscription"
)

// MaterializeJobsForSubscriptionRequest is the public request shape for the
// MaterializeJobsForSubscription use case. Mirrors the internal request struct
// so consumers can build it without depending on internal packages.
//
// See docs/plan/20260429-auto-spawn-jobs-from-subscription/plan.md §3.
type MaterializeJobsForSubscriptionRequest struct {
	SubscriptionID string
	SpawnJobs      bool
}

// MaterializeJobsForSubscriptionResponse is the public response shape.
// Only counts + skip reason are surfaced; the full Job proto records are
// intentionally omitted from the public surface (callers fetch by origin if
// they need the records).
type MaterializeJobsForSubscriptionResponse struct {
	JobCount      int
	SkippedReason string
}

// Skip reason constants exposed for consumer-side switching.
const (
	SpawnSkipNoTemplateFound = "no_template_found"
	SpawnSkipOperatorOptOut  = "operator_opt_out"
)

// MaterializeJobsForSubscription invokes the auto-spawn-jobs-from-subscription
// use case (plan §3). Returns (nil, nil) when the use case is not wired
// (caller must guard) — typically because the operation domain is unavailable
// in the current composition.
func MaterializeJobsForSubscription(
	useCases *UseCases,
	ctx context.Context,
	req *MaterializeJobsForSubscriptionRequest,
) (*MaterializeJobsForSubscriptionResponse, error) {
	if useCases == nil || useCases.Subscription == nil {
		return nil, nil
	}
	uc := useCases.Subscription.MaterializeJobsForSubscription
	if uc == nil {
		return nil, nil
	}
	espResp, err := uc.Execute(ctx, subscription.MaterializeJobsForSubscriptionRequest{
		SubscriptionId: req.SubscriptionID,
		SpawnJobs:      req.SpawnJobs,
	})
	if err != nil {
		return nil, err
	}
	if espResp == nil {
		return &MaterializeJobsForSubscriptionResponse{}, nil
	}
	return &MaterializeJobsForSubscriptionResponse{
		JobCount:      len(espResp.SpawnedJobs),
		SkippedReason: espResp.SkippedReason,
	}, nil
}
