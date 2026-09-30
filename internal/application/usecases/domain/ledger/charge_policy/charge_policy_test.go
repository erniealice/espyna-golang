package charge_policy

import (
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// AC-CP-01
func TestCreateChargePolicyCreatesDraftV1(t *testing.T) {
	h := newHarness(t)
	ctx := ctxAs("preparer")
	resp, err := h.uc.CreateChargePolicy.Execute(ctx, &policypb.CreateChargePolicyRequest{Data: &policypb.ChargePolicy{
		Code: "UTILITY_RECOVERY", Name: "Utility recovery",
		// caller-chosen lifecycle fields must be ignored
		Status: enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED,
	}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.Data[0].GetStatus() != enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE || !resp.Data[0].GetActive() {
		t.Fatalf("policy must start ACTIVE, got %v", resp.Data[0].GetStatus())
	}
	if resp.Data[0].GetCreatedBy() != "preparer" {
		t.Fatalf("created_by = %q", resp.Data[0].GetCreatedBy())
	}
	d := resp.Draft
	if d.GetVersionNumber() != 1 || d.GetStatus() != enumspbVersionDraft || d.GetPreparedBy() != "preparer" || d.GetChargePolicyId() != resp.Data[0].GetId() {
		t.Fatalf("draft v1 wrong: %+v", d)
	}
	if h.tx.calls != 1 {
		t.Fatalf("policy + draft must share ONE transaction, got %d", h.tx.calls)
	}

	t.Run("validation", func(t *testing.T) {
		for name, data := range map[string]*policypb.ChargePolicy{
			"bad code":   {Code: "utility recovery", Name: "x"},
			"blank name": {Code: "OK_CODE", Name: "  "},
		} {
			if _, err := h.uc.CreateChargePolicy.Execute(ctx, &policypb.CreateChargePolicyRequest{Data: data}); !usecaseerr.IsCode(err, CodeValidation) {
				t.Errorf("%s: want ErrValidation, got %v", name, err)
			}
		}
	})
	t.Run("rollback when draft create fails", func(t *testing.T) {
		// A second draft for the same policy id violates the one-draft rule -> whole tx rolls back.
		h2 := newHarness(t)
		h2.versions.s.create(&versionpb.ChargePolicyVersion{Id: "pre", ChargePolicyId: "id-001", Status: enumspbVersionDraft})
		if _, err := h2.uc.CreateChargePolicy.Execute(ctxAs("u"), &policypb.CreateChargePolicyRequest{Data: &policypb.ChargePolicy{Code: "X", Name: "x"}}); err == nil {
			t.Fatal("expected failure")
		}
		if h2.tx.rollbacks != 1 {
			t.Fatalf("expected rollback, got %d", h2.tx.rollbacks)
		}
	})
	t.Run("permission", func(t *testing.T) {
		h3 := newHarness(t)
		h3.authz.perms["charge_policy:create"] = false
		if _, err := h3.uc.CreateChargePolicy.Execute(ctx, &policypb.CreateChargePolicyRequest{Data: &policypb.ChargePolicy{Code: "X", Name: "x"}}); err == nil {
			t.Fatal("create without charge_policy:create must be denied")
		}
	})
}

func approve(t *testing.T, h *harness, approver, versionID string) (*versionpb.ApproveChargePolicyVersionResponse, error) {
	t.Helper()
	return h.uc.ApproveChargePolicyVersion.Execute(ctxAs(approver), &versionpb.ApproveChargePolicyVersionRequest{ChargePolicyVersionId: versionID})
}

// AC-CP-02
func TestApprovedChargePolicyVersionIsImmutable(t *testing.T) {
	h := newHarness(t)
	h.authz.perms["charge_policy:approve_own"] = true
	_, draft := h.seedApprovableDraft(t, "prep", "IMMUTABLE")
	if _, err := approve(t, h, "prep", draft.GetId()); err != nil {
		t.Fatalf("approve: %v", err)
	}
	ctx := ctxAs("prep")
	comps := h.comps(ctx, draft.GetId())
	posts := h.posts(ctx, draft.GetId())
	if len(comps) != 1 || len(posts) != 4 {
		t.Fatalf("fixture: comps=%d posts=%d", len(comps), len(posts))
	}
	scope := "changed"
	cases := map[string]func() error{
		"update version": func() error {
			_, err := h.uc.UpdateChargePolicyVersion.Execute(ctx, &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId(), AssessmentScope: &scope}})
			return err
		},
		"delete version": func() error {
			_, err := h.uc.DeleteChargePolicyVersion.Execute(ctx, &versionpb.DeleteChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId()}})
			return err
		},
		"create component": func() error {
			_, err := h.uc.CreateChargePolicyComponent.Execute(ctx, &componentpb.CreateChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{
				ChargePolicyVersionId: draft.GetId(), ComponentRole: enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_FEE,
				DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_INVOICE, BookPresentation: enumspb.BookPresentation_BOOK_PRESENTATION_REVENUE}})
			return err
		},
		"update component": func() error {
			_, err := h.uc.UpdateChargePolicyComponent.Execute(ctx, &componentpb.UpdateChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{Id: comps[0].GetId(), SequenceOrder: 9}})
			return err
		},
		"delete component": func() error {
			_, err := h.uc.DeleteChargePolicyComponent.Execute(ctx, &componentpb.DeleteChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{Id: comps[0].GetId()}})
			return err
		},
		"create posting": func() error {
			_, err := h.uc.CreateChargePolicyPosting.Execute(ctx, &postingpb.CreateChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{
				ChargePolicyVersionId: draft.GetId(), Event: enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_REVERSAL,
				PostingRole: enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH, AccountId: "acct-cash"}})
			return err
		},
		"update posting": func() error {
			_, err := h.uc.UpdateChargePolicyPosting.Execute(ctx, &postingpb.UpdateChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: posts[0].GetId(), AccountId: "acct-cash"}})
			return err
		},
		"delete posting": func() error {
			_, err := h.uc.DeleteChargePolicyPosting.Execute(ctx, &postingpb.DeleteChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: posts[0].GetId()}})
			return err
		},
	}
	for name, fn := range cases {
		if err := fn(); !usecaseerr.IsCode(err, CodeNotDraft) {
			t.Errorf("%s on APPROVED version: want ErrNotDraft, got %v", name, err)
		}
	}
	// nothing changed
	v, _ := h.uc.ReadChargePolicyVersion.Execute(ctx, &versionpb.ReadChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId()}})
	if v.Data[0].GetAssessmentScope() != "Utility pass-through to tenants" || len(v.Components) != 1 || len(v.Postings) != 4 {
		t.Fatalf("approved version mutated: %+v", v.Data[0])
	}
}

// AC-CP-03
func TestCreateDraftVersionClonesAndRefusesSecondDraft(t *testing.T) {
	h := newHarness(t)
	h.authz.perms["charge_policy:approve_own"] = true
	policy, v1 := h.seedApprovableDraft(t, "prep", "CLONE")
	ctx := ctxAs("prep")

	// no approved version yet -> nothing to clone (and a draft exists)
	if _, err := h.uc.CreateDraftChargePolicyVersion.Execute(ctx, &versionpb.CreateDraftChargePolicyVersionRequest{ChargePolicyId: policy.GetId()}); !usecaseerr.IsCode(err, CodeDraftExists) {
		t.Fatalf("draft exists: got %v", err)
	}
	if _, err := approve(t, h, "prep", v1.GetId()); err != nil {
		t.Fatalf("approve v1: %v", err)
	}
	resp, err := h.uc.CreateDraftChargePolicyVersion.Execute(ctx, &versionpb.CreateDraftChargePolicyVersionRequest{ChargePolicyId: policy.GetId()})
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	d := resp.Draft
	if d.GetVersionNumber() != 2 || d.GetStatus() != enumspbVersionDraft || d.GetClonedFromVersionId() != v1.GetId() {
		t.Fatalf("clone header wrong: %+v", d)
	}
	if d.GetAccountingRole() != enumspb.AccountingRole_ACCOUNTING_ROLE_AGENT || d.GetAssessmentScope() == "" || d.GetPreparedBy() != "prep" {
		t.Fatalf("clone must carry classification and set prepared_by: %+v", d)
	}
	if d.GetApprovedBy() != "" || d.GetSelfApproved() {
		t.Fatalf("clone must not inherit approval stamps: %+v", d)
	}
	if len(resp.Components) != 1 || len(resp.Postings) != 4 {
		t.Fatalf("components=%d postings=%d", len(resp.Components), len(resp.Postings))
	}
	for _, c := range resp.Components {
		if c.GetChargePolicyVersionId() != d.GetId() {
			t.Fatal("cloned component not re-parented")
		}
	}
	// second draft refused
	if _, err := h.uc.CreateDraftChargePolicyVersion.Execute(ctx, &versionpb.CreateDraftChargePolicyVersionRequest{ChargePolicyId: policy.GetId()}); !usecaseerr.IsCode(err, CodeDraftExists) {
		t.Fatalf("second draft: got %v", err)
	}
	// the source stays untouched and APPROVED
	src, _ := h.uc.ReadChargePolicyVersion.Execute(ctx, &versionpb.ReadChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: v1.GetId()}})
	if src.Data[0].GetStatus() != enumspbVersionApproved || len(src.Components) != 1 || len(src.Postings) != 4 {
		t.Fatalf("source version changed: %+v", src.Data[0])
	}
	// approving v2 supersedes v1 in the same transaction
	res, err := approve(t, h, "prep", d.GetId())
	if err != nil {
		t.Fatalf("approve v2: %v", err)
	}
	if res.Superseded.GetId() != v1.GetId() || res.Superseded.GetStatus() != enumspbVersionSuperseded || res.Superseded.GetSupersededAt() == 0 {
		t.Fatalf("v1 not superseded: %+v", res.Superseded)
	}
	// retired policy refuses a new draft
	if _, err := h.uc.RetireChargePolicy.Execute(ctx, &policypb.RetireChargePolicyRequest{ChargePolicyId: policy.GetId()}); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if _, err := h.uc.CreateDraftChargePolicyVersion.Execute(ctx, &versionpb.CreateDraftChargePolicyVersionRequest{ChargePolicyId: policy.GetId()}); !usecaseerr.IsCode(err, CodeRetired) {
		t.Fatalf("retired clone: got %v", err)
	}
}

// AC-CP-04 / AC-UC-36
func TestApproveChargePolicyVersionRefusals(t *testing.T) {
	t.Run("missing approve permission", func(t *testing.T) {
		h := newHarness(t)
		_, draft := h.seedApprovableDraft(t, "prep", "P1")
		h.authz.perms["charge_policy:approve"] = false
		if _, err := approve(t, h, "boss", draft.GetId()); err == nil {
			t.Fatal("expected permission denial")
		}
		assertStatus(t, h, draft.GetId(), enumspbVersionDraft)
	})
	t.Run("self approval without approve_own", func(t *testing.T) {
		h := newHarness(t)
		_, draft := h.seedApprovableDraft(t, "prep", "P2")
		if _, err := approve(t, h, "prep", draft.GetId()); !usecaseerr.IsCode(err, CodeSelfApproval) {
			t.Fatalf("want ErrSelfApproval, got %v", err)
		}
		assertStatus(t, h, draft.GetId(), enumspbVersionDraft)
	})
	t.Run("approve_own sets self_approved", func(t *testing.T) {
		h := newHarness(t)
		h.authz.perms["charge_policy:approve_own"] = true
		_, draft := h.seedApprovableDraft(t, "prep", "P3")
		res, err := approve(t, h, "prep", draft.GetId())
		if err != nil {
			t.Fatalf("approve: %v", err)
		}
		if !res.Data.GetSelfApproved() || res.Data.GetApprovedBy() != "prep" || res.Data.GetApprovedAt() == 0 {
			t.Fatalf("self approval stamps missing: %+v", res.Data)
		}
	})
	t.Run("different approver is not self approved", func(t *testing.T) {
		h := newHarness(t)
		_, draft := h.seedApprovableDraft(t, "prep", "P3B")
		res, err := approve(t, h, "boss", draft.GetId())
		if err != nil {
			t.Fatalf("approve: %v", err)
		}
		if res.Data.GetSelfApproved() || res.Data.GetApprovedBy() != "boss" {
			t.Fatalf("unexpected stamps: %+v", res.Data)
		}
	})
	t.Run("failed checklist", func(t *testing.T) {
		h := newHarness(t)
		_, draft := h.seedApprovableDraft(t, "prep", "P4")
		blankScope := " "
		// blank scope needs a direct store edit: the draft update use case would accept it too
		if _, err := h.uc.UpdateChargePolicyVersion.Execute(ctxAs("prep"), &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: draft.GetId(), AssessmentScope: &blankScope}}); err != nil {
			t.Fatal(err)
		}
		_, err := approve(t, h, "boss", draft.GetId())
		var ae interface {
			Checklist() *versionpb.ChargePolicyApprovalChecklist
		}
		if !usecaseerr.IsCode(err, CodeChecklistFailed) || !errors.As(err, &ae) || ae.Checklist().GetPassed() {
			t.Fatalf("want checklist_failed, got %v", err)
		}
		if !itemFailed(ae.Checklist(), ChecklistAssessmentScope) {
			t.Fatalf("assessment_scope item should fail: %+v", ae.Checklist().GetItems())
		}
		assertStatus(t, h, draft.GetId(), enumspbVersionDraft)
	})
	t.Run("missing required posting and inactive account", func(t *testing.T) {
		h := newHarness(t)
		_, draft := h.seedApprovableDraft(t, "prep", "P4B")
		ctx := ctxAs("prep")
		posts := h.posts(ctx, draft.GetId())
		// remove ISSUE:CLEARING, repoint ISSUE:RECEIVABLE at an inactive account
		for _, p := range posts {
			if p.GetPostingRole() == enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CLEARING {
				_, _ = h.uc.DeleteChargePolicyPosting.Execute(ctx, &postingpb.DeleteChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: p.GetId()}})
			}
			if p.GetEvent() == enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE && p.GetPostingRole() == enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE {
				if _, err := h.uc.UpdateChargePolicyPosting.Execute(ctx, &postingpb.UpdateChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: p.GetId(), AccountId: "acct-inactive"}}); err != nil {
					t.Fatal(err)
				}
			}
		}
		_, err := approve(t, h, "boss", draft.GetId())
		var ae interface {
			Checklist() *versionpb.ChargePolicyApprovalChecklist
		}
		if !usecaseerr.IsCode(err, CodeChecklistFailed) || !errors.As(err, &ae) {
			t.Fatalf("want checklist_failed, got %v", err)
		}
		if !itemFailed(ae.Checklist(), "posting:ISSUE:CLEARING") || !itemFailed(ae.Checklist(), "posting_account:ISSUE:RECEIVABLE") {
			t.Fatalf("items: %+v", ae.Checklist().GetItems())
		}
	})
	t.Run("unsupported S1 combination", func(t *testing.T) {
		cases := map[string]struct {
			mutate func(h *harness, versionID string)
			reason string
		}{
			"principal + excluded": {func(h *harness, id string) {
				r := enumspb.AccountingRole_ACCOUNTING_ROLE_PRINCIPAL
				mustUpdate(t, h, id, &versionpb.ChargePolicyVersion{Id: id, AccountingRole: &r})
			}, ReasonPrincipalExcluded},
			"own supply": {func(h *harness, id string) {
				x := enumspb.TaxPosition_TAX_POSITION_OWN_SUPPLY
				mustUpdate(t, h, id, &versionpb.ChargePolicyVersion{Id: id, TaxPosition: &x})
			}, ReasonOwnSupply},
			"fee component": {func(h *harness, id string) {
				setComponent(t, h, id, &componentpb.ChargePolicyComponent{ComponentRole: enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_FEE})
			}, ReasonFee},
			"invoice document": {func(h *harness, id string) {
				setComponent(t, h, id, &componentpb.ChargePolicyComponent{DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_INVOICE})
			}, ReasonInvoice},
			"two components": {func(h *harness, id string) {
				_, err := h.uc.CreateChargePolicyComponent.Execute(ctxAs("prep"), &componentpb.CreateChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{
					ChargePolicyVersionId: id, ComponentRole: enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_OWN_REVENUE,
					DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT, BookPresentation: enumspb.BookPresentation_BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT}})
				if err != nil {
					t.Fatal(err)
				}
			}, ReasonComponentCount},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				h := newHarness(t)
				_, draft := h.seedApprovableDraft(t, "prep", "PX")
				tc.mutate(h, draft.GetId())
				_, err := approve(t, h, "boss", draft.GetId())
				var ae interface {
					Checklist() *versionpb.ChargePolicyApprovalChecklist
				}
				if !usecaseerr.IsCode(err, CodeUnsupportedCombination) || !errors.As(err, &ae) {
					t.Fatalf("want unsupported_combination, got %v", err)
				}
				found := false
				for _, r := range ae.Checklist().GetReasons() {
					found = found || r == tc.reason
				}
				if !found {
					t.Fatalf("reason %q missing from %v", tc.reason, ae.Checklist().GetReasons())
				}
				assertStatus(t, h, draft.GetId(), enumspbVersionDraft)
			})
		}
	})
	t.Run("retired policy cannot approve", func(t *testing.T) {
		h := newHarness(t)
		p, draft := h.seedApprovableDraft(t, "prep", "P5")
		if _, err := h.uc.RetireChargePolicy.Execute(ctxAs("prep"), &policypb.RetireChargePolicyRequest{ChargePolicyId: p.GetId()}); err != nil {
			t.Fatal(err)
		}
		if _, err := approve(t, h, "boss", draft.GetId()); !usecaseerr.IsCode(err, CodeRetired) {
			t.Fatalf("want ErrRetired, got %v", err)
		}
	})
	t.Run("already approved is not draft", func(t *testing.T) {
		h := newHarness(t)
		_, draft := h.seedApprovableDraft(t, "prep", "P6")
		if _, err := approve(t, h, "boss", draft.GetId()); err != nil {
			t.Fatal(err)
		}
		if _, err := approve(t, h, "boss", draft.GetId()); !usecaseerr.IsCode(err, CodeNotDraft) {
			t.Fatalf("want ErrNotDraft, got %v", err)
		}
	})
}

func assertStatus(t *testing.T, h *harness, versionID string, want enumspb.ChargePolicyVersionStatus) {
	t.Helper()
	m, _ := h.versions.s.read(versionID)
	if got := m.(*versionpb.ChargePolicyVersion).GetStatus(); got != want {
		t.Fatalf("version status = %v, want %v", got, want)
	}
}

func itemFailed(v *versionpb.ChargePolicyApprovalChecklist, code string) bool {
	for _, it := range v.GetItems() {
		if it.GetCode() == code {
			return !it.GetPassed()
		}
	}
	return false
}

func mustUpdate(t *testing.T, h *harness, _ string, patch *versionpb.ChargePolicyVersion) {
	t.Helper()
	if _, err := h.uc.UpdateChargePolicyVersion.Execute(ctxAs("prep"), &versionpb.UpdateChargePolicyVersionRequest{Data: patch}); err != nil {
		t.Fatal(err)
	}
}

func setComponent(t *testing.T, h *harness, versionID string, patch *componentpb.ChargePolicyComponent) {
	t.Helper()
	comps := h.comps(ctxAs("prep"), versionID)
	patch.Id = comps[0].GetId()
	if _, err := h.uc.UpdateChargePolicyComponent.Execute(ctxAs("prep"), &componentpb.UpdateChargePolicyComponentRequest{Data: patch}); err != nil {
		t.Fatal(err)
	}
}

// AC-CP-05
func TestRetireChargePolicyKeepsPinnedTerms(t *testing.T) {
	h := newHarness(t)
	policyA, vA := h.seedApprovableDraft(t, "prep", "KEEP_A")
	policyB, vB := h.seedApprovableDraft(t, "prep", "KEEP_B")
	for _, v := range []string{vA.GetId(), vB.GetId()} {
		if _, err := approve(t, h, "boss", v); err != nil {
			t.Fatal(err)
		}
	}
	ctx := ctxAs("prep")
	pickerResp, err := h.uc.ListPickerChargePolicies.Execute(ctx, &policypb.ListPickerChargePoliciesRequest{})
	if err != nil || len(pickerResp.Data) != 2 {
		t.Fatalf("picker before retire: %v %+v", err, pickerResp)
	}
	// an open draft of policy A (clone) that must survive retirement but be unapprovable
	draftA, err := h.uc.CreateDraftChargePolicyVersion.Execute(ctx, &versionpb.CreateDraftChargePolicyVersionRequest{ChargePolicyId: policyA.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.uc.RetireChargePolicy.Execute(ctx, &policypb.RetireChargePolicyRequest{ChargePolicyId: policyA.GetId()}); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if _, err := h.uc.RetireChargePolicy.Execute(ctx, &policypb.RetireChargePolicyRequest{ChargePolicyId: policyA.GetId()}); !usecaseerr.IsCode(err, CodeRetired) {
		t.Fatalf("double retire: %v", err)
	}
	// excluded from the picker list, still listed for management
	pickerResp, _ = h.uc.ListPickerChargePolicies.Execute(ctx, &policypb.ListPickerChargePoliciesRequest{})
	picker := pickerResp.Data
	if len(picker) != 1 || picker[0].GetId() != policyB.GetId() {
		t.Fatalf("retired policy must leave the picker: %+v", picker)
	}
	all, _ := h.uc.ListChargePolicies.Execute(ctx, &policypb.ListChargePoliciesRequest{})
	if len(all.Data) != 2 {
		t.Fatalf("management list must still show retired: %d", len(all.Data))
	}
	got, _ := h.uc.ReadChargePolicy.Execute(ctx, &policypb.ReadChargePolicyRequest{Data: &policypb.ChargePolicy{Id: policyA.GetId()}})
	if got.Data[0].GetStatus() != enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED || got.Data[0].GetRetiredBy() != "prep" || got.Data[0].GetRetiredAt() == 0 {
		t.Fatalf("retire stamps: %+v", got.Data[0])
	}
	// existing references intact: approved v1 still APPROVED; the open draft stays but cannot be approved
	assertStatus(t, h, vA.GetId(), enumspbVersionApproved)
	assertStatus(t, h, draftA.Draft.GetId(), enumspbVersionDraft)
	if _, err := approve(t, h, "boss", draftA.Draft.GetId()); !usecaseerr.IsCode(err, CodeRetired) {
		t.Fatalf("approve on retired policy: %v", err)
	}
	// delete: an approved policy is never deletable; a never-approved unreferenced one is
	if _, err := h.uc.DeleteChargePolicy.Execute(ctx, &policypb.DeleteChargePolicyRequest{Data: &policypb.ChargePolicy{Id: policyA.GetId()}}); !usecaseerr.IsCode(err, CodeInUse) {
		t.Fatalf("delete approved: %v", err)
	}
	fresh, _ := h.seedApprovableDraft(t, "prep", "NEVER_APPROVED")
	inUseResp, _ := h.uc.GetChargePolicyInUseIds.Execute(ctx, &policypb.GetChargePolicyInUseIdsRequest{ChargePolicyIds: []string{policyA.GetId(), fresh.GetId()}})
	inUse := map[string]bool{}
	for _, id := range inUseResp.GetInUseChargePolicyIds() {
		inUse[id] = true
	}
	if !inUse[policyA.GetId()] || inUse[fresh.GetId()] {
		t.Fatalf("in-use ids: %v", inUse)
	}
	h.policies.referenced[fresh.GetId()] = true
	if _, err := h.uc.DeleteChargePolicy.Execute(ctx, &policypb.DeleteChargePolicyRequest{Data: &policypb.ChargePolicy{Id: fresh.GetId()}}); !usecaseerr.IsCode(err, CodeInUse) {
		t.Fatalf("delete referenced: %v", err)
	}
	delete(h.policies.referenced, fresh.GetId())
	if _, err := h.uc.DeleteChargePolicy.Execute(ctx, &policypb.DeleteChargePolicyRequest{Data: &policypb.ChargePolicy{Id: fresh.GetId()}}); err != nil {
		t.Fatalf("delete never-approved: %v", err)
	}
	if _, ok := h.policies.s.read(fresh.GetId()); ok {
		t.Fatal("policy row should be gone")
	}
}

// AC-CP-07
func TestResolveChargePolicyPrecedence(t *testing.T) {
	h := newHarness(t)
	ctx := ctxAs("prep")
	// no policy: legacy path (also when a product default would exist: S2 not consulted)
	res, err := h.uc.ResolveChargePolicy.Execute(ctx, &policypb.ResolveChargePolicyRequest{ProductChargePolicyId: strp("ignored-in-s1")})
	if err != nil || res.Source != SourceNone || res.Policy != nil {
		t.Fatalf("none: %+v %v", res, err)
	}
	p, v1 := h.seedApprovableDraft(t, "prep", "RESOLVE")
	// policy with only a draft: named refusal
	if _, err := h.uc.ResolveChargePolicy.Execute(ctx, &policypb.ResolveChargePolicyRequest{PackageLineChargePolicyId: strp(p.GetId())}); !usecaseerr.IsCode(err, CodeNoApprovedVersion) {
		t.Fatalf("no approved: %v", err)
	}
	if _, err := approve(t, h, "boss", v1.GetId()); err != nil {
		t.Fatal(err)
	}
	// package-line policy wins over any product default and resolves to the APPROVED version
	res, err = h.uc.ResolveChargePolicy.Execute(ctx, &policypb.ResolveChargePolicyRequest{PackageLineChargePolicyId: strp(p.GetId()), ProductChargePolicyId: strp("other")})
	if err != nil || res.Source != SourcePackageLine || res.Version.GetId() != v1.GetId() || res.Policy.GetId() != p.GetId() {
		t.Fatalf("package line: %+v %v", res, err)
	}
	// after a newer version is approved the resolver returns the new current version
	d2, _ := h.uc.CreateDraftChargePolicyVersion.Execute(ctx, &versionpb.CreateDraftChargePolicyVersionRequest{ChargePolicyId: p.GetId()})
	if _, err := approve(t, h, "boss", d2.Draft.GetId()); err != nil {
		t.Fatal(err)
	}
	res, _ = h.uc.ResolveChargePolicy.Execute(ctx, &policypb.ResolveChargePolicyRequest{PackageLineChargePolicyId: strp(p.GetId())})
	if res.Version.GetId() != d2.Draft.GetId() {
		t.Fatalf("resolver must return current approved version, got %s", res.Version.GetId())
	}
	// unknown policy id
	if _, err := h.uc.ResolveChargePolicy.Execute(ctx, &policypb.ResolveChargePolicyRequest{PackageLineChargePolicyId: strp("nope")}); !usecaseerr.IsCode(err, CodeNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	// retired: named refusal (never a silent fallback)
	if _, err := h.uc.RetireChargePolicy.Execute(ctx, &policypb.RetireChargePolicyRequest{ChargePolicyId: p.GetId()}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.uc.ResolveChargePolicy.Execute(ctx, &policypb.ResolveChargePolicyRequest{PackageLineChargePolicyId: strp(p.GetId())}); !usecaseerr.IsCode(err, CodeRetired) {
		t.Fatalf("retired: %v", err)
	}
}
