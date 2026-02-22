package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/inventory"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeInventory creates all inventory use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeInventory(
	repos *domain.InventoryRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*inventory.InventoryUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return inventory.NewUseCases(
		inventory.InventoryRepositories{
			InventoryItem:          repos.InventoryItem,
			InventorySerial:        repos.InventorySerial,
			InventoryTransaction:   repos.InventoryTransaction,
			InventoryAttribute:     repos.InventoryAttribute,
			InventoryDepreciation:  repos.InventoryDepreciation,
			InventorySerialHistory: repos.InventorySerialHistory,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
