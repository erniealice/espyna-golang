package eventoccurrence

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventoccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
)

// GetEventOccurrenceItemPageDataRepositories groups all repository dependencies
type GetEventOccurrenceItemPageDataRepositories struct {
	EventOccurrence eventoccurrencepb.EventOccurrenceDomainServiceServer // Primary entity repository
}

// GetEventOccurrenceItemPageDataServices groups all business service dependencies
type GetEventOccurrenceItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetEventOccurrenceItemPageDataUseCase handles the business logic for getting event occurrence item page data
type GetEventOccurrenceItemPageDataUseCase struct {
	repositories GetEventOccurrenceItemPageDataRepositories
	services     GetEventOccurrenceItemPageDataServices
}

// NewGetEventOccurrenceItemPageDataUseCase creates a new GetEventOccurrenceItemPageDataUseCase
func NewGetEventOccurrenceItemPageDataUseCase(
	repositories GetEventOccurrenceItemPageDataRepositories,
	services GetEventOccurrenceItemPageDataServices,
) *GetEventOccurrenceItemPageDataUseCase {
	return &GetEventOccurrenceItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event occurrence item page data operation
func (uc *GetEventOccurrenceItemPageDataUseCase) Execute(ctx context.Context, req *eventoccurrencepb.GetEventOccurrenceItemPageDataRequest) (*eventoccurrencepb.GetEventOccurrenceItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventOccurrence, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventOccurrence.GetEventOccurrenceItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventOccurrenceItemPageDataUseCase) validateInput(ctx context.Context, req *eventoccurrencepb.GetEventOccurrenceItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_occurrence.validation.request_required",
			"request is required",
		))
	}

	if req.EventOccurrenceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_occurrence.validation.id_required",
			"event occurrence ID is required",
		))
	}

	return nil
}
