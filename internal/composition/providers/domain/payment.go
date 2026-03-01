package domain

import (
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// PaymentRepositories contains payment domain repositories.
// The legacy Payment, PaymentAttribute, PaymentMethod, and PaymentProfile entities
// have been removed. Their functionality is superseded by:
//   - Collection (money IN) — revenue settlements
//   - Disbursement (money OUT) — expenditure settlements
type PaymentRepositories struct {
	// Reserved for future payment domain repositories if needed
}

// NewPaymentRepositories creates and returns a new set of PaymentRepositories.
// Currently returns an empty struct since all legacy payment entities have been removed.
func NewPaymentRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*PaymentRepositories, error) {
	return &PaymentRepositories{}, nil
}
