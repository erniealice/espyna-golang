package service

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/resourcegate"
	performanceuc "github.com/erniealice/espyna-golang/internal/application/usecases/service/performance"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// initServicePerformance wires GetPerformancePanelData (resource-gated cross-join
// over subscription_seat × evaluation). The ResourceGatekeeper (Gate 2, CR-5)
// holds the triage_all bypass (via Authorizer) and the two scope checkers (via
// interface assertion on the postgres junction adapters). A mock/no-postgres build
// leaves the scope checkers nil → the gatekeeper fail-closes (deny unless the
// caller holds evaluation:triage_all). Returns nil when operationRepos is nil.
func initServicePerformance(
	operationRepos *domain.OperationRepositories,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *performanceuc.UseCase {
	if operationRepos == nil {
		return nil
	}

	// The postgres junction adapters implement IsActiveAccountTeamMember /
	// IsActiveServicer as extension methods. Assert the concrete adapter
	// satisfies the resourcegate adapter's Reader interface, then wrap.
	// A mock/no-postgres build fails the assertion → nil → fail-closes.
	type accountTeamReader = resourcegate.ClientScopeAdapter
	type servicerReader = resourcegate.SubscriptionScopeAdapter
	var clientScope resourcegate.ClientScopeChecker
	if r, ok := operationRepos.ClientWorkspaceUser.(interface {
		IsActiveAccountTeamMember(ctx context.Context, principalID string, clientID string) (bool, error)
	}); ok {
		clientScope = accountTeamReader{Reader: r}
	}
	var subScope resourcegate.SubscriptionScopeChecker
	if r, ok := operationRepos.SubscriptionWorkspaceUser.(interface {
		IsActiveServicer(ctx context.Context, principalID string, subscriptionID string) (bool, error)
	}); ok {
		subScope = servicerReader{Reader: r}
	}

	return performanceuc.NewUseCase(
		performanceuc.Repositories{
			SubscriptionSeat: operationRepos.SubscriptionSeat,
			Evaluation:       operationRepos.Evaluation,
		},
		performanceuc.Services{
			Authorizer:         authSvc,
			Translator:         i18nSvc,
			ResourceGatekeeper: resourcegate.NewResourceGatekeeper(authSvc, clientScope, subScope),
		},
	)
}
