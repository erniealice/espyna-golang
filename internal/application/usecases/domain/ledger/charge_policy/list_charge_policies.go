package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
)

// ListChargePoliciesRepositories groups the repository dependencies of ListChargePolicies.
type ListChargePoliciesRepositories struct {
	ChargePolicy policypb.ChargePolicyDomainServiceServer
}

// ListChargePoliciesServices groups the service dependencies of ListChargePolicies.
type ListChargePoliciesServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListChargePoliciesUseCase lists policies of the caller's workspace (charge_policy:list).
type ListChargePoliciesUseCase struct{ c *core }

// NewListChargePoliciesUseCase creates the use case with grouped dependencies.
func NewListChargePoliciesUseCase(r ListChargePoliciesRepositories, s ListChargePoliciesServices) *ListChargePoliciesUseCase {
	return &ListChargePoliciesUseCase{c: &core{
		repos: Repositories{
			ChargePolicy: r.ChargePolicy,
		},
		svc: Services(s),
	}}
}

// Execute lists policies (paginated by the request).
func (uc *ListChargePoliciesUseCase) Execute(ctx context.Context, req *policypb.ListChargePoliciesRequest) (*policypb.ListChargePoliciesResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		req = &policypb.ListChargePoliciesRequest{}
	}
	resp, err := c.repos.ChargePolicy.ListChargePolicies(ctx, req)
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy", err)
	}
	return resp, nil
}
