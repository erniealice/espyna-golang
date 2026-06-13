package subscription_seat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// GetSubscriptionSeatItemPageDataRepositories groups all repository dependencies
type GetSubscriptionSeatItemPageDataRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// GetSubscriptionSeatItemPageDataServices groups all business service dependencies
type GetSubscriptionSeatItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetSubscriptionSeatItemPageDataUseCase handles the business logic for getting subscription seat item page data
type GetSubscriptionSeatItemPageDataUseCase struct {
	repositories GetSubscriptionSeatItemPageDataRepositories
	services     GetSubscriptionSeatItemPageDataServices
}

// NewGetSubscriptionSeatItemPageDataUseCase creates a new GetSubscriptionSeatItemPageDataUseCase
func NewGetSubscriptionSeatItemPageDataUseCase(
	repositories GetSubscriptionSeatItemPageDataRepositories,
	services GetSubscriptionSeatItemPageDataServices,
) *GetSubscriptionSeatItemPageDataUseCase {
	return &GetSubscriptionSeatItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get subscription seat item page data operation
func (uc *GetSubscriptionSeatItemPageDataUseCase) Execute(ctx context.Context, req *subscriptionseatpb.GetSubscriptionSeatItemPageDataRequest) (*subscriptionseatpb.GetSubscriptionSeatItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionSeat,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.SubscriptionSeat.GetSubscriptionSeatItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.item_page_data_failed", "Failed to retrieve subscription seat item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *GetSubscriptionSeatItemPageDataUseCase) validateInput(ctx context.Context, req *subscriptionseatpb.GetSubscriptionSeatItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.request_required", "Request is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.SubscriptionSeatId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.id_required", "Subscription seat ID is required [DEFAULT]"))
	}
	return nil
}
