package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
)

// DeleteChargePolicyComponentRepositories groups the repository dependencies of DeleteChargePolicyComponent.
type DeleteChargePolicyComponentRepositories struct {
	ChargePolicyComponent     componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// DeleteChargePolicyComponentServices groups the service dependencies of DeleteChargePolicyComponent.
type DeleteChargePolicyComponentServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteChargePolicyComponentUseCase removes a component from a DRAFT version. The actor is recorded as a
// draft editor (C12).
type DeleteChargePolicyComponentUseCase struct{ c *core }

// NewDeleteChargePolicyComponentUseCase creates the use case with grouped dependencies.
func NewDeleteChargePolicyComponentUseCase(r DeleteChargePolicyComponentRepositories, s DeleteChargePolicyComponentServices) *DeleteChargePolicyComponentUseCase {
	return &DeleteChargePolicyComponentUseCase{c: &core{
		repos: Repositories{
			ChargePolicyComponent:     r.ChargePolicyComponent,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute deletes the component under its draft version's row lock.
func (uc *DeleteChargePolicyComponentUseCase) Execute(ctx context.Context, req *componentpb.DeleteChargePolicyComponentRequest) (*componentpb.DeleteChargePolicyComponentResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	actor, err := c.actor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, c.refuse(ctx, CodeValidation, "id is required")
	}
	err = c.inTx(ctx, func(txCtx context.Context) error {
		cur, err := c.repos.ChargePolicyComponent.ReadChargePolicyComponent(txCtx, &componentpb.ReadChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{Id: req.Data.GetId()}})
		if err != nil {
			if usecaseerr.IsNotFound(err) {
				return c.refuse(txCtx, CodeNotFound, "component %s", req.Data.GetId())
			}
			return usecaseerr.RepoErr("charge_policy", "read charge_policy_component", err, req.Data.GetId())
		}
		if cur == nil || len(cur.Data) == 0 {
			return c.refuse(txCtx, CodeNotFound, "component %s", req.Data.GetId())
		}
		if _, err := c.editDraftVersion(txCtx, cur.Data[0].GetChargePolicyVersionId(), actor); err != nil {
			return err
		}
		if _, err := c.repos.ChargePolicyComponent.DeleteChargePolicyComponent(txCtx, &componentpb.DeleteChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{Id: req.Data.GetId()}}); err != nil {
			return usecaseerr.RepoErr("charge_policy", "delete charge_policy_component", err, req.Data.GetId())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &componentpb.DeleteChargePolicyComponentResponse{Success: true}, nil
}
