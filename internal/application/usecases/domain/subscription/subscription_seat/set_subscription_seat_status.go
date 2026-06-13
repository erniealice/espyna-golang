package subscription_seat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// SetSubscriptionSeatStatusRequest is the Go-shaped input for the guarded
// status-transition operation. (No proto request type exists for this op; the
// seat lifecycle arcs are an application-layer concern.)
type SetSubscriptionSeatStatusRequest struct {
	SubscriptionSeatID string
	NewStatus          subscriptionseatpb.SubscriptionSeatStatus
}

// SetSubscriptionSeatStatusRepositories groups all repository dependencies
type SetSubscriptionSeatStatusRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// SetSubscriptionSeatStatusServices groups all business service dependencies
type SetSubscriptionSeatStatusServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SetSubscriptionSeatStatusUseCase enforces the SR-2 guarded status lifecycle.
//
// Whitelisted arcs:
//   - PROPOSED -> ACTIVE
//   - ACTIVE   -> REPLACED
//   - ACTIVE   -> ENDED
//
// REPLACED and ENDED are terminal. All other transitions are rejected. The
// ACTIVE -> REPLACED arc is normally driven by the replace flow; it is also
// reachable here for completeness.
type SetSubscriptionSeatStatusUseCase struct {
	repositories SetSubscriptionSeatStatusRepositories
	services     SetSubscriptionSeatStatusServices
}

// NewSetSubscriptionSeatStatusUseCase creates a new SetSubscriptionSeatStatusUseCase
func NewSetSubscriptionSeatStatusUseCase(
	repositories SetSubscriptionSeatStatusRepositories,
	services SetSubscriptionSeatStatusServices,
) *SetSubscriptionSeatStatusUseCase {
	return &SetSubscriptionSeatStatusUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs a guarded status transition on a single seat.
func (uc *SetSubscriptionSeatStatusUseCase) Execute(ctx context.Context, req *SetSubscriptionSeatStatusRequest) (*subscriptionseatpb.UpdateSubscriptionSeatResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionSeat,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.SubscriptionSeatID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.id_required", "Subscription seat ID is required [DEFAULT]"))
	}

	// Read the current seat to learn its status.
	readResp, err := uc.repositories.SubscriptionSeat.ReadSubscriptionSeat(ctx, &subscriptionseatpb.ReadSubscriptionSeatRequest{
		Data: &subscriptionseatpb.SubscriptionSeat{Id: req.SubscriptionSeatID},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.not_found", "Subscription seat not found [DEFAULT]"))
	}
	seat := readResp.Data[0]

	if !isAllowedSeatStatusArc(seat.Status, req.NewStatus) {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.invalid_status_transition", "Invalid subscription seat status transition [DEFAULT]"))
	}

	seat.Status = req.NewStatus
	// Terminal statuses retire the seat from active surfaces.
	if req.NewStatus == subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_REPLACED ||
		req.NewStatus == subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ENDED {
		seat.Active = false
		if seat.DateEnd == nil {
			now := time.Now().UnixMilli()
			seat.DateEnd = &now
		}
	}
	now := time.Now()
	seat.DateModified = &[]int64{now.UnixMilli()}[0]
	seat.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	resp, err := uc.repositories.SubscriptionSeat.UpdateSubscriptionSeat(ctx, &subscriptionseatpb.UpdateSubscriptionSeatRequest{Data: seat})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.update_failed", "Subscription seat update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// isAllowedSeatStatusArc returns true only for the SR-2 whitelisted transitions.
func isAllowedSeatStatusArc(from, to subscriptionseatpb.SubscriptionSeatStatus) bool {
	type arc struct {
		from subscriptionseatpb.SubscriptionSeatStatus
		to   subscriptionseatpb.SubscriptionSeatStatus
	}
	allowed := map[arc]bool{
		{subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_PROPOSED, subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE}: true,
		{subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE, subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_REPLACED}: true,
		{subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE, subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ENDED}:    true,
	}
	return allowed[arc{from, to}]
}
