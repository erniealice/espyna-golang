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
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "collection_plan", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresCollectionPlanRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionPlanRepository implements collection_plan CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_collection_plan_active ON collection_plan(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_collection_plan_collection_id ON collection_plan(collection_id) - FK lookup on collection_id
//   - CREATE INDEX idx_collection_plan_plan_id ON collection_plan(plan_id) - FK lookup on plan_id
//   - CREATE INDEX idx_collection_plan_date_created ON collection_plan(date_created DESC) - Default sorting
type PostgresCollectionPlanRepository struct {
	collectionplanpb.UnimplementedCollectionPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresCollectionPlanRepository creates a new PostgreSQL collection plan repository
func NewPostgresCollectionPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionplanpb.CollectionPlanDomainServiceServer {
	if tableName == "" {
		tableName = "collection_plan" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionPlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollectionPlan creates a new collection plan using common PostgreSQL operations
func (r *PostgresCollectionPlanRepository) CreateCollectionPlan(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) (*collectionplanpb.CreateCollectionPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection plan data is required")
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
		return nil, fmt.Errorf("failed to create collection plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionPlan := &collectionplanpb.CollectionPlan{}
	if err := protojson.Unmarshal(resultJSON, collectionPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionplanpb.CreateCollectionPlanResponse{
		Data: []*collectionplanpb.CollectionPlan{collectionPlan},
	}, nil
}

// ReadCollectionPlan retrieves a collection plan using common PostgreSQL operations
func (r *PostgresCollectionPlanRepository) ReadCollectionPlan(ctx context.Context, req *collectionplanpb.ReadCollectionPlanRequest) (*collectionplanpb.ReadCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection plan: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionPlan := &collectionplanpb.CollectionPlan{}
	if err := protojson.Unmarshal(resultJSON, collectionPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionplanpb.ReadCollectionPlanResponse{
		Data: []*collectionplanpb.CollectionPlan{collectionPlan},
	}, nil
}

// UpdateCollectionPlan updates a collection plan using common PostgreSQL operations
func (r *PostgresCollectionPlanRepository) UpdateCollectionPlan(ctx context.Context, req *collectionplanpb.UpdateCollectionPlanRequest) (*collectionplanpb.UpdateCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required")
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
		return nil, fmt.Errorf("failed to update collection plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionPlan := &collectionplanpb.CollectionPlan{}
	if err := protojson.Unmarshal(resultJSON, collectionPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionplanpb.UpdateCollectionPlanResponse{
		Data: []*collectionplanpb.CollectionPlan{collectionPlan},
	}, nil
}

// DeleteCollectionPlan deletes a collection plan using common PostgreSQL operations
func (r *PostgresCollectionPlanRepository) DeleteCollectionPlan(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) (*collectionplanpb.DeleteCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection plan: %w", err)
	}

	return &collectionplanpb.DeleteCollectionPlanResponse{
		Success: true,
	}, nil
}

// ListCollectionPlans lists collection plans using common PostgreSQL operations
func (r *PostgresCollectionPlanRepository) ListCollectionPlans(ctx context.Context, req *collectionplanpb.ListCollectionPlansRequest) (*collectionplanpb.ListCollectionPlansResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var collectionPlans []*collectionplanpb.CollectionPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		collectionPlan := &collectionplanpb.CollectionPlan{}
		if err := protojson.Unmarshal(resultJSON, collectionPlan); err != nil {
			// Log error and continue with next item
			continue
		}
		collectionPlans = append(collectionPlans, collectionPlan)
	}

	return &collectionplanpb.ListCollectionPlansResponse{
		Data: collectionPlans,
	}, nil
}

// GetCollectionPlanListPageData retrieves collection plans with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresCollectionPlanRepository) GetCollectionPlanListPageData(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanListPageDataRequest,
) (*collectionplanpb.GetCollectionPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values
	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Junction table pattern
	query := `
		WITH enriched AS (
			SELECT
				cp.id,
				cp.collection_id,
				cp.plan_id,
				cp.active,
				cp.date_created,
				cp.date_modified
			FROM collection_plan cp
			WHERE cp.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       cp.collection_id ILIKE $1 OR
			       cp.plan_id ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var collectionPlans []*collectionplanpb.CollectionPlan
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			collectionId       string
			planId             string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)

		if err := rows.Scan(
			&id,
			&collectionId,
			&planId,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		totalCount = total

		collectionPlan := &collectionplanpb.CollectionPlan{
			Id:           id,
			CollectionId: collectionId,
			PlanId:       planId,
			Active:       active,
		}

		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		collectionPlan.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		collectionPlan.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		collectionPlan.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		collectionPlan.DateModifiedString = &dmStr
	}

		collectionPlans = append(collectionPlans, collectionPlan)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &collectionplanpb.GetCollectionPlanListPageDataResponse{
		CollectionPlanList: collectionPlans,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetCollectionPlanItemPageData retrieves a single collection plan with enhanced item page data
func (r *PostgresCollectionPlanRepository) GetCollectionPlanItemPageData(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanItemPageDataRequest,
) (*collectionplanpb.GetCollectionPlanItemPageDataResponse, error) {
	if req == nil || req.CollectionPlanId == "" {
		return nil, fmt.Errorf("collection plan ID required")
	}

	query := `
		SELECT
			cp.id,
			cp.collection_id,
			cp.plan_id,
			cp.active,
			cp.date_created,
			cp.date_modified
		FROM collection_plan cp
		WHERE cp.id = $1 AND cp.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.CollectionPlanId)

	var (
		id                 string
		collectionId       string
		planId             string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
	)

	if err := row.Scan(
		&id,
		&collectionId,
		&planId,
		&active,
		&dateCreated,
		&dateModified,
	); err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	collectionPlan := &collectionplanpb.CollectionPlan{
		Id:           id,
		CollectionId: collectionId,
		PlanId:       planId,
		Active:       active,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		collectionPlan.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		collectionPlan.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		collectionPlan.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		collectionPlan.DateModifiedString = &dmStr
	}

	return &collectionplanpb.GetCollectionPlanItemPageDataResponse{
		CollectionPlan: collectionPlan,
		Success:        true,
	}, nil
}

// parseCollectionPlanTimestamp parses various timestamp formats to Unix milliseconds
func parseCollectionPlanTimestamp(ts string) (int64, error) {
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

// NewCollectionPlanRepository creates a new PostgreSQL collection_plan repository (old-style constructor)
func NewCollectionPlanRepository(db *sql.DB, tableName string) collectionplanpb.CollectionPlanDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresCollectionPlanRepository(dbOps, tableName)
}
