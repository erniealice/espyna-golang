package fulfillment

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- UpdateFulfillment ----

type UpdateFulfillmentRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type UpdateFulfillmentServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type UpdateFulfillmentUseCase struct {
	repositories UpdateFulfillmentRepositories
	services     UpdateFulfillmentServices
}

// Execute updates non-status fields only.
// Returns an error if the caller attempts to change the status field via this use case;
// status transitions must go through TransitionStatus instead.
func (uc *UpdateFulfillmentUseCase) Execute(ctx context.Context, req *pb.UpdateFulfillmentRequest) (*pb.UpdateFulfillmentResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "fulfillment", Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}

	// Guard: status changes must use TransitionStatus, not UpdateFulfillment.
	if req.Data.Status != "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.status_via_transition", "status must be changed via TransitionStatus, not UpdateFulfillment [DEFAULT]"))
	}

	now := time.Now()
	dm := now.UnixMilli()
	req.Data.DateModified = &dm

	result, err := uc.repositories.Fulfillment.UpdateFulfillment(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.errors.update_failed", "fulfillment update failed [DEFAULT]"))
	}
	return result, nil
}
