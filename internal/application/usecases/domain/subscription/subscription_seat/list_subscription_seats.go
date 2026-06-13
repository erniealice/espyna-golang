package subscription_seat

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// ListSubscriptionSeatsRepositories groups all repository dependencies
type ListSubscriptionSeatsRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// ListSubscriptionSeatsServices groups all business service dependencies
type ListSubscriptionSeatsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListSubscriptionSeatsUseCase handles the business logic for listing subscription seats
type ListSubscriptionSeatsUseCase struct {
	repositories ListSubscriptionSeatsRepositories
	services     ListSubscriptionSeatsServices
}

// NewListSubscriptionSeatsUseCase creates a new ListSubscriptionSeatsUseCase
func NewListSubscriptionSeatsUseCase(
	repositories ListSubscriptionSeatsRepositories,
	services ListSubscriptionSeatsServices,
) *ListSubscriptionSeatsUseCase {
	return &ListSubscriptionSeatsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list subscription seats operation. Filters by
// subscription_id and client_id (the IDOR denorm) ride on req.Filters.
func (uc *ListSubscriptionSeatsUseCase) Execute(ctx context.Context, req *subscriptionseatpb.ListSubscriptionSeatsRequest) (*subscriptionseatpb.ListSubscriptionSeatsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionSeat,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionSeat.ListSubscriptionSeats(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ListSubscriptionSeatsUseCase) validateInput(ctx context.Context, req *subscriptionseatpb.ListSubscriptionSeatsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
