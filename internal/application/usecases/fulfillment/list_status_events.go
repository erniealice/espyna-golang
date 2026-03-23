package fulfillment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- ListStatusEvents ----

type ListStatusEventsRepositories struct{ Fulfillment pb.FulfillmentDomainServiceServer }
type ListStatusEventsServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ListStatusEventsUseCase struct {
	repositories ListStatusEventsRepositories
	services     ListStatusEventsServices
}

// Execute returns the full append-only status event log for a fulfillment.
func (uc *ListStatusEventsUseCase) Execute(ctx context.Context, req *pb.ListStatusEventsRequest) (*pb.ListStatusEventsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "fulfillment", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.FulfillmentId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}
	result, err := uc.repositories.Fulfillment.ListStatusEvents(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.events_list_failed", "status event listing failed [DEFAULT]"))
	}
	return result, nil
}
