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
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductVariantOption, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_variant_option repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductVariantOptionRepository(dbOps, tableName), nil
	})
}

// SQLServerProductVariantOptionRepository implements product_variant_option CRUD using SQL Server.
type SQLServerProductVariantOptionRepository struct {
	productvariantoptionpb.UnimplementedProductVariantOptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductVariantOptionRepository creates a new SQL Server product_variant_option repository.
func NewSQLServerProductVariantOptionRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantoptionpb.ProductVariantOptionDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant_option"
	}
	return &SQLServerProductVariantOptionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductVariantOptionRepository) CreateProductVariantOption(ctx context.Context, req *productvariantoptionpb.CreateProductVariantOptionRequest) (*productvariantoptionpb.CreateProductVariantOptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_variant_option data is required")
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
		return nil, fmt.Errorf("failed to create product_variant_option: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pvo := &productvariantoptionpb.ProductVariantOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantoptionpb.CreateProductVariantOptionResponse{Data: []*productvariantoptionpb.ProductVariantOption{pvo}}, nil
}

func (r *SQLServerProductVariantOptionRepository) ReadProductVariantOption(ctx context.Context, req *productvariantoptionpb.ReadProductVariantOptionRequest) (*productvariantoptionpb.ReadProductVariantOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant_option ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_variant_option: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pvo := &productvariantoptionpb.ProductVariantOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantoptionpb.ReadProductVariantOptionResponse{Data: []*productvariantoptionpb.ProductVariantOption{pvo}}, nil
}

func (r *SQLServerProductVariantOptionRepository) UpdateProductVariantOption(ctx context.Context, req *productvariantoptionpb.UpdateProductVariantOptionRequest) (*productvariantoptionpb.UpdateProductVariantOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant_option ID is required")
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
		return nil, fmt.Errorf("failed to update product_variant_option: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pvo := &productvariantoptionpb.ProductVariantOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantoptionpb.UpdateProductVariantOptionResponse{Data: []*productvariantoptionpb.ProductVariantOption{pvo}}, nil
}

func (r *SQLServerProductVariantOptionRepository) DeleteProductVariantOption(ctx context.Context, req *productvariantoptionpb.DeleteProductVariantOptionRequest) (*productvariantoptionpb.DeleteProductVariantOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant_option ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_variant_option: %w", err)
	}
	return &productvariantoptionpb.DeleteProductVariantOptionResponse{Success: true}, nil
}

func (r *SQLServerProductVariantOptionRepository) ListProductVariantOptions(ctx context.Context, req *productvariantoptionpb.ListProductVariantOptionsRequest) (*productvariantoptionpb.ListProductVariantOptionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_variant_options: %w", err)
	}
	var pvos []*productvariantoptionpb.ProductVariantOption
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pvo := &productvariantoptionpb.ProductVariantOption{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvo); err != nil {
			continue
		}
		pvos = append(pvos, pvo)
	}
	return &productvariantoptionpb.ListProductVariantOptionsResponse{Data: pvos}, nil
}

func (r *SQLServerProductVariantOptionRepository) GetProductVariantOptionListPageData(ctx context.Context, req *productvariantoptionpb.GetProductVariantOptionListPageDataRequest) (*productvariantoptionpb.GetProductVariantOptionListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductVariantOptionListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductVariantOptionRepository) GetProductVariantOptionItemPageData(ctx context.Context, req *productvariantoptionpb.GetProductVariantOptionItemPageDataRequest) (*productvariantoptionpb.GetProductVariantOptionItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductVariantOptionItemPageData not yet implemented — Phase 2")
}
