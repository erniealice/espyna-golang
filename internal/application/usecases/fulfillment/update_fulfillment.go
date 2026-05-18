package fulfillment

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- UpdateFulfillment ----

type UpdateFulfillmentRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type UpdateFulfillmentServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type UpdateFulfillmentUseCase struct {
	repositories UpdateFulfillmentRepositories
	services     UpdateFulfillmentServices
}

// Execute updates non-status fields only.
// Returns an error if the caller attempts to change the status field via this use case;
// status transitions must go through TransitionStatus instead.
func (uc *UpdateFulfillmentUseCase) Execute(ctx context.Context, req *pb.UpdateFulfillmentRequest) (*pb.UpdateFulfillmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "fulfillment", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}

	// Guard: status changes must use TransitionStatus, not UpdateFulfillment.
	if req.Data.Status != "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.status_via_transition", "status must be changed via TransitionStatus, not UpdateFulfillment [DEFAULT]"))
	}

	now := time.Now()
	dm := now.UnixMilli()
	req.Data.DateModified = &dm

	result, err := uc.repositories.Fulfillment.UpdateFulfillment(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.update_failed", "fulfillment update failed [DEFAULT]"))
	}
	return result, nil
}
