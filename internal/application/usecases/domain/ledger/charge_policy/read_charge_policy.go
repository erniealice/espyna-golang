package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

// ReadChargePolicyRepositories groups the repository dependencies of ReadChargePolicy.
type ReadChargePolicyRepositories struct {
	ChargePolicy        policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion versionpb.ChargePolicyVersionDomainServiceServer
}

// ReadChargePolicyServices groups the service dependencies of ReadChargePolicy.
type ReadChargePolicyServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadChargePolicyUseCase returns the header and all its versions (newest first is up to the view).
type ReadChargePolicyUseCase struct{ c *core }

// NewReadChargePolicyUseCase creates the use case with grouped dependencies.
func NewReadChargePolicyUseCase(r ReadChargePolicyRepositories, s ReadChargePolicyServices) *ReadChargePolicyUseCase {
	return &ReadChargePolicyUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:        r.ChargePolicy,
			ChargePolicyVersion: r.ChargePolicyVersion,
		},
		svc: Services(s),
	}}
}

// Execute reads one policy with its versions.
func (uc *ReadChargePolicyUseCase) Execute(ctx context.Context, req *policypb.ReadChargePolicyRequest) (*policypb.ReadChargePolicyResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, c.refuse(ctx, CodeValidation, "id is required")
	}
	p, err := c.readPolicy(ctx, req.Data.GetId())
	if err != nil {
		return nil, err
	}
	vs, err := c.listVersions(ctx, p.GetId())
	if err != nil {
		return nil, err
	}
	return &policypb.ReadChargePolicyResponse{Data: []*policypb.ChargePolicy{p}, Versions: vs, Success: true}, nil
}
