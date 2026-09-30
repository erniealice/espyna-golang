package document_series

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"
)

// Refusals are usecaseerr.Error values (build-spec §7 C2, §7c C29): ErrorCode() = <code>, the
// message is translated from the Lyngua key <entity>.errors.<code> by usecaseerr.Localize.

// fallbackMessages are the English defaults of the general-tier Lyngua keys document_series.errors.<code>
// (labels-s1.md); vertical wording comes only from Lyngua overlays.
var fallbackMessages = map[string]string{
	"validation":           "The document series is not valid.",
	"not_found":            "Series not found.",
	"code_taken":           "That code is already used.",
	"retired":              "This document series is retired.",
	"numbering_locked":     "The prefix and number padding cannot change once numbers have been issued.",
	"transaction_required": "This action needs a database transaction, which is not available.",
}

func newErr(code string) *usecaseerr.Error {
	return usecaseerr.New("document_series: ", code, fallbackMessages[code])
}

var (
	errValidation          = newErr("validation")
	errNotFound            = newErr("not_found")
	errCodeTaken           = newErr("code_taken")
	errRetired             = newErr("retired")
	errNumberingLocked     = newErr("numbering_locked")
	errTransactionRequired = newErr("transaction_required")
)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", errValidation, fmt.Sprintf(format, a...))
}
