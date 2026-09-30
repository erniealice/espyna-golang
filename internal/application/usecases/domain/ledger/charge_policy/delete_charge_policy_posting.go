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

// DeleteChargePolicyPostingRepositories groups the repository dependencies of DeleteChargePolicyPosting.
type DeleteChargePolicyPostingRepositories struct {
	ChargePolicyPosting       postingpb.ChargePolicyPostingDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// DeleteChargePolicyPostingServices groups the service dependencies of DeleteChargePolicyPosting.
type DeleteChargePolicyPostingServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteChargePolicyPostingUseCase removes a posting from a DRAFT version. The actor is recorded as a
// draft editor (C12).
type DeleteChargePolicyPostingUseCase struct{ c *core }

// NewDeleteChargePolicyPostingUseCase creates the use case with grouped dependencies.
func NewDeleteChargePolicyPostingUseCase(r DeleteChargePolicyPostingRepositories, s DeleteChargePolicyPostingServices) *DeleteChargePolicyPostingUseCase {
	return &DeleteChargePolicyPostingUseCase{c: &core{
		repos: Repositories{
			ChargePolicyPosting:       r.ChargePolicyPosting,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute deletes the posting under its draft version's row lock.
func (uc *DeleteChargePolicyPostingUseCase) Execute(ctx context.Context, req *postingpb.DeleteChargePolicyPostingRequest) (*postingpb.DeleteChargePolicyPostingResponse, error) {
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
		if _, err := c.repos.ChargePolicyPosting.DeleteChargePolicyPosting(txCtx, &postingpb.DeleteChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: req.Data.GetId()}}); err != nil {
			return usecaseerr.RepoErr("charge_policy", "delete charge_policy_posting", err, req.Data.GetId())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &postingpb.DeleteChargePolicyPostingResponse{Success: true}, nil
}
