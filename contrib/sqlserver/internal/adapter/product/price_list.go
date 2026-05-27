//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PriceList, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver price_list repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPriceListRepository(dbOps, tableName), nil
	})
}

// SQLServerPriceListRepository implements price_list CRUD using SQL Server.
type SQLServerPriceListRepository struct {
	pricelistpb.UnimplementedPriceListDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerPriceListRepository creates a new SQL Server price_list repository.
func NewSQLServerPriceListRepository(dbOps interfaces.DatabaseOperation, tableName string) pricelistpb.PriceListDomainServiceServer {
	if tableName == "" {
		tableName = "price_list"
	}
	return &SQLServerPriceListRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerPriceListRepository) CreatePriceList(ctx context.Context, req *pricelistpb.CreatePriceListRequest) (*pricelistpb.CreatePriceListResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price_list data is required")
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
		return nil, fmt.Errorf("failed to create price_list: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pl := &pricelistpb.PriceList{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pricelistpb.CreatePriceListResponse{Data: []*pricelistpb.PriceList{pl}}, nil
}

func (r *SQLServerPriceListRepository) ReadPriceList(ctx context.Context, req *pricelistpb.ReadPriceListRequest) (*pricelistpb.ReadPriceListResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price_list ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price_list: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pl := &pricelistpb.PriceList{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pricelistpb.ReadPriceListResponse{Data: []*pricelistpb.PriceList{pl}}, nil
}

func (r *SQLServerPriceListRepository) UpdatePriceList(ctx context.Context, req *pricelistpb.UpdatePriceListRequest) (*pricelistpb.UpdatePriceListResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price_list ID is required")
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
		return nil, fmt.Errorf("failed to update price_list: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pl := &pricelistpb.PriceList{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pricelistpb.UpdatePriceListResponse{Data: []*pricelistpb.PriceList{pl}}, nil
}

func (r *SQLServerPriceListRepository) DeletePriceList(ctx context.Context, req *pricelistpb.DeletePriceListRequest) (*pricelistpb.DeletePriceListResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price_list ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete price_list: %w", err)
	}
	return &pricelistpb.DeletePriceListResponse{Success: true}, nil
}

func (r *SQLServerPriceListRepository) ListPriceLists(ctx context.Context, req *pricelistpb.ListPriceListsRequest) (*pricelistpb.ListPriceListsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list price_lists: %w", err)
	}
	var pls []*pricelistpb.PriceList
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pl := &pricelistpb.PriceList{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
			continue
		}
		pls = append(pls, pl)
	}
	return &pricelistpb.ListPriceListsResponse{Data: pls}, nil
}

func (r *SQLServerPriceListRepository) GetPriceListListPageData(ctx context.Context, req *pricelistpb.GetPriceListListPageDataRequest) (*pricelistpb.GetPriceListListPageDataResponse, error) {
	return nil, fmt.Errorf("GetPriceListListPageData not yet implemented — Phase 2")
}

func (r *SQLServerPriceListRepository) GetPriceListItemPageData(ctx context.Context, req *pricelistpb.GetPriceListItemPageDataRequest) (*pricelistpb.GetPriceListItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetPriceListItemPageData not yet implemented — Phase 2")
}
