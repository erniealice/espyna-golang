package eventoccurrence

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventoccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
)

// ListEventOccurrencesRepositories groups all repository dependencies
type ListEventOccurrencesRepositories struct {
	EventOccurrence eventoccurrencepb.EventOccurrenceDomainServiceServer // Primary entity repository
}

// ListEventOccurrencesServices groups all business service dependencies
type ListEventOccurrencesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListEventOccurrencesUseCase handles the business logic for listing event occurrences
type ListEventOccurrencesUseCase struct {
	repositories ListEventOccurrencesRepositories
	services     ListEventOccurrencesServices
}

// NewListEventOccurrencesUseCase creates use case with grouped dependencies
func NewListEventOccurrencesUseCase(
	repositories ListEventOccurrencesRepositories,
	services ListEventOccurrencesServices,
) *ListEventOccurrencesUseCase {
	return &ListEventOccurrencesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event occurrences operation
func (uc *ListEventOccurrencesUseCase) Execute(ctx context.Context, req *eventoccurrencepb.ListEventOccurrencesRequest) (*eventoccurrencepb.ListEventOccurrencesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventOccurrence,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &eventoccurrencepb.ListEventOccurrencesRequest{}
	}

	// Call repository
	resp, err := uc.repositories.EventOccurrence.ListEventOccurrences(ctx, req)
	if err != nil {
		errorMessage := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_occurrence.errors.list_failed", "Failed to retrieve event occurrences [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	return resp, nil
}
