package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// ApproveChargePolicyVersionRepositories groups the repository dependencies of ApproveChargePolicyVersion.
type ApproveChargePolicyVersionRepositories struct {
	ChargePolicy              policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent     componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyPosting       postingpb.ChargePolicyPostingDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
	Account                   accountpb.AccountDomainServiceServer
}

// ApproveChargePolicyVersionServices groups the service dependencies of ApproveChargePolicyVersion.
type ApproveChargePolicyVersionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ApproveChargePolicyVersionUseCase approves a DRAFT version (AC-CP-04 / AC-UC-36):
//
//  1. permission charge_policy:approve (strict, shadow-immune);
//  2. separation of duties (C12): the actor is "self" when they prepared the draft, when ANY
//     recorded draft editor is the actor, or when prepared_by is empty (unknown authorship fails
//     closed). A self approval also requires charge_policy:approve_own and stamps
//     self_approved = true (audit flag);
//  3. S1 matrix (unsupported_combination) then S1 checklist (checklist_failed);
//  4. in ONE transaction: the version becomes APPROVED (approved_by/at) and the previously
//     APPROVED version becomes SUPERSEDED (superseded_at).
//
// A retired policy can no longer be approved (an open draft stays but is inert).
type ApproveChargePolicyVersionUseCase struct{ c *core }

// NewApproveChargePolicyVersionUseCase creates the use case with grouped dependencies.
func NewApproveChargePolicyVersionUseCase(r ApproveChargePolicyVersionRepositories, s ApproveChargePolicyVersionServices) *ApproveChargePolicyVersionUseCase {
	return &ApproveChargePolicyVersionUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:              r.ChargePolicy,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyComponent:     r.ChargePolicyComponent,
			ChargePolicyPosting:       r.ChargePolicyPosting,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
			Account:                   r.Account,
		},
		svc: Services(s),
	}}
}

// Execute approves the draft.
func (uc *ApproveChargePolicyVersionUseCase) Execute(ctx context.Context, req *versionpb.ApproveChargePolicyVersionRequest) (*versionpb.ApproveChargePolicyVersionResponse, error) {
	c := uc.c
	if err := c.gateStrict(ctx, entityid.ActionApprove); err != nil {
		return nil, err
	}
	actor, err := c.actor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || blank(req.GetChargePolicyVersionId()) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy version id is required")
	}
	out := &versionpb.ApproveChargePolicyVersionResponse{Success: true}
	err = c.inTx(ctx, func(txCtx context.Context) error {
		// Lock order policy -> version everywhere (draft clone locks the policy only), so
		// read the version unlocked to learn its policy first.
		probe, lErr := c.readVersion(txCtx, req.GetChargePolicyVersionId())
		if lErr != nil {
			return lErr
		}
		p, lErr := c.lockPolicy(txCtx, probe.GetChargePolicyId())
		if lErr != nil {
			return lErr
		}
		v, lErr := c.lockVersion(txCtx, req.GetChargePolicyVersionId())
		if lErr != nil {
			return lErr
		}
		if v.GetStatus() != enumspbVersionDraft {
			return c.refuse(txCtx, CodeNotDraft, "version %s is %s", v.GetId(), v.GetStatus())
		}
		if p.GetStatus() == enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED {
			return c.refuse(txCtx, CodeRetired, "charge policy %s is retired", p.GetId())
		}
		self, sErr := c.isSelfApproval(txCtx, v, actor)
		if sErr != nil {
			return sErr
		}
		if self {
			if e := c.gateStrict(txCtx, entityid.ActionApproveOwn); e != nil {
				return c.refuse(txCtx, CodeSelfApproval, "%s is required", entityid.EntityPermission(entityid.ChargePolicy, entityid.ActionApproveOwn))
			}
		}
		res, vErr := c.validate(txCtx, v)
		if vErr != nil {
			return vErr
		}
		out.Checklist = res
		if !res.GetSupported() {
			return &checklistError{err: c.refuse(txCtx, CodeUnsupportedCombination, "%s", strings.Join(res.GetReasons(), ",")).(*usecaseerr.Error), checklist: res}
		}
		if !res.GetPassed() {
			return &checklistError{err: c.refuse(txCtx, CodeChecklistFailed, "%s", failedCodes(res.GetItems())).(*usecaseerr.Error), checklist: res}
		}
		all, lErr := c.listVersions(txCtx, p.GetId())
		if lErr != nil {
			return lErr
		}
		ms, ts := stamp()
		// Supersede the current APPROVED first (frees the "one approved" invariant), then approve.
		for _, prev := range all {
			if prev.GetStatus() != enumspbVersionApproved || prev.GetId() == v.GetId() {
				continue
			}
			r, uErr := c.repos.ChargePolicyVersion.UpdateChargePolicyVersion(txCtx, &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{
				Id:                 prev.GetId(),
				Status:             enumspbVersionSuperseded,
				SupersededAt:       i64p(ms),
				DateModified:       i64p(ms),
				DateModifiedString: strp(ts),
			}})
			if uErr != nil {
				return usecaseerr.RepoErr("charge_policy", "supersede charge_policy_version", uErr, prev.GetId())
			}
			if r != nil && len(r.Data) > 0 {
				out.Superseded = r.Data[0]
			}
		}
		patch := &versionpb.ChargePolicyVersion{
			Id:                 v.GetId(),
			Status:             enumspbVersionApproved,
			ApprovedBy:         strp(actor),
			ApprovedAt:         i64p(ms),
			SelfApproved:       self,
			DateModified:       i64p(ms),
			DateModifiedString: strp(ts),
		}
		r, uErr := c.repos.ChargePolicyVersion.UpdateChargePolicyVersion(txCtx, &versionpb.UpdateChargePolicyVersionRequest{Data: patch})
		if uErr != nil {
			return usecaseerr.RepoErr("charge_policy", "approve charge_policy_version", uErr, v.GetId())
		}
		if r != nil && len(r.Data) > 0 {
			out.Data = r.Data[0]
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// isSelfApproval implements the C12 separation-of-duties rule: the actor is "self" when they
// prepared the draft, when any recorded editor is the actor, or when authorship is unknown
// (empty prepared_by). A read error fails closed (returned, approval refused).
func (c *core) isSelfApproval(ctx context.Context, v *versionpb.ChargePolicyVersion, actor string) (bool, error) {
	if v.GetPreparedBy() == "" || v.GetPreparedBy() == actor {
		return true, nil
	}
	eds, err := c.listEditors(ctx, v.GetId())
	if err != nil {
		return false, err
	}
	for _, e := range eds {
		if e.GetUserId() == actor {
			return true, nil
		}
	}
	return false, nil
}
