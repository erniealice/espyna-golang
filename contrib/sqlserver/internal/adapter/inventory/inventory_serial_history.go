//go:build sqlserver

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.InventorySerialHistory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver inventory_serial_history repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInventorySerialHistoryRepository(dbOps, tableName), nil
	})
}

// SQLServerInventorySerialHistoryRepository implements inventory_serial_history operations using SQL Server.
// This is an IMMUTABLE audit trail — records are never updated, only appended.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerInventorySerialHistoryRepository struct {
	serialhistorypb.UnimplementedInventorySerialHistoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerInventorySerialHistoryRepository creates a new SQL Server inventory serial history repository.
func NewSQLServerInventorySerialHistoryRepository(dbOps interfaces.DatabaseOperation, tableName string) serialhistorypb.InventorySerialHistoryDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_serial_history"
	}
	return &SQLServerInventorySerialHistoryRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerInventorySerialHistoryRepository) CreateInventorySerialHistory(ctx context.Context, req *serialhistorypb.CreateInventorySerialHistoryRequest) (*serialhistorypb.CreateInventorySerialHistoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory serial history data is required")
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
		return nil, fmt.Errorf("failed to create inventory serial history: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &serialhistorypb.InventorySerialHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &serialhistorypb.CreateInventorySerialHistoryResponse{Data: []*serialhistorypb.InventorySerialHistory{obj}}, nil
}

func (r *SQLServerInventorySerialHistoryRepository) ReadInventorySerialHistory(ctx context.Context, req *serialhistorypb.ReadInventorySerialHistoryRequest) (*serialhistorypb.ReadInventorySerialHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial history ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory serial history: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &serialhistorypb.InventorySerialHistory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &serialhistorypb.ReadInventorySerialHistoryResponse{Data: []*serialhistorypb.InventorySerialHistory{obj}}, nil
}

func (r *SQLServerInventorySerialHistoryRepository) ListInventorySerialHistory(ctx context.Context, req *serialhistorypb.ListInventorySerialHistoryRequest) (*serialhistorypb.ListInventorySerialHistoryResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory serial history: %w", err)
	}
	var items []*serialhistorypb.InventorySerialHistory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &serialhistorypb.InventorySerialHistory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &serialhistorypb.ListInventorySerialHistoryResponse{Data: items}, nil
}

func (r *SQLServerInventorySerialHistoryRepository) GetInventorySerialHistoryListPageData(ctx context.Context, req *serialhistorypb.GetInventorySerialHistoryListPageDataRequest) (*serialhistorypb.GetInventorySerialHistoryListPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventorySerialHistoryListPageData not yet implemented")
}

func (r *SQLServerInventorySerialHistoryRepository) GetInventorySerialHistoryItemPageData(ctx context.Context, req *serialhistorypb.GetInventorySerialHistoryItemPageDataRequest) (*serialhistorypb.GetInventorySerialHistoryItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventorySerialHistoryItemPageData not yet implemented")
}
