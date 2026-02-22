package inventory

import (
	// Inventory use cases
	inventoryItemUC "github.com/erniealice/espyna-golang/internal/application/usecases/inventory/inventory_item"
	inventorySerialUC "github.com/erniealice/espyna-golang/internal/application/usecases/inventory/inventory_serial"
	inventoryTransactionUC "github.com/erniealice/espyna-golang/internal/application/usecases/inventory/inventory_transaction"
	inventoryAttributeUC "github.com/erniealice/espyna-golang/internal/application/usecases/inventory/inventory_attribute"
	inventoryDepreciationUC "github.com/erniealice/espyna-golang/internal/application/usecases/inventory/inventory_depreciation"
	serialHistoryUC "github.com/erniealice/espyna-golang/internal/application/usecases/inventory/inventory_serial_history"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for inventory repositories
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// InventoryRepositories contains all inventory domain repositories
type InventoryRepositories struct {
	InventoryItem          inventoryitempb.InventoryItemDomainServiceServer
	InventorySerial        inventoryserialpb.InventorySerialDomainServiceServer
	InventoryTransaction   inventorytransactionpb.InventoryTransactionDomainServiceServer
	InventoryAttribute     inventoryattributepb.InventoryAttributeDomainServiceServer
	InventoryDepreciation  inventorydepreciationpb.InventoryDepreciationDomainServiceServer
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer
}

// InventoryUseCases contains all inventory-related use cases
type InventoryUseCases struct {
	InventoryItem          *inventoryItemUC.UseCases
	InventorySerial        *inventorySerialUC.UseCases
	InventoryTransaction   *inventoryTransactionUC.UseCases
	InventoryAttribute     *inventoryAttributeUC.UseCases
	InventoryDepreciation  *inventoryDepreciationUC.UseCases
	InventorySerialHistory *serialHistoryUC.UseCases
}

// NewUseCases creates all inventory use cases with proper constructor injection
func NewUseCases(
	repos InventoryRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *InventoryUseCases {
	// Create inventory item use cases
	inventoryItemUseCases := inventoryItemUC.NewUseCases(
		inventoryItemUC.InventoryItemRepositories{
			InventoryItem: repos.InventoryItem,
		},
		inventoryItemUC.InventoryItemServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Create inventory serial use cases
	inventorySerialUseCases := inventorySerialUC.NewUseCases(
		inventorySerialUC.InventorySerialRepositories{
			InventorySerial: repos.InventorySerial,
		},
		inventorySerialUC.InventorySerialServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Create inventory transaction use cases
	inventoryTransactionUseCases := inventoryTransactionUC.NewUseCases(
		inventoryTransactionUC.InventoryTransactionRepositories{
			InventoryTransaction: repos.InventoryTransaction,
		},
		inventoryTransactionUC.InventoryTransactionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Create inventory attribute use cases
	inventoryAttributeUseCases := inventoryAttributeUC.NewUseCases(
		inventoryAttributeUC.InventoryAttributeRepositories{
			InventoryAttribute: repos.InventoryAttribute,
		},
		inventoryAttributeUC.InventoryAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Create inventory depreciation use cases
	inventoryDepreciationUseCases := inventoryDepreciationUC.NewUseCases(
		inventoryDepreciationUC.InventoryDepreciationRepositories{
			InventoryDepreciation: repos.InventoryDepreciation,
		},
		inventoryDepreciationUC.InventoryDepreciationServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Create inventory serial history use cases
	inventorySerialHistoryUseCases := serialHistoryUC.NewUseCases(
		serialHistoryUC.InventorySerialHistoryRepositories{
			InventorySerialHistory: repos.InventorySerialHistory,
		},
		serialHistoryUC.InventorySerialHistoryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &InventoryUseCases{
		InventoryItem:          inventoryItemUseCases,
		InventorySerial:        inventorySerialUseCases,
		InventoryTransaction:   inventoryTransactionUseCases,
		InventoryAttribute:     inventoryAttributeUseCases,
		InventoryDepreciation:  inventoryDepreciationUseCases,
		InventorySerialHistory: inventorySerialHistoryUseCases,
	}
}
