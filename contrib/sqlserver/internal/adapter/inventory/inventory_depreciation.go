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
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.InventoryDepreciation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver inventory_depreciation repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInventoryDepreciationRepository(dbOps, tableName), nil
	})
}

// SQLServerInventoryDepreciationRepository implements inventory_depreciation CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerInventoryDepreciationRepository struct {
	inventorydepreciationpb.UnimplementedInventoryDepreciationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerInventoryDepreciationRepository creates a new SQL Server inventory_depreciation repository.
func NewSQLServerInventoryDepreciationRepository(dbOps interfaces.DatabaseOperation, tableName string) inventorydepreciationpb.InventoryDepreciationDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_depreciation"
	}
	return &SQLServerInventoryDepreciationRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerInventoryDepreciationRepository) CreateInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.CreateInventoryDepreciationRequest) (*inventorydepreciationpb.CreateInventoryDepreciationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory_depreciation data is required")
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
		return nil, fmt.Errorf("failed to create inventory_depreciation: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventorydepreciationpb.InventoryDepreciation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventorydepreciationpb.CreateInventoryDepreciationResponse{Data: []*inventorydepreciationpb.InventoryDepreciation{obj}}, nil
}

func (r *SQLServerInventoryDepreciationRepository) ReadInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.ReadInventoryDepreciationRequest) (*inventorydepreciationpb.ReadInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_depreciation ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory_depreciation: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventorydepreciationpb.InventoryDepreciation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventorydepreciationpb.ReadInventoryDepreciationResponse{Data: []*inventorydepreciationpb.InventoryDepreciation{obj}}, nil
}

func (r *SQLServerInventoryDepreciationRepository) UpdateInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.UpdateInventoryDepreciationRequest) (*inventorydepreciationpb.UpdateInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_depreciation ID is required")
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
		return nil, fmt.Errorf("failed to update inventory_depreciation: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &inventorydepreciationpb.InventoryDepreciation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventorydepreciationpb.UpdateInventoryDepreciationResponse{Data: []*inventorydepreciationpb.InventoryDepreciation{obj}}, nil
}

func (r *SQLServerInventoryDepreciationRepository) DeleteInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.DeleteInventoryDepreciationRequest) (*inventorydepreciationpb.DeleteInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory_depreciation ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete inventory_depreciation: %w", err)
	}
	return &inventorydepreciationpb.DeleteInventoryDepreciationResponse{Success: true}, nil
}

func (r *SQLServerInventoryDepreciationRepository) ListInventoryDepreciations(ctx context.Context, req *inventorydepreciationpb.ListInventoryDepreciationsRequest) (*inventorydepreciationpb.ListInventoryDepreciationsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory_depreciations: %w", err)
	}
	var items []*inventorydepreciationpb.InventoryDepreciation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &inventorydepreciationpb.InventoryDepreciation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &inventorydepreciationpb.ListInventoryDepreciationsResponse{Data: items}, nil
}

func (r *SQLServerInventoryDepreciationRepository) GetInventoryDepreciationListPageData(ctx context.Context, req *inventorydepreciationpb.GetInventoryDepreciationListPageDataRequest) (*inventorydepreciationpb.GetInventoryDepreciationListPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventoryDepreciationListPageData not yet implemented")
}

func (r *SQLServerInventoryDepreciationRepository) GetInventoryDepreciationItemPageData(ctx context.Context, req *inventorydepreciationpb.GetInventoryDepreciationItemPageDataRequest) (*inventorydepreciationpb.GetInventoryDepreciationItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventoryDepreciationItemPageData not yet implemented")
}
