package billable_charge

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// Named refusals (C2, build-spec §7). The returned error's message is Translator-translated from
// key billable_charge.errors.<code> and the type exposes ErrorCode() = <code> (errors.As on
// interface{ ErrorCode() string }); nothing here is an exported sentinel. Precedent:
// subscription/product_price_plan/charge_policy_guard.go (GuardError) +
// expenditure/supplier_contract/approve.go:51 (translated message).
const (
	codeValidation          = "validation"
	codeNotFound            = "not_found"
	codeNotIssued           = "not_issued"
	codeAdjustNotDownward   = "adjust_not_downward"
	codeTransactionRequired = "transaction_required"
	codeLockUnavailable     = "lock_unavailable"
	codeObligationConflict  = "obligation_conflict"
)

// fallbackMessages are the English defaults of the general-tier Lyngua keys.
var fallbackMessages = map[string]string{
	codeValidation:          "The charge is not valid.",
	codeNotFound:            "Charge not found.",
	codeNotIssued:           "Only issued charges can be adjusted.",
	codeAdjustNotDownward:   "The new amount must be lower than the current amount.",
	codeTransactionRequired: "This action needs a database transaction, which is not available.",
	codeLockUnavailable:     "This action needs row locking, which the storage provider does not support.",
	codeObligationConflict:  "This charge already exists with different details.",
}

// refuse builds the translated named refusal; detail (never row data) extends the English fallback only.
func refuse(ctx context.Context, tr ports.Translator, code string, detail ...string) error {
	fb := fallbackMessages[code]
	if len(detail) > 0 && detail[0] != "" {
		fb += " (" + detail[0] + ")"
	}
	return usecaseerr.New("", code, contextutil.GetTranslatedMessageWithContext(ctx, tr, "billable_charge.errors."+code, fb))
}
