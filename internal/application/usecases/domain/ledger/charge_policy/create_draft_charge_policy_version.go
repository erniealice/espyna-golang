package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// CreateDraftChargePolicyVersionRepositories groups the repository dependencies of CreateDraftChargePolicyVersion.
type CreateDraftChargePolicyVersionRepositories struct {
	ChargePolicy              policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion       versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent     componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyPosting       postingpb.ChargePolicyPostingDomainServiceServer
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
}

// CreateDraftChargePolicyVersionServices groups the service dependencies of CreateDraftChargePolicyVersion.
type CreateDraftChargePolicyVersionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// CreateDraftChargePolicyVersionUseCase clones the latest APPROVED version into a new DRAFT
// (components + postings copied, cloned_from_version_id set, number = max+1). Refused when a
// DRAFT already exists, when the policy is retired, or when no APPROVED version exists (AC-CP-03).
type CreateDraftChargePolicyVersionUseCase struct{ c *core }

// NewCreateDraftChargePolicyVersionUseCase creates the use case with grouped dependencies.
func NewCreateDraftChargePolicyVersionUseCase(r CreateDraftChargePolicyVersionRepositories, s CreateDraftChargePolicyVersionServices) *CreateDraftChargePolicyVersionUseCase {
	return &CreateDraftChargePolicyVersionUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:              r.ChargePolicy,
			ChargePolicyVersion:       r.ChargePolicyVersion,
			ChargePolicyComponent:     r.ChargePolicyComponent,
			ChargePolicyPosting:       r.ChargePolicyPosting,
			ChargePolicyVersionEditor: r.ChargePolicyVersionEditor,
		},
		svc: Services(s),
	}}
}

// Execute clones the current approved version into a new draft owned (prepared_by) by the actor.
func (uc *CreateDraftChargePolicyVersionUseCase) Execute(ctx context.Context, req *versionpb.CreateDraftChargePolicyVersionRequest) (*versionpb.CreateDraftChargePolicyVersionResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	actor, err := c.actor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || blank(req.GetChargePolicyId()) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy id is required")
	}
	out := &versionpb.CreateDraftChargePolicyVersionResponse{Success: true}
	err = c.inTx(ctx, func(txCtx context.Context) error {
		p, lErr := c.lockPolicy(txCtx, req.GetChargePolicyId())
		if lErr != nil {
			return lErr
		}
		if p.GetStatus() == enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED {
			return c.refuse(txCtx, CodeRetired, "charge policy %s is retired", p.GetId())
		}
		vs, lErr := c.listVersions(txCtx, p.GetId())
		if lErr != nil {
			return lErr
		}
		if draftOf(vs) != nil {
			return c.refuse(txCtx, CodeDraftExists, "charge policy %s already has a draft version", p.GetId())
		}
		src := currentApproved(vs)
		if src == nil {
			return c.refuse(txCtx, CodeNoApprovedVersion, "charge policy %s has no approved version to clone", p.GetId())
		}
		ms, ts := stamp()
		draft := &versionpb.ChargePolicyVersion{
			Id:                  c.newID(),
			ChargePolicyId:      p.GetId(),
			VersionNumber:       maxVersionNumber(vs) + 1,
			Status:              enumspbVersionDraft,
			AccountingRole:      src.AccountingRole,
			BookPresentation:    src.BookPresentation,
			TaxPosition:         src.TaxPosition,
			TaxTreatmentId:      src.TaxTreatmentId,
			AssessmentScope:     src.AssessmentScope,
			AssessmentNote:      src.AssessmentNote,
			ClonedFromVersionId: strp(src.GetId()),
			PreparedBy:          strp(actor),
			Active:              true,
			DateCreated:         i64p(ms),
			DateCreatedString:   strp(ts),
			DateModified:        i64p(ms),
			DateModifiedString:  strp(ts),
		}
		created, cErr := c.repos.ChargePolicyVersion.CreateChargePolicyVersion(txCtx, &versionpb.CreateChargePolicyVersionRequest{Data: draft})
		if cErr != nil {
			return usecaseerr.RepoErr("charge_policy", "create charge_policy_version", cErr, p.GetId())
		}
		if created == nil || len(created.Data) == 0 {
			return usecaseerr.RepoErr("charge_policy", "create charge_policy_version", fmt.Errorf("no row returned"), p.GetId())
		}
		out.Draft = created.Data[0]
		if err := c.recordEditor(txCtx, out.Draft.GetId(), actor); err != nil {
			return err
		}
		comps, cErr := c.listComponents(txCtx, src.GetId())
		if cErr != nil {
			return cErr
		}
		for _, sc := range comps {
			r, e := c.repos.ChargePolicyComponent.CreateChargePolicyComponent(txCtx, &componentpb.CreateChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{
				Id:                    c.newID(),
				ChargePolicyVersionId: out.Draft.GetId(),
				ComponentRole:         sc.GetComponentRole(),
				DocumentKind:          sc.GetDocumentKind(),
				BookPresentation:      sc.GetBookPresentation(),
				SequenceOrder:         sc.GetSequenceOrder(),
				Active:                true,
				DateCreated:           i64p(ms),
				DateCreatedString:     strp(ts),
				DateModified:          i64p(ms),
				DateModifiedString:    strp(ts),
			}})
			if e != nil {
				return usecaseerr.RepoErr("charge_policy", "clone charge_policy_component", e, sc.GetId())
			}
			out.Components = append(out.Components, r.Data...)
		}
		posts, cErr := c.listPostings(txCtx, src.GetId())
		if cErr != nil {
			return cErr
		}
		for _, sp := range posts {
			r, e := c.repos.ChargePolicyPosting.CreateChargePolicyPosting(txCtx, &postingpb.CreateChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{
				Id:                    c.newID(),
				ChargePolicyVersionId: out.Draft.GetId(),
				Event:                 sp.GetEvent(),
				PostingRole:           sp.GetPostingRole(),
				AccountId:             sp.GetAccountId(),
				Active:                true,
				DateCreated:           i64p(ms),
				DateCreatedString:     strp(ts),
				DateModified:          i64p(ms),
				DateModifiedString:    strp(ts),
			}})
			if e != nil {
				return usecaseerr.RepoErr("charge_policy", "clone charge_policy_posting", e, sp.GetId())
			}
			out.Postings = append(out.Postings, r.Data...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
