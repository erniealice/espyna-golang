package cost_source_component

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// Named refusals (C2, build-spec §7). The returned error's message is Translator-translated from
// key cost_source_component.errors.<code> and the type exposes ErrorCode() = <code>, so a view
// branches with errors.As on interface{ ErrorCode() string } without importing this package.
// Precedent: subscription/product_price_plan/charge_policy_guard.go (GuardError) +
// supplier_contract/approve.go:51 (translated message).
const (
	codeValidation          = "validation"
	codeNotFound            = "not_found"
	codeClaimed             = "claimed"
	codeTransactionRequired = "transaction_required"
	codeReferenceInvalid    = "reference_invalid"
	codeLockUnavailable     = "lock_unavailable"
)

// fallbackMessages are the English defaults of the general-tier Lyngua keys.
var fallbackMessages = map[string]string{
	codeValidation:          "The cost component is not valid.",
	codeNotFound:            "Cost line not found.",
	codeClaimed:             "This cost component is already claimed by an allocation or a recognition and can no longer be changed.",
	codeTransactionRequired: "This action needs a database transaction, which is not available.",
	codeReferenceInvalid:    "A referenced expenditure, line item or tax treatment was not found in this workspace.",
	codeLockUnavailable:     "This action needs row locking, which the storage provider does not support.",
}

// refuse builds the translated named refusal for code; detail is appended to the English fallback
// only (never row data).
func refuse(ctx context.Context, tr ports.Translator, code string, detail ...string) error {
	fb := fallbackMessages[code]
	if len(detail) > 0 && detail[0] != "" {
		fb += " (" + detail[0] + ")"
	}
	return usecaseerr.New("", code, contextutil.GetTranslatedMessageWithContext(ctx, tr, "cost_source_component.errors."+code, fb))
}
