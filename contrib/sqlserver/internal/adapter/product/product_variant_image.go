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
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductVariantImage, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_variant_image repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductVariantImageRepository(dbOps, tableName), nil
	})
}

// SQLServerProductVariantImageRepository implements product_variant_image CRUD using SQL Server.
type SQLServerProductVariantImageRepository struct {
	productvariantimagepb.UnimplementedProductVariantImageDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductVariantImageRepository creates a new SQL Server product_variant_image repository.
func NewSQLServerProductVariantImageRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantimagepb.ProductVariantImageDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant_image"
	}
	return &SQLServerProductVariantImageRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductVariantImageRepository) CreateProductVariantImage(ctx context.Context, req *productvariantimagepb.CreateProductVariantImageRequest) (*productvariantimagepb.CreateProductVariantImageResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_variant_image data is required")
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
		return nil, fmt.Errorf("failed to create product_variant_image: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pvi := &productvariantimagepb.ProductVariantImage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvi); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantimagepb.CreateProductVariantImageResponse{Data: []*productvariantimagepb.ProductVariantImage{pvi}}, nil
}

func (r *SQLServerProductVariantImageRepository) ReadProductVariantImage(ctx context.Context, req *productvariantimagepb.ReadProductVariantImageRequest) (*productvariantimagepb.ReadProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant_image ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_variant_image: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pvi := &productvariantimagepb.ProductVariantImage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvi); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantimagepb.ReadProductVariantImageResponse{Data: []*productvariantimagepb.ProductVariantImage{pvi}}, nil
}

func (r *SQLServerProductVariantImageRepository) UpdateProductVariantImage(ctx context.Context, req *productvariantimagepb.UpdateProductVariantImageRequest) (*productvariantimagepb.UpdateProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant_image ID is required")
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
		return nil, fmt.Errorf("failed to update product_variant_image: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pvi := &productvariantimagepb.ProductVariantImage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvi); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productvariantimagepb.UpdateProductVariantImageResponse{Data: []*productvariantimagepb.ProductVariantImage{pvi}}, nil
}

func (r *SQLServerProductVariantImageRepository) DeleteProductVariantImage(ctx context.Context, req *productvariantimagepb.DeleteProductVariantImageRequest) (*productvariantimagepb.DeleteProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_variant_image ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_variant_image: %w", err)
	}
	return &productvariantimagepb.DeleteProductVariantImageResponse{Success: true}, nil
}

func (r *SQLServerProductVariantImageRepository) ListProductVariantImages(ctx context.Context, req *productvariantimagepb.ListProductVariantImagesRequest) (*productvariantimagepb.ListProductVariantImagesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_variant_images: %w", err)
	}
	var pvis []*productvariantimagepb.ProductVariantImage
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pvi := &productvariantimagepb.ProductVariantImage{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pvi); err != nil {
			continue
		}
		pvis = append(pvis, pvi)
	}
	return &productvariantimagepb.ListProductVariantImagesResponse{Data: pvis}, nil
}

func (r *SQLServerProductVariantImageRepository) GetProductVariantImageListPageData(ctx context.Context, req *productvariantimagepb.GetProductVariantImageListPageDataRequest) (*productvariantimagepb.GetProductVariantImageListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductVariantImageListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductVariantImageRepository) GetProductVariantImageItemPageData(ctx context.Context, req *productvariantimagepb.GetProductVariantImageItemPageDataRequest) (*productvariantimagepb.GetProductVariantImageItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductVariantImageItemPageData not yet implemented — Phase 2")
}
