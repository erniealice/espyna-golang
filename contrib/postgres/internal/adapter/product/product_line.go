//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ProductLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresProductLineRepository(dbOps, tableName), nil
	})
}

// PostgresProductLineRepository implements product_line CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_line_active ON product_line(active) WHERE active = true
//   - CREATE INDEX idx_product_line_product_id ON product_line(product_id)
//   - CREATE INDEX idx_product_line_line_id ON product_line(line_id)
//   - CREATE INDEX idx_product_line_sort_order ON product_line(sort_order)
//   - CREATE INDEX idx_product_line_date_created ON product_line(date_created DESC)
type PostgresProductLineRepository struct {
	productlinepb.UnimplementedProductLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresProductLineRepository creates a new PostgreSQL product line repository.
func NewPostgresProductLineRepository(dbOps interfaces.DatabaseOperation, tableName string) productlinepb.ProductLineDomainServiceServer {
	if tableName == "" {
		tableName = "product_line"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductLineRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *PostgresProductLineRepository) CreateProductLine(ctx context.Context, req *productlinepb.CreateProductLineRequest) (*productlinepb.CreateProductLineResponse, error) {
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

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productLine := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productLine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productlinepb.CreateProductLineResponse{Data: []*productlinepb.ProductLine{productLine}}, nil
}

func (r *PostgresProductLineRepository) ReadProductLine(ctx context.Context, req *productlinepb.ReadProductLineRequest) (*productlinepb.ReadProductLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product line: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productLine := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productLine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productlinepb.ReadProductLineResponse{Data: []*productlinepb.ProductLine{productLine}}, nil
}

func (r *PostgresProductLineRepository) UpdateProductLine(ctx context.Context, req *productlinepb.UpdateProductLineRequest) (*productlinepb.UpdateProductLineResponse, error) {
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

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productLine := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productLine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productlinepb.UpdateProductLineResponse{Data: []*productlinepb.ProductLine{productLine}}, nil
}

func (r *PostgresProductLineRepository) DeleteProductLine(ctx context.Context, req *productlinepb.DeleteProductLineRequest) (*productlinepb.DeleteProductLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product line: %w", err)
	}

	return &productlinepb.DeleteProductLineResponse{Success: true}, nil
}

func (r *PostgresProductLineRepository) ListProductLines(ctx context.Context, req *productlinepb.ListProductLinesRequest) (*productlinepb.ListProductLinesResponse, error) {
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

	productLines := make([]*productlinepb.ProductLine, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		productLine := &productlinepb.ProductLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productLine); err != nil {
			continue
		}
		productLines = append(productLines, productLine)
	}

	return &productlinepb.ListProductLinesResponse{Data: productLines}, nil
}

func (r *PostgresProductLineRepository) GetProductLineListPageData(ctx context.Context, req *productlinepb.GetProductLineListPageDataRequest) (*productlinepb.GetProductLineListPageDataResponse, error) {
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

	productLines := make([]*productlinepb.ProductLine, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		productLine := &productlinepb.ProductLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productLine); err != nil {
			continue
		}
		productLines = append(productLines, productLine)
	}

	return &productlinepb.GetProductLineListPageDataResponse{
		ProductLineList: productLines,
		Pagination:      listResult.Pagination,
		SearchResults:   []*commonpb.SearchResult{},
		Success:         true,
	}, nil
}

func (r *PostgresProductLineRepository) GetProductLineItemPageData(ctx context.Context, req *productlinepb.GetProductLineItemPageDataRequest) (*productlinepb.GetProductLineItemPageDataResponse, error) {
	if req == nil || req.ProductLineId == "" {
		return nil, fmt.Errorf("product line ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.ProductLineId)
	if err != nil {
		return nil, fmt.Errorf("failed to get product line item page data: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productLine := &productlinepb.ProductLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productLine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productlinepb.GetProductLineItemPageDataResponse{
		ProductLine: productLine,
		Success:     true,
	}, nil
}