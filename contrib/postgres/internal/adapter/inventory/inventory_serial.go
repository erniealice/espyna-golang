//go:build postgresql

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.InventorySerial, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_serial repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresInventorySerialRepository(dbOps, tableName), nil
	})
}

// PostgresInventorySerialRepository implements inventory_serial CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_serial_active ON inventory_serial(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_serial_inventory_item_id ON inventory_serial(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_serial_serial_number ON inventory_serial(serial_number) - Search on serial_number
//   - CREATE INDEX idx_inventory_serial_imei ON inventory_serial(imei) - Search on imei
//   - CREATE INDEX idx_inventory_serial_status ON inventory_serial(status) - Filter by status
//   - CREATE INDEX idx_inventory_serial_date_created ON inventory_serial(date_created DESC) - Default sorting
//
// Every method routes through dbOps, so the workspace decorator is always in the
// path; the repository holds no raw *sql.DB that could bypass it.
type PostgresInventorySerialRepository struct {
	inventoryserialpb.UnimplementedInventorySerialDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresInventorySerialRepository creates a new PostgreSQL inventory serial repository
func NewPostgresInventorySerialRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryserialpb.InventorySerialDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_serial" // default fallback
	}

	return &PostgresInventorySerialRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateInventorySerial creates a new inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) CreateInventorySerial(ctx context.Context, req *inventoryserialpb.CreateInventorySerialRequest) (*inventoryserialpb.CreateInventorySerialResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory serial data is required")
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
		return nil, fmt.Errorf("failed to create inventory serial: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventorySerial := &inventoryserialpb.InventorySerial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventorySerial); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryserialpb.CreateInventorySerialResponse{
		Data: []*inventoryserialpb.InventorySerial{inventorySerial},
	}, nil
}

// ReadInventorySerial retrieves an inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) ReadInventorySerial(ctx context.Context, req *inventoryserialpb.ReadInventorySerialRequest) (*inventoryserialpb.ReadInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory serial: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventorySerial := &inventoryserialpb.InventorySerial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventorySerial); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryserialpb.ReadInventorySerialResponse{
		Data: []*inventoryserialpb.InventorySerial{inventorySerial},
	}, nil
}

// UpdateInventorySerial updates an inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) UpdateInventorySerial(ctx context.Context, req *inventoryserialpb.UpdateInventorySerialRequest) (*inventoryserialpb.UpdateInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial ID is required")
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
		return nil, fmt.Errorf("failed to update inventory serial: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventorySerial := &inventoryserialpb.InventorySerial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventorySerial); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryserialpb.UpdateInventorySerialResponse{
		Data: []*inventoryserialpb.InventorySerial{inventorySerial},
	}, nil
}

// DeleteInventorySerial deletes an inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) DeleteInventorySerial(ctx context.Context, req *inventoryserialpb.DeleteInventorySerialRequest) (*inventoryserialpb.DeleteInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory serial: %w", err)
	}

	return &inventoryserialpb.DeleteInventorySerialResponse{
		Success: true,
	}, nil
}

// ListInventorySerials lists inventory serials using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) ListInventorySerials(ctx context.Context, req *inventoryserialpb.ListInventorySerialsRequest) (*inventoryserialpb.ListInventorySerialsResponse, error) {
	// Build filter params, honoring entity-specific InventoryItemId field
	var params *interfaces.ListParams
	if req != nil {
		filters := req.Filters
		if itemID := req.GetInventoryItemId(); itemID != "" {
			itemFilter := &commonpb.TypedFilter{
				Field: "inventory_item_id",
				FilterType: &commonpb.TypedFilter_StringFilter{
					StringFilter: &commonpb.StringFilter{
						Value:    itemID,
						Operator: commonpb.StringOperator_STRING_EQUALS,
					},
				},
			}
			if filters == nil {
				filters = &commonpb.FilterRequest{}
			}
			filters.Filters = append(filters.Filters, itemFilter)
		}
		if filters != nil {
			params = &interfaces.ListParams{Filters: filters}
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory serials: %w", err)
	}

	// Convert results to protobuf slice using protojson
	unmarshalOpts := protojson.UnmarshalOptions{DiscardUnknown: true}
	var inventorySerials []*inventoryserialpb.InventorySerial
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		inventorySerial := &inventoryserialpb.InventorySerial{}
		if err := unmarshalOpts.Unmarshal(resultJSON, inventorySerial); err != nil {
			continue
		}
		inventorySerials = append(inventorySerials, inventorySerial)
	}

	return &inventoryserialpb.ListInventorySerialsResponse{
		Data: inventorySerials,
	}, nil
}

// NewInventorySerialRepository creates a new PostgreSQL inventory serial repository (old-style constructor)
func NewInventorySerialRepository(db *sql.DB, tableName string) inventoryserialpb.InventorySerialDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresInventorySerialRepository(dbOps, tableName)
}
