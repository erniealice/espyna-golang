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
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductOption, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_option repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductOptionRepository(dbOps, tableName), nil
	})
}

// SQLServerProductOptionRepository implements product_option CRUD using SQL Server.
type SQLServerProductOptionRepository struct {
	productoptionpb.UnimplementedProductOptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductOptionRepository creates a new SQL Server product_option repository.
func NewSQLServerProductOptionRepository(dbOps interfaces.DatabaseOperation, tableName string) productoptionpb.ProductOptionDomainServiceServer {
	if tableName == "" {
		tableName = "product_option"
	}
	return &SQLServerProductOptionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductOptionRepository) CreateProductOption(ctx context.Context, req *productoptionpb.CreateProductOptionRequest) (*productoptionpb.CreateProductOptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_option data is required")
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
		return nil, fmt.Errorf("failed to create product_option: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	po := &productoptionpb.ProductOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productoptionpb.CreateProductOptionResponse{Data: []*productoptionpb.ProductOption{po}}, nil
}

func (r *SQLServerProductOptionRepository) ReadProductOption(ctx context.Context, req *productoptionpb.ReadProductOptionRequest) (*productoptionpb.ReadProductOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_option ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_option: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	po := &productoptionpb.ProductOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productoptionpb.ReadProductOptionResponse{Data: []*productoptionpb.ProductOption{po}}, nil
}

func (r *SQLServerProductOptionRepository) UpdateProductOption(ctx context.Context, req *productoptionpb.UpdateProductOptionRequest) (*productoptionpb.UpdateProductOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_option ID is required")
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
		return nil, fmt.Errorf("failed to update product_option: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	po := &productoptionpb.ProductOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productoptionpb.UpdateProductOptionResponse{Data: []*productoptionpb.ProductOption{po}}, nil
}

func (r *SQLServerProductOptionRepository) DeleteProductOption(ctx context.Context, req *productoptionpb.DeleteProductOptionRequest) (*productoptionpb.DeleteProductOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_option ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_option: %w", err)
	}
	return &productoptionpb.DeleteProductOptionResponse{Success: true}, nil
}

func (r *SQLServerProductOptionRepository) ListProductOptions(ctx context.Context, req *productoptionpb.ListProductOptionsRequest) (*productoptionpb.ListProductOptionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_options: %w", err)
	}
	var pos []*productoptionpb.ProductOption
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		po := &productoptionpb.ProductOption{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
			continue
		}
		pos = append(pos, po)
	}
	return &productoptionpb.ListProductOptionsResponse{Data: pos}, nil
}

func (r *SQLServerProductOptionRepository) GetProductOptionListPageData(ctx context.Context, req *productoptionpb.GetProductOptionListPageDataRequest) (*productoptionpb.GetProductOptionListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductOptionListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductOptionRepository) GetProductOptionItemPageData(ctx context.Context, req *productoptionpb.GetProductOptionItemPageDataRequest) (*productoptionpb.GetProductOptionItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductOptionItemPageData not yet implemented — Phase 2")
}
