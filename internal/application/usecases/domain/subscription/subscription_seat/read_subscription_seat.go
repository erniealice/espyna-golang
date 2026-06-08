package subscription_seat

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// ReadSubscriptionSeatRepositories groups all repository dependencies
type ReadSubscriptionSeatRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// ReadSubscriptionSeatServices groups all business service dependencies
type ReadSubscriptionSeatServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadSubscriptionSeatUseCase handles the business logic for reading subscription seats
type ReadSubscriptionSeatUseCase struct {
	repositories ReadSubscriptionSeatRepositories
	services     ReadSubscriptionSeatServices
}

// NewReadSubscriptionSeatUseCase creates a new ReadSubscriptionSeatUseCase
func NewReadSubscriptionSeatUseCase(
	repositories ReadSubscriptionSeatRepositories,
	services ReadSubscriptionSeatServices,
) *ReadSubscriptionSeatUseCase {
	return &ReadSubscriptionSeatUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read subscription seat operation
func (uc *ReadSubscriptionSeatUseCase) Execute(ctx context.Context, req *subscriptionseatpb.ReadSubscriptionSeatRequest) (*subscriptionseatpb.ReadSubscriptionSeatResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionSeat, entityid.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionSeat.ReadSubscriptionSeat(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ReadSubscriptionSeatUseCase) validateInput(ctx context.Context, req *subscriptionseatpb.ReadSubscriptionSeatRequest) error {
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
