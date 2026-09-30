package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

// GetChargePolicyInUseIdsRepositories groups the repository dependencies of GetChargePolicyInUseIds.
type GetChargePolicyInUseIdsRepositories struct {
	ChargePolicy        policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion versionpb.ChargePolicyVersionDomainServiceServer
}

// GetChargePolicyInUseIdsServices groups the service dependencies of GetChargePolicyInUseIds.
type GetChargePolicyInUseIdsServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetChargePolicyInUseIdsUseCase returns, of the given policies, the ids that cannot be
// deleted: some version was ever APPROVED/SUPERSEDED, or a product_price_plan references it.
type GetChargePolicyInUseIdsUseCase struct{ c *core }

// NewGetChargePolicyInUseIdsUseCase creates the use case with grouped dependencies.
func NewGetChargePolicyInUseIdsUseCase(r GetChargePolicyInUseIdsRepositories, s GetChargePolicyInUseIdsServices) *GetChargePolicyInUseIdsUseCase {
	return &GetChargePolicyInUseIdsUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:        r.ChargePolicy,
			ChargePolicyVersion: r.ChargePolicyVersion,
		},
		svc: Services(s),
	}}
}

// Execute reports the in-use subset of the requested ids.
func (uc *GetChargePolicyInUseIdsUseCase) Execute(ctx context.Context, req *policypb.GetChargePolicyInUseIdsRequest) (*policypb.GetChargePolicyInUseIdsResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	used, err := c.inUse(ctx, req.GetChargePolicyIds())
	if err != nil {
		return nil, err
	}
	out := &policypb.GetChargePolicyInUseIdsResponse{Success: true}
	for _, id := range req.GetChargePolicyIds() {
		if used[id] {
			out.InUseChargePolicyIds = append(out.InUseChargePolicyIds, id)
		}
	}
	return out, nil
}
