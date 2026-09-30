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

// CreateChargePolicyPostingRepositories groups the repository dependencies of CreateChargePolicyPosting.
type CreateChargePolicyPostingRepositories struct {
	ChargePolicyPosting       postingpb.ChargePolicyPostingDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// CreateChargePolicyPostingServices groups the service dependencies of CreateChargePolicyPosting.
type CreateChargePolicyPostingServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// CreateChargePolicyPostingUseCase adds a posting to a DRAFT version (AC-CP-02: any other version
// status is refused with not_draft). The parent version row is locked FOR UPDATE first, so an
// approval and a concurrent edit serialize; the actor is recorded as a draft editor (C12).
type CreateChargePolicyPostingUseCase struct{ c *core }

// NewCreateChargePolicyPostingUseCase creates the use case with grouped dependencies.
func NewCreateChargePolicyPostingUseCase(r CreateChargePolicyPostingRepositories, s CreateChargePolicyPostingServices) *CreateChargePolicyPostingUseCase {
	return &CreateChargePolicyPostingUseCase{c: &core{
		repos: Repositories{
			ChargePolicyPosting:       r.ChargePolicyPosting,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute adds the posting to its draft version.
func (uc *CreateChargePolicyPostingUseCase) Execute(ctx context.Context, req *postingpb.CreateChargePolicyPostingRequest) (*postingpb.CreateChargePolicyPostingResponse, error) {
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
	if req.Data.GetEvent() == 0 || req.Data.GetPostingRole() == 0 || blank(req.Data.GetAccountId()) {
		return nil, c.refuse(ctx, CodeValidation, "event, posting role and account are required")
	}
	var resp *postingpb.CreateChargePolicyPostingResponse
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
		r, err := c.repos.ChargePolicyPosting.CreateChargePolicyPosting(txCtx, &postingpb.CreateChargePolicyPostingRequest{Data: row})
		if err != nil {
			return usecaseerr.RepoErr("charge_policy", "create charge_policy_posting", err, row.GetChargePolicyVersionId())
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
