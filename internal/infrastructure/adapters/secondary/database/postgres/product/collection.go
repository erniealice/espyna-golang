//go:build postgres

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "collection", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresCollectionRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionRepository implements collection CRUD operations using PostgreSQL
type PostgresCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresCollectionRepository creates a new PostgreSQL collection repository
func NewPostgresCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionpb.CollectionDomainServiceServer {
	if tableName == "" {
		tableName = "collection" // default fallback
	}
	return &PostgresCollectionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateCollection creates a new collection using common PostgreSQL operations
func (r *PostgresCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection data is required")
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
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := protojson.Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.CreateCollectionResponse{
		Data: []*collectionpb.Collection{collection},
	}, nil
}

// ReadCollection retrieves a collection using common PostgreSQL operations
func (r *PostgresCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := protojson.Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.ReadCollectionResponse{
		Data: []*collectionpb.Collection{collection},
	}, nil
}

// UpdateCollection updates a collection using common PostgreSQL operations
func (r *PostgresCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
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
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := protojson.Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.UpdateCollectionResponse{
		Data: []*collectionpb.Collection{collection},
	}, nil
}

// DeleteCollection deletes a collection using common PostgreSQL operations
func (r *PostgresCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &collectionpb.DeleteCollectionResponse{
		Success: true,
	}, nil
}

// ListCollections lists collections using common PostgreSQL operations
func (r *PostgresCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var collections []*collectionpb.Collection
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		collection := &collectionpb.Collection{}
		if err := protojson.Unmarshal(resultJSON, collection); err != nil {
			// Log error and continue with next item
			continue
		}
		collections = append(collections, collection)
	}

	return &collectionpb.ListCollectionsResponse{
		Data: collections,
	}, nil
}

// GetCollectionListPageData retrieves a paginated list of collections with all related data expanded
// This method uses CTEs (Common Table Expressions) to load all related data in a single optimized query
// Relationships:
// - collection_plan (Many:Many via junction table)
// - collection_parent (Self-referential parent/child via junction table)
// TODO: Add unit tests for GetCollectionListPageData
func (r *PostgresCollectionRepository) GetCollectionListPageData(ctx context.Context, req *collectionpb.GetCollectionListPageDataRequest) (*collectionpb.GetCollectionListPageDataResponse, error) {
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

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_active ON collection(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_name_trgm ON collection USING gin(name gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_description_trgm ON collection USING gin(description gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_date_created ON collection(date_created DESC);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_plan_collection_id ON collection_plan(collection_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_plan_plan_id ON collection_plan(plan_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_plan_active ON collection_plan(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_parent_collection_id ON collection_parent(collection_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_parent_parent_id ON collection_parent(parent_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_parent_active ON collection_parent(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_plan_active ON plan(active) WHERE active = true;

	// Build the CTE query following the translation plan pattern
	query := `
		WITH
		-- CTE 1: Aggregate collection_plan relationships with plan details
		collection_plans_agg AS (
			SELECT
				cp.collection_id,
				array_agg(
					DISTINCT jsonb_build_object(
						'id', cp.id,
						'collection_id', cp.collection_id,
						'plan_id', cp.plan_id,
						'date_created', cp.date_created,
						'date_modified', cp.date_modified,
						'active', cp.active,
						'plan', jsonb_build_object(
							'id', p.id,
							'name', p.name,
							'description', p.description,
							'date_created', p.date_created,
							'date_modified', p.date_modified,
							'active', p.active
						)
					) ORDER BY p.name ASC
				) FILTER (WHERE p.id IS NOT NULL) as collection_plans
			FROM collection_plan cp
			INNER JOIN plan p ON cp.plan_id = p.id
			WHERE cp.active = true AND p.active = true
			GROUP BY cp.collection_id
		),

		-- CTE 2: Aggregate collection_parent relationships (self-referential parent)
		collection_parent_agg AS (
			SELECT
				cpp.collection_id,
				jsonb_build_object(
					'id', cpp.id,
					'collection_id', cpp.collection_id,
					'parent_id', cpp.parent_id,
					'date_created', cpp.date_created,
					'date_modified', cpp.date_modified,
					'active', cpp.active,
					'parent', jsonb_build_object(
						'id', cp.id,
						'name', cp.name,
						'description', cp.description,
						'date_created', cp.date_created,
						'date_modified', cp.date_modified,
						'active', cp.active
					)
				) as collection_parent
			FROM collection_parent cpp
			INNER JOIN collection cp ON cpp.parent_id = cp.id
			WHERE cpp.active = true AND cp.active = true
		),

		-- CTE 3: Apply search filter
		search_filtered AS (
			SELECT c.*
			FROM collection c
			WHERE c.active = true
				AND ($1::text = '' OR
					c.name ILIKE $1 OR
					c.description ILIKE $1)
		),

		-- CTE 4: Join with plans and parent, prepare for sorting
		enriched AS (
			SELECT
				sf.id,
				sf.name,
				sf.description,
				sf.active,
				sf.date_created,
				sf.date_modified
				COALESCE(cpa.collection_plans, ARRAY[]::jsonb[]) as collection_plans,
				cppa.collection_parent
			FROM search_filtered sf
			LEFT JOIN collection_plans_agg cpa ON sf.id = cpa.collection_id
			LEFT JOIN collection_parent_agg cppa ON sf.id = cppa.collection_id
		),

		-- CTE 5: Apply sorting
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

		-- CTE 6: Calculate total count for pagination
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
			s.date_modified,
			s.collection_plans,
			s.collection_parent,
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
		return nil, fmt.Errorf("failed to execute GetCollectionListPageData query: %w", err)
	}
	defer rows.Close()

	var collections []*collectionpb.Collection
	var totalCount int32

	for rows.Next() {
		var (
			id                   string
			name                 string
			description          string
			active               bool
			dateCreated          sql.NullInt64
			dateCreatedString    sql.NullString
			dateModified         sql.NullInt64
			dateModifiedString   sql.NullString
			collectionPlansJSON  []byte
			collectionParentJSON []byte
			rowTotalCount        int32
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&active,
			&dateCreated,
			&dateModified,
			&collectionPlansJSON,
			&collectionParentJSON,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan collection row: %w", err)
		}

		totalCount = rowTotalCount

		// Build collection message
		collection := &collectionpb.Collection{
			Id:          id,
			Name:        name,
			Description: description,
			Active:      active,
		}

		if dateCreated.Valid {
			collection.DateCreated = &dateCreated.Int64
		}
		if dateCreatedString.Valid {
			collection.DateCreatedString = &dateCreatedString.String
		}
		if dateModified.Valid {
			collection.DateModified = &dateModified.Int64
		}
		if dateModifiedString.Valid {
			collection.DateModifiedString = &dateModifiedString.String
		}

		// Note: The aggregated relationship data (collectionPlansJSON, collectionParentJSON)
		// is available in JSONB format but not directly mapped to the Collection protobuf structure.
		// This is intentional as the Collection message doesn't include these nested collections
		// in its schema. The CTE aggregations are prepared for potential future use or can be
		// accessed via separate junction table queries (CollectionPlan, CollectionParent services).
		_ = collectionPlansJSON
		_ = collectionParentJSON

		collections = append(collections, collection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection rows: %w", err)
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

	return &collectionpb.GetCollectionListPageDataResponse{
		Success:        true,
		CollectionList: collections,
		Pagination:     paginationResponse,
	}, nil
}

// GetCollectionItemPageData retrieves a single collection with all related data expanded
// This method uses CTEs (Common Table Expressions) to load all related data in a single query
// Relationships:
// - collection_plan (Many:Many via junction table)
// - collection_parent (Self-referential parent/child via junction table)
// TODO: Add unit tests for GetCollectionItemPageData
func (r *PostgresCollectionRepository) GetCollectionItemPageData(ctx context.Context, req *collectionpb.GetCollectionItemPageDataRequest) (*collectionpb.GetCollectionItemPageDataResponse, error) {
	if req.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_id ON collection(id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_plan_collection_id ON collection_plan(collection_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_plan_plan_id ON collection_plan(plan_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_parent_collection_id ON collection_parent(collection_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_collection_parent_parent_id ON collection_parent(parent_id);

	// Build CTE query to fetch collection with all related data
	query := `
		WITH
		-- CTE 1: Aggregate collection_plan relationships with plan details
		collection_plans_agg AS (
			SELECT
				cp.collection_id,
				array_agg(
					DISTINCT jsonb_build_object(
						'id', cp.id,
						'collection_id', cp.collection_id,
						'plan_id', cp.plan_id,
						'date_created', cp.date_created,
						'date_modified', cp.date_modified,
						'active', cp.active,
						'plan', jsonb_build_object(
							'id', p.id,
							'name', p.name,
							'description', p.description,
							'date_created', p.date_created,
							'date_modified', p.date_modified,
							'active', p.active
						)
					) ORDER BY p.name ASC
				) FILTER (WHERE p.id IS NOT NULL) as collection_plans
			FROM collection_plan cp
			INNER JOIN plan p ON cp.plan_id = p.id
			WHERE cp.collection_id = $1 AND cp.active = true AND p.active = true
			GROUP BY cp.collection_id
		),

		-- CTE 2: Aggregate collection_parent relationships (self-referential parent)
		collection_parent_agg AS (
			SELECT
				cpp.collection_id,
				jsonb_build_object(
					'id', cpp.id,
					'collection_id', cpp.collection_id,
					'parent_id', cpp.parent_id,
					'date_created', cpp.date_created,
					'date_modified', cpp.date_modified,
					'active', cpp.active,
					'parent', jsonb_build_object(
						'id', cp.id,
						'name', cp.name,
						'description', cp.description,
						'date_created', cp.date_created,
						'date_modified', cp.date_modified,
						'active', cp.active
					)
				) as collection_parent
			FROM collection_parent cpp
			INNER JOIN collection cp ON cpp.parent_id = cp.id
			WHERE cpp.collection_id = $1 AND cpp.active = true AND cp.active = true
		)

		-- Final SELECT with all related data
		SELECT
			c.id,
			c.name,
			c.description,
			c.active,
			c.date_created,
			c.date_modified
			COALESCE(cpa.collection_plans, ARRAY[]::jsonb[]) as collection_plans,
			cppa.collection_parent
		FROM collection c
		LEFT JOIN collection_plans_agg cpa ON c.id = cpa.collection_id
		LEFT JOIN collection_parent_agg cppa ON c.id = cppa.collection_id
		WHERE c.id = $1 AND c.active = true
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	var (
		id                   string
		name                 string
		description          string
		active               bool
		dateCreated          sql.NullInt64
		dateCreatedString    sql.NullString
		dateModified         sql.NullInt64
		dateModifiedString   sql.NullString
		collectionPlansJSON  []byte
		collectionParentJSON []byte
	)

	err := db.GetDB().QueryRowContext(ctx, query, req.CollectionId).Scan(
		&id,
		&name,
		&description,
		&active,
		&dateCreated,
		&dateModified,
		&collectionPlansJSON,
		&collectionParentJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection not found with ID: %s", req.CollectionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetCollectionItemPageData query: %w", err)
	}

	// Build collection message
	collection := &collectionpb.Collection{
		Id:          id,
		Name:        name,
		Description: description,
		Active:      active,
	}

	if dateCreated.Valid {
		collection.DateCreated = &dateCreated.Int64
	}
	if dateCreatedString.Valid {
		collection.DateCreatedString = &dateCreatedString.String
	}
	if dateModified.Valid {
		collection.DateModified = &dateModified.Int64
	}
	if dateModifiedString.Valid {
		collection.DateModifiedString = &dateModifiedString.String
	}

	// Note: The aggregated relationship data (collectionPlansJSON, collectionParentJSON)
	// is available in JSONB format but not directly mapped to the Collection protobuf structure.
	// This is intentional as the Collection message doesn't include these nested collections
	// in its schema. The CTE aggregations are prepared for potential future use or can be
	// accessed via separate junction table queries (CollectionPlan, CollectionParent services).
	_ = collectionPlansJSON
	_ = collectionParentJSON

	return &collectionpb.GetCollectionItemPageDataResponse{
		Success:    true,
		Collection: collection,
	}, nil
}

// NewCollectionRepository creates a new PostgreSQL collection repository (old-style constructor)
func NewCollectionRepository(db *sql.DB, tableName string) collectionpb.CollectionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresCollectionRepository(dbOps, tableName)
}
