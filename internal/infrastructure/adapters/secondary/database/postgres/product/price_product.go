//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "price_product", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres price_product repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPriceProductRepository(dbOps, tableName), nil
	})
}

// PostgresPriceProductRepository implements price_product CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_price_product_active ON price_product(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_price_product_product_id ON price_product(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_price_product_currency ON price_product(currency) - Filter by currency
//   - CREATE INDEX idx_price_product_amount ON price_product(amount) - Sort/filter by amount
//   - CREATE INDEX idx_price_product_date_created ON price_product(date_created DESC) - Default sorting
type PostgresPriceProductRepository struct {
	priceproductpb.UnimplementedPriceProductDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresPriceProductRepository creates a new PostgreSQL price product repository
func NewPostgresPriceProductRepository(dbOps interfaces.DatabaseOperation, tableName string) priceproductpb.PriceProductDomainServiceServer {
	if tableName == "" {
		tableName = "price_product" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPriceProductRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePriceProduct creates a new price product using common PostgreSQL operations
func (r *PostgresPriceProductRepository) CreatePriceProduct(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) (*priceproductpb.CreatePriceProductResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price product data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price product: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceProduct := &priceproductpb.PriceProduct{}
	if err := protojson.Unmarshal(resultJSON, priceProduct); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceproductpb.CreatePriceProductResponse{
		Data: []*priceproductpb.PriceProduct{priceProduct},
	}, nil
}

// ReadPriceProduct retrieves a price product using common PostgreSQL operations
func (r *PostgresPriceProductRepository) ReadPriceProduct(ctx context.Context, req *priceproductpb.ReadPriceProductRequest) (*priceproductpb.ReadPriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price product: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceProduct := &priceproductpb.PriceProduct{}
	if err := protojson.Unmarshal(resultJSON, priceProduct); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceproductpb.ReadPriceProductResponse{
		Data: []*priceproductpb.PriceProduct{priceProduct},
	}, nil
}

// UpdatePriceProduct updates a price product using common PostgreSQL operations
func (r *PostgresPriceProductRepository) UpdatePriceProduct(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) (*priceproductpb.UpdatePriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price product: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceProduct := &priceproductpb.PriceProduct{}
	if err := protojson.Unmarshal(resultJSON, priceProduct); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceproductpb.UpdatePriceProductResponse{
		Data: []*priceproductpb.PriceProduct{priceProduct},
	}, nil
}

// DeletePriceProduct deletes a price product using common PostgreSQL operations
func (r *PostgresPriceProductRepository) DeletePriceProduct(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) (*priceproductpb.DeletePriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete price product: %w", err)
	}

	return &priceproductpb.DeletePriceProductResponse{
		Success: true,
	}, nil
}

// ListPriceProducts lists price products using common PostgreSQL operations
func (r *PostgresPriceProductRepository) ListPriceProducts(ctx context.Context, req *priceproductpb.ListPriceProductsRequest) (*priceproductpb.ListPriceProductsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list price products: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var priceProducts []*priceproductpb.PriceProduct
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		priceProduct := &priceproductpb.PriceProduct{}
		if err := protojson.Unmarshal(resultJSON, priceProduct); err != nil {
			// Log error and continue with next item
			continue
		}
		priceProducts = append(priceProducts, priceProduct)
	}

	return &priceproductpb.ListPriceProductsResponse{
		Data: priceProducts,
	}, nil
}

// GetPriceProductListPageData retrieves price products with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresPriceProductRepository) GetPriceProductListPageData(
	ctx context.Context,
	req *priceproductpb.GetPriceProductListPageDataRequest,
) (*priceproductpb.GetPriceProductListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}

	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Pricing pattern with amount/currency
	query := `WITH enriched AS (SELECT id, product_id, amount, currency, active, date_created, date_modified FROM price_product WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR product_id ILIKE $1 OR currency ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var priceProducts []*priceproductpb.PriceProduct
	var totalCount int64
	for rows.Next() {
		var id, productId, currency string
		var amount int64
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &productId, &amount, &currency, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		priceProduct := &priceproductpb.PriceProduct{Id: id, ProductId: productId, Amount: amount, Currency: currency, Active: active}
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		priceProduct.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		priceProduct.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		priceProduct.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		priceProduct.DateModifiedString = &dmStr
	}
		priceProducts = append(priceProducts, priceProduct)
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &priceproductpb.GetPriceProductListPageDataResponse{PriceProductList: priceProducts, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetPriceProductItemPageData retrieves price product item page data
func (r *PostgresPriceProductRepository) GetPriceProductItemPageData(ctx context.Context, req *priceproductpb.GetPriceProductItemPageDataRequest) (*priceproductpb.GetPriceProductItemPageDataResponse, error) {
	if req == nil || req.PriceProductId == "" {
		return nil, fmt.Errorf("price product ID required")
	}
	query := `SELECT id, product_id, amount, currency, active, date_created, date_modified FROM price_product WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.PriceProductId)
	var id, productId, currency string
	var amount int64
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &productId, &amount, &currency, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("price product not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	priceProduct := &priceproductpb.PriceProduct{Id: id, ProductId: productId, Amount: amount, Currency: currency, Active: active}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		priceProduct.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		priceProduct.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		priceProduct.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		priceProduct.DateModifiedString = &dmStr
	}
	return &priceproductpb.GetPriceProductItemPageDataResponse{PriceProduct: priceProduct, Success: true}, nil
}

// parsePriceProductTimestamp parses various timestamp formats to Unix milliseconds
func parsePriceProductTimestamp(ts string) (int64, error) {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.UnixMilli(), nil
	}
	for _, fmt := range []string{"2006-01-02T15:04:05Z", "2006-01-02 15:04:05", "2006-01-02T15:04:05.000Z"} {
		if t, err := time.Parse(fmt, ts); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, nil
}

// NewPriceProductRepository creates a new PostgreSQL price_product repository (old-style constructor)
func NewPriceProductRepository(db *sql.DB, tableName string) priceproductpb.PriceProductDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPriceProductRepository(dbOps, tableName)
}
