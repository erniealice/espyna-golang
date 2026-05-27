//go:build mysql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/consumer"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Collection, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLCollectionRepository(dbOps, tableName), nil
	})
}

// MySQLCollectionRepository implements collection CRUD operations using MySQL 8.0+.
type MySQLCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLCollectionRepository creates a new MySQL collection repository.
func NewMySQLCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionpb.CollectionDomainServiceServer {
	if tableName == "" {
		tableName = "collection" // default fallback
	}
	return &MySQLCollectionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateCollection creates a new collection using common MySQL operations.
func (r *MySQLCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection data is required")
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
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.CreateCollectionResponse{
		Data: []*collectionpb.Collection{collection},
	}, nil
}

// ReadCollection retrieves a collection using common MySQL operations.
func (r *MySQLCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.ReadCollectionResponse{
		Data: []*collectionpb.Collection{collection},
	}, nil
}

// UpdateCollection updates a collection using common MySQL operations.
func (r *MySQLCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
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
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.UpdateCollectionResponse{
		Data: []*collectionpb.Collection{collection},
	}, nil
}

// DeleteCollection deletes a collection using common MySQL operations (soft delete).
func (r *MySQLCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &collectionpb.DeleteCollectionResponse{
		Success: true,
	}, nil
}

// ListCollections lists collections using common MySQL operations.
func (r *MySQLCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var collections []*collectionpb.Collection
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		collection := &collectionpb.Collection{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
			continue
		}
		collections = append(collections, collection)
	}

	return &collectionpb.ListCollectionsResponse{
		Data: collections,
	}, nil
}

var collectionSortableSQLCols = []string{
	"id", "name", "description", "active", "date_created", "date_modified",
}

// GetCollectionListPageData retrieves a paginated list of collections with all
// related data expanded.
//
// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders, args in same left-to-right order)
//   - ILIKE → LIKE (MySQL ci collation handles case-insensitivity)
//   - array_agg(DISTINCT jsonb_build_object(...) ORDER BY ...) FILTER (WHERE ...) →
//     JSON_ARRAYAGG(JSON_OBJECT(...)) with a dedupe sub-CTE ordered before
//     aggregation (MySQL 8.0 JSON_ARRAYAGG has no DISTINCT/ORDER BY; dedupe +
//     order in a subquery, then aggregate in the outer CTE).
//   - COALESCE(..., ARRAY[]::jsonb[]) → COALESCE(..., JSON_ARRAY())
//   - LIMIT/OFFSET use positional ?
//   - core.BuildOrderBy used for sort (backtick quoting)
//   - COUNT(*) OVER () — MySQL 8.0+ supports window functions
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLCollectionRepository) GetCollectionListPageData(ctx context.Context, req *collectionpb.GetCollectionListPageDataRequest) (*collectionpb.GetCollectionListPageDataResponse, error) {
	// Extract workspace_id from context (REQUIRED for multi-tenancy).
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Extract pagination parameters with defaults.
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	// Extract search query.
	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	// Build sort — fail-closed against per-entity whitelist (A2 guard).
	orderByClause, err := mysqlCore.BuildOrderBy(collectionSortableSQLCols, req.GetSort(), "`date_created` DESC")
	if err != nil {
		return nil, err
	}

	// Build filter/search WHERE clauses.
	searchFields := []string{"c.name", "c.description"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE c.active = 1"
	if workspaceID != "" {
		whereSQL += " AND c.workspace_id = ?"
	}
	if searchQuery != "" {
		whereSQL += " AND (c.name LIKE ? OR c.description LIKE ?)"
	}
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	// Build args list:
	// [workspaceID (optional), searchQuery x2 (optional), ...filterArgs, limit, offset]
	queryArgs := []any{}
	if workspaceID != "" {
		queryArgs = append(queryArgs, workspaceID)
	}
	if searchQuery != "" {
		queryArgs = append(queryArgs, searchQuery, searchQuery)
	}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

	// CTE query — MySQL 8.0+ supports CTEs and COUNT(*) OVER().
	//
	// collection_plans_agg: MySQL JSON_ARRAYAGG has no DISTINCT/ORDER BY in 8.0.x.
	// We dedupe + order in a sub-CTE (plans_ordered), then aggregate in plans_agg.
	// COALESCE(... , JSON_ARRAY()) replaces COALESCE(..., ARRAY[]::jsonb[]).
	//
	// collection_parent_agg: JSON_OBJECT replaces jsonb_build_object.
	query := fmt.Sprintf(`
		WITH
		-- Sub-CTE: dedupe + order plan rows before aggregation (MySQL JSON_ARRAYAGG
		-- has no DISTINCT/ORDER BY; pre-sort ensures deterministic output).
		plans_ordered AS (
			SELECT
				cp.collection_id,
				cp.id        AS cp_id,
				cp.plan_id,
				cp.date_created  AS cp_date_created,
				cp.date_modified AS cp_date_modified,
				cp.active    AS cp_active,
				p.id         AS p_id,
				p.name       AS p_name,
				p.description AS p_description,
				p.date_created  AS p_date_created,
				p.date_modified AS p_date_modified,
				p.active     AS p_active,
				ROW_NUMBER() OVER (PARTITION BY cp.collection_id, cp.plan_id ORDER BY p.name ASC) AS rn
			FROM collection_plan cp
			INNER JOIN plan p ON cp.plan_id = p.id
			WHERE cp.active = 1 AND p.active = 1
		),

		-- CTE 1: Aggregate deduplicated plan rows per collection.
		collection_plans_agg AS (
			SELECT
				collection_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', cp_id,
						'collection_id', collection_id,
						'plan_id', plan_id,
						'date_created', cp_date_created,
						'date_modified', cp_date_modified,
						'active', cp_active,
						'plan', JSON_OBJECT(
							'id', p_id,
							'name', p_name,
							'description', p_description,
							'date_created', p_date_created,
							'date_modified', p_date_modified,
							'active', p_active
						)
					)
				) AS collection_plans
			FROM plans_ordered
			WHERE rn = 1
			GROUP BY collection_id
		),

		-- CTE 2: Aggregate collection_parent relationships (self-referential parent).
		collection_parent_agg AS (
			SELECT
				cpp.collection_id,
				JSON_OBJECT(
					'id', cpp.id,
					'collection_id', cpp.collection_id,
					'parent_id', cpp.parent_id,
					'date_created', cpp.date_created,
					'date_modified', cpp.date_modified,
					'active', cpp.active,
					'parent', JSON_OBJECT(
						'id', cp.id,
						'name', cp.name,
						'description', cp.description,
						'date_created', cp.date_created,
						'date_modified', cp.date_modified,
						'active', cp.active
					)
				) AS collection_parent
			FROM collection_parent cpp
			INNER JOIN collection cp ON cpp.parent_id = cp.id
			WHERE cpp.active = 1 AND cp.active = 1
		),

		-- CTE 3: Apply search/workspace filter.
		enriched AS (
			SELECT
				c.id,
				c.name,
				c.description,
				c.active,
				c.date_created,
				c.date_modified,
				COALESCE(cpa.collection_plans, JSON_ARRAY()) AS collection_plans,
				cppa.collection_parent
			FROM collection c
			LEFT JOIN collection_plans_agg cpa ON c.id = cpa.collection_id
			LEFT JOIN collection_parent_agg cppa ON c.id = cppa.collection_id
			%s
		),

		-- CTE 4: Count total rows for pagination (before LIMIT/OFFSET).
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)

		-- Final SELECT with pagination.
		SELECT
			e.id,
			e.name,
			e.description,
			e.active,
			e.date_created,
			e.date_modified,
			e.collection_plans,
			e.collection_parent,
			c.total
		FROM enriched e, counted c
		%s
		LIMIT ? OFFSET ?;
	`, whereSQL, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
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
			dateModified         sql.NullInt64
			collectionPlansJSON  []byte
			collectionParentJSON []byte
			rowTotalCount        int32
		)

		if err := rows.Scan(
			&id,
			&name,
			&description,
			&active,
			&dateCreated,
			&dateModified,
			&collectionPlansJSON,
			&collectionParentJSON,
			&rowTotalCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection row: %w", err)
		}

		totalCount = rowTotalCount

		collection := &collectionpb.Collection{
			Id:          id,
			Name:        name,
			Description: description,
			Active:      active,
		}

		if dateCreated.Valid {
			collection.DateCreated = &dateCreated.Int64
		}
		if dateModified.Valid {
			collection.DateModified = &dateModified.Int64
		}

		// JSON aggregations are available for future use; not mapped to proto yet.
		_ = collectionPlansJSON
		_ = collectionParentJSON

		collections = append(collections, collection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection rows: %w", err)
	}

	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	return &collectionpb.GetCollectionListPageDataResponse{
		Success:        true,
		CollectionList: collections,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalCount,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetCollectionItemPageData retrieves a single collection with all related data expanded.
//
// Dialect translation from postgres gold standard:
//   - $1 → ? (MySQL positional placeholder)
//   - ARRAY[]::jsonb[] → JSON_ARRAY()
//   - jsonb_build_object → JSON_OBJECT
//   - array_agg(DISTINCT ... ORDER BY ...) FILTER (WHERE ...) → sub-CTE dedupe + JSON_ARRAYAGG
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLCollectionRepository) GetCollectionItemPageData(ctx context.Context, req *collectionpb.GetCollectionItemPageDataRequest) (*collectionpb.GetCollectionItemPageDataResponse, error) {
	if req.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Sub-CTE dedupe approach: MySQL JSON_ARRAYAGG has no DISTINCT/ORDER BY.
	// Deduplicate + order in plans_ordered (ROW_NUMBER), aggregate in plans_agg.
	query := `
		WITH
		plans_ordered AS (
			SELECT
				cp.collection_id,
				cp.id        AS cp_id,
				cp.plan_id,
				cp.date_created  AS cp_date_created,
				cp.date_modified AS cp_date_modified,
				cp.active    AS cp_active,
				p.id         AS p_id,
				p.name       AS p_name,
				p.description AS p_description,
				p.date_created  AS p_date_created,
				p.date_modified AS p_date_modified,
				p.active     AS p_active,
				ROW_NUMBER() OVER (PARTITION BY cp.collection_id, cp.plan_id ORDER BY p.name ASC) AS rn
			FROM collection_plan cp
			INNER JOIN plan p ON cp.plan_id = p.id
			WHERE cp.collection_id = ? AND cp.active = 1 AND p.active = 1
		),
		collection_plans_agg AS (
			SELECT
				collection_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', cp_id,
						'collection_id', collection_id,
						'plan_id', plan_id,
						'date_created', cp_date_created,
						'date_modified', cp_date_modified,
						'active', cp_active,
						'plan', JSON_OBJECT(
							'id', p_id,
							'name', p_name,
							'description', p_description,
							'date_created', p_date_created,
							'date_modified', p_date_modified,
							'active', p_active
						)
					)
				) AS collection_plans
			FROM plans_ordered
			WHERE rn = 1
			GROUP BY collection_id
		),
		collection_parent_agg AS (
			SELECT
				cpp.collection_id,
				JSON_OBJECT(
					'id', cpp.id,
					'collection_id', cpp.collection_id,
					'parent_id', cpp.parent_id,
					'date_created', cpp.date_created,
					'date_modified', cpp.date_modified,
					'active', cpp.active,
					'parent', JSON_OBJECT(
						'id', cp.id,
						'name', cp.name,
						'description', cp.description,
						'date_created', cp.date_created,
						'date_modified', cp.date_modified,
						'active', cp.active
					)
				) AS collection_parent
			FROM collection_parent cpp
			INNER JOIN collection cp ON cpp.parent_id = cp.id
			WHERE cpp.collection_id = ? AND cpp.active = 1 AND cp.active = 1
		)
		SELECT
			c.id,
			c.name,
			c.description,
			c.active,
			c.date_created,
			c.date_modified,
			COALESCE(cpa.collection_plans, JSON_ARRAY()) AS collection_plans,
			cppa.collection_parent
		FROM collection c
		LEFT JOIN collection_plans_agg cpa ON c.id = cpa.collection_id
		LEFT JOIN collection_parent_agg cppa ON c.id = cppa.collection_id
		WHERE c.id = ? AND c.active = 1
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	// Arg order: collectionId (for plans_ordered WHERE), collectionId (for
	// collection_parent_agg WHERE), collectionId (for final WHERE).
	row := exec.QueryRowContext(ctx, query, req.CollectionId, req.CollectionId, req.CollectionId)

	var (
		id                   string
		name                 string
		description          string
		active               bool
		dateCreated          sql.NullInt64
		dateModified         sql.NullInt64
		collectionPlansJSON  []byte
		collectionParentJSON []byte
	)

	err := row.Scan(
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

	collection := &collectionpb.Collection{
		Id:          id,
		Name:        name,
		Description: description,
		Active:      active,
	}

	if dateCreated.Valid {
		collection.DateCreated = &dateCreated.Int64
	}
	if dateModified.Valid {
		collection.DateModified = &dateModified.Int64
	}

	// JSON aggregations are available for future use; not mapped to proto yet.
	_ = collectionPlansJSON
	_ = collectionParentJSON

	return &collectionpb.GetCollectionItemPageDataResponse{
		Success:    true,
		Collection: collection,
	}, nil
}

// NewCollectionRepository creates a new MySQL collection repository (old-style constructor).
func NewCollectionRepository(db *sql.DB, tableName string) collectionpb.CollectionDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLCollectionRepository(dbOps, tableName)
}
