//go:build postgres

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_plan", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductPlanRepository(dbOps, tableName), nil
	})
}

// PostgresProductPlanRepository implements product_plan CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_plan_active ON product_plan(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_plan_product_id ON product_plan(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_product_plan_plan_id ON product_plan(plan_id) - FK lookup on plan_id
//   - CREATE INDEX idx_product_plan_date_created ON product_plan(date_created DESC) - Default sorting
type PostgresProductPlanRepository struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductPlanRepository creates a new PostgreSQL product plan repository
func NewPostgresProductPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) productplanpb.ProductPlanDomainServiceServer {
	if tableName == "" {
		tableName = "product_plan" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductPlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductPlan creates a new product plan using common PostgreSQL operations
func (r *PostgresProductPlanRepository) CreateProductPlan(ctx context.Context, req *productplanpb.CreateProductPlanRequest) (*productplanpb.CreateProductPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product plan data is required")
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
		return nil, fmt.Errorf("failed to create product plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPlan := &productplanpb.ProductPlan{}
	if err := protojson.Unmarshal(resultJSON, productPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productplanpb.CreateProductPlanResponse{
		Data: []*productplanpb.ProductPlan{productPlan},
	}, nil
}

// ReadProductPlan retrieves a product plan using common PostgreSQL operations
func (r *PostgresProductPlanRepository) ReadProductPlan(ctx context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product plan: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPlan := &productplanpb.ProductPlan{}
	if err := protojson.Unmarshal(resultJSON, productPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productplanpb.ReadProductPlanResponse{
		Data: []*productplanpb.ProductPlan{productPlan},
	}, nil
}

// UpdateProductPlan updates a product plan using common PostgreSQL operations
func (r *PostgresProductPlanRepository) UpdateProductPlan(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) (*productplanpb.UpdateProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
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
		return nil, fmt.Errorf("failed to update product plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPlan := &productplanpb.ProductPlan{}
	if err := protojson.Unmarshal(resultJSON, productPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productplanpb.UpdateProductPlanResponse{
		Data: []*productplanpb.ProductPlan{productPlan},
	}, nil
}

// DeleteProductPlan deletes a product plan using common PostgreSQL operations
func (r *PostgresProductPlanRepository) DeleteProductPlan(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) (*productplanpb.DeleteProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product plan: %w", err)
	}

	return &productplanpb.DeleteProductPlanResponse{
		Success: true,
	}, nil
}

// ListProductPlans lists product plans using common PostgreSQL operations
func (r *PostgresProductPlanRepository) ListProductPlans(ctx context.Context, req *productplanpb.ListProductPlansRequest) (*productplanpb.ListProductPlansResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list product plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productPlans []*productplanpb.ProductPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		productPlan := &productplanpb.ProductPlan{}
		if err := protojson.Unmarshal(resultJSON, productPlan); err != nil {
			// Log error and continue with next item
			continue
		}
		productPlans = append(productPlans, productPlan)
	}

	return &productplanpb.ListProductPlansResponse{
		Data: productPlans,
	}, nil
}

// GetProductPlanListPageData retrieves product plans with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresProductPlanRepository) GetProductPlanListPageData(
	ctx context.Context,
	req *productplanpb.GetProductPlanListPageDataRequest,
) (*productplanpb.GetProductPlanListPageDataResponse, error) {
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

	query := `WITH enriched AS (SELECT id, name, description, price, currency, product_id, active, date_created, date_created_string, date_modified, date_modified_string FROM product_plan WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR name ILIKE $1 OR description ILIKE $1 OR product_id ILIKE $1 OR currency ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var productPlans []*productplanpb.ProductPlan
	var totalCount int64
	for rows.Next() {
		var id, name, productId, currency string
		var description *string
		var price float64
		var active bool
		var dateCreated, dateCreatedString, dateModified, dateModifiedString *string
		var total int64
		if err := rows.Scan(&id, &name, &description, &price, &currency, &productId, &active, &dateCreated, &dateCreatedString, &dateModified, &dateModifiedString, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		productPlan := &productplanpb.ProductPlan{Id: id, Name: name, Price: price, Currency: currency, ProductId: productId, Active: active}
		if description != nil {
			productPlan.Description = description
		}
		if dateCreatedString != nil {
			productPlan.DateCreatedString = dateCreatedString
		}
		if dateModifiedString != nil {
			productPlan.DateModifiedString = dateModifiedString
		}
		if dateCreated != nil && *dateCreated != "" {
			if ts, _ := parseProductPlanTimestamp(*dateCreated); ts > 0 {
				productPlan.DateCreated = &ts
			}
		}
		if dateModified != nil && *dateModified != "" {
			if ts, _ := parseProductPlanTimestamp(*dateModified); ts > 0 {
				productPlan.DateModified = &ts
			}
		}
		productPlans = append(productPlans, productPlan)
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &productplanpb.GetProductPlanListPageDataResponse{ProductPlanList: productPlans, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetProductPlanItemPageData retrieves product plan item page data
func (r *PostgresProductPlanRepository) GetProductPlanItemPageData(ctx context.Context, req *productplanpb.GetProductPlanItemPageDataRequest) (*productplanpb.GetProductPlanItemPageDataResponse, error) {
	if req == nil || req.ProductPlanId == "" {
		return nil, fmt.Errorf("product plan ID required")
	}
	query := `SELECT id, name, description, price, currency, product_id, active, date_created, date_created_string, date_modified, date_modified_string FROM product_plan WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.ProductPlanId)
	var id, name, productId, currency string
	var description *string
	var price float64
	var active bool
	var dateCreated, dateCreatedString, dateModified, dateModifiedString *string
	if err := row.Scan(&id, &name, &description, &price, &currency, &productId, &active, &dateCreated, &dateCreatedString, &dateModified, &dateModifiedString); err == sql.ErrNoRows {
		return nil, fmt.Errorf("product plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	productPlan := &productplanpb.ProductPlan{Id: id, Name: name, Price: price, Currency: currency, ProductId: productId, Active: active}
	if description != nil {
		productPlan.Description = description
	}
	if dateCreatedString != nil {
		productPlan.DateCreatedString = dateCreatedString
	}
	if dateModifiedString != nil {
		productPlan.DateModifiedString = dateModifiedString
	}
	if dateCreated != nil && *dateCreated != "" {
		if ts, _ := parseProductPlanTimestamp(*dateCreated); ts > 0 {
			productPlan.DateCreated = &ts
		}
	}
	if dateModified != nil && *dateModified != "" {
		if ts, _ := parseProductPlanTimestamp(*dateModified); ts > 0 {
			productPlan.DateModified = &ts
		}
	}
	return &productplanpb.GetProductPlanItemPageDataResponse{ProductPlan: productPlan, Success: true}, nil
}

// parseProductPlanTimestamp parses various timestamp formats to Unix milliseconds
func parseProductPlanTimestamp(ts string) (int64, error) {
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

// NewProductPlanRepository creates a new PostgreSQL product_plan repository (old-style constructor)
func NewProductPlanRepository(db *sql.DB, tableName string) productplanpb.ProductPlanDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductPlanRepository(dbOps, tableName)
}
