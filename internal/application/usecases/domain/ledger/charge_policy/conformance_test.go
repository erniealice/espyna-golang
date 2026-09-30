package charge_policy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

func (h *harness) editorIDs(versionID string) map[string]bool {
	out := map[string]bool{}
	for _, m := range h.editors.s.rows {
		e := m.(*editorpb.ChargePolicyVersionEditor)
		if e.GetChargePolicyVersionId() == versionID {
			out[e.GetUserId()] = true
		}
	}
	return out
}

// C12: every draft/component/posting edit records the actor; approval treats any recorded
// editor as the preparer (self) and requires approve_own.
func TestEveryDraftEditorIsSelfForApproval(t *testing.T) {
	edits := map[string]func(h *harness, versionID, posting, component, user string) error{
		"update version": func(h *harness, vid, _, _, u string) error {
			s := "edited"
			_, err := h.uc.UpdateChargePolicyVersion.Execute(ctxAs(u), &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: vid, AssessmentNote: &s}})
			return err
		},
		"create component": func(h *harness, vid, _, _, u string) error {
			_, err := h.uc.CreateChargePolicyComponent.Execute(ctxAs(u), &componentpb.CreateChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{
				ChargePolicyVersionId: vid, ComponentRole: enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST,
				DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT, BookPresentation: enumspb.BookPresentation_BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT}})
			return err
		},
		"update component": func(h *harness, _, _, comp, u string) error {
			_, err := h.uc.UpdateChargePolicyComponent.Execute(ctxAs(u), &componentpb.UpdateChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{Id: comp, SequenceOrder: 7}})
			return err
		},
		"delete component": func(h *harness, _, _, comp, u string) error {
			_, err := h.uc.DeleteChargePolicyComponent.Execute(ctxAs(u), &componentpb.DeleteChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{Id: comp}})
			return err
		},
		"create posting": func(h *harness, vid, _, _, u string) error {
			_, err := h.uc.CreateChargePolicyPosting.Execute(ctxAs(u), &postingpb.CreateChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{
				ChargePolicyVersionId: vid, Event: enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_REVERSAL, PostingRole: enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH, AccountId: "acct-cash"}})
			return err
		},
		"update posting": func(h *harness, _, post, _, u string) error {
			_, err := h.uc.UpdateChargePolicyPosting.Execute(ctxAs(u), &postingpb.UpdateChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: post, AccountId: "acct-cash"}})
			return err
		},
		"delete posting": func(h *harness, _, post, _, u string) error {
			_, err := h.uc.DeleteChargePolicyPosting.Execute(ctxAs(u), &postingpb.DeleteChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: post}})
			return err
		},
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			_, draft := h.seedApprovableDraft(t, "prep", "SOD")
			vid := draft.GetId()
			post := h.posts(ctxAs("prep"), vid)[0].GetId()
			comp := h.comps(ctxAs("prep"), vid)[0].GetId()
			if err := edit(h, vid, post, comp, "editor2"); err != nil {
				t.Fatalf("edit as editor2: %v", err)
			}
			if !h.editorIDs(vid)["editor2"] || !h.editorIDs(vid)["prep"] {
				t.Fatalf("editors = %v, want prep and editor2", h.editorIDs(vid))
			}
			// a second edit by the same user must not duplicate the editor row (UNIQUE(version,user))
			if err := edit(h, vid, post, comp, "editor2"); err != nil && usecaseerr.IsCode(err, CodeNotFound) == false && !strings.Contains(err.Error(), "not found") {
				t.Fatalf("repeat edit: %v", err)
			}
			if len(h.editors.s.rows) != 2 {
				t.Fatalf("editor rows = %d, want 2", len(h.editors.s.rows))
			}
			// editor2 is "self": approve without approve_own is refused, the draft stays a draft
			if _, err := approve(t, h, "editor2", vid); !usecaseerr.IsCode(err, CodeSelfApproval) {
				t.Fatalf("editor approving without approve_own: %v", err)
			}
			assertStatus(t, h, vid, enumspbVersionDraft)
			// with approve_own it is allowed and flagged self_approved
			h.authz.perms["charge_policy:approve_own"] = true
			// (some edits leave the draft outside the S1 matrix/checklist; the SoD gate runs before
			// those, so either an approval flagged self_approved or a matrix/checklist refusal proves it passed)
			res, err := approve(t, h, "editor2", vid)
			switch {
			case err == nil:
				if !res.Data.GetSelfApproved() {
					t.Fatalf("editor approval must be flagged self_approved: %+v", res)
				}
			case usecaseerr.IsCode(err, CodeChecklistFailed), usecaseerr.IsCode(err, CodeUnsupportedCombination):
			default:
				t.Fatalf("editor with approve_own: %v", err)
			}
		})
	}
	t.Run("a non-editor approver is not self", func(t *testing.T) {
		h := newHarness(t)
		_, draft := h.seedApprovableDraft(t, "prep", "SOD2")
		res, err := approve(t, h, "boss", draft.GetId())
		if err != nil || res.Data.GetSelfApproved() {
			t.Fatalf("boss: %v %+v", err, res)
		}
	})
}

func TestApprovalEmptyPreparedByRequiresApproveOwn(t *testing.T) {
	h := newHarness(t)
	_, draft := h.seedApprovableDraft(t, "prep", "NOPREP")
	// unknown authorship: blank prepared_by (direct store edit; lifecycle rows are never client-writable)
	h.versions.s.rows[draft.GetId()].(*versionpb.ChargePolicyVersion).PreparedBy = nil
	if _, err := approve(t, h, "boss", draft.GetId()); !usecaseerr.IsCode(err, CodeSelfApproval) {
		t.Fatalf("empty prepared_by without approve_own: %v", err)
	}
	h.authz.perms["charge_policy:approve_own"] = true
	res, err := approve(t, h, "boss", draft.GetId())
	if err != nil || !res.Data.GetSelfApproved() {
		t.Fatalf("with approve_own: %v %+v", err, res)
	}
}

type failingEditorRepo struct {
	editorpb.ChargePolicyVersionEditorDomainServiceServer
}

func (failingEditorRepo) ListChargePolicyVersionEditors(context.Context, *editorpb.ListChargePolicyVersionEditorsRequest) (*editorpb.ListChargePolicyVersionEditorsResponse, error) {
	return nil, fmt.Errorf("connection reset")
}

func TestApprovalFailsClosedWhenEditorsUnreadable(t *testing.T) {
	h := newHarness(t)
	_, draft := h.seedApprovableDraft(t, "prep", "CLOSED")
	h.uc = NewUseCases(Repositories{ChargePolicy: h.policies, ChargePolicyVersion: h.versions, ChargePolicyComponent: h.components,
		ChargePolicyPosting: h.postings, ChargePolicyVersionEditor: failingEditorRepo{h.editors}, Account: h.accounts, TaxTreatment: h.taxes}, h.svc())
	_, err := approve(t, h, "boss", draft.GetId())
	if err == nil || usecaseerr.IsCode(err, CodeNotFound) {
		t.Fatalf("editor read error must refuse approval (and never as not_found): %v", err)
	}
	assertStatus(t, h, draft.GetId(), enumspbVersionDraft)

	// editor repository not wired at all: refused as unverifiable
	h.uc = NewUseCases(Repositories{ChargePolicy: h.policies, ChargePolicyVersion: h.versions, ChargePolicyComponent: h.components,
		ChargePolicyPosting: h.postings, Account: h.accounts, TaxTreatment: h.taxes}, h.svc())
	if _, err := approve(t, h, "boss", draft.GetId()); !usecaseerr.IsCode(err, CodeUnverifiable) {
		t.Fatalf("unwired editors: %v", err)
	}
}

type noLockVersionRepo struct {
	versionpb.ChargePolicyVersionDomainServiceServer
}

// C4: a repository that cannot lock rows is refused, never read unlocked.
func TestRowLockersFailClosed(t *testing.T) {
	h := newHarness(t)
	_, draft := h.seedApprovableDraft(t, "prep", "NOLOCK")
	h.uc = NewUseCases(Repositories{ChargePolicy: h.policies, ChargePolicyVersion: noLockVersionRepo{h.versions}, ChargePolicyComponent: h.components,
		ChargePolicyPosting: h.postings, ChargePolicyVersionEditor: h.editors, Account: h.accounts, TaxTreatment: h.taxes}, h.svc())
	scope := "x"
	if _, err := h.uc.UpdateChargePolicyVersion.Execute(ctxAs("prep"), &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId(), AssessmentScope: &scope}}); !usecaseerr.IsCode(err, CodeUnverifiable) {
		t.Fatalf("update without locker: %v", err)
	}
	if _, err := approve(t, h, "boss", draft.GetId()); !usecaseerr.IsCode(err, CodeUnverifiable) {
		t.Fatalf("approve without locker: %v", err)
	}
}

// C5: a request-supplied tax_treatment_id is read before it is stored.
func TestUpdateVersionChecksTaxTreatment(t *testing.T) {
	h := newHarness(t)
	_, draft := h.seedApprovableDraft(t, "prep", "TAX")
	upd := func(id string) error {
		_, err := h.uc.UpdateChargePolicyVersion.Execute(ctxAs("prep"), &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId(), TaxTreatmentId: &id}})
		return err
	}
	if err := upd("tax-foreign"); !usecaseerr.IsCode(err, CodeTaxTreatmentNotFound) {
		t.Fatalf("unknown tax treatment: %v", err)
	}
	if err := upd("tax-ok"); err != nil {
		t.Fatalf("known tax treatment: %v", err)
	}
	if m, _ := h.versions.s.read(draft.GetId()); m.(*versionpb.ChargePolicyVersion).GetTaxTreatmentId() != "tax-ok" {
		t.Fatal("tax treatment not stored")
	}
	h.uc = NewUseCases(Repositories{ChargePolicy: h.policies, ChargePolicyVersion: h.versions, ChargePolicyComponent: h.components,
		ChargePolicyPosting: h.postings, ChargePolicyVersionEditor: h.editors, Account: h.accounts}, h.svc())
	if err := upd("tax-ok"); !usecaseerr.IsCode(err, CodeUnverifiable) {
		t.Fatalf("tax repo not wired: %v", err)
	}
}

type brokenVersionRepo struct {
	versionpb.ChargePolicyVersionDomainServiceServer
}

func (brokenVersionRepo) ReadChargePolicyVersion(context.Context, *versionpb.ReadChargePolicyVersionRequest) (*versionpb.ReadChargePolicyVersionResponse, error) {
	return nil, errors.New("deadline exceeded")
}

// C9: a repository failure is not reported as not_found.
func TestRepositoryErrorsAreNotNotFound(t *testing.T) {
	h := newHarness(t)
	h.uc = NewUseCases(Repositories{ChargePolicy: h.policies, ChargePolicyVersion: brokenVersionRepo{h.versions}, ChargePolicyComponent: h.components,
		ChargePolicyPosting: h.postings, ChargePolicyVersionEditor: h.editors, Account: h.accounts}, h.svc())
	_, err := h.uc.ReadChargePolicyVersion.Execute(ctxAs("u"), &versionpb.ReadChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: "v"}})
	if err == nil || usecaseerr.IsCode(err, CodeNotFound) || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("repo error must surface as itself: %v", err)
	}
	// a genuinely absent row still is not_found
	h2 := newHarness(t)
	if _, err := h2.uc.ReadChargePolicyVersion.Execute(ctxAs("u"), &versionpb.ReadChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: "absent"}}); !usecaseerr.IsCode(err, CodeNotFound) {
		t.Fatalf("absent: %v", err)
	}
}

type xlator struct{}

func (xlator) Get(_ context.Context, _, key string, _ ...any) string { return "XL[" + key + "]" }
func (xlator) GetWithDefault(_ context.Context, _, key, _ string, _ ...any) string {
	return "XL[" + key + "]"
}

// C2: named refusals are translated from charge_policy.errors.<code> and expose ErrorCode().
func TestRefusalsAreTranslatedAndCoded(t *testing.T) {
	h := newHarness(t)
	svc := h.svc()
	svc.Translator = xlator{}
	h.uc = NewUseCases(h.repos(), svc)
	_, draft := h.seedApprovableDraft(t, "prep", "XLATE")
	if _, err := approve(t, h, "boss", draft.GetId()); err != nil {
		t.Fatal(err)
	}
	scope := "x"
	_, err := h.uc.UpdateChargePolicyVersion.Execute(ctxAs("prep"), &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId(), AssessmentScope: &scope}})
	var ce interface{ ErrorCode() string }
	if !errors.As(err, &ce) || ce.ErrorCode() != "not_draft" {
		t.Fatalf("code: %v", err)
	}
	if !strings.HasPrefix(err.Error(), "XL[charge_policy.errors.not_draft]") {
		t.Fatalf("message not translated from charge_policy.errors.not_draft: %q", err.Error())
	}
}

// R3: deleting a draft version is gated by charge_policy:delete (as the view is), not update.
func TestDeleteVersionUsesDeletePermission(t *testing.T) {
	h := newHarness(t)
	_, draft := h.seedApprovableDraft(t, "prep", "DELPERM")
	h.authz.perms["charge_policy:delete"] = false
	if _, err := h.uc.DeleteChargePolicyVersion.Execute(ctxAs("prep"), &versionpb.DeleteChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId()}}); err == nil {
		t.Fatal("update permission alone must not delete a version")
	}
	h.authz.perms["charge_policy:delete"] = true
	h.authz.perms["charge_policy:update"] = false
	if _, err := h.uc.DeleteChargePolicyVersion.Execute(ctxAs("prep"), &versionpb.DeleteChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId()}}); err != nil {
		t.Fatalf("delete permission deletes a draft: %v", err)
	}
	if len(h.editors.s.rows) != 0 || len(h.components.s.rows) != 0 || len(h.postings.s.rows) != 0 {
		t.Fatalf("children (incl. editors) must go with the version: editors=%d comps=%d posts=%d", len(h.editors.s.rows), len(h.components.s.rows), len(h.postings.s.rows))
	}
}

// C11: scoped lists require their scope, AND it onto caller filters, and refuse OR logic.
func TestScopedListsRequireScope(t *testing.T) {
	h := newHarness(t)
	_, draft := h.seedApprovableDraft(t, "prep", "LISTS")
	ctx := ctxAs("prep")
	if _, err := h.uc.ListChargePolicyVersions.Execute(ctx, &versionpb.ListChargePolicyVersionsRequest{}); !usecaseerr.IsCode(err, CodeValidation) {
		t.Fatalf("unscoped versions: %v", err)
	}
	if _, err := h.uc.ListChargePolicyComponents.Execute(ctx, &componentpb.ListChargePolicyComponentsRequest{}); !usecaseerr.IsCode(err, CodeValidation) {
		t.Fatalf("unscoped components: %v", err)
	}
	if _, err := h.uc.ListChargePolicyPostings.Execute(ctx, &postingpb.ListChargePolicyPostingsRequest{}); !usecaseerr.IsCode(err, CodeValidation) {
		t.Fatalf("unscoped postings: %v", err)
	}
	vs, err := h.uc.ListChargePolicyVersions.Execute(ctx, &versionpb.ListChargePolicyVersionsRequest{ChargePolicyId: strp(draft.GetChargePolicyId()), Pagination: &commonpb.PaginationRequest{Limit: 10}})
	if err != nil || len(vs.Data) != 1 || !vs.Success {
		t.Fatalf("scoped versions: %v %+v", err, vs)
	}
	or := &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: listdata.EqFilter("status", "DRAFT").Filters}
	if _, err := h.uc.ListChargePolicyPostings.Execute(ctx, &postingpb.ListChargePolicyPostingsRequest{ChargePolicyVersionId: strp(draft.GetId()), Filters: or}); !usecaseerr.IsCode(err, CodeValidation) {
		t.Fatalf("OR filters must be refused: %v", err)
	}
}

// New draft (clone) records only the cloner as editor: prior editors are not carried over.
func TestCloneRecordsOnlyClonerAsEditor(t *testing.T) {
	h := newHarness(t)
	h.authz.perms["charge_policy:approve_own"] = true
	policy, v1 := h.seedApprovableDraft(t, "prep", "CLONEED")
	if _, err := approve(t, h, "prep", v1.GetId()); err != nil {
		t.Fatal(err)
	}
	d2, err := h.uc.CreateDraftChargePolicyVersion.Execute(ctxAs("cloner"), &versionpb.CreateDraftChargePolicyVersionRequest{ChargePolicyId: policy.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	eds := h.editorIDs(d2.Draft.GetId())
	if len(eds) != 1 || !eds["cloner"] {
		t.Fatalf("clone editors = %v", eds)
	}
	// prep (editor of v1 only) is NOT self on v2
	if _, err := approve(t, h, "prep", d2.Draft.GetId()); err != nil {
		t.Fatalf("prep approving cloner's v2 (not an editor): %v", err)
	}
}

var _ = policypb.ChargePolicy{}
