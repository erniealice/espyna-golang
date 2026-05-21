package fulfillment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- ListStatusEvents ----

type ListStatusEventsRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type ListStatusEventsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}
type ListStatusEventsUseCase struct {
	repositories ListStatusEventsRepositories
	services     ListStatusEventsServices
}

// Execute returns the full append-only status event log for a fulfillment.
func (uc *ListStatusEventsUseCase) Execute(ctx context.Context, req *pb.ListStatusEventsRequest) (*pb.ListStatusEventsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator, "fulfillment", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.FulfillmentId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}
	result, err := uc.repositories.Fulfillment.ListStatusEvents(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.errors.events_list_failed", "status event listing failed [DEFAULT]"))
	}
	return result, nil
}
