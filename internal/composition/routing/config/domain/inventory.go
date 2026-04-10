package domain

import (
	"fmt"

	inventoryuc "github.com/erniealice/espyna-golang/internal/application/usecases/inventory"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
)

// ConfigureInventoryDomain configures routes for the Inventory domain.
func ConfigureInventoryDomain(inventoryUseCases *inventoryuc.InventoryUseCases) contracts.DomainRouteConfiguration {
	if inventoryUseCases == nil {
		fmt.Printf("Inventory use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "inventory",
			Prefix:  "/inventory",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// InventoryItem routes
	if inventoryUseCases.InventoryItem != nil {
		if inventoryUseCases.InventoryItem.CreateInventoryItem != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-item/create",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryItem.CreateInventoryItem, &inventoryitempb.CreateInventoryItemRequest{}),
			})
		}
		if inventoryUseCases.InventoryItem.ReadInventoryItem != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-item/read",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryItem.ReadInventoryItem, &inventoryitempb.ReadInventoryItemRequest{}),
			})
		}
		if inventoryUseCases.InventoryItem.UpdateInventoryItem != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-item/update",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryItem.UpdateInventoryItem, &inventoryitempb.UpdateInventoryItemRequest{}),
			})
		}
		if inventoryUseCases.InventoryItem.DeleteInventoryItem != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-item/delete",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryItem.DeleteInventoryItem, &inventoryitempb.DeleteInventoryItemRequest{}),
			})
		}
		if inventoryUseCases.InventoryItem.ListInventoryItems != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-item/list",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryItem.ListInventoryItems, &inventoryitempb.ListInventoryItemsRequest{}),
			})
		}
	}

	// InventorySerial routes
	if inventoryUseCases.InventorySerial != nil {
		if inventoryUseCases.InventorySerial.CreateInventorySerial != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-serial/create",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventorySerial.CreateInventorySerial, &inventoryserialpb.CreateInventorySerialRequest{}),
			})
		}
		if inventoryUseCases.InventorySerial.ReadInventorySerial != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-serial/read",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventorySerial.ReadInventorySerial, &inventoryserialpb.ReadInventorySerialRequest{}),
			})
		}
		if inventoryUseCases.InventorySerial.UpdateInventorySerial != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-serial/update",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventorySerial.UpdateInventorySerial, &inventoryserialpb.UpdateInventorySerialRequest{}),
			})
		}
		if inventoryUseCases.InventorySerial.DeleteInventorySerial != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-serial/delete",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventorySerial.DeleteInventorySerial, &inventoryserialpb.DeleteInventorySerialRequest{}),
			})
		}
		if inventoryUseCases.InventorySerial.ListInventorySerials != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-serial/list",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventorySerial.ListInventorySerials, &inventoryserialpb.ListInventorySerialsRequest{}),
			})
		}
	}

	// InventoryTransaction routes
	if inventoryUseCases.InventoryTransaction != nil {
		if inventoryUseCases.InventoryTransaction.CreateInventoryTransaction != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-transaction/create",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryTransaction.CreateInventoryTransaction, &inventorytransactionpb.CreateInventoryTransactionRequest{}),
			})
		}
		if inventoryUseCases.InventoryTransaction.ReadInventoryTransaction != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-transaction/read",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryTransaction.ReadInventoryTransaction, &inventorytransactionpb.ReadInventoryTransactionRequest{}),
			})
		}
		if inventoryUseCases.InventoryTransaction.ListInventoryTransactions != nil {
			routes = append(routes, contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/inventory/inventory-transaction/list",
				Handler: contracts.NewGenericHandler(inventoryUseCases.InventoryTransaction.ListInventoryTransactions, &inventorytransactionpb.ListInventoryTransactionsRequest{}),
			})
		}
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "inventory",
		Prefix:  "/inventory",
		Enabled: len(routes) > 0,
		Routes:  routes,
	}
}
