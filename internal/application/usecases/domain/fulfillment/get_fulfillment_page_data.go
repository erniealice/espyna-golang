package fulfillment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- GetFulfillmentItemPageData ----

type GetFulfillmentItemPageDataRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type GetFulfillmentItemPageDataServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}
type GetFulfillmentItemPageDataUseCase struct {
	repositories GetFulfillmentItemPageDataRepositories
	services     GetFulfillmentItemPageDataServices
}

// Execute fetches the full fulfillment detail page: fulfillment, line items, status events,
// returns, and resolved supplier/revenue names.
//
// It calls AllowedEvents from the state machine and injects the result into the
// response's AllowedEvents field so the view can render available action buttons
// without importing the state machine directly.
func (uc *GetFulfillmentItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetFulfillmentItemPageDataRequest) (*pb.GetFulfillmentItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator, "fulfillment", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}

	result, err := uc.repositories.Fulfillment.GetFulfillmentItemPageData(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.errors.not_found", "fulfillment not found [DEFAULT]"))
	}

	// Resolve allowed events from the state machine and inject into the response.
	// The proto field AllowedEvents []string maps to FulfillmentEvent strings.
	if result != nil && result.Fulfillment != nil {
		currentStatus := FulfillmentStatus(result.Fulfillment.Status)
		allowed := AllowedEvents(currentStatus)
		events := make([]string, len(allowed))
		for i, e := range allowed {
			events[i] = string(e)
		}
		result.AllowedEvents = events
	}

	return result, nil
}
