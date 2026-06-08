package subscription_seat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// UpdateSubscriptionSeatRepositories groups all repository dependencies
type UpdateSubscriptionSeatRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// UpdateSubscriptionSeatServices groups all business service dependencies
type UpdateSubscriptionSeatServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateSubscriptionSeatUseCase handles the business logic for updating subscription seats
type UpdateSubscriptionSeatUseCase struct {
	repositories UpdateSubscriptionSeatRepositories
	services     UpdateSubscriptionSeatServices
}

// NewUpdateSubscriptionSeatUseCase creates a new UpdateSubscriptionSeatUseCase
func NewUpdateSubscriptionSeatUseCase(
	repositories UpdateSubscriptionSeatRepositories,
	services UpdateSubscriptionSeatServices,
) *UpdateSubscriptionSeatUseCase {
	return &UpdateSubscriptionSeatUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update subscription seat operation.
//
// Note: status transitions and contracted_amount-while-active changes are NOT
// done here — use SetSubscriptionSeatStatus (guarded arcs) and the replace flow
// respectively. The DB trigger backstops contracted_amount immutability while
// active.
//
// status is NEVER changed through this generic Update: a caller-supplied status
// would bypass the SetSubscriptionSeatStatus arc whitelist (SR-2) and the
// active/date_end coupling that the set-status closure owns. We re-read the
// persisted seat and PRESERVE its current status (plus the active/date_end derived
// from it), overwriting any caller-supplied value before the write. Status changes
// must go only through SetSubscriptionSeatStatus.
func (uc *UpdateSubscriptionSeatUseCase) Execute(ctx context.Context, req *subscriptionseatpb.UpdateSubscriptionSeatRequest) (*subscriptionseatpb.UpdateSubscriptionSeatResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionSeat, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Preserve the persisted status (and its derived active/date_end) so the
	// generic update cannot drive a status transition out-of-band.
	if err := uc.preserveStatus(ctx, req.Data); err != nil {
		return nil, err
	}

	uc.enrich(req.Data)

	resp, err := uc.repositories.SubscriptionSeat.UpdateSubscriptionSeat(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.update_failed", "Subscription seat update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// preserveStatus re-reads the persisted seat and forces the request's status,
// active, and date_end back to the persisted values, so a status change cannot be
// smuggled through the generic Update (it must go through SetSubscriptionSeatStatus).
func (uc *UpdateSubscriptionSeatUseCase) preserveStatus(ctx context.Context, seat *subscriptionseatpb.SubscriptionSeat) error {
	readResp, err := uc.repositories.SubscriptionSeat.ReadSubscriptionSeat(ctx, &subscriptionseatpb.ReadSubscriptionSeatRequest{
		Data: &subscriptionseatpb.SubscriptionSeat{Id: seat.Id},
	})
	if err != nil {
		return err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.not_found", "Subscription seat not found [DEFAULT]"))
	}
	current := readResp.Data[0]
	// status↔active↔date_end are owned by the set-status closure; never flip them
	// independently through the generic update.
	seat.Status = current.Status
	seat.Active = current.Active
	seat.DateEnd = current.DateEnd
	return nil
}

func (uc *UpdateSubscriptionSeatUseCase) validateInput(ctx context.Context, req *subscriptionseatpb.UpdateSubscriptionSeatRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.id_required", "Subscription seat ID is required [DEFAULT]"))
	}
	if req.Data.ContractedAmount != nil && req.Data.GetContractedAmount() < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.contracted_amount_negative", "Contracted amount cannot be negative [DEFAULT]"))
	}
	return nil
}

func (uc *UpdateSubscriptionSeatUseCase) enrich(seat *subscriptionseatpb.SubscriptionSeat) {
	now := time.Now()
	seat.DateModified = &[]int64{now.UnixMilli()}[0]
	seat.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}
