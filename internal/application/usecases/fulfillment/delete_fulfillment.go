package fulfillment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- DeleteFulfillment ----

type DeleteFulfillmentRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type DeleteFulfillmentServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type DeleteFulfillmentUseCase struct {
	repositories DeleteFulfillmentRepositories
	services     DeleteFulfillmentServices
}

// Execute soft-deletes a fulfillment (sets active=false).
// Deletion is only allowed when the fulfillment is in PENDING or CANCELLED status.
// All other statuses (in-flight, delivered) are guarded to prevent data loss.
func (uc *DeleteFulfillmentUseCase) Execute(ctx context.Context, req *pb.DeleteFulfillmentRequest) (*pb.DeleteFulfillmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "fulfillment", ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}

	// Read current fulfillment to enforce status guard.
	current, err := uc.repositories.Fulfillment.GetFulfillment(ctx, &pb.GetFulfillmentRequest{Id: req.Id})
	if err != nil || current == nil || current.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.not_found", "fulfillment not found [DEFAULT]"))
	}

	currentStatus := FulfillmentStatus(current.Data.Status)
	if currentStatus != StatusPending && currentStatus != StatusCancelled {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.delete_not_allowed", "only PENDING or CANCELLED fulfillments may be deleted [DEFAULT]"))
	}

	result, err := uc.repositories.Fulfillment.DeleteFulfillment(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.deletion_failed", "fulfillment deletion failed [DEFAULT]"))
	}
	return result, nil
}
