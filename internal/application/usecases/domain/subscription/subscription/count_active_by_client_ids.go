package subscription

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// CountActiveByClientIdsRepositories groups repository dependencies for
// the CountActiveByClientIds use case.
type CountActiveByClientIdsRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

// CountActiveByClientIdsServices groups service dependencies for
// the CountActiveByClientIds use case.
type CountActiveByClientIdsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// CountActiveByClientIdsUseCase counts active subscriptions grouped by client ID.
type CountActiveByClientIdsUseCase struct {
	repositories CountActiveByClientIdsRepositories
	services     CountActiveByClientIdsServices
}

// NewCountActiveByClientIdsUseCase creates a new CountActiveByClientIdsUseCase.
func NewCountActiveByClientIdsUseCase(
	repos CountActiveByClientIdsRepositories,
	svcs CountActiveByClientIdsServices,
) *CountActiveByClientIdsUseCase {
	return &CountActiveByClientIdsUseCase{
		repositories: repos,
		services:     svcs,
	}
}

// Execute performs an authorization check then delegates to the repository.
func (uc *CountActiveByClientIdsUseCase) Execute(ctx context.Context, req *subscriptionpb.CountActiveByClientIdsRequest) (*subscriptionpb.CountActiveByClientIdsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Subscription,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.Subscription.CountActiveByClientIds(ctx, req)
}
