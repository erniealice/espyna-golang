//go:build postgres

package subscription

import (
	"context"
	"database/sql"
	"time"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	priceplanpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/price_plan"
)

// PostgresPricePlanRepository implements price_plan CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_price_plan_active ON price_plan(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_price_plan_plan_id ON price_plan(plan_id) - Filter by plan
//   - CREATE INDEX idx_price_plan_amount ON price_plan(amount) - Sort/filter by price
//   - CREATE INDEX idx_price_plan_date_created ON price_plan(date_created DESC) - Default sorting
type PostgresPricePlanRepository struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "price_plan", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres price_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPricePlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresPricePlanRepository creates a new PostgreSQL price plan repository
func NewPostgresPricePlanRepository(dbOps interfaces.DatabaseOperation, tableName string) priceplanpb.PricePlanDomainServiceServer {
	if tableName == "" {
		tableName = "price_plan" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPricePlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePricePlan creates a new price plan using common PostgreSQL operations
func (r *PostgresPricePlanRepository) CreatePricePlan(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price plan data is required")
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
		return nil, fmt.Errorf("failed to create price plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pricePlan := &priceplanpb.PricePlan{}
	if err := protojson.Unmarshal(resultJSON, pricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceplanpb.CreatePricePlanResponse{
		Data: []*priceplanpb.PricePlan{pricePlan},
	}, nil
}

// ReadPricePlan retrieves a price plan using common PostgreSQL operations
func (r *PostgresPricePlanRepository) ReadPricePlan(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price plan: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pricePlan := &priceplanpb.PricePlan{}
	if err := protojson.Unmarshal(resultJSON, pricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceplanpb.ReadPricePlanResponse{
		Data: []*priceplanpb.PricePlan{pricePlan},
	}, nil
}

// UpdatePricePlan updates a price plan using common PostgreSQL operations
func (r *PostgresPricePlanRepository) UpdatePricePlan(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
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
		return nil, fmt.Errorf("failed to update price plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pricePlan := &priceplanpb.PricePlan{}
	if err := protojson.Unmarshal(resultJSON, pricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceplanpb.UpdatePricePlanResponse{
		Data: []*priceplanpb.PricePlan{pricePlan},
	}, nil
}

// DeletePricePlan deletes a price plan using common PostgreSQL operations
func (r *PostgresPricePlanRepository) DeletePricePlan(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) (*priceplanpb.DeletePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete price plan: %w", err)
	}

	return &priceplanpb.DeletePricePlanResponse{
		Success: true,
	}, nil
}

// ListPricePlans lists price plans using common PostgreSQL operations
func (r *PostgresPricePlanRepository) ListPricePlans(ctx context.Context, req *priceplanpb.ListPricePlansRequest) (*priceplanpb.ListPricePlansResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list price plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var pricePlans []*priceplanpb.PricePlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		pricePlan := &priceplanpb.PricePlan{}
		if err := protojson.Unmarshal(resultJSON, pricePlan); err != nil {
			// Log error and continue with next item
			continue
		}
		pricePlans = append(pricePlans, pricePlan)
	}

	return &priceplanpb.ListPricePlansResponse{
		Data: pricePlans,
	}, nil
}

// GetPricePlanListPageData retrieves paginated price plan list data with CTE
func (r *PostgresPricePlanRepository) GetPricePlanListPageData(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) (*priceplanpb.GetPricePlanListPageDataResponse, error) {
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

	query := `SELECT id, plan_id, amount, currency, name, description, active, date_created, date_modified FROM price_plan WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR plan_id ILIKE $1 OR currency ILIKE $1) ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var pricePlans []*priceplanpb.PricePlan
	var totalCount int64
	for rows.Next() {
		var id, planId, currency, name, description string
		var amount float64
		var active bool
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &planId, &amount, &currency, &name, &description, &active, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount++
		pricePlan := &priceplanpb.PricePlan{Id: id, PlanId: planId, Name: name, Description: description, Amount: float64(amount), Currency: currency, Active: active}
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		pricePlan.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		pricePlan.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		pricePlan.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		pricePlan.DateModifiedString = &dmStr
	}
		pricePlans = append(pricePlans, pricePlan)
	}
	return &priceplanpb.GetPricePlanListPageDataResponse{PricePlanList: pricePlans, Success: true}, nil
}
// Note: Pagination removed - not available in current protobuf schema

// GetPricePlanItemPageData retrieves price plan item page data
func (r *PostgresPricePlanRepository) GetPricePlanItemPageData(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) (*priceplanpb.GetPricePlanItemPageDataResponse, error) {
	if req == nil || req.PricePlanId == "" {
		return nil, fmt.Errorf("price plan ID required")
	}
	query := `SELECT id, plan_id, amount, currency, name, description, active, date_created, date_modified FROM price_plan WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.PricePlanId)
	var id, planId, currency, name, description string
	var amount float64
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &planId, &amount, &currency, &name, &description, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("price plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	pricePlan := &priceplanpb.PricePlan{Id: id, PlanId: planId, Name: name, Description: description, Amount: float64(amount), Currency: currency, Active: active}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		pricePlan.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		pricePlan.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		pricePlan.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		pricePlan.DateModifiedString = &dmStr
	}
	return &priceplanpb.GetPricePlanItemPageDataResponse{PricePlan: pricePlan, Success: true}, nil
}

// Note: Duplicate operations.ParseTimestamp function removed - use the one from common operations package

// NewPricePlanRepository creates a new PostgreSQL price_plan repository (old-style constructor)
func NewPricePlanRepository(db *sql.DB, tableName string) priceplanpb.PricePlanDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPricePlanRepository(dbOps, tableName)
}
