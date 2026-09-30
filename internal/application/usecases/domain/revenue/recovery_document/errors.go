package recovery_document

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"
)

// Refusals are usecaseerr.Error values (build-spec §7 C2/C29): Error() = "recovery_document: " + the
// message translated from recovery_document.errors.<code>, ErrorCode() = <code>, so views branch
// without importing espyna. The refusal values are unexported: callers use usecaseerr.IsCode or the
// ErrorCode() interface.
// fallbackMessages are the English defaults of the general-tier Lyngua keys recovery_document.errors.<code>
// (labels-s1.md); vertical wording comes only from Lyngua overlays.
var fallbackMessages = map[string]string{
	"nothing_selected":        "Select at least one charge.",
	"not_open":                "Only open charges can be issued.",
	"not_found":               "Document not found.",
	"not_issued":              "The document being corrected is not issued.",
	"currency_mismatch":       "The currencies do not match.",
	"missing_posting":         "The charge policy has no account mapping for this step.",
	"series_retired":          "This document series is retired.",
	"series_kind_mismatch":    "This document series does not issue this kind of document.",
	"component_kind_mismatch": "A charge on this document cannot be issued on this kind of document.",
	"issuance_conflict":       "These charges were already issued with different content.",
	"has_applications":        "Payments have been applied. Reverse them first.",
	"has_credit_notes":        "Credit notes correct this document. Void them first.",
	"already_void":            "This document is already void.",
	"void_reason_required":    "Enter a reason.",
	"validation":              "The document is not valid.",
	"transaction_required":    "This action needs a database transaction, which is not available.",
}

func newErr(code string) *usecaseerr.Error {
	return usecaseerr.New("recovery_document: ", code, fallbackMessages[code])
}

var (
	errNothingSelected       = newErr("nothing_selected")
	errNotOpen               = newErr("not_open")
	errNotFound              = newErr("not_found")
	errNotIssued             = newErr("not_issued")
	errCurrencyMismatch      = newErr("currency_mismatch")
	errMissingPosting        = newErr("missing_posting")
	errSeriesRetired         = newErr("series_retired")
	errSeriesKindMismatch    = newErr("series_kind_mismatch")
	errComponentKindMismatch = newErr("component_kind_mismatch")
	errIssuanceConflict      = newErr("issuance_conflict")
	errHasApplications       = newErr("has_applications")
	errHasCreditNotes        = newErr("has_credit_notes")
	errAlreadyVoid           = newErr("already_void")
	errVoidReasonRequired    = newErr("void_reason_required")
	errValidation            = newErr("validation")
	errTransactionRequired   = newErr("transaction_required")
)

// invalid is a validation refusal whose translated message keeps the detail out of the code.
func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", errValidation, fmt.Sprintf(format, a...))
}
