package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresProductPricePlanRepository implements product_price_plan CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_price_plan_active ON product_price_plan(active) WHERE active = true
//   - CREATE INDEX idx_product_price_plan_price_plan_id ON product_price_plan(price_plan_id) - Filter by price plan
//   - CREATE INDEX idx_product_price_plan_product_id ON product_price_plan(product_id) - Filter by product
//   - CREATE INDEX idx_product_price_plan_date_created ON product_price_plan(date_created DESC) - Default sorting
type PostgresProductPricePlanRepository struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ProductPricePlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_price_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductPricePlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresProductPricePlanRepository creates a new PostgreSQL product price plan repository
func NewPostgresProductPricePlanRepository(dbOps interfaces.DatabaseOperation, tableName string) productpriceplanpb.ProductPricePlanDomainServiceServer {
	if tableName == "" {
		tableName = "product_price_plan" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductPricePlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductPricePlan creates a new product price plan using common PostgreSQL operations
func (r *PostgresProductPricePlanRepository) CreateProductPricePlan(ctx context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) (*productpriceplanpb.CreateProductPricePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product price plan data is required")
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
		return nil, fmt.Errorf("failed to create product price plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPricePlan := &productpriceplanpb.ProductPricePlan{}
	if err := protojson.Unmarshal(resultJSON, productPricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpriceplanpb.CreateProductPricePlanResponse{
		Data: []*productpriceplanpb.ProductPricePlan{productPricePlan},
	}, nil
}

// ReadProductPricePlan retrieves a product price plan using common PostgreSQL operations
func (r *PostgresProductPricePlanRepository) ReadProductPricePlan(ctx context.Context, req *productpriceplanpb.ReadProductPricePlanRequest) (*productpriceplanpb.ReadProductPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product price plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product price plan: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPricePlan := &productpriceplanpb.ProductPricePlan{}
	if err := protojson.Unmarshal(resultJSON, productPricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpriceplanpb.ReadProductPricePlanResponse{
		Data: []*productpriceplanpb.ProductPricePlan{productPricePlan},
	}, nil
}

// UpdateProductPricePlan updates a product price plan using common PostgreSQL operations
func (r *PostgresProductPricePlanRepository) UpdateProductPricePlan(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) (*productpriceplanpb.UpdateProductPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product price plan ID is required")
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
		return nil, fmt.Errorf("failed to update product price plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPricePlan := &productpriceplanpb.ProductPricePlan{}
	if err := protojson.Unmarshal(resultJSON, productPricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpriceplanpb.UpdateProductPricePlanResponse{
		Data: []*productpriceplanpb.ProductPricePlan{productPricePlan},
	}, nil
}

// DeleteProductPricePlan deletes a product price plan using common PostgreSQL operations
func (r *PostgresProductPricePlanRepository) DeleteProductPricePlan(ctx context.Context, req *productpriceplanpb.DeleteProductPricePlanRequest) (*productpriceplanpb.DeleteProductPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product price plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product price plan: %w", err)
	}

	return &productpriceplanpb.DeleteProductPricePlanResponse{
		Success: true,
	}, nil
}

// ListProductPricePlans lists product price plans using common PostgreSQL operations
func (r *PostgresProductPricePlanRepository) ListProductPricePlans(ctx context.Context, req *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product price plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productPricePlans []*productpriceplanpb.ProductPricePlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		productPricePlan := &productpriceplanpb.ProductPricePlan{}
		if err := protojson.Unmarshal(resultJSON, productPricePlan); err != nil {
			// Log error and continue with next item
			continue
		}
		productPricePlans = append(productPricePlans, productPricePlan)
	}

	return &productpriceplanpb.ListProductPricePlansResponse{
		Data: productPricePlans,
	}, nil
}

// GetProductPricePlanListPageData retrieves paginated product price plan list data
func (r *PostgresProductPricePlanRepository) GetProductPricePlanListPageData(ctx context.Context, req *productpriceplanpb.GetProductPricePlanListPageDataRequest) (*productpriceplanpb.GetProductPricePlanListPageDataResponse, error) {
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

	query := `SELECT id, price_plan_id, product_id, price, currency, active, date_created, date_modified FROM product_price_plan WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR price_plan_id ILIKE $1 OR product_id ILIKE $1 OR currency ILIKE $1) ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var productPricePlans []*productpriceplanpb.ProductPricePlan
	for rows.Next() {
		var id, pricePlanId, productId, currency string
		var price int64
		var active bool
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &pricePlanId, &productId, &price, &currency, &active, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		productPricePlan := &productpriceplanpb.ProductPricePlan{
			Id:          id,
			PricePlanId: pricePlanId,
			ProductId:   productId,
			Price:       price,
			Currency:    currency,
			Active:      active,
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productPricePlan.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productPricePlan.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productPricePlan.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productPricePlan.DateModifiedString = &dmStr
		}
		productPricePlans = append(productPricePlans, productPricePlan)
	}
	return &productpriceplanpb.GetProductPricePlanListPageDataResponse{ProductPricePlanList: productPricePlans, Success: true}, nil
}

// GetProductPricePlanItemPageData retrieves product price plan item page data
func (r *PostgresProductPricePlanRepository) GetProductPricePlanItemPageData(ctx context.Context, req *productpriceplanpb.GetProductPricePlanItemPageDataRequest) (*productpriceplanpb.GetProductPricePlanItemPageDataResponse, error) {
	if req == nil || req.ProductPricePlanId == "" {
		return nil, fmt.Errorf("product price plan ID required")
	}
	query := `SELECT id, price_plan_id, product_id, price, currency, active, date_created, date_modified FROM product_price_plan WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.ProductPricePlanId)
	var id, pricePlanId, productId, currency string
	var price int64
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &pricePlanId, &productId, &price, &currency, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("product price plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	productPricePlan := &productpriceplanpb.ProductPricePlan{
		Id:          id,
		PricePlanId: pricePlanId,
		ProductId:   productId,
		Price:       price,
		Currency:    currency,
		Active:      active,
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productPricePlan.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productPricePlan.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productPricePlan.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productPricePlan.DateModifiedString = &dmStr
	}
	return &productpriceplanpb.GetProductPricePlanItemPageDataResponse{ProductPricePlan: productPricePlan, Success: true}, nil
}

// NewProductPricePlanRepository creates a new PostgreSQL product_price_plan repository (old-style constructor)
func NewProductPricePlanRepository(db *sql.DB, tableName string) productpriceplanpb.ProductPricePlanDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductPricePlanRepository(dbOps, tableName)
}
