//go:build postgres

package entity

import (
	"time"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "group", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres group repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresGroupRepository(dbOps, tableName), nil
	})
}

// PostgresGroupRepository implements group CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_group_active ON "group"(active) WHERE active = true - Filter active groups
//   - CREATE INDEX idx_group_name ON "group"(name) - Search on name field
//   - CREATE INDEX idx_group_name_trgm ON "group" USING gin(name gin_trgm_ops) - Fuzzy search support
//   - CREATE INDEX idx_group_date_created ON "group"(date_created DESC) - Default sorting
type PostgresGroupRepository struct {
	grouppb.UnimplementedGroupDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresGroupRepository creates a new PostgreSQL group repository
func NewPostgresGroupRepository(dbOps interfaces.DatabaseOperation, tableName string) grouppb.GroupDomainServiceServer {
	if tableName == "" {
		tableName = "group" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresGroupRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateGroup creates a new group using common PostgreSQL operations
func (r *PostgresGroupRepository) CreateGroup(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("group data is required")
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
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	group := &grouppb.Group{}
	if err := protojson.Unmarshal(resultJSON, group); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &grouppb.CreateGroupResponse{
		Data: []*grouppb.Group{group},
	}, nil
}

// ReadGroup retrieves a group using common PostgreSQL operations
func (r *PostgresGroupRepository) ReadGroup(ctx context.Context, req *grouppb.ReadGroupRequest) (*grouppb.ReadGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read group: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	group := &grouppb.Group{}
	if err := protojson.Unmarshal(resultJSON, group); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &grouppb.ReadGroupResponse{
		Data: []*grouppb.Group{group},
	}, nil
}

// UpdateGroup updates a group using common PostgreSQL operations
func (r *PostgresGroupRepository) UpdateGroup(ctx context.Context, req *grouppb.UpdateGroupRequest) (*grouppb.UpdateGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required")
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
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	group := &grouppb.Group{}
	if err := protojson.Unmarshal(resultJSON, group); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &grouppb.UpdateGroupResponse{
		Data: []*grouppb.Group{group},
	}, nil
}

// DeleteGroup deletes a group using common PostgreSQL operations
func (r *PostgresGroupRepository) DeleteGroup(ctx context.Context, req *grouppb.DeleteGroupRequest) (*grouppb.DeleteGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete group: %w", err)
	}

	return &grouppb.DeleteGroupResponse{
		Success: true,
	}, nil
}

// ListGroups lists groups using common PostgreSQL operations
func (r *PostgresGroupRepository) ListGroups(ctx context.Context, req *grouppb.ListGroupsRequest) (*grouppb.ListGroupsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var groups []*grouppb.Group
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		group := &grouppb.Group{}
		if err := protojson.Unmarshal(resultJSON, group); err != nil {
			// Log error and continue with next item
			continue
		}
		groups = append(groups, group)
	}

	return &grouppb.ListGroupsResponse{
		Data: groups,
	}, nil
}

// GetGroupListPageData retrieves groups with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresGroupRepository) GetGroupListPageData(
	ctx context.Context,
	req *grouppb.GetGroupListPageDataRequest,
) (*grouppb.GetGroupListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get group list page data request is required")
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
		// Handle offset pagination
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

	// CTE Query - Single round-trip with filtering and pagination
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on "group".active for filtering active records
	// - INDEX RECOMMENDATION: Create index on "group".name for search performance
	// - INDEX RECOMMENDATION: Create index on "group".date_created for default sorting
	query := `
		WITH enriched AS (
			SELECT
				g.id,
				g.name,
				g.description,
				g.active,
				g.date_created,
				g.date_modified
			FROM "group" g
			WHERE g.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   g.name ILIKE $1 OR
				   g.description ILIKE $1)
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
		return nil, fmt.Errorf("failed to query group list page data: %w", err)
	}
	defer rows.Close()

	var groups []*grouppb.Group
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			name               string
			description        *string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group row: %w", err)
		}

		totalCount = total

		group := &grouppb.Group{
			Id:     id,
			Name:   name,
			Active: active,
		}

		if description != nil {
			group.Description = *description
		}

		// Handle nullable timestamp fields

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		group.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		group.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		group.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		group.DateModifiedString = &dmStr
	}

		groups = append(groups, group)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &grouppb.GetGroupListPageDataResponse{
		GroupList: groups,
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

// GetGroupItemPageData retrieves a single group with enhanced item page data
func (r *PostgresGroupRepository) GetGroupItemPageData(
	ctx context.Context,
	req *grouppb.GetGroupItemPageDataRequest,
) (*grouppb.GetGroupItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get group item page data request is required")
	}
	if req.GroupId == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	// Simple query for single group item
	query := `
		SELECT
			g.id,
			g.name,
			g.description,
			g.active,
			g.date_created,
			g.date_modified
		FROM "group" g
		WHERE g.id = $1 AND g.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.GroupId)

	var (
		id                 string
		name               string
		description        *string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group with ID '%s' not found", req.GroupId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query group item page data: %w", err)
	}

	group := &grouppb.Group{
		Id:     id,
		Name:   name,
		Active: active,
	}

	if description != nil {
		group.Description = *description
	}

	// Handle nullable timestamp fields

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		group.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		group.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		group.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		group.DateModifiedString = &dmStr
	}

	return &grouppb.GetGroupItemPageDataResponse{
		Group:   group,
		Success: true,
	}, nil
}


// NewGroupRepository creates a new PostgreSQL group repository (old-style constructor)
func NewGroupRepository(db *sql.DB, tableName string) grouppb.GroupDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresGroupRepository(dbOps, tableName)
}
