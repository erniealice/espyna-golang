package fulfillment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ListStatusEventsUseCase struct {
	repositories ListStatusEventsRepositories
	services     ListStatusEventsServices
}

// Execute returns the full append-only status event log for a fulfillment.
func (uc *ListStatusEventsUseCase) Execute(ctx context.Context, req *pb.ListStatusEventsRequest) (*pb.ListStatusEventsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "fulfillment", Action: entityid.ActionRead}); err != nil {
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
