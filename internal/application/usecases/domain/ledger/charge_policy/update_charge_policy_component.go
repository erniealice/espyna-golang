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

// UpdateChargePolicyComponentRepositories groups the repository dependencies of UpdateChargePolicyComponent.
type UpdateChargePolicyComponentRepositories struct {
	ChargePolicyComponent     componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// UpdateChargePolicyComponentServices groups the service dependencies of UpdateChargePolicyComponent.
type UpdateChargePolicyComponentServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateChargePolicyComponentUseCase edits a component of a DRAFT version. The owning version can never
// change. The actor is recorded as a draft editor (C12).
type UpdateChargePolicyComponentUseCase struct{ c *core }

// NewUpdateChargePolicyComponentUseCase creates the use case with grouped dependencies.
func NewUpdateChargePolicyComponentUseCase(r UpdateChargePolicyComponentRepositories, s UpdateChargePolicyComponentServices) *UpdateChargePolicyComponentUseCase {
	return &UpdateChargePolicyComponentUseCase{c: &core{
		repos: Repositories{
			ChargePolicyComponent:     r.ChargePolicyComponent,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute updates the component under its draft version's row lock.
func (uc *UpdateChargePolicyComponentUseCase) Execute(ctx context.Context, req *componentpb.UpdateChargePolicyComponentRequest) (*componentpb.UpdateChargePolicyComponentResponse, error) {
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
	var resp *componentpb.UpdateChargePolicyComponentResponse
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
		ms, ts := stamp()
		patch := req.Data
		patch.ChargePolicyVersionId = "" // immutable owner
		patch.DateModified, patch.DateModifiedString = i64p(ms), strp(ts)
		r, err := c.repos.ChargePolicyComponent.UpdateChargePolicyComponent(txCtx, &componentpb.UpdateChargePolicyComponentRequest{Data: patch})
		if err != nil {
			return usecaseerr.RepoErr("charge_policy", "update charge_policy_component", err, patch.GetId())
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
