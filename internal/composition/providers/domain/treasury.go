package domain

import (
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// TreasuryRepositories contains treasury domain repositories.
// The legacy Payment, PaymentAttribute, PaymentMethod, and PaymentProfile entities
// have been removed. Their functionality is superseded by:
//   - Collection (money IN) — revenue settlements
//   - Disbursement (money OUT) — expenditure settlements
type TreasuryRepositories struct {
	// Reserved for future treasury domain repositories if needed
}

// NewTreasuryRepositories creates and returns a new set of TreasuryRepositories.
// Currently returns an empty struct since all legacy payment entities have been removed.
func NewTreasuryRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*TreasuryRepositories, error) {
	return &TreasuryRepositories{}, nil
}
