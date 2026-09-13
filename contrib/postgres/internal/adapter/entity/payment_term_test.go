//go:build postgresql

package entity

import (
	"context"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// paymentTermWriteFixture embeds the operation interface so this focused test
// only has to observe the create/update payloads used by the adapter.
type paymentTermWriteFixture struct {
	interfaces.DatabaseOperation
	createData map[string]any
	updateData map[string]any
}

func (f *paymentTermWriteFixture) Create(_ context.Context, _ string, data map[string]any) (map[string]any, error) {
	f.createData = data
	return map[string]any{"id": "payment-term-1", "net_days": int32(0), "is_default": false}, nil
}

func (f *paymentTermWriteFixture) Update(_ context.Context, _ string, _ string, data map[string]any) (map[string]any, error) {
	f.updateData = data
	return map[string]any{"id": "payment-term-1", "net_days": int32(0), "is_default": false}, nil
}

func TestPaymentTermWritesPersistZeroValuedScalars(t *testing.T) {
	fixture := &paymentTermWriteFixture{}
	repo := NewPostgresPaymentTermRepository(fixture, "payment_term").(*PostgresPaymentTermRepository)
	term := &paymenttermpb.PaymentTerm{Name: "Immediate", Code: "cod"}

	if _, err := repo.CreatePaymentTerm(context.Background(), &paymenttermpb.CreatePaymentTermRequest{Data: term}); err != nil {
		t.Fatalf("CreatePaymentTerm() error = %v", err)
	}
	assertPaymentTermScalarPayload(t, fixture.createData)

	term.Id = "payment-term-1"
	if _, err := repo.UpdatePaymentTerm(context.Background(), &paymenttermpb.UpdatePaymentTermRequest{Data: term}); err != nil {
		t.Fatalf("UpdatePaymentTerm() error = %v", err)
	}
	assertPaymentTermScalarPayload(t, fixture.updateData)
}

func assertPaymentTermScalarPayload(t *testing.T, data map[string]any) {
	t.Helper()
	if got, ok := data["netDays"].(int32); !ok || got != 0 {
		t.Fatalf("netDays payload = %#v, want int32(0)", data["netDays"])
	}
	if got, ok := data["isDefault"].(bool); !ok || got {
		t.Fatalf("isDefault payload = %#v, want false", data["isDefault"])
	}
}
