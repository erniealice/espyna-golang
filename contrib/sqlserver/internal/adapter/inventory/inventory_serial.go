//go:build sqlserver

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.InventorySerial, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver inventory_serial repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInventorySerialRepository(dbOps, tableName), nil
	})
}

// SQLServerInventorySerialRepository implements inventory_serial CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerInventorySerialRepository struct {
	inventoryserialpb.UnimplementedInventorySerialDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerInventorySerialRepository creates a new SQL Server inventory_serial repository.
func NewSQLServerInventorySerialRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryserialpb.InventorySerialDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_serial"
	}
	return &SQLServerInventorySerialRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerInventorySerialRepository) CreateInventorySerial(ctx context.Context, req *inventoryserialpb.CreateInventorySerialRequest) (*inventoryserialpb.CreateInventorySerialResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory_serial data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory_serial: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventoryserialpb.InventorySerial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventoryserialpb.CreateInventorySerialResponse{Data: []*inventoryserialpb.InventorySerial{obj}}, nil
}

func (r *SQLServerInventorySerialRepository) ReadInventorySerial(ctx context.Context, req *inventoryserialpb.ReadInventorySerialRequest) (*inventoryserialpb.ReadInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_serial ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory_serial: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventoryserialpb.InventorySerial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventoryserialpb.ReadInventorySerialResponse{Data: []*inventoryserialpb.InventorySerial{obj}}, nil
}

func (r *SQLServerInventorySerialRepository) UpdateInventorySerial(ctx context.Context, req *inventoryserialpb.UpdateInventorySerialRequest) (*inventoryserialpb.UpdateInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_serial ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory_serial: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventoryserialpb.InventorySerial{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventoryserialpb.UpdateInventorySerialResponse{Data: []*inventoryserialpb.InventorySerial{obj}}, nil
}

func (r *SQLServerInventorySerialRepository) DeleteInventorySerial(ctx context.Context, req *inventoryserialpb.DeleteInventorySerialRequest) (*inventoryserialpb.DeleteInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_serial ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete inventory_serial: %w", err)
	}
	return &inventoryserialpb.DeleteInventorySerialResponse{Success: true}, nil
}

func (r *SQLServerInventorySerialRepository) ListInventorySerials(ctx context.Context, req *inventoryserialpb.ListInventorySerialsRequest) (*inventoryserialpb.ListInventorySerialsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory_serials: %w", err)
	}
	var items []*inventoryserialpb.InventorySerial
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &inventoryserialpb.InventorySerial{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &inventoryserialpb.ListInventorySerialsResponse{Data: items}, nil
}

func (r *SQLServerInventorySerialRepository) GetInventorySerialListPageData(ctx context.Context, req *inventoryserialpb.GetInventorySerialListPageDataRequest) (*inventoryserialpb.GetInventorySerialListPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventorySerialListPageData not yet implemented")
}

func (r *SQLServerInventorySerialRepository) GetInventorySerialItemPageData(ctx context.Context, req *inventoryserialpb.GetInventorySerialItemPageDataRequest) (*inventoryserialpb.GetInventorySerialItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventorySerialItemPageData not yet implemented")
}
