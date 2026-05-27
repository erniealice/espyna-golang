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
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductVariant, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_variant repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductVariantRepository(dbOps, tableName), nil
	})
}

// SQLServerProductVariantRepository implements product_variant CRUD using SQL Server.
type SQLServerProductVariantRepository struct {
	productvariantpb.UnimplementedProductVariantDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductVariantRepository creates a new SQL Server product_variant repository.
func NewSQLServerProductVariantRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantpb.ProductVariantDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant"
	}
	return &SQLServerProductVariantRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductVariantRepository) CreateProductVariant(ctx context.Context, req *productvariantpb.CreateProductVariantRequest) (*productvariantpb.CreateProductVariantResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_variant data is required")
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
		return nil, fmt.Errorf("failed to create product_variant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pv := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantpb.CreateProductVariantResponse{Data: []*productvariantpb.ProductVariant{pv}}, nil
}

func (r *SQLServerProductVariantRepository) ReadProductVariant(ctx context.Context, req *productvariantpb.ReadProductVariantRequest) (*productvariantpb.ReadProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_variant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pv := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantpb.ReadProductVariantResponse{Data: []*productvariantpb.ProductVariant{pv}}, nil
}

func (r *SQLServerProductVariantRepository) UpdateProductVariant(ctx context.Context, req *productvariantpb.UpdateProductVariantRequest) (*productvariantpb.UpdateProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant ID is required")
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
		return nil, fmt.Errorf("failed to update product_variant: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pv := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantpb.UpdateProductVariantResponse{Data: []*productvariantpb.ProductVariant{pv}}, nil
}

func (r *SQLServerProductVariantRepository) DeleteProductVariant(ctx context.Context, req *productvariantpb.DeleteProductVariantRequest) (*productvariantpb.DeleteProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_variant: %w", err)
	}
	return &productvariantpb.DeleteProductVariantResponse{Success: true}, nil
}

func (r *SQLServerProductVariantRepository) ListProductVariants(ctx context.Context, req *productvariantpb.ListProductVariantsRequest) (*productvariantpb.ListProductVariantsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_variants: %w", err)
	}
	var pvs []*productvariantpb.ProductVariant
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pv := &productvariantpb.ProductVariant{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pv); err != nil {
			continue
		}
		pvs = append(pvs, pv)
	}
	return &productvariantpb.ListProductVariantsResponse{Data: pvs}, nil
}

func (r *SQLServerProductVariantRepository) GetProductVariantListPageData(ctx context.Context, req *productvariantpb.GetProductVariantListPageDataRequest) (*productvariantpb.GetProductVariantListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductVariantListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductVariantRepository) GetProductVariantItemPageData(ctx context.Context, req *productvariantpb.GetProductVariantItemPageDataRequest) (*productvariantpb.GetProductVariantItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductVariantItemPageData not yet implemented — Phase 2")
}
