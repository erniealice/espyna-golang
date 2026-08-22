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
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.InventorySerialHistory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_serial_history repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresInventorySerialHistoryRepository(dbOps, tableName), nil
	})
}

// PostgresInventorySerialHistoryRepository implements inventory_serial_history operations using PostgreSQL
// This is an IMMUTABLE audit trail — records are never updated, only appended.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_serial_history_inventory_serial_id ON inventory_serial_history(inventory_serial_id) - FK lookup
//   - CREATE INDEX idx_inventory_serial_history_inventory_item_id ON inventory_serial_history(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_serial_history_from_status ON inventory_serial_history(from_status) - Filter by from_status
//   - CREATE INDEX idx_inventory_serial_history_to_status ON inventory_serial_history(to_status) - Filter by to_status
//   - CREATE INDEX idx_inventory_serial_history_reference_type ON inventory_serial_history(reference_type) - Filter by reference_type
//   - CREATE INDEX idx_inventory_serial_history_date_created ON inventory_serial_history(date_created DESC) - Default sorting
//
// Every method routes through dbOps, so the workspace decorator is always in the
// path; the repository holds no raw *sql.DB that could bypass it.
type PostgresInventorySerialHistoryRepository struct {
	serialhistorypb.UnimplementedInventorySerialHistoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresInventorySerialHistoryRepository creates a new PostgreSQL inventory serial history repository
func NewPostgresInventorySerialHistoryRepository(dbOps interfaces.DatabaseOperation, tableName string) serialhistorypb.InventorySerialHistoryDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_serial_history" // default fallback
	}

	return &PostgresInventorySerialHistoryRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateInventorySerialHistory creates a new inventory serial history record using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) CreateInventorySerialHistory(ctx context.Context, req *serialhistorypb.CreateInventorySerialHistoryRequest) (*serialhistorypb.CreateInventorySerialHistoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory serial history data is required")
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
		return nil, fmt.Errorf("failed to create inventory serial history: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	serialHistory := &serialhistorypb.InventorySerialHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, serialHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &serialhistorypb.CreateInventorySerialHistoryResponse{
		Data: []*serialhistorypb.InventorySerialHistory{serialHistory},
	}, nil
}

// ReadInventorySerialHistory retrieves an inventory serial history record using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) ReadInventorySerialHistory(ctx context.Context, req *serialhistorypb.ReadInventorySerialHistoryRequest) (*serialhistorypb.ReadInventorySerialHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial history ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory serial history: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	serialHistory := &serialhistorypb.InventorySerialHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, serialHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &serialhistorypb.ReadInventorySerialHistoryResponse{
		Data: []*serialhistorypb.InventorySerialHistory{serialHistory},
	}, nil
}

// NOTE: UpdateInventorySerialHistory is intentionally NOT implemented.
// This is an immutable audit trail — records are never updated, only appended.
// The Unimplemented method from the embedded server will return codes.Unimplemented.

// DeleteInventorySerialHistory deletes an inventory serial history record using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) DeleteInventorySerialHistory(ctx context.Context, req *serialhistorypb.DeleteInventorySerialHistoryRequest) (*serialhistorypb.DeleteInventorySerialHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial history ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory serial history: %w", err)
	}

	return &serialhistorypb.DeleteInventorySerialHistoryResponse{
		Success: true,
	}, nil
}

// ListInventorySerialHistory lists inventory serial history records using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) ListInventorySerialHistory(ctx context.Context, req *serialhistorypb.ListInventorySerialHistoryRequest) (*serialhistorypb.ListInventorySerialHistoryResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory serial history: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var serialHistories []*serialhistorypb.InventorySerialHistory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		serialHistory := &serialhistorypb.InventorySerialHistory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, serialHistory); err != nil {
			// Log error and continue with next item
			continue
		}
		serialHistories = append(serialHistories, serialHistory)
	}

	return &serialhistorypb.ListInventorySerialHistoryResponse{
		Data: serialHistories,
	}, nil
}

// NewInventorySerialHistoryRepository creates a new PostgreSQL inventory serial history repository (old-style constructor)
func NewInventorySerialHistoryRepository(db *sql.DB, tableName string) serialhistorypb.InventorySerialHistoryDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresInventorySerialHistoryRepository(dbOps, tableName)
}
