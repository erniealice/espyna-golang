package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
)

// ListChargePolicyComponentsRepositories groups the repository dependencies of ListChargePolicyComponents.
type ListChargePolicyComponentsRepositories struct {
	ChargePolicyComponent componentpb.ChargePolicyComponentDomainServiceServer
}

// ListChargePolicyComponentsServices groups the service dependencies of ListChargePolicyComponents.
type ListChargePolicyComponentsServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListChargePolicyComponentsUseCase lists the components of one version (scope charge_policy_version_id).
type ListChargePolicyComponentsUseCase struct{ c *core }

// NewListChargePolicyComponentsUseCase creates the use case with grouped dependencies.
func NewListChargePolicyComponentsUseCase(r ListChargePolicyComponentsRepositories, s ListChargePolicyComponentsServices) *ListChargePolicyComponentsUseCase {
	return &ListChargePolicyComponentsUseCase{c: &core{
		repos: Repositories{
			ChargePolicyComponent: r.ChargePolicyComponent,
		},
		svc: Services(s),
	}}
}

// Execute lists the components of the scoped version (paginated by the request).
func (uc *ListChargePolicyComponentsUseCase) Execute(ctx context.Context, req *componentpb.ListChargePolicyComponentsRequest) (*componentpb.ListChargePolicyComponentsResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || blank(req.GetChargePolicyVersionId()) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy version id is required")
	}
	filters, err := c.scopeFilter(ctx, req.GetFilters(), "charge_policy_version_id", req.GetChargePolicyVersionId())
	if err != nil {
		return nil, err
	}
	scoped := &componentpb.ListChargePolicyComponentsRequest{Search: req.Search, Filters: filters, Sort: req.Sort, Pagination: req.Pagination, ChargePolicyVersionId: req.ChargePolicyVersionId}
	resp, err := c.repos.ChargePolicyComponent.ListChargePolicyComponents(ctx, scoped)
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_component", err, req.GetChargePolicyVersionId())
	}
	return resp, nil
}
