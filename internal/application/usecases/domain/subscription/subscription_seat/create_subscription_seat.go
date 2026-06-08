package subscription_seat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// CreateSubscriptionSeatRepositories groups all repository dependencies
type CreateSubscriptionSeatRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer // Primary entity repository
	Subscription     subscriptionpb.SubscriptionDomainServiceServer         // Entity reference + client_id stamping
}

// CreateSubscriptionSeatServices groups all business service dependencies
type CreateSubscriptionSeatServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateSubscriptionSeatUseCase handles the business logic for creating subscription seats
type CreateSubscriptionSeatUseCase struct {
	repositories CreateSubscriptionSeatRepositories
	services     CreateSubscriptionSeatServices
}

// NewCreateSubscriptionSeatUseCase creates use case with grouped dependencies
func NewCreateSubscriptionSeatUseCase(
	repositories CreateSubscriptionSeatRepositories,
	services CreateSubscriptionSeatServices,
) *CreateSubscriptionSeatUseCase {
	return &CreateSubscriptionSeatUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create subscription seat operation.
//
// client_id is a denormalized IDOR anchor; it is always stamped from the parent
// subscription (single-write boundary), NEVER taken from caller input. status
// defaults to PROPOSED when unset (entity-status-conventions: never NULL).
func (uc *CreateSubscriptionSeatUseCase) Execute(ctx context.Context, req *subscriptionseatpb.CreateSubscriptionSeatRequest) (*subscriptionseatpb.CreateSubscriptionSeatResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionSeat, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Stamp client_id from the subscription (single-write boundary) + validate FK.
	if err := uc.stampClientFromSubscription(ctx, req.Data); err != nil {
		return nil, err
	}

	uc.enrich(req.Data)

	resp, err := uc.repositories.SubscriptionSeat.CreateSubscriptionSeat(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.creation_failed", "Subscription seat creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *CreateSubscriptionSeatUseCase) validateInput(ctx context.Context, req *subscriptionseatpb.CreateSubscriptionSeatRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.SubscriptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.subscription_id_required", "Subscription ID is required [DEFAULT]"))
	}
	if req.Data.StaffId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.staff_id_required", "Staff ID is required [DEFAULT]"))
	}
	if req.Data.ProductPlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.product_plan_id_required", "Product plan ID is required [DEFAULT]"))
	}
	if req.Data.ContractedAmount != nil && req.Data.GetContractedAmount() < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.contracted_amount_negative", "Contracted amount cannot be negative [DEFAULT]"))
	}
	return nil
}

// stampClientFromSubscription reads the parent subscription, validates it exists +
// is active, and stamps the seat's client_id from subscription.client_id.
func (uc *CreateSubscriptionSeatUseCase) stampClientFromSubscription(ctx context.Context, seat *subscriptionseatpb.SubscriptionSeat) error {
	subscription, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: seat.SubscriptionId},
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.subscription_reference_validation_failed", "Failed to validate subscription entity reference [DEFAULT]")
		return fmt.Errorf("%s: %w", translatedError, err)
	}
	if subscription == nil || len(subscription.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.subscription_not_found", "Subscription not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{subscriptionId}", seat.SubscriptionId)
		return errors.New(translatedError)
	}
	if !subscription.Data[0].Active {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.subscription_not_active", "Referenced subscription is not active [DEFAULT]")
		return errors.New(translatedError)
	}
	// Single-write boundary: client_id ALWAYS comes from the subscription.
	seat.ClientId = subscription.Data[0].ClientId
	return nil
}

func (uc *CreateSubscriptionSeatUseCase) enrich(seat *subscriptionseatpb.SubscriptionSeat) {
	now := time.Now()
	if seat.Id == "" {
		seat.Id = uc.services.IDGenerator.GenerateID()
	}
	// Default status to PROPOSED (never NULL — entity-status-conventions).
	if seat.Status == subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_UNSPECIFIED {
		seat.Status = subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_PROPOSED
	}
	// position must be non-empty for active seats so the partial unique
	// (subscription_id, position) WHERE status='active' is a real backstop —
	// NULL/empty positions are distinct in a partial unique and would let two
	// active seats coexist (SR-2 bypass). An original seat is its own lineage
	// root: position_id = id (per _seat-replacement-spec.md). Replace inherits a
	// non-empty old.Position from here, preserving the lineage.
	if seat.Position == nil || *seat.Position == "" {
		pos := seat.Id
		seat.Position = &pos
	}
	seat.Active = true
	seat.DateCreated = &[]int64{now.UnixMilli()}[0]
	seat.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	seat.DateModified = &[]int64{now.UnixMilli()}[0]
	seat.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}
