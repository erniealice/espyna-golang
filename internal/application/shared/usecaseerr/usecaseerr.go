// Package usecaseerr is the one coded-refusal contract of the espyna use cases
// (20260927-usage-and-pass-through-charges build-spec §7 C2/C9, §7c C29): a stable
// refusal code that views branch on through the ErrorCode() interface, a message
// translated from the general-tier Lyngua key <entity>.errors.<code>, and the
// repository-failure helpers that keep infrastructure errors from ever being
// reported as a named refusal.
//
// Charter — this package MUST NOT import:
//   - proto entity/service types (esqyma/.../domain/<entity>, .../service/<X>);
//   - DB drivers or adapter packages;
//   - anything under internal/application/usecases/...
//
// It depends on the standard library, ports (Translator), ports/domain (the
// ErrLockedRowNotFound sentinel) and shared/context (translation lookup).
//
// Consumers (keep in sync): usecases/domain/{treasury/collection,
// treasury/collection_application, revenue/{recovery_document,document_series,
// revenue_payment}, ledger/{charge_policy,charge_effect}, expenditure/{allocation_batch,
// cost_source_component,expense_recognition}, subscription/{billable_charge,
// agreement_line_term}}.
package usecaseerr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// Error is a stable named refusal. ErrorCode() returns the code (the last segment of the Lyngua
// key <entity>.errors.<code>); Error() returns prefix + message, where prefix is the owning
// package's label ("recovery_document: ") or empty.
type Error struct {
	prefix string
	code   string
	msg    string
}

// New builds a refusal. prefix is prepended to the message by Error() and may be empty.
func New(prefix, code, msg string) *Error { return &Error{prefix: prefix, code: code, msg: msg} }

func (e *Error) Error() string     { return e.prefix + e.msg }
func (e *Error) ErrorCode() string { return e.code }

// Message returns the message without the package prefix.
func (e *Error) Message() string { return e.msg }

// WithMessage returns a copy of the refusal (same prefix and code) carrying msg.
func (e *Error) WithMessage(msg string) *Error {
	return &Error{prefix: e.prefix, code: e.code, msg: msg}
}

// IsCode reports whether err (or anything it wraps) carries the given stable code.
func IsCode(err error, code string) bool {
	var e interface{ ErrorCode() string }
	return errors.As(err, &e) && e.ErrorCode() == code
}

// Localize returns err with its message translated from <entity>.errors.<code>, falling back to
// the refusal's own message; the prefix and ErrorCode() are unchanged. Errors that carry no
// *Error pass through unchanged, and a nil translator keeps the fallback message.
func Localize(ctx context.Context, tr ports.Translator, entity string, err error) error {
	var e *Error
	if err == nil || !errors.As(err, &e) {
		return err
	}
	msg := contextutil.GetTranslatedMessageWithContext(ctx, tr, entity+".errors."+e.code, e.msg)
	return e.WithMessage(msg)
}

// RepoErr logs a repository failure (package + operation + ids only, never row data) and wraps
// it as "<pkg>: <op>: <err>". It is never mapped to a named refusal (C9).
func RepoErr(pkg, op string, err error, ids ...string) error {
	log.Printf("%s: %s failed (ids=%v): %v", pkg, op, ids, err)
	return fmt.Errorf("%s: %s: %w", pkg, op, err)
}

// IsNotFound recognises the adapters' "row absent (or foreign workspace)" outcomes: a locker's
// wrapped domainports.ErrLockedRowNotFound or the generic single-row read's "record not found".
// Any other error is an infrastructure failure and must never be reported as not_found (C9).
func IsNotFound(err error) bool {
	return err != nil && (errors.Is(err, domainports.ErrLockedRowNotFound) ||
		strings.Contains(strings.ToLower(err.Error()), "record not found"))
}
