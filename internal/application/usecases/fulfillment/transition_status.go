package fulfillment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- TransitionStatus ----

type TransitionStatusRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type TransitionStatusServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type TransitionStatusUseCase struct {
	repositories TransitionStatusRepositories
	services     TransitionStatusServices
}

// Execute drives the fulfillment state machine.
//
// It reads the current status, calls ValidateTransition to assert the event is
// legal, then atomically updates the status and inserts a status event via the
// adapter's TransitionStatus RPC.
func (uc *TransitionStatusUseCase) Execute(ctx context.Context, req *pb.TransitionStatusRequest) (*pb.TransitionStatusResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "fulfillment", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.FulfillmentId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}
	if req.Event == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.event_required", "transition event is required [DEFAULT]"))
	}

	// Read current fulfillment to obtain current status.
	current, err := uc.repositories.Fulfillment.GetFulfillment(ctx, &pb.GetFulfillmentRequest{Id: req.FulfillmentId})
	if err != nil || current == nil || current.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.not_found", "fulfillment not found [DEFAULT]"))
	}

	currentStatus := FulfillmentStatus(current.Data.Status)
	event := FulfillmentEvent(req.Event)

	// Validate via state machine — returns target status or ErrInvalidTransition.
	_, err = ValidateTransition(currentStatus, event)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.invalid_transition", err.Error()))
	}

	// Execute the transition (atomically updates status + inserts event log) within
	// a transaction if available.
	if uc.services.TransactionService != nil {
		var result *pb.TransitionStatusResponse
		txErr := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.Fulfillment.TransitionStatus(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if txErr != nil {
			return nil, txErr
		}
		return result, nil
	}
	return uc.repositories.Fulfillment.TransitionStatus(ctx, req)
}
