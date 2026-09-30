package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
)

// UpdateChargePolicyRepositories groups the repository dependencies of UpdateChargePolicy.
type UpdateChargePolicyRepositories struct {
	ChargePolicy policypb.ChargePolicyDomainServiceServer
}

// UpdateChargePolicyServices groups the service dependencies of UpdateChargePolicy.
type UpdateChargePolicyServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateChargePolicyUseCase edits name/description only. Code, status and lifecycle fields
// are immutable through this path.
type UpdateChargePolicyUseCase struct{ c *core }

// NewUpdateChargePolicyUseCase creates the use case with grouped dependencies.
func NewUpdateChargePolicyUseCase(r UpdateChargePolicyRepositories, s UpdateChargePolicyServices) *UpdateChargePolicyUseCase {
	return &UpdateChargePolicyUseCase{c: &core{
		repos: Repositories{
			ChargePolicy: r.ChargePolicy,
		},
		svc: Services(s),
	}}
}

// Execute updates the header's name/description.
func (uc *UpdateChargePolicyUseCase) Execute(ctx context.Context, req *policypb.UpdateChargePolicyRequest) (*policypb.UpdateChargePolicyResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, c.refuse(ctx, CodeValidation, "id is required")
	}
	if req.Data.Name != "" && blank(req.Data.GetName()) {
		return nil, c.refuse(ctx, CodeValidation, "name must not be blank")
	}
	if _, err := c.readPolicy(ctx, req.Data.GetId()); err != nil {
		return nil, err
	}
	ms, ts := stamp()
	patch := &policypb.ChargePolicy{
		Id:                 req.Data.GetId(),
		Name:               req.Data.GetName(),
		Description:        req.Data.Description,
		DateModified:       i64p(ms),
		DateModifiedString: strp(ts),
	}
	resp, err := c.repos.ChargePolicy.UpdateChargePolicy(ctx, &policypb.UpdateChargePolicyRequest{Data: patch})
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "update charge_policy", err, patch.Id)
	}
	return resp, nil
}
