package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

// ListChargePolicyVersionsRepositories groups the repository dependencies of ListChargePolicyVersions.
type ListChargePolicyVersionsRepositories struct {
	ChargePolicyVersion versionpb.ChargePolicyVersionDomainServiceServer
}

// ListChargePolicyVersionsServices groups the service dependencies of ListChargePolicyVersions.
type ListChargePolicyVersionsServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListChargePolicyVersionsUseCase lists the versions of one policy (scope charge_policy_id).
type ListChargePolicyVersionsUseCase struct{ c *core }

// NewListChargePolicyVersionsUseCase creates the use case with grouped dependencies.
func NewListChargePolicyVersionsUseCase(r ListChargePolicyVersionsRepositories, s ListChargePolicyVersionsServices) *ListChargePolicyVersionsUseCase {
	return &ListChargePolicyVersionsUseCase{c: &core{
		repos: Repositories{
			ChargePolicyVersion: r.ChargePolicyVersion,
		},
		svc: Services(s),
	}}
}

// Execute lists the versions of the scoped policy (paginated by the request).
func (uc *ListChargePolicyVersionsUseCase) Execute(ctx context.Context, req *versionpb.ListChargePolicyVersionsRequest) (*versionpb.ListChargePolicyVersionsResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || blank(req.GetChargePolicyId()) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy id is required")
	}
	filters, err := c.scopeFilter(ctx, req.GetFilters(), "charge_policy_id", req.GetChargePolicyId())
	if err != nil {
		return nil, err
	}
	scoped := &versionpb.ListChargePolicyVersionsRequest{Search: req.Search, Filters: filters, Sort: req.Sort, Pagination: req.Pagination, ChargePolicyId: req.ChargePolicyId}
	resp, err := c.repos.ChargePolicyVersion.ListChargePolicyVersions(ctx, scoped)
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_version", err, req.GetChargePolicyId())
	}
	return resp, nil
}
