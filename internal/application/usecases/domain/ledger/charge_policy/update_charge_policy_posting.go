package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
)

// UpdateChargePolicyPostingRepositories groups the repository dependencies of UpdateChargePolicyPosting.
type UpdateChargePolicyPostingRepositories struct {
	ChargePolicyPosting       postingpb.ChargePolicyPostingDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// UpdateChargePolicyPostingServices groups the service dependencies of UpdateChargePolicyPosting.
type UpdateChargePolicyPostingServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateChargePolicyPostingUseCase edits a posting of a DRAFT version. The owning version can never
// change. The actor is recorded as a draft editor (C12).
type UpdateChargePolicyPostingUseCase struct{ c *core }

// NewUpdateChargePolicyPostingUseCase creates the use case with grouped dependencies.
func NewUpdateChargePolicyPostingUseCase(r UpdateChargePolicyPostingRepositories, s UpdateChargePolicyPostingServices) *UpdateChargePolicyPostingUseCase {
	return &UpdateChargePolicyPostingUseCase{c: &core{
		repos: Repositories{
			ChargePolicyPosting:       r.ChargePolicyPosting,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute updates the posting under its draft version's row lock.
func (uc *UpdateChargePolicyPostingUseCase) Execute(ctx context.Context, req *postingpb.UpdateChargePolicyPostingRequest) (*postingpb.UpdateChargePolicyPostingResponse, error) {
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
	var resp *postingpb.UpdateChargePolicyPostingResponse
	err = c.inTx(ctx, func(txCtx context.Context) error {
		cur, err := c.repos.ChargePolicyPosting.ReadChargePolicyPosting(txCtx, &postingpb.ReadChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: req.Data.GetId()}})
		if err != nil {
			if usecaseerr.IsNotFound(err) {
				return c.refuse(txCtx, CodeNotFound, "posting %s", req.Data.GetId())
			}
			return usecaseerr.RepoErr("charge_policy", "read charge_policy_posting", err, req.Data.GetId())
		}
		if cur == nil || len(cur.Data) == 0 {
			return c.refuse(txCtx, CodeNotFound, "posting %s", req.Data.GetId())
		}
		if _, err := c.editDraftVersion(txCtx, cur.Data[0].GetChargePolicyVersionId(), actor); err != nil {
			return err
		}
		ms, ts := stamp()
		patch := req.Data
		patch.ChargePolicyVersionId = "" // immutable owner
		patch.DateModified, patch.DateModifiedString = i64p(ms), strp(ts)
		r, err := c.repos.ChargePolicyPosting.UpdateChargePolicyPosting(txCtx, &postingpb.UpdateChargePolicyPostingRequest{Data: patch})
		if err != nil {
			return usecaseerr.RepoErr("charge_policy", "update charge_policy_posting", err, patch.GetId())
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
