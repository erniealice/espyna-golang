package subscription_seat

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// DeleteSubscriptionSeatRepositories groups all repository dependencies
type DeleteSubscriptionSeatRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// DeleteSubscriptionSeatServices groups all business service dependencies
type DeleteSubscriptionSeatServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteSubscriptionSeatUseCase handles the business logic for deleting subscription seats (soft-delete)
type DeleteSubscriptionSeatUseCase struct {
	repositories DeleteSubscriptionSeatRepositories
	services     DeleteSubscriptionSeatServices
}

// NewDeleteSubscriptionSeatUseCase creates a new DeleteSubscriptionSeatUseCase
func NewDeleteSubscriptionSeatUseCase(
	repositories DeleteSubscriptionSeatRepositories,
	services DeleteSubscriptionSeatServices,
) *DeleteSubscriptionSeatUseCase {
	return &DeleteSubscriptionSeatUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete subscription seat operation (soft-delete: active=false).
func (uc *DeleteSubscriptionSeatUseCase) Execute(ctx context.Context, req *subscriptionseatpb.DeleteSubscriptionSeatRequest) (*subscriptionseatpb.DeleteSubscriptionSeatResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionSeat,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionSeat.DeleteSubscriptionSeat(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.deletion_failed", "Subscription seat deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *DeleteSubscriptionSeatUseCase) validateInput(ctx context.Context, req *subscriptionseatpb.DeleteSubscriptionSeatRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.id_required", "Subscription seat ID is required [DEFAULT]"))
	}
	return nil
}
