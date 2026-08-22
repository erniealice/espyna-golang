//go:build postgresql

package inventory

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.InventoryDepreciation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_depreciation repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresInventoryDepreciationRepository(dbOps, tableName), nil
	})
}

// PostgresInventoryDepreciationRepository implements inventory_depreciation CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_depreciation_active ON inventory_depreciation(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_depreciation_inventory_item_id ON inventory_depreciation(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_depreciation_method ON inventory_depreciation(method) - Search on method
//   - CREATE INDEX idx_inventory_depreciation_start_date ON inventory_depreciation(start_date) - Sort/filter by start_date
//   - CREATE INDEX idx_inventory_depreciation_date_created ON inventory_depreciation(date_created DESC) - Default sorting
//
// Every method routes through dbOps, so the workspace decorator is always in the
// path; the repository holds no raw *sql.DB that could bypass it.
type PostgresInventoryDepreciationRepository struct {
	inventorydepreciationpb.UnimplementedInventoryDepreciationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresInventoryDepreciationRepository creates a new PostgreSQL inventory depreciation repository
func NewPostgresInventoryDepreciationRepository(dbOps interfaces.DatabaseOperation, tableName string) inventorydepreciationpb.InventoryDepreciationDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_depreciation" // default fallback
	}

	return &PostgresInventoryDepreciationRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateInventoryDepreciation creates a new inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) CreateInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.CreateInventoryDepreciationRequest) (*inventorydepreciationpb.CreateInventoryDepreciationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory depreciation data is required")
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
		return nil, fmt.Errorf("failed to create inventory depreciation: %w", err)
	}

	// Convert result back to protobuf using protojson
	postgresCore.ConvertMillisToDateStr(result, "start_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryDepreciation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorydepreciationpb.CreateInventoryDepreciationResponse{
		Data: []*inventorydepreciationpb.InventoryDepreciation{inventoryDepreciation},
	}, nil
}

// ReadInventoryDepreciation retrieves an inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) ReadInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.ReadInventoryDepreciationRequest) (*inventorydepreciationpb.ReadInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory depreciation ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory depreciation: %w", err)
	}

	// Convert result to protobuf using protojson
	postgresCore.ConvertMillisToDateStr(result, "start_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryDepreciation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorydepreciationpb.ReadInventoryDepreciationResponse{
		Data: []*inventorydepreciationpb.InventoryDepreciation{inventoryDepreciation},
	}, nil
}

// UpdateInventoryDepreciation updates an inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) UpdateInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.UpdateInventoryDepreciationRequest) (*inventorydepreciationpb.UpdateInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory depreciation ID is required")
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
		return nil, fmt.Errorf("failed to update inventory depreciation: %w", err)
	}

	// Convert result back to protobuf using protojson
	postgresCore.ConvertMillisToDateStr(result, "start_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryDepreciation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorydepreciationpb.UpdateInventoryDepreciationResponse{
		Data: []*inventorydepreciationpb.InventoryDepreciation{inventoryDepreciation},
	}, nil
}

// DeleteInventoryDepreciation deletes an inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) DeleteInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.DeleteInventoryDepreciationRequest) (*inventorydepreciationpb.DeleteInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory depreciation ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory depreciation: %w", err)
	}

	return &inventorydepreciationpb.DeleteInventoryDepreciationResponse{
		Success: true,
	}, nil
}

// ListInventoryDepreciations lists inventory depreciations using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) ListInventoryDepreciations(ctx context.Context, req *inventorydepreciationpb.ListInventoryDepreciationsRequest) (*inventorydepreciationpb.ListInventoryDepreciationsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory depreciations: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var inventoryDepreciations []*inventorydepreciationpb.InventoryDepreciation
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(result, "start_date")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryDepreciation); err != nil {
			// Log error and continue with next item
			continue
		}
		inventoryDepreciations = append(inventoryDepreciations, inventoryDepreciation)
	}

	return &inventorydepreciationpb.ListInventoryDepreciationsResponse{
		Data: inventoryDepreciations,
	}, nil
}

// NewInventoryDepreciationRepository creates a new PostgreSQL inventory depreciation repository (old-style constructor)
func NewInventoryDepreciationRepository(db *sql.DB, tableName string) inventorydepreciationpb.InventoryDepreciationDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresInventoryDepreciationRepository(dbOps, tableName)
}
