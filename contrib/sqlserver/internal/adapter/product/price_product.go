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
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PriceProduct, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver price_product repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPriceProductRepository(dbOps, tableName), nil
	})
}

// SQLServerPriceProductRepository implements price_product CRUD using SQL Server.
type SQLServerPriceProductRepository struct {
	priceproductpb.UnimplementedPriceProductDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerPriceProductRepository creates a new SQL Server price_product repository.
func NewSQLServerPriceProductRepository(dbOps interfaces.DatabaseOperation, tableName string) priceproductpb.PriceProductDomainServiceServer {
	if tableName == "" {
		tableName = "price_product"
	}
	return &SQLServerPriceProductRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerPriceProductRepository) CreatePriceProduct(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) (*priceproductpb.CreatePriceProductResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price_product data is required")
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
		return nil, fmt.Errorf("failed to create price_product: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pp := &priceproductpb.PriceProduct{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &priceproductpb.CreatePriceProductResponse{Data: []*priceproductpb.PriceProduct{pp}}, nil
}

func (r *SQLServerPriceProductRepository) ReadPriceProduct(ctx context.Context, req *priceproductpb.ReadPriceProductRequest) (*priceproductpb.ReadPriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price_product ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price_product: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pp := &priceproductpb.PriceProduct{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &priceproductpb.ReadPriceProductResponse{Data: []*priceproductpb.PriceProduct{pp}}, nil
}

func (r *SQLServerPriceProductRepository) UpdatePriceProduct(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) (*priceproductpb.UpdatePriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price_product ID is required")
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
		return nil, fmt.Errorf("failed to update price_product: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pp := &priceproductpb.PriceProduct{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &priceproductpb.UpdatePriceProductResponse{Data: []*priceproductpb.PriceProduct{pp}}, nil
}

func (r *SQLServerPriceProductRepository) DeletePriceProduct(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) (*priceproductpb.DeletePriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price_product ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete price_product: %w", err)
	}
	return &priceproductpb.DeletePriceProductResponse{Success: true}, nil
}

func (r *SQLServerPriceProductRepository) ListPriceProducts(ctx context.Context, req *priceproductpb.ListPriceProductsRequest) (*priceproductpb.ListPriceProductsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list price_products: %w", err)
	}
	var pps []*priceproductpb.PriceProduct
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pp := &priceproductpb.PriceProduct{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
			continue
		}
		pps = append(pps, pp)
	}
	return &priceproductpb.ListPriceProductsResponse{Data: pps}, nil
}

func (r *SQLServerPriceProductRepository) GetPriceProductListPageData(ctx context.Context, req *priceproductpb.GetPriceProductListPageDataRequest) (*priceproductpb.GetPriceProductListPageDataResponse, error) {
	return nil, fmt.Errorf("GetPriceProductListPageData not yet implemented — Phase 2")
}

func (r *SQLServerPriceProductRepository) GetPriceProductItemPageData(ctx context.Context, req *priceproductpb.GetPriceProductItemPageDataRequest) (*priceproductpb.GetPriceProductItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetPriceProductItemPageData not yet implemented — Phase 2")
}
