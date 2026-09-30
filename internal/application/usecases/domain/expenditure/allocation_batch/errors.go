package allocation_batch

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// Named refusals (C2, build-spec §7). The returned error's message is Translator-translated from
// key allocation_batch.errors.<code> and the type exposes ErrorCode() = <code> (errors.As on
// interface{ ErrorCode() string }); nothing here is an exported sentinel. Precedent:
// subscription/product_price_plan/charge_policy_guard.go (GuardError) +
// expenditure/supplier_contract/approve.go:51 (translated message).
const (
	codeValidation                 = "validation"
	codeNotFound                   = "not_found"
	codeNotDraft                   = "not_draft"
	codeAlreadyPublished           = "already_published"
	codeSourceClaimedByRecognition = "source_claimed_by_recognition"
	codeSourceClaimedByAllocation  = "source_claimed_by_allocation"
	codeSharesTotalMismatch        = "shares_total_mismatch"
	codeDenominatorInvalid         = "denominator_invalid"
	codeNoRecoverableShare         = "no_recoverable_share"
	codeServicePeriodInvalid       = "service_period_invalid"
	codePolicyComponentInvalid     = "policy_component_invalid"
	codeTransactionRequired        = "transaction_required"
	codeLockUnavailable            = "lock_unavailable"
	// Refusals raised by the agreement-term rules (agreement_line_term package) and re-issued
	// here under allocation_batch.errors.<code>.
	codeAgreementTermMissing = "agreement_term_missing"
	codeTermBoundaryCrossed  = "term_boundary_crossed"
	codeOverlap              = "overlap"
)

// fallbackMessages are the English defaults of the general-tier Lyngua keys.
var fallbackMessages = map[string]string{
	codeValidation:                 "The allocation is not valid.",
	codeNotFound:                   "Allocation not found.",
	codeNotDraft:                   "Only drafts can be changed.",
	codeAlreadyPublished:           "Already published.",
	codeSourceClaimedByRecognition: "This cost is already recognised as an expense.",
	codeSourceClaimedByAllocation:  "This cost is already allocated.",
	codeSharesTotalMismatch:        "The shares do not add up to the cost line amount.",
	codeDenominatorInvalid:         "Total weight must be above zero.",
	codeNoRecoverableShare:         "Add at least one recoverable share.",
	codeServicePeriodInvalid:       "The cost component has no valid service period.",
	codePolicyComponentInvalid:     "The charge policy version must have exactly one complete component.",
	codeTransactionRequired:        "This action needs a database transaction, which is not available.",
	codeLockUnavailable:            "This action needs row locking, which the storage provider does not support.",
	codeAgreementTermMissing:       "The agreement has no charge term for this customer.",
	codeTermBoundaryCrossed:        "The service period crosses a change in the agreement's charge terms.",
	codeOverlap:                    "Charge terms of one line cannot overlap.",
}

// refuse builds the translated named refusal; detail (never row data) extends the English fallback only.
func refuse(ctx context.Context, tr ports.Translator, code string, detail ...string) error {
	fb := fallbackMessages[code]
	if len(detail) > 0 && detail[0] != "" {
		fb += " (" + detail[0] + ")"
	}
	return usecaseerr.New("", code, contextutil.GetTranslatedMessageWithContext(ctx, tr, "allocation_batch.errors."+code, fb))
}

// reissue re-issues a coded refusal of a collaborating package (agreement terms) as this
// package's own named refusal, keeping the code.
func reissue(ctx context.Context, tr ports.Translator, err error) error {
	var e interface{ ErrorCode() string }
	if errors.As(err, &e) {
		if _, known := fallbackMessages[e.ErrorCode()]; known {
			return refuse(ctx, tr, e.ErrorCode())
		}
	}
	return err
}
