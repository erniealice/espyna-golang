//go:build postgresql

package core

import (
	"errors"
	"fmt"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
)

// ImmutableRecordError is returned by the Delete method of an adapter whose rows are immutable
// financial records (billable_charge, charge_component, recovery_document(+line),
// collection_application, charge_effect, published allocation). Such rows are corrected by a new
// reversing/correcting row, never removed at the port (build-spec §7 C6). ErrorCode() lets callers
// branch without importing this package.
type ImmutableRecordError struct{ Table string }

func (e *ImmutableRecordError) Error() string {
	return fmt.Sprintf("%s rows are immutable and cannot be deleted", e.Table)
}

// Unwrap makes errors.Is(err, domainports.ErrImmutableRow) true: ONE port-level sentinel for every
// immutable-row refusal, whichever adapter (S1 allocation, charge policy, recovery documents,
// collection applications, charge effects) raised it.
func (e *ImmutableRecordError) Unwrap() error { return domainports.ErrImmutableRow }

// ErrorCode is the stable refusal code.
func (e *ImmutableRecordError) ErrorCode() string { return "immutable_record" }

// ErrImmutableRecord builds the refusal for table.
func ErrImmutableRecord(table string) error { return &ImmutableRecordError{Table: table} }

// IsImmutableRecord reports whether err is an ImmutableRecordError.
func IsImmutableRecord(err error) bool {
	var e *ImmutableRecordError
	return errors.As(err, &e)
}
