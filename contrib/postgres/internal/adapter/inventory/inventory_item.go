//go:build postgresql

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.InventoryItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresInventoryItemRepository(dbOps, tableName), nil
	})
}

// PostgresInventoryItemRepository implements inventory_item CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_item_active ON inventory_item(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_item_product_id ON inventory_item(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_inventory_item_location_id ON inventory_item(location_id) - FK lookup on location_id
//   - CREATE INDEX idx_inventory_item_name ON inventory_item(name) - Search on name field
//   - CREATE INDEX idx_inventory_item_sku ON inventory_item(sku) - Search on sku field
//   - CREATE INDEX idx_inventory_item_date_created ON inventory_item(date_created DESC) - Default sorting
//
// Every method routes through dbOps, so the workspace decorator is always in the
// path; the repository holds no raw *sql.DB that could bypass it.
type PostgresInventoryItemRepository struct {
	inventoryitempb.UnimplementedInventoryItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresInventoryItemRepository creates a new PostgreSQL inventory item repository
func NewPostgresInventoryItemRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_item" // default fallback
	}

	return &PostgresInventoryItemRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateInventoryItem creates a new inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) CreateInventoryItem(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) (*inventoryitempb.CreateInventoryItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory item data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory item: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.CreateInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// ReadInventoryItem retrieves an inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) ReadInventoryItem(ctx context.Context, req *inventoryitempb.ReadInventoryItemRequest) (*inventoryitempb.ReadInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory item: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.ReadInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// UpdateInventoryItem updates an inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) UpdateInventoryItem(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) (*inventoryitempb.UpdateInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory item: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.UpdateInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// DeleteInventoryItem deletes an inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) DeleteInventoryItem(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) (*inventoryitempb.DeleteInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory item: %w", err)
	}

	return &inventoryitempb.DeleteInventoryItemResponse{
		Success: true,
	}, nil
}

var inventoryItemSortableSQLCols = []string{
	"id", "active", "name", "product_id", "location_id", "sku",
	"quantity_on_hand", "quantity_reserved", "quantity_available",
	"reorder_level", "unit_of_measure", "date_created", "date_modified",
}

var inventoryItemSortSpec = espynahttp.SortSpec{AllowedCols: inventoryItemSortableSQLCols}

// ListInventoryItems lists inventory items using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) ListInventoryItems(ctx context.Context, req *inventoryitempb.ListInventoryItemsRequest) (*inventoryitempb.ListInventoryItemsResponse, error) {
	if err := espynahttp.ValidateSortColumns(inventoryItemSortSpec, req.GetSort(), "inventory_item"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory items: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var inventoryItems []*inventoryitempb.InventoryItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		inventoryItem := &inventoryitempb.InventoryItem{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
			// Log error and continue with next item
			continue
		}
		inventoryItems = append(inventoryItems, inventoryItem)
	}

	return &inventoryitempb.ListInventoryItemsResponse{
		Data: inventoryItems,
	}, nil
}

// NewInventoryItemRepository creates a new PostgreSQL inventory item repository (old-style constructor)
func NewInventoryItemRepository(db *sql.DB, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresInventoryItemRepository(dbOps, tableName)
}
