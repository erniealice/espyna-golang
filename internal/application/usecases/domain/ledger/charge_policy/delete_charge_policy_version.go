package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
)

// DeleteChargePolicyVersionRepositories groups the repository dependencies of DeleteChargePolicyVersion.
type DeleteChargePolicyVersionRepositories struct {
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent     componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyPosting       postingpb.ChargePolicyPostingDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// DeleteChargePolicyVersionServices groups the service dependencies of DeleteChargePolicyVersion.
type DeleteChargePolicyVersionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteChargePolicyVersionUseCase deletes a DRAFT version with its components, postings and
// editors. A version that was ever approved (APPROVED / SUPERSEDED) can never be deleted. Gated
// by charge_policy:delete, matching the view's delete action (R3: was `update`).
type DeleteChargePolicyVersionUseCase struct{ c *core }

// NewDeleteChargePolicyVersionUseCase creates the use case with grouped dependencies.
func NewDeleteChargePolicyVersionUseCase(r DeleteChargePolicyVersionRepositories, s DeleteChargePolicyVersionServices) *DeleteChargePolicyVersionUseCase {
	return &DeleteChargePolicyVersionUseCase{c: &core{
		repos: Repositories{
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyComponent:     r.ChargePolicyComponent,
			ChargePolicyPosting:       r.ChargePolicyPosting,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute deletes a draft version tree under the version row lock.
func (uc *DeleteChargePolicyVersionUseCase) Execute(ctx context.Context, req *versionpb.DeleteChargePolicyVersionRequest) (*versionpb.DeleteChargePolicyVersionResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, c.refuse(ctx, CodeValidation, "id is required")
	}
	err := c.inTx(ctx, func(txCtx context.Context) error {
		if _, err := c.requireDraftVersion(txCtx, req.Data.GetId()); err != nil {
			return err
		}
		return c.deleteVersionTree(txCtx, req.Data.GetId())
	})
	if err != nil {
		return nil, err
	}
	return &versionpb.DeleteChargePolicyVersionResponse{Success: true}, nil
}
