package consumer

import (
	"context"

	plan "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/plan"
)

// CustomizePlanForClientRequest is the public request shape for the
// CustomizePlanForClient use case. Mirrors the internal request struct so
// consumers can build it without depending on internal packages.
//
// See docs/plan/20260427-plan-client-scope/plan.md §4.1.
type CustomizePlanForClientRequest struct {
	SourcePlanID      string
	SourcePricePlanID string
	ClientID          string
	SubscriptionID    string
	NewScheduleName   string
}

// CustomizePlanForClientResponse is the public response shape. Only the IDs +
// reuse flag are surfaced; the full proto records are intentionally omitted
// from the public surface so consumers don't depend on esqyma proto types
// transitively. Callers needing the full records can re-read by ID.
type CustomizePlanForClientResponse struct {
	NewPlanID      string
	NewPricePlanID string
	NewScheduleID  string
	Reused         bool
}

// CustomizePlanForClient invokes the espyna use case that clones a master
// Plan + PricePlan tree into a client-scoped private copy and (optionally)
// repoints a subscription onto the new PricePlan. Single transaction.
//
// Returns nil + nil error when the use case is not wired (caller must guard).
func CustomizePlanForClient(
	useCases *UseCases,
	ctx context.Context,
	req *CustomizePlanForClientRequest,
) (*CustomizePlanForClientResponse, error) {
	if useCases == nil || useCases.Subscription == nil || useCases.Subscription.Plan == nil {
		return nil, nil
	}
	uc := useCases.Subscription.Plan.CustomizePlanForClient
	if uc == nil {
		return nil, nil
	}
	espResp, err := uc.Execute(ctx, &plan.CustomizePlanForClientRequest{
		SourcePlanID:      req.SourcePlanID,
		SourcePricePlanID: req.SourcePricePlanID,
		ClientID:          req.ClientID,
		SubscriptionID:    req.SubscriptionID,
		NewScheduleName:   req.NewScheduleName,
	})
	if err != nil {
		return nil, err
	}
	if espResp == nil {
		return &CustomizePlanForClientResponse{}, nil
	}
	return &CustomizePlanForClientResponse{
		NewPlanID:      espResp.Plan.GetId(),
		NewPricePlanID: espResp.PricePlan.GetId(),
		NewScheduleID:  espResp.PriceSchedule.GetId(),
		Reused:         espResp.Reused,
	}, nil
}
