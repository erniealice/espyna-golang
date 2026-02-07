//go:build postgres

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
	planlocationpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_location"
)

// PostgresPlanRepository implements plan CRUD operations using PostgreSQL
type PostgresPlanRepository struct {
	planpb.UnimplementedPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "plan", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresPlanRepository creates a new PostgreSQL plan repository
func NewPostgresPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) planpb.PlanDomainServiceServer {
	if tableName == "" {
		tableName = "plan" // default fallback
	}
	return &PostgresPlanRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreatePlan creates a new plan using common PostgreSQL operations
func (r *PostgresPlanRepository) CreatePlan(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan data is required")
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
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	plan := &planpb.Plan{}
	if err := protojson.Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planpb.CreatePlanResponse{
		Data: []*planpb.Plan{plan},
	}, nil
}

// ReadPlan retrieves a plan using common PostgreSQL operations
func (r *PostgresPlanRepository) ReadPlan(ctx context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, *req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	plan := &planpb.Plan{}
	if err := protojson.Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planpb.ReadPlanResponse{
		Data: []*planpb.Plan{plan},
	}, nil
}

// UpdatePlan updates a plan using common PostgreSQL operations
func (r *PostgresPlanRepository) UpdatePlan(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
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
	result, err := r.dbOps.Update(ctx, r.tableName, *req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	plan := &planpb.Plan{}
	if err := protojson.Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planpb.UpdatePlanResponse{
		Data: []*planpb.Plan{plan},
	}, nil
}

// DeletePlan deletes a plan using common PostgreSQL operations
func (r *PostgresPlanRepository) DeletePlan(ctx context.Context, req *planpb.DeletePlanRequest) (*planpb.DeletePlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, *req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete plan: %w", err)
	}

	return &planpb.DeletePlanResponse{
		Success: true,
	}, nil
}

// ListPlans lists plans using common PostgreSQL operations
func (r *PostgresPlanRepository) ListPlans(ctx context.Context, req *planpb.ListPlansRequest) (*planpb.ListPlansResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var plans []*planpb.Plan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		plan := &planpb.Plan{}
		if err := protojson.Unmarshal(resultJSON, plan); err != nil {
			// Log error and continue with next item
			continue
		}
		plans = append(plans, plan)
	}

	return &planpb.ListPlansResponse{
		Data: plans,
	}, nil
}

// GetPlanListPageData retrieves a paginated, filtered, sorted, and searchable list of plans with location relationships
// This method uses CTEs (Common Table Expressions) to optimize query performance by loading all data in a single query
// TODO: Add unit tests for GetPlanListPageData
func (r *PostgresPlanRepository) GetPlanListPageData(ctx context.Context, req *planpb.GetPlanListPageDataRequest) (*planpb.GetPlanListPageDataResponse, error) {
	// Extract pagination parameters with defaults
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100 // Cap at 100 items per page
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	// Extract search query
	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	// Extract sort parameters with defaults
	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 { // DESC enum value
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_active ON plan(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_name_trgm ON plan USING gin(name gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_description_trgm ON plan USING gin(description gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_date_created ON plan(date_created DESC);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_location_plan_id ON plan_location(plan_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_location_location_id ON plan_location(location_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_location_active ON plan_location(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_location_active ON location(active) WHERE active = true;

	// Build the CTE query following the translation plan pattern
	query := `
		WITH
		-- CTE 1: Aggregate plan_location relationships with location details
		plan_locations_agg AS (
			SELECT
				pl.plan_id,
				array_agg(
					DISTINCT jsonb_build_object(
						'id', pl.id,
						'plan_id', pl.plan_id,
						'location_id', pl.location_id,
						'date_created', pl.date_created,
						'date_created_string', pl.date_created_string,
						'date_modified', pl.date_modified,
						'date_modified_string', pl.date_modified_string,
						'active', pl.active,
						'location', jsonb_build_object(
							'id', l.id,
							'name', l.name,
							'address', l.address,
							'date_created', l.date_created,
							'date_created_string', l.date_created_string,
							'date_modified', l.date_modified,
							'date_modified_string', l.date_modified_string,
							'active', l.active,
							'description', l.description
						)
					) ORDER BY l.name ASC
				) FILTER (WHERE l.id IS NOT NULL) as plan_locations
			FROM plan_location pl
			INNER JOIN location l ON pl.location_id = l.id
			WHERE pl.active = true AND l.active = true
			GROUP BY pl.plan_id
		),

		-- CTE 2: Apply search filter
		search_filtered AS (
			SELECT p.*
			FROM plan p
			WHERE p.active = true
				AND ($1::text = '' OR
					p.name ILIKE $1 OR
					p.description ILIKE $1)
		),

		-- CTE 3: Join with locations and prepare for sorting
		enriched AS (
			SELECT
				sf.id,
				sf.name,
				sf.description,
				sf.active,
				sf.date_created,
				sf.date_created_string,
				sf.date_modified,
				sf.date_modified_string,
				COALESCE(pla.plan_locations, ARRAY[]::jsonb[]) as plan_locations
			FROM search_filtered sf
			LEFT JOIN plan_locations_agg pla ON sf.id = pla.plan_id
		),

		-- CTE 4: Apply sorting
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN $4 = 'name' AND $5 = 'ASC' THEN name END ASC,
				CASE WHEN $4 = 'name' AND $5 = 'DESC' THEN name END DESC,
				CASE WHEN $4 = 'description' AND $5 = 'ASC' THEN description END ASC,
				CASE WHEN $4 = 'description' AND $5 = 'DESC' THEN description END DESC,
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC
		),

		-- CTE 5: Calculate total count for pagination
		total_count AS (
			SELECT count(*) as total FROM sorted
		)

		-- Final SELECT with pagination
		SELECT
			s.id,
			s.name,
			s.description,
			s.active,
			s.date_created,
			s.date_created_string,
			s.date_modified,
			s.date_modified_string,
			s.plan_locations,
			tc.total as _total_count
		FROM sorted s
		CROSS JOIN total_count tc
		LIMIT $2 OFFSET $3
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	rows, err := db.GetDB().QueryContext(ctx, query,
		searchQuery,   // $1
		limit,         // $2
		offset,        // $3
		sortField,     // $4
		sortDirection, // $5
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetPlanListPageData query: %w", err)
	}
	defer rows.Close()

	var plans []*planpb.Plan
	var totalCount int32

	for rows.Next() {
		var (
			id                 string
			name               string
			description        string
			active             bool
			dateCreated        sql.NullInt64
			dateCreatedString  sql.NullString
			dateModified       sql.NullInt64
			dateModifiedString sql.NullString
			planLocationsJSON  []byte
			rowTotalCount      int32
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&active,
			&dateCreated,
			&dateCreatedString,
			&dateModified,
			&dateModifiedString,
			&planLocationsJSON,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan row: %w", err)
		}

		totalCount = rowTotalCount

		// Build plan message
		plan := &planpb.Plan{
			Id:          &id,
			Name:        name,
			Description: &description,
			Active:      active,
		}

		if dateCreated.Valid {
			plan.DateCreated = &dateCreated.Int64
		}
		if dateCreatedString.Valid {
			plan.DateCreatedString = &dateCreatedString.String
		}
		if dateModified.Valid {
			plan.DateModified = &dateModified.Int64
		}
		if dateModifiedString.Valid {
			plan.DateModifiedString = &dateModifiedString.String
		}

		// Parse plan_locations JSON array
		if len(planLocationsJSON) > 0 {
			var planLocations []map[string]any
			if err := json.Unmarshal(planLocationsJSON, &planLocations); err == nil {
				// Convert to protobuf PlanLocation messages
				for _, plData := range planLocations {
					plJSON, _ := json.Marshal(plData)
					var planLocation planlocationpb.PlanLocation
					if err := protojson.Unmarshal(plJSON, &planLocation); err == nil {
						plan.PlanLocations = append(plan.PlanLocations, &planLocation)
					}
				}
			}
		}

		plans = append(plans, plan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating plan rows: %w", err)
	}

	// Build pagination response
	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	paginationResponse := &commonpb.PaginationResponse{
		TotalItems:  totalCount,
		CurrentPage: &page,
		TotalPages:  &totalPages,
		HasNext:     hasNext,
		HasPrev:     hasPrev,
	}

	return &planpb.GetPlanListPageDataResponse{
		Success:    true,
		PlanList:   plans,
		Pagination: paginationResponse,
	}, nil
}

// GetPlanItemPageData retrieves a single plan with all related location data expanded
// This method uses CTEs (Common Table Expressions) to load all related data in a single query
// TODO: Add unit tests for GetPlanItemPageData
func (r *PostgresPlanRepository) GetPlanItemPageData(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) (*planpb.GetPlanItemPageDataResponse, error) {
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_id ON plan(id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_location_plan_id ON plan_location(plan_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_location_location_id ON plan_location(location_id);

	// Build CTE query to fetch plan with all related data
	query := `
		WITH
		-- CTE 1: Aggregate plan_location relationships with location details
		plan_locations_agg AS (
			SELECT
				pl.plan_id,
				array_agg(
					DISTINCT jsonb_build_object(
						'id', pl.id,
						'plan_id', pl.plan_id,
						'location_id', pl.location_id,
						'date_created', pl.date_created,
						'date_created_string', pl.date_created_string,
						'date_modified', pl.date_modified,
						'date_modified_string', pl.date_modified_string,
						'active', pl.active,
						'location', jsonb_build_object(
							'id', l.id,
							'name', l.name,
							'address', l.address,
							'date_created', l.date_created,
							'date_created_string', l.date_created_string,
							'date_modified', l.date_modified,
							'date_modified_string', l.date_modified_string,
							'active', l.active,
							'description', l.description
						)
					) ORDER BY l.name ASC
				) FILTER (WHERE l.id IS NOT NULL) as plan_locations
			FROM plan_location pl
			INNER JOIN location l ON pl.location_id = l.id
			WHERE pl.plan_id = $1 AND pl.active = true AND l.active = true
			GROUP BY pl.plan_id
		)

		-- Final SELECT with all related data
		SELECT
			p.id,
			p.name,
			p.description,
			p.active,
			p.date_created,
			p.date_created_string,
			p.date_modified,
			p.date_modified_string,
			COALESCE(pla.plan_locations, ARRAY[]::jsonb[]) as plan_locations
		FROM plan p
		LEFT JOIN plan_locations_agg pla ON p.id = pla.plan_id
		WHERE p.id = $1 AND p.active = true
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	var (
		id                 string
		name               string
		description        string
		active             bool
		dateCreated        sql.NullInt64
		dateCreatedString  sql.NullString
		dateModified       sql.NullInt64
		dateModifiedString sql.NullString
		planLocationsJSON  []byte
	)

	err := db.GetDB().QueryRowContext(ctx, query, req.PlanId).Scan(
		&id,
		&name,
		&description,
		&active,
		&dateCreated,
		&dateCreatedString,
		&dateModified,
		&dateModifiedString,
		&planLocationsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plan not found with ID: %s", req.PlanId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetPlanItemPageData query: %w", err)
	}

	// Build plan message
	plan := &planpb.Plan{
		Id:          &id,
		Name:        name,
		Description: &description,
		Active:      active,
	}

	if dateCreated.Valid {
		plan.DateCreated = &dateCreated.Int64
	}
	if dateCreatedString.Valid {
		plan.DateCreatedString = &dateCreatedString.String
	}
	if dateModified.Valid {
		plan.DateModified = &dateModified.Int64
	}
	if dateModifiedString.Valid {
		plan.DateModifiedString = &dateModifiedString.String
	}

	// Parse plan_locations JSON array
	if len(planLocationsJSON) > 0 {
		var planLocations []map[string]any
		if err := json.Unmarshal(planLocationsJSON, &planLocations); err == nil {
			// Convert to protobuf PlanLocation messages
			for _, plData := range planLocations {
				plJSON, _ := json.Marshal(plData)
				var planLocation planlocationpb.PlanLocation
				if err := protojson.Unmarshal(plJSON, &planLocation); err == nil {
					plan.PlanLocations = append(plan.PlanLocations, &planLocation)
				}
			}
		}
	}

	return &planpb.GetPlanItemPageDataResponse{
		Success: true,
		Plan:    plan,
	}, nil
}

// NewPlanRepository creates a new PostgreSQL plan repository (old-style constructor)
func NewPlanRepository(db *sql.DB, tableName string) planpb.PlanDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPlanRepository(dbOps, tableName)
}
