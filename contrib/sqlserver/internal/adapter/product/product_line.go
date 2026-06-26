//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductLineRepository(dbOps, tableName), nil
	})
}

// SQLServerProductLineRepository implements product_line CRUD using SQL Server.
type SQLServerProductLineRepository struct {
	productlinepb.UnimplementedProductLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductLineRepository creates a new SQL Server product_line repository.
func NewSQLServerProductLineRepository(dbOps interfaces.DatabaseOperation, tableName string) productlinepb.ProductLineDomainServiceServer {
	if tableName == "" {
		tableName = "product_line"
	}
	return &SQLServerProductLineRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductLineRepository) CreateProductLine(ctx context.Context, req *productlinepb.CreateProductLineRequest) (*productlinepb.CreateProductLineResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("product line data is required")
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
		return nil, fmt.Errorf("failed to create product line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pl := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productlinepb.CreateProductLineResponse{Data: []*productlinepb.ProductLine{pl}}, nil
}

func (r *SQLServerProductLineRepository) ReadProductLine(ctx context.Context, req *productlinepb.ReadProductLineRequest) (*productlinepb.ReadProductLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pl := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productlinepb.ReadProductLineResponse{Data: []*productlinepb.ProductLine{pl}}, nil
}

func (r *SQLServerProductLineRepository) UpdateProductLine(ctx context.Context, req *productlinepb.UpdateProductLineRequest) (*productlinepb.UpdateProductLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// Always include active flag — proto3 omits bool=false during JSON marshal,
	// which would silently skip deactivation via the form toggle.
	data["active"] = req.Data.GetActive()
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pl := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productlinepb.UpdateProductLineResponse{Data: []*productlinepb.ProductLine{pl}}, nil
}

func (r *SQLServerProductLineRepository) DeleteProductLine(ctx context.Context, req *productlinepb.DeleteProductLineRequest) (*productlinepb.DeleteProductLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product line: %w", err)
	}
	return &productlinepb.DeleteProductLineResponse{Success: true}, nil
}

func (r *SQLServerProductLineRepository) ListProductLines(ctx context.Context, req *productlinepb.ListProductLinesRequest) (*productlinepb.ListProductLinesResponse, error) {
	var params *interfaces.ListParams
	if req != nil {
		params = &interfaces.ListParams{
			Search:     req.Search,
			Filters:    req.Filters,
			Sort:       req.Sort,
			Pagination: req.Pagination,
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product lines: %w", err)
	}
	pls := make([]*productlinepb.ProductLine, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pl := &productlinepb.ProductLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
			continue
		}
		pls = append(pls, pl)
	}
	return &productlinepb.ListProductLinesResponse{Data: pls}, nil
}

func (r *SQLServerProductLineRepository) GetProductLineListPageData(ctx context.Context, req *productlinepb.GetProductLineListPageDataRequest) (*productlinepb.GetProductLineListPageDataResponse, error) {
	var params *interfaces.ListParams
	if req != nil {
		params = &interfaces.ListParams{
			Search:     req.Search,
			Filters:    req.Filters,
			Sort:       req.Sort,
			Pagination: req.Pagination,
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get product line list page data: %w", err)
	}
	pls := make([]*productlinepb.ProductLine, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pl := &productlinepb.ProductLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
			continue
		}
		pls = append(pls, pl)
	}
	return &productlinepb.GetProductLineListPageDataResponse{
		ProductLineList: pls,
		Pagination:      listResult.Pagination,
		SearchResults:   []*commonpb.SearchResult{},
		Success:         true,
	}, nil
}

func (r *SQLServerProductLineRepository) GetProductLineItemPageData(ctx context.Context, req *productlinepb.GetProductLineItemPageDataRequest) (*productlinepb.GetProductLineItemPageDataResponse, error) {
	if req == nil || req.ProductLineId == "" {
		return nil, fmt.Errorf("product line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ProductLineId)
	if err != nil {
		return nil, fmt.Errorf("failed to get product line item page data: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pl := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productlinepb.GetProductLineItemPageDataResponse{
		ProductLine: pl,
		Success:     true,
	}, nil
}
