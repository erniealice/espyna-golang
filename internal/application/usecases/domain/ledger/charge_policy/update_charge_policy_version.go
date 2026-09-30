package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

// UpdateChargePolicyVersionRepositories groups the repository dependencies of UpdateChargePolicyVersion.
type UpdateChargePolicyVersionRepositories struct {
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
	TaxTreatment              taxtreatmentpb.TaxTreatmentDomainServiceServer
}

// UpdateChargePolicyVersionServices groups the service dependencies of UpdateChargePolicyVersion.
type UpdateChargePolicyVersionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateChargePolicyVersionUseCase edits the classification fields of a DRAFT version. All
// lifecycle-owned fields (status, number, approval and supersede stamps, self_approved,
// cloned_from, prepared_by, policy id) are ignored: they can only change through the
// lifecycle use cases. Anything but a DRAFT is refused with not_draft (AC-CP-02). The actor is
// recorded as a draft editor (C12).
type UpdateChargePolicyVersionUseCase struct{ c *core }

// NewUpdateChargePolicyVersionUseCase creates the use case with grouped dependencies.
func NewUpdateChargePolicyVersionUseCase(r UpdateChargePolicyVersionRepositories, s UpdateChargePolicyVersionServices) *UpdateChargePolicyVersionUseCase {
	return &UpdateChargePolicyVersionUseCase{c: &core{
		repos: Repositories{
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
			TaxTreatment:              r.TaxTreatment,
		},
		svc: Services(s),
	}}
}

// Execute updates the draft's classification fields.
func (uc *UpdateChargePolicyVersionUseCase) Execute(ctx context.Context, req *versionpb.UpdateChargePolicyVersionRequest) (*versionpb.UpdateChargePolicyVersionResponse, error) {
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
	var resp *versionpb.UpdateChargePolicyVersionResponse
	err = c.inTx(ctx, func(txCtx context.Context) error {
		if _, err := c.editDraftVersion(txCtx, req.Data.GetId(), actor); err != nil {
			return err
		}
		if err := c.requireTaxTreatment(txCtx, req.Data.GetTaxTreatmentId()); err != nil {
			return err
		}
		ms, ts := stamp()
		patch := &versionpb.ChargePolicyVersion{
			Id:                 req.Data.GetId(),
			AccountingRole:     req.Data.AccountingRole,
			BookPresentation:   req.Data.BookPresentation,
			TaxPosition:        req.Data.TaxPosition,
			TaxTreatmentId:     req.Data.TaxTreatmentId,
			AssessmentScope:    req.Data.AssessmentScope,
			AssessmentNote:     req.Data.AssessmentNote,
			DateModified:       i64p(ms),
			DateModifiedString: strp(ts),
		}
		r, err := c.repos.ChargePolicyVersion.UpdateChargePolicyVersion(txCtx, &versionpb.UpdateChargePolicyVersionRequest{Data: patch})
		if err != nil {
			return usecaseerr.RepoErr("charge_policy", "update charge_policy_version", err, patch.Id)
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// requireTaxTreatment verifies a request-supplied tax_treatment_id exists before it is stored
// (C5: request-supplied reference ids are read before use). A blank id is not a reference.
func (c *core) requireTaxTreatment(ctx context.Context, id string) error {
	if blank(id) {
		return nil
	}
	if c.repos.TaxTreatment == nil {
		return c.refuse(ctx, CodeUnverifiable, "tax treatment %s cannot be verified", id)
	}
	resp, err := c.repos.TaxTreatment.ReadTaxTreatment(ctx, &taxtreatmentpb.ReadTaxTreatmentRequest{Data: &taxtreatmentpb.TaxTreatment{Id: id}})
	if err != nil {
		if usecaseerr.IsNotFound(err) {
			return c.refuse(ctx, CodeTaxTreatmentNotFound, "tax treatment %s", id)
		}
		return usecaseerr.RepoErr("charge_policy", "read tax_treatment", err, id)
	}
	if resp == nil || len(resp.Data) == 0 {
		return c.refuse(ctx, CodeTaxTreatmentNotFound, "tax treatment %s", id)
	}
	return nil
}
