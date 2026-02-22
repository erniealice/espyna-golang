package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Protobuf domain services - Inventory domain
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// InventoryRepositories contains all 6 inventory domain repositories
// Inventory domain: InventoryItem, InventorySerial, InventoryTransaction, InventoryAttribute, InventoryDepreciation, InventorySerialHistory
type InventoryRepositories struct {
	InventoryItem          inventoryitempb.InventoryItemDomainServiceServer
	InventorySerial        inventoryserialpb.InventorySerialDomainServiceServer
	InventoryTransaction   inventorytransactionpb.InventoryTransactionDomainServiceServer
	InventoryAttribute     inventoryattributepb.InventoryAttributeDomainServiceServer
	InventoryDepreciation  inventorydepreciationpb.InventoryDepreciationDomainServiceServer
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer
}

// NewInventoryRepositories creates and returns a new set of InventoryRepositories
func NewInventoryRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*InventoryRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names directly from dbTableConfig
	inventoryItemRepo, err := repoCreator.CreateRepository("inventory_item", conn, dbTableConfig.InventoryItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory_item repository: %w", err)
	}

	inventorySerialRepo, err := repoCreator.CreateRepository("inventory_serial", conn, dbTableConfig.InventorySerial)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory_serial repository: %w", err)
	}

	inventoryTransactionRepo, err := repoCreator.CreateRepository("inventory_transaction", conn, dbTableConfig.InventoryTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory_transaction repository: %w", err)
	}

	inventoryAttributeRepo, err := repoCreator.CreateRepository("inventory_attribute", conn, dbTableConfig.InventoryAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory_attribute repository: %w", err)
	}

	inventoryDepreciationRepo, err := repoCreator.CreateRepository("inventory_depreciation", conn, dbTableConfig.InventoryDepreciation)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory_depreciation repository: %w", err)
	}

	inventorySerialHistoryRepo, err := repoCreator.CreateRepository("inventory_serial_history", conn, dbTableConfig.InventorySerialHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory_serial_history repository: %w", err)
	}

	// Type assert each repository to its interface
	return &InventoryRepositories{
		InventoryItem:          inventoryItemRepo.(inventoryitempb.InventoryItemDomainServiceServer),
		InventorySerial:        inventorySerialRepo.(inventoryserialpb.InventorySerialDomainServiceServer),
		InventoryTransaction:   inventoryTransactionRepo.(inventorytransactionpb.InventoryTransactionDomainServiceServer),
		InventoryAttribute:     inventoryAttributeRepo.(inventoryattributepb.InventoryAttributeDomainServiceServer),
		InventoryDepreciation:  inventoryDepreciationRepo.(inventorydepreciationpb.InventoryDepreciationDomainServiceServer),
		InventorySerialHistory: inventorySerialHistoryRepo.(serialhistorypb.InventorySerialHistoryDomainServiceServer),
	}, nil
}
