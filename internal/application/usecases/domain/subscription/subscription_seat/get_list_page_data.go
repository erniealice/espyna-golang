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

// GetSubscriptionSeatListPageDataRepositories groups all repository dependencies
type GetSubscriptionSeatListPageDataRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// GetSubscriptionSeatListPageDataServices groups all business service dependencies
type GetSubscriptionSeatListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetSubscriptionSeatListPageDataUseCase handles the business logic for getting subscription seat list page data
type GetSubscriptionSeatListPageDataUseCase struct {
	repositories GetSubscriptionSeatListPageDataRepositories
	services     GetSubscriptionSeatListPageDataServices
}

// NewGetSubscriptionSeatListPageDataUseCase creates a new GetSubscriptionSeatListPageDataUseCase
func NewGetSubscriptionSeatListPageDataUseCase(
	repositories GetSubscriptionSeatListPageDataRepositories,
	services GetSubscriptionSeatListPageDataServices,
) *GetSubscriptionSeatListPageDataUseCase {
	return &GetSubscriptionSeatListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get subscription seat list page data operation
func (uc *GetSubscriptionSeatListPageDataUseCase) Execute(ctx context.Context, req *subscriptionseatpb.GetSubscriptionSeatListPageDataRequest) (*subscriptionseatpb.GetSubscriptionSeatListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionSeat,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionSeat.GetSubscriptionSeatListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *GetSubscriptionSeatListPageDataUseCase) validateInput(ctx context.Context, req *subscriptionseatpb.GetSubscriptionSeatListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
