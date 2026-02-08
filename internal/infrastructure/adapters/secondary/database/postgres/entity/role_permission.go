//go:build postgres

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
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "role_permission", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres role_permission repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresRolePermissionRepository(dbOps, tableName), nil
	})
}

// PostgresRolePermissionRepository implements role permission CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_role_permission_active ON role_permission(active) WHERE active = true - Filter active role permissions
//   - CREATE INDEX idx_role_permission_role_id ON role_permission(role_id) - Filter by role
//   - CREATE INDEX idx_role_permission_permission_id ON role_permission(permission_id) - Filter by permission
//   - CREATE INDEX idx_role_permission_date_created ON role_permission(date_created DESC) - Default sorting
type PostgresRolePermissionRepository struct {
	rolepermissionpb.UnimplementedRolePermissionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresRolePermissionRepository creates a new PostgreSQL role permission repository
func NewPostgresRolePermissionRepository(dbOps interfaces.DatabaseOperation, tableName string) rolepermissionpb.RolePermissionDomainServiceServer {
	if tableName == "" {
		tableName = "role_permission" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresRolePermissionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRolePermission creates a new role permission using common PostgreSQL operations
func (r *PostgresRolePermissionRepository) CreateRolePermission(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) (*rolepermissionpb.CreateRolePermissionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("role permission data is required")
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
		return nil, fmt.Errorf("failed to create role permission: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	rolePermission := &rolepermissionpb.RolePermission{}
	if err := protojson.Unmarshal(resultJSON, rolePermission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepermissionpb.CreateRolePermissionResponse{
		Data: []*rolepermissionpb.RolePermission{rolePermission},
	}, nil
}

// ReadRolePermission retrieves a role permission using common PostgreSQL operations
func (r *PostgresRolePermissionRepository) ReadRolePermission(ctx context.Context, req *rolepermissionpb.ReadRolePermissionRequest) (*rolepermissionpb.ReadRolePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read role permission: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	rolePermission := &rolepermissionpb.RolePermission{}
	if err := protojson.Unmarshal(resultJSON, rolePermission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepermissionpb.ReadRolePermissionResponse{
		Data: []*rolepermissionpb.RolePermission{rolePermission},
	}, nil
}

// UpdateRolePermission updates a role permission using common PostgreSQL operations
func (r *PostgresRolePermissionRepository) UpdateRolePermission(ctx context.Context, req *rolepermissionpb.UpdateRolePermissionRequest) (*rolepermissionpb.UpdateRolePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required")
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
		return nil, fmt.Errorf("failed to update role permission: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	rolePermission := &rolepermissionpb.RolePermission{}
	if err := protojson.Unmarshal(resultJSON, rolePermission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepermissionpb.UpdateRolePermissionResponse{
		Data: []*rolepermissionpb.RolePermission{rolePermission},
	}, nil
}

// DeleteRolePermission deletes a role permission using common PostgreSQL operations
func (r *PostgresRolePermissionRepository) DeleteRolePermission(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) (*rolepermissionpb.DeleteRolePermissionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete role permission: %w", err)
	}

	return &rolepermissionpb.DeleteRolePermissionResponse{
		Success: true,
	}, nil
}

// ListRolePermissions lists role permissions using common PostgreSQL operations
func (r *PostgresRolePermissionRepository) ListRolePermissions(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) (*rolepermissionpb.ListRolePermissionsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role permissions: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var rolePermissions []*rolepermissionpb.RolePermission
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		rolePermission := &rolepermissionpb.RolePermission{}
		if err := protojson.Unmarshal(resultJSON, rolePermission); err != nil {
			// Log error and continue with next item
			continue
		}
		rolePermissions = append(rolePermissions, rolePermission)
	}

	return &rolepermissionpb.ListRolePermissionsResponse{
		Data: rolePermissions,
	}, nil
}

// GetRolePermissionListPageData retrieves role permissions with advanced filtering, sorting, and pagination using CTE
func (r *PostgresRolePermissionRepository) GetRolePermissionListPageData(
	ctx context.Context,
	req *rolepermissionpb.GetRolePermissionListPageDataRequest,
) (*rolepermissionpb.GetRolePermissionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role permission list page data request is required")
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

	// Build WHERE clause for filtering
	whereClause := "WHERE rp.active = true"
	args := []interface{}{limit, offset}

	// Note: Removed direct RoleId/PermissionId filtering as GetRolePermissionListPageDataRequest doesn't have these fields
	// Filtering should be done through req.Filters field instead

	// CTE Query - Junction table pattern
	query := `
		WITH enriched AS (
			SELECT
				rp.id,
				rp.role_id,
				rp.permission_id,
				rp.active,
				rp.date_created,
				rp.date_modified
			FROM role_permission rp
			` + whereClause + `
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query role permission list page data: %w", err)
	}
	defer rows.Close()

	var rolePermissions []*rolepermissionpb.RolePermission
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			roleID             string
			permissionID       string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)

		err := rows.Scan(
			&id,
			&roleID,
			&permissionID,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role permission row: %w", err)
		}

		totalCount = total

		rolePermission := &rolepermissionpb.RolePermission{
			Id:           id,
			RoleId:       roleID,
			PermissionId: permissionID,
			Active:       active,
		}

		// Handle nullable timestamp fields

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		rolePermission.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		rolePermission.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		rolePermission.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		rolePermission.DateModifiedString = &dmStr
	}

		rolePermissions = append(rolePermissions, rolePermission)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role permission rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &rolepermissionpb.GetRolePermissionListPageDataResponse{
		RolePermissionList: rolePermissions,
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

// GetRolePermissionItemPageData retrieves a single role permission with enhanced item page data
func (r *PostgresRolePermissionRepository) GetRolePermissionItemPageData(
	ctx context.Context,
	req *rolepermissionpb.GetRolePermissionItemPageDataRequest,
) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role permission item page data request is required")
	}
	if req.RolePermissionId == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	// Simple query for single role permission item
	query := `
		SELECT
			rp.id,
			rp.role_id,
			rp.permission_id,
			rp.active,
			rp.date_created,
			rp.date_modified
		FROM role_permission rp
		WHERE rp.id = $1 AND rp.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.RolePermissionId)

	var (
		id                 string
		roleID             string
		permissionID       string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
	)

	err := row.Scan(
		&id,
		&roleID,
		&permissionID,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role permission with ID '%s' not found", req.RolePermissionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query role permission item page data: %w", err)
	}

	rolePermission := &rolepermissionpb.RolePermission{
		Id:           id,
		RoleId:       roleID,
		PermissionId: permissionID,
		Active:       active,
	}

	// Handle nullable timestamp fields

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		rolePermission.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		rolePermission.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		rolePermission.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		rolePermission.DateModifiedString = &dmStr
	}

	return &rolepermissionpb.GetRolePermissionItemPageDataResponse{
		RolePermission: rolePermission,
		Success:        true,
	}, nil
}


// NewRolePermissionRepository creates a new PostgreSQL role_permission repository (old-style constructor)
func NewRolePermissionRepository(db *sql.DB, tableName string) rolepermissionpb.RolePermissionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresRolePermissionRepository(dbOps, tableName)
}
