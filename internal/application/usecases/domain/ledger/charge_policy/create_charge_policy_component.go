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

// CreateChargePolicyComponentRepositories groups the repository dependencies of CreateChargePolicyComponent.
type CreateChargePolicyComponentRepositories struct {
	ChargePolicyComponent     componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// CreateChargePolicyComponentServices groups the service dependencies of CreateChargePolicyComponent.
type CreateChargePolicyComponentServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// CreateChargePolicyComponentUseCase adds a component to a DRAFT version (AC-CP-02: any other version
// status is refused with not_draft). The parent version row is locked FOR UPDATE first, so an
// approval and a concurrent edit serialize; the actor is recorded as a draft editor (C12).
type CreateChargePolicyComponentUseCase struct{ c *core }

// NewCreateChargePolicyComponentUseCase creates the use case with grouped dependencies.
func NewCreateChargePolicyComponentUseCase(r CreateChargePolicyComponentRepositories, s CreateChargePolicyComponentServices) *CreateChargePolicyComponentUseCase {
	return &CreateChargePolicyComponentUseCase{c: &core{
		repos: Repositories{
			ChargePolicyComponent:     r.ChargePolicyComponent,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute adds the component to its draft version.
func (uc *CreateChargePolicyComponentUseCase) Execute(ctx context.Context, req *componentpb.CreateChargePolicyComponentRequest) (*componentpb.CreateChargePolicyComponentResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	actor, err := c.actor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetChargePolicyVersionId()) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy version id is required")
	}
	if req.Data.GetComponentRole() == 0 || req.Data.GetDocumentKind() == 0 || req.Data.GetBookPresentation() == 0 {
		return nil, c.refuse(ctx, CodeValidation, "component role, document kind and book presentation are required")
	}
	var resp *componentpb.CreateChargePolicyComponentResponse
	err = c.inTx(ctx, func(txCtx context.Context) error {
		if _, err := c.editDraftVersion(txCtx, req.Data.GetChargePolicyVersionId(), actor); err != nil {
			return err
		}
		ms, ts := stamp()
		row := req.Data
		row.Id = c.newID()
		row.Active = true
		row.DateCreated, row.DateCreatedString = i64p(ms), strp(ts)
		row.DateModified, row.DateModifiedString = i64p(ms), strp(ts)
		r, err := c.repos.ChargePolicyComponent.CreateChargePolicyComponent(txCtx, &componentpb.CreateChargePolicyComponentRequest{Data: row})
		if err != nil {
			return usecaseerr.RepoErr("charge_policy", "create charge_policy_component", err, row.GetChargePolicyVersionId())
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
