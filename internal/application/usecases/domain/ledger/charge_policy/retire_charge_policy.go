package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// RetireChargePolicyRepositories groups the repository dependencies of RetireChargePolicy.
type RetireChargePolicyRepositories struct {
	ChargePolicy policypb.ChargePolicyDomainServiceServer
}

// RetireChargePolicyServices groups the service dependencies of RetireChargePolicy.
type RetireChargePolicyServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// RetireChargePolicyUseCase moves ACTIVE -> RETIRED at any time (AC-CP-05). An open draft is
// kept but can no longer be approved; existing product_price_plan references are untouched.
type RetireChargePolicyUseCase struct{ c *core }

// NewRetireChargePolicyUseCase creates the use case with grouped dependencies.
func NewRetireChargePolicyUseCase(r RetireChargePolicyRepositories, s RetireChargePolicyServices) *RetireChargePolicyUseCase {
	return &RetireChargePolicyUseCase{c: &core{
		repos: Repositories{
			ChargePolicy: r.ChargePolicy,
		},
		svc: Services(s),
	}}
}

// Execute retires the policy under its row lock.
func (uc *RetireChargePolicyUseCase) Execute(ctx context.Context, req *policypb.RetireChargePolicyRequest) (*policypb.RetireChargePolicyResponse, error) {
	c := uc.c
	if err := c.gateStrict(ctx, entityid.ActionRetire); err != nil {
		return nil, err
	}
	actor, err := c.actor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || blank(req.GetChargePolicyId()) {
		return nil, c.refuse(ctx, CodeValidation, "id is required")
	}
	var out *policypb.ChargePolicy
	err = c.inTx(ctx, func(txCtx context.Context) error {
		p, lErr := c.lockPolicy(txCtx, req.GetChargePolicyId())
		if lErr != nil {
			return lErr
		}
		if p.GetStatus() == enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED {
			return c.refuse(txCtx, CodeRetired, "charge policy %s is already retired", p.GetId())
		}
		ms, ts := stamp()
		resp, uErr := c.repos.ChargePolicy.UpdateChargePolicy(txCtx, &policypb.UpdateChargePolicyRequest{Data: &policypb.ChargePolicy{
			Id:                 p.GetId(),
			Status:             enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED,
			RetiredAt:          i64p(ms),
			RetiredBy:          strp(actor),
			DateModified:       i64p(ms),
			DateModifiedString: strp(ts),
		}})
		if uErr != nil {
			return usecaseerr.RepoErr("charge_policy", "retire charge_policy", uErr, p.GetId())
		}
		if resp != nil && len(resp.Data) > 0 {
			out = resp.Data[0]
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &policypb.RetireChargePolicyResponse{Data: out, Success: true}, nil
}
