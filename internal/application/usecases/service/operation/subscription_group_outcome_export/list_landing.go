package subscription_group_outcome_export

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

// ListSubscriptionGroupOutcomeLandingUseCase serves the report landing
// aggregate. It deliberately shares the export aggregate's repository and
// reportScope so principal/workspace policy cannot drift between the landing
// and the detailed report.
type ListSubscriptionGroupOutcomeLandingUseCase struct {
	repositories Repositories
	services     Services
}

func (uc *ListSubscriptionGroupOutcomeLandingUseCase) Execute(
	ctx context.Context,
	req *exportpb.ListSubscriptionGroupOutcomeLandingRequest,
) (*exportpb.ListSubscriptionGroupOutcomeLandingResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"subscription_group_outcome_export.validation.request_required", "subscription group outcome landing request is required"))
	}

	scope, err := uc.services.reportScope(ctx)
	if err != nil {
		return nil, err
	}
	// The landing is a report projection and therefore requires the underlying
	// outcome-summary list capability in addition to the export read capability.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobOutcomeSummary,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if uc.repositories.LandingQuery == nil {
		return &exportpb.ListSubscriptionGroupOutcomeLandingResponse{Success: true, Rows: []*exportpb.SubscriptionGroupOutcomeLandingRow{}}, nil
	}
	return uc.repositories.LandingQuery.ListSubscriptionGroupOutcomeLandingScoped(ctx, req, scope)
}
