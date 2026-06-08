package eventrecurrence

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
)

// GetEventRecurrenceItemPageDataRepositories groups all repository dependencies
type GetEventRecurrenceItemPageDataRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer
}

// GetEventRecurrenceItemPageDataServices groups all business service dependencies
type GetEventRecurrenceItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetEventRecurrenceItemPageDataUseCase handles the business logic for getting event recurrence item page data
type GetEventRecurrenceItemPageDataUseCase struct {
	repositories GetEventRecurrenceItemPageDataRepositories
	services     GetEventRecurrenceItemPageDataServices
}

// NewGetEventRecurrenceItemPageDataUseCase creates a new GetEventRecurrenceItemPageDataUseCase
func NewGetEventRecurrenceItemPageDataUseCase(
	repositories GetEventRecurrenceItemPageDataRepositories,
	services GetEventRecurrenceItemPageDataServices,
) *GetEventRecurrenceItemPageDataUseCase {
	return &GetEventRecurrenceItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event recurrence item page data operation
func (uc *GetEventRecurrenceItemPageDataUseCase) Execute(
	ctx context.Context,
	req *eventrecurrencepb.GetEventRecurrenceItemPageDataRequest,
) (*eventrecurrencepb.GetEventRecurrenceItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"event_recurrence", entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event recurrence item page data retrieval within a transaction
func (uc *GetEventRecurrenceItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *eventrecurrencepb.GetEventRecurrenceItemPageDataRequest,
) (*eventrecurrencepb.GetEventRecurrenceItemPageDataResponse, error) {
	var result *eventrecurrencepb.GetEventRecurrenceItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"event_recurrence.errors.item_page_data_failed",
				"event recurrence item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting event recurrence item page data
func (uc *GetEventRecurrenceItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *eventrecurrencepb.GetEventRecurrenceItemPageDataRequest,
) (*eventrecurrencepb.GetEventRecurrenceItemPageDataResponse, error) {
	// Create read request for the event recurrence
	readReq := &eventrecurrencepb.ReadEventRecurrenceRequest{
		Data: &eventrecurrencepb.EventRecurrence{
			Id: req.EventRecurrenceId,
		},
	}

	// Retrieve the event recurrence
	readResp, err := uc.repositories.EventRecurrence.ReadEventRecurrence(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"event_recurrence.errors.read_failed",
			"failed to retrieve event recurrence: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"event_recurrence.errors.not_found",
			"event recurrence not found",
		))
	}

	// Get the event recurrence (should be only one)
	eventRecurrence := readResp.Data[0]

	// Validate that we got the expected event recurrence
	if eventRecurrence.Id != req.EventRecurrenceId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"event_recurrence.errors.id_mismatch",
			"retrieved event recurrence ID does not match requested ID",
		))
	}

	return &eventrecurrencepb.GetEventRecurrenceItemPageDataResponse{
		EventRecurrence: eventRecurrence,
		Success:         true,
	}, nil
}

// validateInput validates the input request
func (uc *GetEventRecurrenceItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *eventrecurrencepb.GetEventRecurrenceItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"event_recurrence.validation.request_required",
			"request is required",
		))
	}

	if req.EventRecurrenceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"event_recurrence.validation.id_required",
			"event recurrence ID is required",
		))
	}

	return nil
}
