package charge_policy

import (
	"context"
	"log"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// Checklist item codes (stable; the fycha view maps each to a Lyngua label key).
const (
	ChecklistAssessmentScope = "assessment_scope"
	// posting:<EVENT>:<ROLE> items: the four required S1 postings.
	// posting_account:<EVENT>:<ROLE> items: the posting's account exists in the workspace and is active.
	checklistPostingPrefix        = "posting:"
	checklistPostingAccountPrefix = "posting_account:"
)

// Unsupported-combination reason codes (S1 matrix).
const (
	ReasonPrincipalExcluded      = "principal_excluded" // principal role with an excluded-reimbursement tax position
	ReasonAccountingRoleNotAgent = "accounting_role_not_agent"
	ReasonOwnSupply              = "own_supply"
	ReasonTaxPositionNotExcluded = "tax_position_not_excluded_reimbursement"
	ReasonComponentCount         = "component_count_not_one"
	ReasonFee                    = "fee"
	ReasonOwnRevenue             = "own_revenue"
	ReasonComponentRole          = "component_role_not_recovery_cost"
	ReasonInvoice                = "invoice"
	ReasonDocumentKind           = "document_kind_not_recovery_document"
	ReasonComponentBook          = "component_book_presentation_not_excluded_reimbursement"
)

// requiredPostings are the four S1 postings that must exist (build-spec §3).
var requiredPostings = []struct {
	event enumspb.ChargePostingEvent
	role  enumspb.ChargePostingRole
}{
	{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE},
	{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CLEARING},
	{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH},
	{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE},
}

func postingKey(e enumspb.ChargePostingEvent, r enumspb.ChargePostingRole) string {
	return strings.TrimPrefix(e.String(), "CHARGE_POSTING_EVENT_") + ":" + strings.TrimPrefix(r.String(), "CHARGE_POSTING_ROLE_")
}

// classifyCombination applies the S1 matrix: the ONLY approvable combination is
// accounting_role AGENT + tax_position EXCLUDED_REIMBURSEMENT + exactly one component
// RECOVERY_COST on a RECOVERY_DOCUMENT with book presentation EXCLUDED_REIMBURSEMENT.
func classifyCombination(v *versionpb.ChargePolicyVersion, comps []*componentpb.ChargePolicyComponent) []string {
	var reasons []string
	role, tax := v.GetAccountingRole(), v.GetTaxPosition()
	if role != enumspb.AccountingRole_ACCOUNTING_ROLE_AGENT {
		if role == enumspb.AccountingRole_ACCOUNTING_ROLE_PRINCIPAL && tax == enumspb.TaxPosition_TAX_POSITION_EXCLUDED_REIMBURSEMENT {
			reasons = append(reasons, ReasonPrincipalExcluded)
		} else {
			reasons = append(reasons, ReasonAccountingRoleNotAgent)
		}
	}
	switch tax {
	case enumspb.TaxPosition_TAX_POSITION_EXCLUDED_REIMBURSEMENT:
	case enumspb.TaxPosition_TAX_POSITION_OWN_SUPPLY:
		reasons = append(reasons, ReasonOwnSupply)
	default:
		reasons = append(reasons, ReasonTaxPositionNotExcluded)
	}
	if len(comps) != 1 {
		reasons = append(reasons, ReasonComponentCount)
	}
	for _, c := range comps {
		switch c.GetComponentRole() {
		case enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST:
		case enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_FEE:
			reasons = append(reasons, ReasonFee)
		case enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_OWN_REVENUE:
			reasons = append(reasons, ReasonOwnRevenue)
		default:
			reasons = append(reasons, ReasonComponentRole)
		}
		switch c.GetDocumentKind() {
		case enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT:
		case enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_INVOICE:
			reasons = append(reasons, ReasonInvoice)
		default:
			reasons = append(reasons, ReasonDocumentKind)
		}
		if c.GetBookPresentation() != enumspb.BookPresentation_BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT {
			reasons = append(reasons, ReasonComponentBook)
		}
	}
	return reasons
}

// checklist evaluates the S1 approval checklist for a version.
func (c *core) checklist(ctx context.Context, v *versionpb.ChargePolicyVersion, posts []*postingpb.ChargePolicyPosting) []*versionpb.ChargePolicyChecklistItem {
	items := []*versionpb.ChargePolicyChecklistItem{{
		Code:   ChecklistAssessmentScope,
		Passed: !blank(v.GetAssessmentScope()),
	}}
	present := map[string]bool{}
	for _, p := range posts {
		present[postingKey(p.GetEvent(), p.GetPostingRole())] = true
	}
	for _, rp := range requiredPostings {
		k := postingKey(rp.event, rp.role)
		items = append(items, &versionpb.ChargePolicyChecklistItem{Code: checklistPostingPrefix + k, Passed: present[k]})
	}
	for _, p := range posts {
		k := postingKey(p.GetEvent(), p.GetPostingRole())
		ok, detail := c.accountUsable(ctx, p.GetAccountId())
		items = append(items, &versionpb.ChargePolicyChecklistItem{Code: checklistPostingAccountPrefix + k, Passed: ok, Detail: detail})
	}
	return items
}

// accountUsable reports whether the account exists in the caller's workspace (the generic read
// is workspace-scoped, so a foreign account is indistinguishable from a missing one) and is
// active. Fails closed when no account repository is wired.
func (c *core) accountUsable(ctx context.Context, accountID string) (bool, string) {
	if c.repos.Account == nil {
		return false, "account repository unavailable"
	}
	if blank(accountID) {
		return false, "account id is blank"
	}
	resp, err := c.repos.Account.ReadAccount(ctx, &accountpb.ReadAccountRequest{Data: &accountpb.Account{Id: accountID}})
	if err != nil {
		if !usecaseerr.IsNotFound(err) {
			// a lookup failure fails the checklist item (closed) and is logged (C9), never reported as "not found"
			log.Printf("charge_policy: read account failed (ids=[%s]): %v", accountID, err)
			return false, "account lookup failed"
		}
		return false, "account not found in workspace"
	}
	if resp == nil || len(resp.Data) == 0 {
		return false, "account not found in workspace"
	}
	if !resp.Data[0].GetActive() {
		return false, "account is inactive"
	}
	return true, ""
}

// validate builds the full readiness report for a version (no mutation).
func (c *core) validate(ctx context.Context, v *versionpb.ChargePolicyVersion) (*versionpb.ChargePolicyApprovalChecklist, error) {
	comps, err := c.listComponents(ctx, v.GetId())
	if err != nil {
		return nil, err
	}
	posts, err := c.listPostings(ctx, v.GetId())
	if err != nil {
		return nil, err
	}
	res := &versionpb.ChargePolicyApprovalChecklist{Reasons: classifyCombination(v, comps)}
	res.Supported = len(res.Reasons) == 0
	res.Items = c.checklist(ctx, v, posts)
	res.Passed = true
	for _, it := range res.Items {
		if !it.GetPassed() {
			res.Passed = false
			break
		}
	}
	return res, nil
}

func failedCodes(items []*versionpb.ChargePolicyChecklistItem) string {
	var out []string
	for _, it := range items {
		if !it.GetPassed() {
			out = append(out, it.GetCode())
		}
	}
	return strings.Join(out, ",")
}
