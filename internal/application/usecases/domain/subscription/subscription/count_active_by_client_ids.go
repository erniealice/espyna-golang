package subscription

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
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
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscription, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.Subscription.CountActiveByClientIds(ctx, req)
}
