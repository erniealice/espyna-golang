// Package subscription_group_outcome_export owns the report-authorized,
// subscription-group-scoped composite outcome read and minimal render resolver.
package subscription_group_outcome_export

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

const (
	subscriptionGroupOutcomeExportPermissionEntity = "subscription_group_outcome_export"
	principalTypeOperatorOwner                     = int32(1)
	principalTypeOperatorStaff                     = int32(2)
	principalTypeStaff                             = int32(7)
)

type Repositories struct {
	Query        ports.SubscriptionGroupOutcomeExportQueryService
	LandingQuery ports.SubscriptionGroupOutcomeLandingQueryService
}

type Services struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UseCases struct {
	GetSubscriptionGroupOutcomeExport                *GetUseCase
	ResolveSubscriptionGroupOutcomeDocumentForRender *ResolveDocumentUseCase
	ListSubscriptionGroupOutcomeLanding              *ListSubscriptionGroupOutcomeLandingUseCase
}

func NewUseCases(repositories Repositories, services Services) *UseCases {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	if repositories.LandingQuery == nil {
		if landingQuery, ok := repositories.Query.(ports.SubscriptionGroupOutcomeLandingQueryService); ok {
			repositories.LandingQuery = landingQuery
		}
	}
	return &UseCases{
		GetSubscriptionGroupOutcomeExport:                &GetUseCase{repositories: repositories, services: services},
		ResolveSubscriptionGroupOutcomeDocumentForRender: &ResolveDocumentUseCase{repositories: repositories, services: services},
		ListSubscriptionGroupOutcomeLanding:              &ListSubscriptionGroupOutcomeLandingUseCase{repositories: repositories, services: services},
	}
}

func (s Services) reportScope(ctx context.Context) (ports.SubscriptionGroupOutcomeExportScope, error) {
	if err := s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: subscriptionGroupOutcomeExportPermissionEntity,
		Action: entityid.ActionRead,
	}); err != nil {
		return ports.SubscriptionGroupOutcomeExportScope{}, err
	}

	requestIdentity, ok := identity.FromContext(ctx)
	if !ok || requestIdentity == nil {
		return ports.SubscriptionGroupOutcomeExportScope{}, fmt.Errorf("subscription group outcome export requires an active principal")
	}
	switch requestIdentity.PrincipalType {
	case principalTypeOperatorOwner, principalTypeOperatorStaff, principalTypeStaff:
		// Exact allowlist. Operator principals may bypass group servicing grants
		// only when their active binding also holds workspace:list, matching the
		// established report-card landing scope. Otherwise OPERATOR_STAFF stays
		// servicing-grant scoped. STAFF always keeps that boundary and additionally
		// intersects the reachable-job graph, even if one of its roles happens to
		// carry workspace:list.
	default:
		return ports.SubscriptionGroupOutcomeExportScope{}, fmt.Errorf("subscription group outcome export is unavailable for the active principal")
	}

	operatorPrincipal := requestIdentity.PrincipalType == principalTypeOperatorOwner || requestIdentity.PrincipalType == principalTypeOperatorStaff
	workspaceWide := operatorPrincipal && s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Workspace,
		Action: entityid.ActionList,
	}) == nil
	return ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: workspaceWide}, nil
}
