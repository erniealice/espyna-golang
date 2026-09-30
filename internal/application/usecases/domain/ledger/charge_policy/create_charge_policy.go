package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"fmt"
	"regexp"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// CreateChargePolicyRepositories groups the repository dependencies of CreateChargePolicy.
type CreateChargePolicyRepositories struct {
	ChargePolicy              policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// CreateChargePolicyServices groups the service dependencies of CreateChargePolicy.
type CreateChargePolicyServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// CreateChargePolicyUseCase creates the policy (ACTIVE) and draft version 1 in ONE
// transaction (AC-CP-01). Callers cannot choose status, version number or workspace.
type CreateChargePolicyUseCase struct{ c *core }

// NewCreateChargePolicyUseCase creates the use case with grouped dependencies.
func NewCreateChargePolicyUseCase(r CreateChargePolicyRepositories, s CreateChargePolicyServices) *CreateChargePolicyUseCase {
	return &CreateChargePolicyUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:              r.ChargePolicy,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

var codePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Execute creates the policy and its draft v1; the preparer is recorded as the draft's first editor.
func (uc *CreateChargePolicyUseCase) Execute(ctx context.Context, req *policypb.CreateChargePolicyRequest) (*policypb.CreateChargePolicyResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionCreate); err != nil {
		return nil, err
	}
	actor, err := c.actor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, c.refuse(ctx, CodeValidation, "data is required")
	}
	if !codePattern.MatchString(req.Data.GetCode()) {
		return nil, c.refuse(ctx, CodeValidation, "code must be UPPER_SNAKE_CASE")
	}
	if blank(req.Data.GetName()) {
		return nil, c.refuse(ctx, CodeValidation, "name is required")
	}
	ms, ts := stamp()
	policy := &policypb.ChargePolicy{
		Id:                 c.newID(),
		Code:               req.Data.GetCode(),
		Name:               req.Data.GetName(),
		Description:        req.Data.Description,
		Status:             enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE,
		CreatedBy:          strp(actor),
		Active:             true,
		DateCreated:        i64p(ms),
		DateCreatedString:  strp(ts),
		DateModified:       i64p(ms),
		DateModifiedString: strp(ts),
	}
	out := &policypb.CreateChargePolicyResponse{Success: true}
	err = c.inTx(ctx, func(txCtx context.Context) error {
		created, cErr := c.repos.ChargePolicy.CreateChargePolicy(txCtx, &policypb.CreateChargePolicyRequest{Data: policy})
		if cErr != nil {
			return usecaseerr.RepoErr("charge_policy", "create charge_policy", cErr)
		}
		if created == nil || len(created.Data) == 0 {
			return usecaseerr.RepoErr("charge_policy", "create charge_policy", fmt.Errorf("no row returned"))
		}
		out.Data = []*policypb.ChargePolicy{created.Data[0]}
		draft, dErr := c.newDraftVersion(txCtx, created.Data[0].GetId(), 1, actor, "")
		if dErr != nil {
			return dErr
		}
		out.Draft = draft
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// newDraftVersion inserts a DRAFT version row (no children) and records the preparer as its first editor.
func (c *core) newDraftVersion(ctx context.Context, policyID string, number int32, actor, clonedFrom string) (*versionpb.ChargePolicyVersion, error) {
	ms, ts := stamp()
	v := &versionpb.ChargePolicyVersion{
		Id:                 c.newID(),
		ChargePolicyId:     policyID,
		VersionNumber:      number,
		Status:             enumspbVersionDraft,
		PreparedBy:         strp(actor),
		Active:             true,
		DateCreated:        i64p(ms),
		DateCreatedString:  strp(ts),
		DateModified:       i64p(ms),
		DateModifiedString: strp(ts),
	}
	if clonedFrom != "" {
		v.ClonedFromVersionId = strp(clonedFrom)
	}
	resp, err := c.repos.ChargePolicyVersion.CreateChargePolicyVersion(ctx, &versionpb.CreateChargePolicyVersionRequest{Data: v})
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "create charge_policy_version", err, policyID)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, usecaseerr.RepoErr("charge_policy", "create charge_policy_version", fmt.Errorf("no row returned"), policyID)
	}
	if err := c.recordEditor(ctx, resp.Data[0].GetId(), actor); err != nil {
		return nil, err
	}
	return resp.Data[0], nil
}
