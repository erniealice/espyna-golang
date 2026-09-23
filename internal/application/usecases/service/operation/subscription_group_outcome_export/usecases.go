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
	jobdoctmplpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

const (
	subscriptionGroupOutcomeExportPermissionEntity = "subscription_group_outcome_export"
	principalTypeOperatorOwner                     = int32(1)
	principalTypeOperatorStaff                     = int32(2)
	principalTypeStaff                             = int32(7)
)

type Repositories struct {
	Query                 ports.SubscriptionGroupOutcomeExportQueryService
	ClientReportCardQuery ports.SubscriptionGroupClientReportCardQueryService
	LandingQuery          ports.SubscriptionGroupOutcomeLandingQueryService
	// JobOutcomeSummaryDocumentTemplate is the SAME josdt repository the domain
	// package's FindApplicableUseCase wraps (job_outcome_summary_document_template:
	// list, a MANAGEMENT-only gate). ResolvePublishedReportCardTemplateUseCase
	// calls FindApplicableJobOutcomeSummaryDocumentTemplate on it DIRECTLY,
	// bypassing that gate, and authorizes instead via this package's report
	// scope (subscription_group_outcome_export:read) — see R3 / DEC-3. Nil-safe:
	// an absent repo makes the render-scoped resolver return "unavailable".
	JobOutcomeSummaryDocumentTemplate jobdoctmplpb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
}

type Services struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UseCases struct {
	GetSubscriptionGroupOutcomeExport                *GetUseCase
	GetSubscriptionGroupClientReportCard             *GetClientReportCardUseCase
	ResolveSubscriptionGroupOutcomeDocumentForRender *ResolveDocumentUseCase
	ListSubscriptionGroupOutcomeLanding              *ListSubscriptionGroupOutcomeLandingUseCase
	// ResolvePublishedReportCardTemplate is the render-scoped josdt resolver
	// (R3 / DEC-3): same repository call as the domain package's
	// FindApplicableJobOutcomeSummaryDocumentTemplate, but authorized against
	// subscription_group_outcome_export:read so a STAFF principal (type 7) can
	// resolve the published binding it is already entitled to read/export.
	ResolvePublishedReportCardTemplate *ResolvePublishedReportCardTemplateUseCase
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
		GetSubscriptionGroupClientReportCard:             &GetClientReportCardUseCase{repositories: repositories, services: services},
		ResolveSubscriptionGroupOutcomeDocumentForRender: &ResolveDocumentUseCase{repositories: repositories, services: services},
		ListSubscriptionGroupOutcomeLanding:              &ListSubscriptionGroupOutcomeLandingUseCase{repositories: repositories, services: services},
		ResolvePublishedReportCardTemplate:               &ResolvePublishedReportCardTemplateUseCase{repositories: repositories, services: services},
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
		// servicing-grant scoped. STAFF stays servicing-grant scoped even if one of
		// its roles happens to carry workspace:list. Since the 2026-09-21 owner
		// decision an assigned STAFF principal reads the WHOLE assigned section
		// (read-only): the adapter no longer intersects the reachable-job graph
		// (see narrowStaffReportsToReachableJobs in the postgres adapter).
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
