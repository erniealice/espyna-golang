//go:build postgresql

package entity

import (
	"time"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "permission", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres permission repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPermissionRepository(dbOps, tableName), nil
	})
}

// PostgresPermissionRepository implements permission CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_permission_active ON permission(active) WHERE active = true - Filter active permissions
//   - CREATE INDEX idx_permission_name ON permission(name) - Search on name field
//   - CREATE INDEX idx_permission_name_trgm ON permission USING gin(name gin_trgm_ops) - Fuzzy search support
//   - CREATE INDEX idx_permission_date_created ON permission(date_created DESC) - Default sorting
type PostgresPermissionRepository struct {
	permissionpb.UnimplementedPermissionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresPermissionRepository creates a new PostgreSQL permission repository
func NewPostgresPermissionRepository(dbOps interfaces.DatabaseOperation, tableName string) permissionpb.PermissionDomainServiceServer {
	if tableName == "" {
		tableName = "permission" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPermissionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePermission creates a new permission using common PostgreSQL operations
func (r *PostgresPermissionRepository) CreatePermission(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("permission data is required")
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
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	permission := &permissionpb.Permission{}
	if err := protojson.Unmarshal(resultJSON, permission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &permissionpb.CreatePermissionResponse{
		Data: []*permissionpb.Permission{permission},
	}, nil
}

// ReadPermission retrieves a permission using common PostgreSQL operations
func (r *PostgresPermissionRepository) ReadPermission(ctx context.Context, req *permissionpb.ReadPermissionRequest) (*permissionpb.ReadPermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read permission: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	permission := &permissionpb.Permission{}
	if err := protojson.Unmarshal(resultJSON, permission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &permissionpb.ReadPermissionResponse{
		Data: []*permissionpb.Permission{permission},
	}, nil
}

// UpdatePermission updates a permission using common PostgreSQL operations
func (r *PostgresPermissionRepository) UpdatePermission(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required")
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
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	permission := &permissionpb.Permission{}
	if err := protojson.Unmarshal(resultJSON, permission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &permissionpb.UpdatePermissionResponse{
		Data: []*permissionpb.Permission{permission},
	}, nil
}

// DeletePermission deletes a permission using common PostgreSQL operations
func (r *PostgresPermissionRepository) DeletePermission(ctx context.Context, req *permissionpb.DeletePermissionRequest) (*permissionpb.DeletePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete permission: %w", err)
	}

	return &permissionpb.DeletePermissionResponse{
		Success: true,
	}, nil
}

// ListPermissions lists permissions using common PostgreSQL operations
func (r *PostgresPermissionRepository) ListPermissions(ctx context.Context, req *permissionpb.ListPermissionsRequest) (*permissionpb.ListPermissionsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var permissions []*permissionpb.Permission
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		permission := &permissionpb.Permission{}
		if err := protojson.Unmarshal(resultJSON, permission); err != nil {
			// Log error and continue with next item
			continue
		}
		permissions = append(permissions, permission)
	}

	return &permissionpb.ListPermissionsResponse{
		Data: permissions,
	}, nil
}

// GetPermissionListPageData retrieves permissions with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresPermissionRepository) GetPermissionListPageData(
	ctx context.Context,
	req *permissionpb.GetPermissionListPageDataRequest,
) (*permissionpb.GetPermissionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get permission list page data request is required")
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
	// - INDEX RECOMMENDATION: Create index on permission.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on permission.name for search performance
	// - INDEX RECOMMENDATION: Create index on permission.date_created for default sorting
	query := `
		WITH enriched AS (
			SELECT
				p.id,
				p.name,
				p.description,
				p.permission_code,
				p.permission_type,
				p.active,
				p.date_created,
				p.date_modified
			FROM permission p
			WHERE ($1::text IS NULL OR $1::text = '' OR
				   p.name ILIKE $1 OR
				   p.description ILIKE $1 OR
				   p.permission_code ILIKE $1)
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
		return nil, fmt.Errorf("failed to query permission list page data: %w", err)
	}
	defer rows.Close()

	var permissions []*permissionpb.Permission
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			name             string
			description      *string
			permissionCode   *string
			permissionType   *string
			active           bool
			dateCreated      time.Time
			dateModified     time.Time
			total            int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&permissionCode,
			&permissionType,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission row: %w", err)
		}

		totalCount = total

		permission := &permissionpb.Permission{
			Id:     id,
			Name:   name,
			Active: active,
		}

		if description != nil {
			permission.Description = *description
		}
		if permissionCode != nil {
			permission.PermissionCode = *permissionCode
		}
		if permissionType != nil {
			permission.PermissionType = parsePermissionType(*permissionType)
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			permission.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			permission.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			permission.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			permission.DateModifiedString = &dmStr
		}

		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permission rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &permissionpb.GetPermissionListPageDataResponse{
		PermissionList: permissions,
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

// GetPermissionItemPageData retrieves a single permission with enhanced item page data
func (r *PostgresPermissionRepository) GetPermissionItemPageData(
	ctx context.Context,
	req *permissionpb.GetPermissionItemPageDataRequest,
) (*permissionpb.GetPermissionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get permission item page data request is required")
	}
	if req.PermissionId == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	// Simple query for single permission item
	query := `
		SELECT
			p.id,
			p.name,
			p.description,
			p.permission_code,
			p.permission_type,
			p.active,
			p.date_created,
			p.date_modified
		FROM permission p
		WHERE p.id = $1
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.PermissionId)

	var (
		id             string
		name           string
		description    *string
		permissionCode *string
		permissionType *string
		active         bool
		dateCreated    time.Time
		dateModified   time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&permissionCode,
		&permissionType,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission with ID '%s' not found", req.PermissionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query permission item page data: %w", err)
	}

	permission := &permissionpb.Permission{
		Id:     id,
		Name:   name,
		Active: active,
	}

	if description != nil {
		permission.Description = *description
	}
	if permissionCode != nil {
		permission.PermissionCode = *permissionCode
	}
	if permissionType != nil {
		permission.PermissionType = parsePermissionType(*permissionType)
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		permission.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		permission.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		permission.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		permission.DateModifiedString = &dmStr
	}

	return &permissionpb.GetPermissionItemPageDataResponse{
		Permission: permission,
		Success:    true,
	}, nil
}

// parsePermissionType converts a database permission_type string to the protobuf enum.
func parsePermissionType(s string) permissionpb.PermissionType {
	switch s {
	case "PERMISSION_TYPE_DENY":
		return permissionpb.PermissionType_PERMISSION_TYPE_DENY
	default:
		return permissionpb.PermissionType_PERMISSION_TYPE_ALLOW
	}
}


// NewPermissionRepository creates a new PostgreSQL permission repository (old-style constructor)
func NewPermissionRepository(db *sql.DB, tableName string) permissionpb.PermissionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPermissionRepository(dbOps, tableName)
}
