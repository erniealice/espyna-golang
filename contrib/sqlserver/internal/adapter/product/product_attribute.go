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
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductAttributeRepository(dbOps, tableName), nil
	})
}

// SQLServerProductAttributeRepository implements product_attribute CRUD using SQL Server.
type SQLServerProductAttributeRepository struct {
	productattributepb.UnimplementedProductAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductAttributeRepository creates a new SQL Server product_attribute repository.
func NewSQLServerProductAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) productattributepb.ProductAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "product_attribute"
	}
	return &SQLServerProductAttributeRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductAttributeRepository) CreateProductAttribute(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) (*productattributepb.CreateProductAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_attribute data is required")
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
		return nil, fmt.Errorf("failed to create product_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pa := &productattributepb.ProductAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productattributepb.CreateProductAttributeResponse{Data: []*productattributepb.ProductAttribute{pa}}, nil
}

func (r *SQLServerProductAttributeRepository) ReadProductAttribute(ctx context.Context, req *productattributepb.ReadProductAttributeRequest) (*productattributepb.ReadProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pa := &productattributepb.ProductAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productattributepb.ReadProductAttributeResponse{Data: []*productattributepb.ProductAttribute{pa}}, nil
}

func (r *SQLServerProductAttributeRepository) UpdateProductAttribute(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) (*productattributepb.UpdateProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_attribute ID is required")
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
		return nil, fmt.Errorf("failed to update product_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pa := &productattributepb.ProductAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productattributepb.UpdateProductAttributeResponse{Data: []*productattributepb.ProductAttribute{pa}}, nil
}

func (r *SQLServerProductAttributeRepository) DeleteProductAttribute(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) (*productattributepb.DeleteProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_attribute ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_attribute: %w", err)
	}
	return &productattributepb.DeleteProductAttributeResponse{Success: true}, nil
}

func (r *SQLServerProductAttributeRepository) ListProductAttributes(ctx context.Context, req *productattributepb.ListProductAttributesRequest) (*productattributepb.ListProductAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_attributes: %w", err)
	}
	var pas []*productattributepb.ProductAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pa := &productattributepb.ProductAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
			continue
		}
		pas = append(pas, pa)
	}
	return &productattributepb.ListProductAttributesResponse{Data: pas}, nil
}

func (r *SQLServerProductAttributeRepository) GetProductAttributeListPageData(ctx context.Context, req *productattributepb.GetProductAttributeListPageDataRequest) (*productattributepb.GetProductAttributeListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductAttributeListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductAttributeRepository) GetProductAttributeItemPageData(ctx context.Context, req *productattributepb.GetProductAttributeItemPageDataRequest) (*productattributepb.GetProductAttributeItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductAttributeItemPageData not yet implemented — Phase 2")
}
