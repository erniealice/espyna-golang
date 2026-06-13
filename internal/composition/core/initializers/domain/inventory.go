package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/inventory"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeInventory creates all inventory use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeInventory(
	repos *domain.InventoryRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
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
		actionGate,
	), nil
}
