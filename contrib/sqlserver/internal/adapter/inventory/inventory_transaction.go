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
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.InventoryTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver inventory_transaction repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInventoryTransactionRepository(dbOps, tableName), nil
	})
}

// SQLServerInventoryTransactionRepository implements inventory_transaction CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerInventoryTransactionRepository struct {
	inventorytransactionpb.UnimplementedInventoryTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerInventoryTransactionRepository creates a new SQL Server inventory_transaction repository.
func NewSQLServerInventoryTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) inventorytransactionpb.InventoryTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_transaction"
	}
	return &SQLServerInventoryTransactionRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerInventoryTransactionRepository) CreateInventoryTransaction(ctx context.Context, req *inventorytransactionpb.CreateInventoryTransactionRequest) (*inventorytransactionpb.CreateInventoryTransactionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory_transaction data is required")
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
		return nil, fmt.Errorf("failed to create inventory_transaction: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventorytransactionpb.InventoryTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventorytransactionpb.CreateInventoryTransactionResponse{Data: []*inventorytransactionpb.InventoryTransaction{obj}}, nil
}

func (r *SQLServerInventoryTransactionRepository) ReadInventoryTransaction(ctx context.Context, req *inventorytransactionpb.ReadInventoryTransactionRequest) (*inventorytransactionpb.ReadInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_transaction ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory_transaction: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventorytransactionpb.InventoryTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventorytransactionpb.ReadInventoryTransactionResponse{Data: []*inventorytransactionpb.InventoryTransaction{obj}}, nil
}

func (r *SQLServerInventoryTransactionRepository) UpdateInventoryTransaction(ctx context.Context, req *inventorytransactionpb.UpdateInventoryTransactionRequest) (*inventorytransactionpb.UpdateInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_transaction ID is required")
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
		return nil, fmt.Errorf("failed to update inventory_transaction: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventorytransactionpb.InventoryTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventorytransactionpb.UpdateInventoryTransactionResponse{Data: []*inventorytransactionpb.InventoryTransaction{obj}}, nil
}

func (r *SQLServerInventoryTransactionRepository) DeleteInventoryTransaction(ctx context.Context, req *inventorytransactionpb.DeleteInventoryTransactionRequest) (*inventorytransactionpb.DeleteInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_transaction ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete inventory_transaction: %w", err)
	}
	return &inventorytransactionpb.DeleteInventoryTransactionResponse{Success: true}, nil
}

func (r *SQLServerInventoryTransactionRepository) ListInventoryTransactions(ctx context.Context, req *inventorytransactionpb.ListInventoryTransactionsRequest) (*inventorytransactionpb.ListInventoryTransactionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory_transactions: %w", err)
	}
	var items []*inventorytransactionpb.InventoryTransaction
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &inventorytransactionpb.InventoryTransaction{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &inventorytransactionpb.ListInventoryTransactionsResponse{Data: items}, nil
}

func (r *SQLServerInventoryTransactionRepository) GetInventoryTransactionListPageData(ctx context.Context, req *inventorytransactionpb.GetInventoryTransactionListPageDataRequest) (*inventorytransactionpb.GetInventoryTransactionListPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventoryTransactionListPageData not yet implemented")
}

func (r *SQLServerInventoryTransactionRepository) GetInventoryTransactionItemPageData(ctx context.Context, req *inventorytransactionpb.GetInventoryTransactionItemPageDataRequest) (*inventorytransactionpb.GetInventoryTransactionItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventoryTransactionItemPageData not yet implemented")
}
