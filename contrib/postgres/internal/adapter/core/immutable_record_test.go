//go:build postgresql

package core

import (
	"errors"
	"fmt"
	"testing"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
)

// One immutable-row convention (C6): every adapter refusal is the typed ImmutableRecordError AND
// satisfies the port sentinel, also when wrapped.
func TestImmutableRecordSatisfiesPortSentinel(t *testing.T) {
	err := fmt.Errorf("delete: %w", ErrImmutableRecord("recovery_document"))
	if !errors.Is(err, domainports.ErrImmutableRow) {
		t.Fatal("ErrImmutableRecord must satisfy errors.Is(domainports.ErrImmutableRow)")
	}
	if !IsImmutableRecord(err) {
		t.Fatal("IsImmutableRecord must see the typed error through wrapping")
	}
	var coded interface{ ErrorCode() string }
	if !errors.As(err, &coded) || coded.ErrorCode() != "immutable_record" {
		t.Fatalf("ErrorCode() = %v", coded)
	}
}
