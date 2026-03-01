package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
)

// RegisterPaymentUseCases registers all payment domain use cases with the registry.
// All payment entities (Payment, PaymentMethod, PaymentProfile, PaymentAttribute)
// have been superseded by collection (money IN) and disbursement (money OUT).
func RegisterPaymentUseCases(_ *usecases.Aggregate, _ func(string, ports.ActivityExecutor)) {
	// No-op: payment entities removed — superseded by collection/disbursement
}
