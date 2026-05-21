//go:build postgresql

package product

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
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ProductPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
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
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPlan := &productplanpb.ProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPlan); err != nil {
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
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPlan := &productplanpb.ProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPlan); err != nil {
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
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPlan := &productplanpb.ProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPlan); err != nil {
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

	err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id)
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
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productPlans []*productplanpb.ProductPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		productPlan := &productplanpb.ProductPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPlan); err != nil {
			// Log error and continue with next item
			continue
		}
		productPlans = append(productPlans, productPlan)
	}

	return &productplanpb.ListProductPlansResponse{
		Data: productPlans,
	}, nil
}

// GetProductPlanListPageData retrieves product plans by composing over ListProductPlans.
//
// Canonical pattern (plan 20260429-pagedata-canonicalize §3): delegate field
// projection to ListProductPlans (field-agnostic via protojson DiscardUnknown)
// and only enforce caller intent (active filter, pagination metadata) here.
func (r *PostgresProductPlanRepository) GetProductPlanListPageData(
	ctx context.Context,
	req *productplanpb.GetProductPlanListPageDataRequest,
) (*productplanpb.GetProductPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	limit, page := int32(50), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
		}
	}

	lr, err := r.ListProductPlans(ctx, &productplanpb.ListProductPlansRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list product plans: %w", err)
	}

	// Preserve page-data intent: only active product_plan rows on this surface.
	all := lr.GetData()
	productPlans := make([]*productplanpb.ProductPlan, 0, len(all))
	for _, pp := range all {
		if pp != nil && pp.GetActive() {
			productPlans = append(productPlans, pp)
		}
	}

	totalCount := int64(len(productPlans))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &productplanpb.GetProductPlanListPageDataResponse{
		ProductPlanList: productPlans,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
		Success: true,
	}, nil
}

// GetProductPlanItemPageData retrieves a single product plan by composing over
// ReadProductPlan.
//
// Canonical pattern (plan 20260429-pagedata-canonicalize §3): per the
// ProductPlan caveat, denormalization of the parent Plan + Product would stay
// via dual reads (ReadPlan + ReadProduct), but the response message
// (GetProductPlanItemPageDataResponse) only carries ProductPlan, so no extras
// are needed here. ReadProductPlan handles the full proto field projection via
// dbOps.Read + protojson DiscardUnknown.
func (r *PostgresProductPlanRepository) GetProductPlanItemPageData(
	ctx context.Context,
	req *productplanpb.GetProductPlanItemPageDataRequest,
) (*productplanpb.GetProductPlanItemPageDataResponse, error) {
	if req == nil || req.ProductPlanId == "" {
		return nil, fmt.Errorf("product plan ID required")
	}

	rr, err := r.ReadProductPlan(ctx, &productplanpb.ReadProductPlanRequest{
		Data: &productplanpb.ProductPlan{Id: req.ProductPlanId},
	})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("product plan not found")
	}
	productPlan := rr.GetData()[0]

	// Preserve page-data intent: only active product_plan rows on this surface.
	if !productPlan.GetActive() {
		return nil, fmt.Errorf("product plan not found")
	}

	return &productplanpb.GetProductPlanItemPageDataResponse{
		ProductPlan: productPlan,
		Success:     true,
	}, nil
}

// ListByPlan retrieves all product plans for a given plan, ordered by date_created DESC
func (r *PostgresProductPlanRepository) ListByPlan(
	ctx context.Context,
	req *productplanpb.ListProductPlansByPlanRequest,
) (*productplanpb.ListProductPlansByPlanResponse, error) {
	if req == nil || req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	query := `
		SELECT pp.id, pp.name, pp.description, pp.product_id, pp.plan_id, pp.active, pp.date_created, pp.date_modified
		FROM product_plan pp
		WHERE pp.plan_id = $1 AND pp.active = true
		ORDER BY pp.date_created DESC
	`

	rows, err := r.db.QueryContext(ctx, query, req.PlanId)
	if err != nil {
		return nil, fmt.Errorf("failed to list product plans by plan: %w", err)
	}
	defer rows.Close()

	var items []*productplanpb.ProductPlan
	for rows.Next() {
		var (
			id           string
			name         string
			description  *string
			productId    string
			planId       *string
			active       bool
			dateCreated  time.Time
			dateModified time.Time
		)

		if err := rows.Scan(&id, &name, &description, &productId, &planId, &active, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("failed to scan product plan row: %w", err)
		}

		item := &productplanpb.ProductPlan{Id: id, Name: name, ProductId: productId, Active: active}
		if description != nil {
			item.Description = description
		}
		if planId != nil {
			item.PlanId = *planId
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			item.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			item.DateModifiedString = &dmStr
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product plan rows: %w", err)
	}

	return &productplanpb.ListProductPlansByPlanResponse{ProductPlans: items, Success: true}, nil
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
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresProductPlanRepository(dbOps, tableName)
}
