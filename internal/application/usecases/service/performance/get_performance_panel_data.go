// Package performance hosts the SERVICE-layer GetPerformancePanelData use case
// (§E6 Axis-2: a servicing-gated cross-join over subscription_seat × evaluation ×
// evaluation_cycle → service layer). It takes only the conditional 14-layer
// subset (Layer-7 use case + initializer wiring); NO own proto/entityid/adapter.
//
// The panel is workspace-scoped AND resource-gated (Q-SERVICING-SCOPE-1 / CR-5):
// it returns only seats/evaluations on engagements the caller can access, unless
// they hold "evaluation:triage_all". NULL subscription_id resolves to ACCOUNT
// scope via client_id (never workspace-wide). The gate is the shared
// ResourceGatekeeper (Gate 2), consumed here.
package performance

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/internal/application/shared/resourcegate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// evaluationScoreReader is the narrow interface the evaluation postgres adapter
// satisfies for the batched latest-score read (Q-RATING-XJOIN-1). It is declared
// here (not on the proto interface) because GetLatestEvaluationScore is an
// adapter extension method, not an RPC.
type evaluationScoreReader interface {
	GetLatestEvaluationScore(ctx context.Context, staffIDs []string) (map[string]*float64, error)
}

// Repositories groups the cross-aggregate read dependencies.
type Repositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
	Evaluation       evaluationpb.EvaluationDomainServiceServer
}

// Services groups the business-service + gate dependencies.
type Services struct {
	Authorizer         ports.Authorizer
	Translator         ports.Translator
	ResourceGatekeeper *resourcegate.ResourceGatekeeper
}

// PanelRow is one row of the performance panel: a seat + its subject's latest
// rating. (The cycle-status join is read from the seat's subscription context.)
type PanelRow struct {
	Seat         *subscriptionseatpb.SubscriptionSeat
	LatestRating *float64
}

// PanelData is the assembled panel projection.
type PanelData struct {
	Rows []PanelRow
}

// GetPerformancePanelDataRequest is the Go-shaped input.
type GetPerformancePanelDataRequest struct{}

// UseCase assembles the servicing-gated performance panel.
type UseCase struct {
	r Repositories
	s Services
}

func NewUseCase(r Repositories, s Services) *UseCase {
	return &UseCase{r: r, s: s}
}

func (uc *UseCase) Execute(ctx context.Context, req *GetPerformancePanelDataRequest) (*PanelData, error) {
	// Gate 1: can you DO evaluation:list?
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.Evaluation, entityid.ActionList); err != nil {
		return nil, err
	}
	out := &PanelData{}
	if uc.r.SubscriptionSeat == nil {
		return out, nil
	}

	seatsResp, err := uc.r.SubscriptionSeat.ListSubscriptionSeats(ctx, &subscriptionseatpb.ListSubscriptionSeatsRequest{})
	if err != nil {
		return nil, err
	}
	if seatsResp == nil {
		return out, nil
	}

	// Gate 2: can you SEE this seat's client/subscription?
	var gated []*subscriptionseatpb.SubscriptionSeat
	staffIDset := map[string]bool{}
	for _, seat := range seatsResp.Data {
		var subPtr *string
		if seat.SubscriptionId != "" {
			subID := seat.SubscriptionId
			subPtr = &subID
		}
		if !uc.s.ResourceGatekeeper.CanAccess(ctx, &resourcegate.CheckAccessRequest{
			Entity:         entityid.Evaluation,
			ClientID:       seat.ClientId,
			SubscriptionID: subPtr,
		}) {
			continue
		}
		gated = append(gated, seat)
		if seat.StaffId != "" {
			staffIDset[seat.StaffId] = true
		}
	}

	// Batched latest-rating read per subject_staff_id (Q-RATING-XJOIN-1).
	ratings := map[string]*float64{}
	if reader, ok := uc.r.Evaluation.(evaluationScoreReader); ok && len(staffIDset) > 0 {
		staffIDs := make([]string, 0, len(staffIDset))
		for id := range staffIDset {
			staffIDs = append(staffIDs, id)
		}
		if m, rerr := reader.GetLatestEvaluationScore(ctx, staffIDs); rerr == nil {
			ratings = m
		}
	}

	for _, seat := range gated {
		out.Rows = append(out.Rows, PanelRow{Seat: seat, LatestRating: ratings[seat.StaffId]})
	}
	return out, nil
}
