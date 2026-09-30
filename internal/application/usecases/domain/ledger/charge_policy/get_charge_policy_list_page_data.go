package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
)

// GetChargePolicyListPageDataRepositories groups the repository dependencies of GetChargePolicyListPageData.
type GetChargePolicyListPageDataRepositories struct {
	ChargePolicy policypb.ChargePolicyDomainServiceServer
}

// GetChargePolicyListPageDataServices groups the service dependencies of GetChargePolicyListPageData.
type GetChargePolicyListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetChargePolicyListPageDataUseCase serves the paginated list page (charge_policy:list).
type GetChargePolicyListPageDataUseCase struct{ c *core }

// NewGetChargePolicyListPageDataUseCase creates the use case with grouped dependencies.
func NewGetChargePolicyListPageDataUseCase(r GetChargePolicyListPageDataRepositories, s GetChargePolicyListPageDataServices) *GetChargePolicyListPageDataUseCase {
	return &GetChargePolicyListPageDataUseCase{c: &core{
		repos: Repositories{
			ChargePolicy: r.ChargePolicy,
		},
		svc: Services(s),
	}}
}

// Execute returns one page of policies.
func (uc *GetChargePolicyListPageDataUseCase) Execute(ctx context.Context, req *policypb.GetChargePolicyListPageDataRequest) (*policypb.GetChargePolicyListPageDataResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		req = &policypb.GetChargePolicyListPageDataRequest{}
	}
	resp, err := c.repos.ChargePolicy.GetChargePolicyListPageData(ctx, req)
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "get charge_policy list page", err)
	}
	return resp, nil
}
