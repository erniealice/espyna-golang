//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Collection, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerCollectionRepository(dbOps, tableName), nil
	})
}

// SQLServerCollectionRepository implements collection CRUD using SQL Server.
type SQLServerCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerCollectionRepository creates a new SQL Server collection repository.
func NewSQLServerCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionpb.CollectionDomainServiceServer {
	if tableName == "" {
		tableName = "collection"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerCollectionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
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
	return &collectionpb.CreateCollectionResponse{Data: []*collectionpb.Collection{collection}}, nil
}

func (r *SQLServerCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
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
	return &collectionpb.ReadCollectionResponse{Data: []*collectionpb.Collection{collection}}, nil
}

func (r *SQLServerCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
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
	return &collectionpb.UpdateCollectionResponse{Data: []*collectionpb.Collection{collection}}, nil
}

func (r *SQLServerCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}
	return &collectionpb.DeleteCollectionResponse{Success: true}, nil
}

func (r *SQLServerCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
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
	return &collectionpb.ListCollectionsResponse{Data: collections}, nil
}

// GetCollectionListPageData retrieves a paginated list of collections with related
// data expanded using SQL Server CTEs.
//
// SQL Server translation notes:
//   - `array_agg(DISTINCT jsonb_build_object(...) ORDER BY ...)` →
//     dedupe sub-CTE + `FOR JSON PATH` subquery
//   - `ILIKE` → `LIKE` (SQL Server LIKE is case-insensitive with default collation)
//   - `COALESCE(cpa.collection_plans, ARRAY[]::jsonb[])` → NULL handled via `ISNULL`
//   - `LIMIT $2 OFFSET $3` → `OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY`
//   - `active = true` → `active = 1`
func (r *SQLServerCollectionRepository) GetCollectionListPageData(ctx context.Context, req *collectionpb.GetCollectionListPageDataRequest) (*collectionpb.GetCollectionListPageDataResponse, error) {
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

	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	// SQL Server CTE translation:
	// - collection_plan aggregation uses FOR JSON PATH sub-CTE (with dedupe) instead of array_agg(DISTINCT ...)
	// - collection_parent uses FOR JSON PATH single-row subquery
	// - LIKE instead of ILIKE
	// - active = 1 (BIT)
	// - OFFSET/FETCH pagination
	// - Sort field is safe: validated against the allowed set below.
	allowedSortFields := map[string]bool{
		"name": true, "description": true, "date_created": true, "date_modified": true,
	}
	if !allowedSortFields[sortField] {
		sortField = "date_created"
	}

	query := `
		WITH
		-- CTE 1: Dedupe collection_plan rows (SQL Server has no DISTINCT in FOR JSON PATH directly)
		collection_plan_dedup AS (
			SELECT DISTINCT
				cp.id,
				cp.collection_id,
				cp.plan_id,
				cp.date_created,
				cp.date_modified,
				cp.active,
				p.id         AS plan_id_val,
				p.name       AS plan_name,
				p.description AS plan_description,
				p.date_created AS plan_date_created,
				p.date_modified AS plan_date_modified,
				p.active     AS plan_active
			FROM collection_plan cp
			INNER JOIN plan p ON cp.plan_id = p.id
			WHERE cp.active = 1 AND p.active = 1
		),
		-- CTE 2: Aggregate collection_plan relationships with plan details (JSON)
		collection_plans_agg AS (
			SELECT
				cpd.collection_id,
				(
					SELECT
						cpd2.id              AS id,
						cpd2.collection_id   AS collection_id,
						cpd2.plan_id         AS plan_id,
						cpd2.date_created    AS date_created,
						cpd2.date_modified   AS date_modified,
						cpd2.active          AS active,
						cpd2.plan_id_val     AS [plan.id],
						cpd2.plan_name       AS [plan.name],
						cpd2.plan_description AS [plan.description],
						cpd2.plan_date_created AS [plan.date_created],
						cpd2.plan_date_modified AS [plan.date_modified],
						cpd2.plan_active     AS [plan.active]
					FROM collection_plan_dedup cpd2
					WHERE cpd2.collection_id = cpd.collection_id
					ORDER BY cpd2.plan_name ASC
					FOR JSON PATH
				) AS collection_plans_json
			FROM collection_plan_dedup cpd
			GROUP BY cpd.collection_id
		),
		-- CTE 3: Apply search filter
		search_filtered AS (
			SELECT c.*
			FROM collection c
			WHERE c.active = 1
				AND (@p1 = '' OR
					c.name LIKE @p1 OR
					c.description LIKE @p1)
		),
		-- CTE 4: Join with plans and parent, prepare for sorting
		enriched AS (
			SELECT
				sf.id,
				sf.name,
				sf.description,
				sf.active,
				sf.date_created,
				sf.date_modified,
				cpa.collection_plans_json,
				(
					SELECT TOP 1
						cpp.id              AS id,
						cpp.collection_id   AS collection_id,
						cpp.parent_id       AS parent_id,
						cpp.date_created    AS date_created,
						cpp.date_modified   AS date_modified,
						cpp.active          AS active,
						cp2.id              AS [parent.id],
						cp2.name            AS [parent.name],
						cp2.description     AS [parent.description],
						cp2.date_created    AS [parent.date_created],
						cp2.date_modified   AS [parent.date_modified],
						cp2.active          AS [parent.active]
					FROM collection_parent cpp
					INNER JOIN collection cp2 ON cpp.parent_id = cp2.id
					WHERE cpp.collection_id = sf.id AND cpp.active = 1 AND cp2.active = 1
					FOR JSON PATH, WITHOUT_ARRAY_WRAPPER
				) AS collection_parent_json
			FROM search_filtered sf
			LEFT JOIN collection_plans_agg cpa ON sf.id = cpa.collection_id
		),
		-- CTE 5: Calculate total count for pagination
		total_count AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.id,
			e.name,
			e.description,
			e.active,
			e.date_created,
			e.date_modified,
			e.collection_plans_json,
			e.collection_parent_json,
			tc.total AS _total_count
		FROM enriched e
		CROSS JOIN total_count tc
		ORDER BY
			CASE WHEN @p4 = 'name'         AND @p5 = 'ASC'  THEN e.name         END ASC,
			CASE WHEN @p4 = 'name'         AND @p5 = 'DESC' THEN e.name         END DESC,
			CASE WHEN @p4 = 'description'  AND @p5 = 'ASC'  THEN e.description  END ASC,
			CASE WHEN @p4 = 'description'  AND @p5 = 'DESC' THEN e.description  END DESC,
			CASE WHEN (@p4 = 'date_created' OR @p4 = '') AND @p5 = 'DESC' THEN e.date_created END DESC,
			CASE WHEN @p4 = 'date_created' AND @p5 = 'ASC'  THEN e.date_created END ASC
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	rows, err := r.db.QueryContext(ctx, query,
		searchQuery,   // @p1
		limit,         // @p2
		offset,        // @p3
		sortField,     // @p4
		sortDirection, // @p5
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
			dateModified         sql.NullInt64
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

		// Aggregated relationship data available in JSON format — not mapped to
		// Collection proto (same intentional stub as postgres gold standard).
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

// GetCollectionItemPageData retrieves a single collection with all related data
// expanded using SQL Server CTEs.
func (r *SQLServerCollectionRepository) GetCollectionItemPageData(ctx context.Context, req *collectionpb.GetCollectionItemPageDataRequest) (*collectionpb.GetCollectionItemPageDataResponse, error) {
	if req.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	query := `
		WITH
		collection_plan_dedup AS (
			SELECT DISTINCT
				cp.id,
				cp.collection_id,
				cp.plan_id,
				cp.date_created,
				cp.date_modified,
				cp.active,
				p.id          AS plan_id_val,
				p.name        AS plan_name,
				p.description AS plan_description,
				p.date_created AS plan_date_created,
				p.date_modified AS plan_date_modified,
				p.active      AS plan_active
			FROM collection_plan cp
			INNER JOIN plan p ON cp.plan_id = p.id
			WHERE cp.collection_id = @p1 AND cp.active = 1 AND p.active = 1
		)
		SELECT
			c.id,
			c.name,
			c.description,
			c.active,
			c.date_created,
			c.date_modified,
			(
				SELECT
					cpd.id              AS id,
					cpd.collection_id   AS collection_id,
					cpd.plan_id         AS plan_id,
					cpd.date_created    AS date_created,
					cpd.date_modified   AS date_modified,
					cpd.active          AS active,
					cpd.plan_id_val     AS [plan.id],
					cpd.plan_name       AS [plan.name],
					cpd.plan_description AS [plan.description],
					cpd.plan_date_created AS [plan.date_created],
					cpd.plan_date_modified AS [plan.date_modified],
					cpd.plan_active     AS [plan.active]
				FROM collection_plan_dedup cpd
				ORDER BY cpd.plan_name ASC
				FOR JSON PATH
			) AS collection_plans_json,
			(
				SELECT TOP 1
					cpp.id              AS id,
					cpp.collection_id   AS collection_id,
					cpp.parent_id       AS parent_id,
					cpp.date_created    AS date_created,
					cpp.date_modified   AS date_modified,
					cpp.active          AS active,
					cp2.id              AS [parent.id],
					cp2.name            AS [parent.name],
					cp2.description     AS [parent.description],
					cp2.date_created    AS [parent.date_created],
					cp2.date_modified   AS [parent.date_modified],
					cp2.active          AS [parent.active]
				FROM collection_parent cpp
				INNER JOIN collection cp2 ON cpp.parent_id = cp2.id
				WHERE cpp.collection_id = @p1 AND cpp.active = 1 AND cp2.active = 1
				FOR JSON PATH, WITHOUT_ARRAY_WRAPPER
			) AS collection_parent_json
		FROM collection c
		WHERE c.id = @p1 AND c.active = 1
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

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

	err := r.db.QueryRowContext(ctx, query, req.CollectionId).Scan(
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

	// Aggregated relationship data available in JSON format — not mapped to
	// Collection proto (same intentional stub as postgres gold standard).
	_ = collectionPlansJSON
	_ = collectionParentJSON

	return &collectionpb.GetCollectionItemPageDataResponse{
		Success:    true,
		Collection: collection,
	}, nil
}
