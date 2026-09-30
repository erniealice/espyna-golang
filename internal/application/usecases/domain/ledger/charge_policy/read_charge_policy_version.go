package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

// ReadChargePolicyVersionRepositories groups the repository dependencies of ReadChargePolicyVersion.
type ReadChargePolicyVersionRepositories struct {
	ChargePolicyVersion   versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyPosting   postingpb.ChargePolicyPostingDomainServiceServer
}

// ReadChargePolicyVersionServices groups the service dependencies of ReadChargePolicyVersion.
type ReadChargePolicyVersionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadChargePolicyVersionUseCase returns one version with its components and postings.
type ReadChargePolicyVersionUseCase struct{ c *core }

// NewReadChargePolicyVersionUseCase creates the use case with grouped dependencies.
func NewReadChargePolicyVersionUseCase(r ReadChargePolicyVersionRepositories, s ReadChargePolicyVersionServices) *ReadChargePolicyVersionUseCase {
	return &ReadChargePolicyVersionUseCase{c: &core{
		repos: Repositories{
			ChargePolicyVersion:   r.ChargePolicyVersion,
			ChargePolicyComponent: r.ChargePolicyComponent,
			ChargePolicyPosting:   r.ChargePolicyPosting,
		},
		svc: Services(s),
	}}
}

// Execute reads one version with its children.
func (uc *ReadChargePolicyVersionUseCase) Execute(ctx context.Context, req *versionpb.ReadChargePolicyVersionRequest) (*versionpb.ReadChargePolicyVersionResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, c.refuse(ctx, CodeValidation, "id is required")
	}
	v, err := c.readVersion(ctx, req.Data.GetId())
	if err != nil {
		return nil, err
	}
	comps, err := c.listComponents(ctx, v.GetId())
	if err != nil {
		return nil, err
	}
	posts, err := c.listPostings(ctx, v.GetId())
	if err != nil {
		return nil, err
	}
	return &versionpb.ReadChargePolicyVersionResponse{Data: []*versionpb.ChargePolicyVersion{v}, Components: comps, Postings: posts, Success: true}, nil
}
