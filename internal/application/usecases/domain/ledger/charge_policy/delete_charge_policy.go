package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
)

// DeleteChargePolicyRepositories groups the repository dependencies of DeleteChargePolicy.
type DeleteChargePolicyRepositories struct {
	ChargePolicy              policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent     componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyPosting       postingpb.ChargePolicyPostingDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// DeleteChargePolicyServices groups the service dependencies of DeleteChargePolicy.
type DeleteChargePolicyServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteChargePolicyUseCase deletes a policy only when no version was ever APPROVED and no
// product_price_plan references it; its draft version, components, postings and editors go with it.
type DeleteChargePolicyUseCase struct{ c *core }

// NewDeleteChargePolicyUseCase creates the use case with grouped dependencies.
func NewDeleteChargePolicyUseCase(r DeleteChargePolicyRepositories, s DeleteChargePolicyServices) *DeleteChargePolicyUseCase {
	return &DeleteChargePolicyUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:              r.ChargePolicy,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyComponent:     r.ChargePolicyComponent,
			ChargePolicyPosting:       r.ChargePolicyPosting,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute deletes a never-approved, unreferenced policy under its row lock.
func (uc *DeleteChargePolicyUseCase) Execute(ctx context.Context, req *policypb.DeleteChargePolicyRequest) (*policypb.DeleteChargePolicyResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, c.refuse(ctx, CodeValidation, "id is required")
	}
	err := c.inTx(ctx, func(txCtx context.Context) error {
		p, err := c.lockPolicy(txCtx, req.Data.GetId())
		if err != nil {
			return err
		}
		used, err := c.inUse(txCtx, []string{p.GetId()})
		if err != nil {
			return err
		}
		if used[p.GetId()] {
			return c.refuse(txCtx, CodeInUse, "charge policy %s", p.GetId())
		}
		vs, err := c.listVersions(txCtx, p.GetId())
		if err != nil {
			return err
		}
		for _, v := range vs {
			if err := c.deleteVersionTree(txCtx, v.GetId()); err != nil {
				return err
			}
		}
		if _, err := c.repos.ChargePolicy.DeleteChargePolicy(txCtx, &policypb.DeleteChargePolicyRequest{Data: &policypb.ChargePolicy{Id: p.GetId()}}); err != nil {
			return usecaseerr.RepoErr("charge_policy", "delete charge_policy", err, p.GetId())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &policypb.DeleteChargePolicyResponse{Success: true}, nil
}
