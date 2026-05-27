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
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductOptionValue, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_option_value repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductOptionValueRepository(dbOps, tableName), nil
	})
}

// SQLServerProductOptionValueRepository implements product_option_value CRUD using SQL Server.
type SQLServerProductOptionValueRepository struct {
	productoptionvaluepb.UnimplementedProductOptionValueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductOptionValueRepository creates a new SQL Server product_option_value repository.
func NewSQLServerProductOptionValueRepository(dbOps interfaces.DatabaseOperation, tableName string) productoptionvaluepb.ProductOptionValueDomainServiceServer {
	if tableName == "" {
		tableName = "product_option_value"
	}
	return &SQLServerProductOptionValueRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductOptionValueRepository) CreateProductOptionValue(ctx context.Context, req *productoptionvaluepb.CreateProductOptionValueRequest) (*productoptionvaluepb.CreateProductOptionValueResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_option_value data is required")
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
		return nil, fmt.Errorf("failed to create product_option_value: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pov := &productoptionvaluepb.ProductOptionValue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pov); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productoptionvaluepb.CreateProductOptionValueResponse{Data: []*productoptionvaluepb.ProductOptionValue{pov}}, nil
}

func (r *SQLServerProductOptionValueRepository) ReadProductOptionValue(ctx context.Context, req *productoptionvaluepb.ReadProductOptionValueRequest) (*productoptionvaluepb.ReadProductOptionValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_option_value ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_option_value: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pov := &productoptionvaluepb.ProductOptionValue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pov); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productoptionvaluepb.ReadProductOptionValueResponse{Data: []*productoptionvaluepb.ProductOptionValue{pov}}, nil
}

func (r *SQLServerProductOptionValueRepository) UpdateProductOptionValue(ctx context.Context, req *productoptionvaluepb.UpdateProductOptionValueRequest) (*productoptionvaluepb.UpdateProductOptionValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_option_value ID is required")
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
		return nil, fmt.Errorf("failed to update product_option_value: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pov := &productoptionvaluepb.ProductOptionValue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pov); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productoptionvaluepb.UpdateProductOptionValueResponse{Data: []*productoptionvaluepb.ProductOptionValue{pov}}, nil
}

func (r *SQLServerProductOptionValueRepository) DeleteProductOptionValue(ctx context.Context, req *productoptionvaluepb.DeleteProductOptionValueRequest) (*productoptionvaluepb.DeleteProductOptionValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_option_value ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_option_value: %w", err)
	}
	return &productoptionvaluepb.DeleteProductOptionValueResponse{Success: true}, nil
}

func (r *SQLServerProductOptionValueRepository) ListProductOptionValues(ctx context.Context, req *productoptionvaluepb.ListProductOptionValuesRequest) (*productoptionvaluepb.ListProductOptionValuesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_option_values: %w", err)
	}
	var povs []*productoptionvaluepb.ProductOptionValue
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pov := &productoptionvaluepb.ProductOptionValue{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pov); err != nil {
			continue
		}
		povs = append(povs, pov)
	}
	return &productoptionvaluepb.ListProductOptionValuesResponse{Data: povs}, nil
}

func (r *SQLServerProductOptionValueRepository) GetProductOptionValueListPageData(ctx context.Context, req *productoptionvaluepb.GetProductOptionValueListPageDataRequest) (*productoptionvaluepb.GetProductOptionValueListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductOptionValueListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductOptionValueRepository) GetProductOptionValueItemPageData(ctx context.Context, req *productoptionvaluepb.GetProductOptionValueItemPageDataRequest) (*productoptionvaluepb.GetProductOptionValueItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductOptionValueItemPageData not yet implemented — Phase 2")
}
