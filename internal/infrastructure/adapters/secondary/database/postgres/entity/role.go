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
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "role", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres role repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresRoleRepository(dbOps, tableName), nil
	})
}

// PostgresRoleRepository implements role CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_role_workspace_id ON role(workspace_id) - Multi-tenancy filter
//   - CREATE INDEX idx_role_active ON role(active) WHERE active = true - Filter active roles
//   - CREATE INDEX idx_role_name ON role(name) - Search performance
//   - CREATE INDEX idx_role_date_created ON role(date_created DESC) - Default sorting
//   - CREATE INDEX idx_role_permission_role_id ON role_permission(role_id) - Junction table lookup
//   - CREATE INDEX idx_role_permission_permission_id ON role_permission(permission_id) - Junction table FK
//   - CREATE INDEX idx_role_permission_active ON role_permission(active) - Filter active junction records
//   - CREATE INDEX idx_permission_active ON permission(active) - Filter active permissions
type PostgresRoleRepository struct {
	rolepb.UnimplementedRoleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresRoleRepository creates a new PostgreSQL role repository
func NewPostgresRoleRepository(dbOps interfaces.DatabaseOperation, tableName string) rolepb.RoleDomainServiceServer {
	if tableName == "" {
		tableName = "role" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresRoleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRole creates a new role using common PostgreSQL operations
func (r *PostgresRoleRepository) CreateRole(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("role data is required")
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
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	role := &rolepb.Role{}
	if err := protojson.Unmarshal(resultJSON, role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepb.CreateRoleResponse{
		Data: []*rolepb.Role{role},
	}, nil
}

// ReadRole retrieves a role using common PostgreSQL operations
func (r *PostgresRoleRepository) ReadRole(ctx context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read role: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	role := &rolepb.Role{}
	if err := protojson.Unmarshal(resultJSON, role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepb.ReadRoleResponse{
		Data: []*rolepb.Role{role},
	}, nil
}

// UpdateRole updates a role using common PostgreSQL operations
func (r *PostgresRoleRepository) UpdateRole(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
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
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	role := &rolepb.Role{}
	if err := protojson.Unmarshal(resultJSON, role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepb.UpdateRoleResponse{
		Data: []*rolepb.Role{role},
	}, nil
}

// DeleteRole deletes a role using common PostgreSQL operations
func (r *PostgresRoleRepository) DeleteRole(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete role: %w", err)
	}

	return &rolepb.DeleteRoleResponse{
		Success: true,
	}, nil
}

// ListRoles lists roles using common PostgreSQL operations
func (r *PostgresRoleRepository) ListRoles(ctx context.Context, req *rolepb.ListRolesRequest) (*rolepb.ListRolesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var roles []*rolepb.Role
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		role := &rolepb.Role{}
		if err := protojson.Unmarshal(resultJSON, role); err != nil {
			// Log error and continue with next item
			continue
		}
		roles = append(roles, role)
	}

	return &rolepb.ListRolesResponse{
		Data: roles,
	}, nil
}

// GetRoleListPageData retrieves roles with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresRoleRepository) GetRoleListPageData(
	ctx context.Context,
	req *rolepb.GetRoleListPageDataRequest,
) (*rolepb.GetRoleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role list page data request is required")
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

	// CTE Query - Single round-trip with enriched role_permission data
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on role.workspace_id (multi-tenancy filter)
	// - INDEX RECOMMENDATION: Create index on role_permission.role_id (junction table lookup)
	// - INDEX RECOMMENDATION: Create index on role_permission.permission_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on role_permission.active (filter active junction records)
	// - INDEX RECOMMENDATION: Create index on permission.active (filter active permissions)
	query := `
		WITH role_permissions_agg AS (
			SELECT
				rp.role_id,
				jsonb_agg(
					jsonb_build_object(
						'id', rp.id,
						'role_id', rp.role_id,
						'permission_id', rp.permission_id,
						'permission', jsonb_build_object(
							'id', p.id,
							'name', p.name,
							'description', p.description,
							'active', p.active
						),
						'active', rp.active
					) ORDER BY p.name
				) FILTER (WHERE rp.id IS NOT NULL) as permissions
			FROM role_permission rp
			JOIN permission p ON rp.permission_id = p.id
			WHERE rp.active = true AND p.active = true
			GROUP BY rp.role_id
		),
		enriched AS (
			SELECT
				r.id,
				r.workspace_id,
				r.name,
				r.description,
				r.color,
				r.active,
				r.date_created,
				r.date_modified
				COALESCE(rpa.permissions, '[]'::jsonb) as role_permissions
			FROM role r
			LEFT JOIN role_permissions_agg rpa ON r.id = rpa.role_id
			WHERE r.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   r.name ILIKE $1 OR
				   r.description ILIKE $1)
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
		return nil, fmt.Errorf("failed to query role list page data: %w", err)
	}
	defer rows.Close()

	var roles []*rolepb.Role
	var totalCount int64

	for rows.Next() {
		var (
			id                   string
			workspaceId          *string
			name                 string
			description          string
			color                string
			active               bool
			dateCreated          time.Time
			dateModified         time.Time
			rolePermissionsJSON  []byte
			total                int64
		)

		err := rows.Scan(
			&id,
			&workspaceId,
			&name,
			&description,
			&color,
			&active,
			&dateCreated,
			&dateModified,
			&rolePermissionsJSON,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}

		totalCount = total

		role := &rolepb.Role{
			Id:          id,
			Name:        name,
			Description: description,
			Color:       color,
			Active:      active,
		}

		if workspaceId != nil {
			role.WorkspaceId = workspaceId
		}

		// Handle nullable timestamp fields

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		role.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		role.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		role.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		role.DateModifiedString = &dmStr
	}

		// Parse role_permissions JSONB array
		if len(rolePermissionsJSON) > 0 && string(rolePermissionsJSON) != "[]" {
			var permissionsData []map[string]interface{}
			if err := json.Unmarshal(rolePermissionsJSON, &permissionsData); err == nil {
				for _, permData := range permissionsData {
					// Convert map to RolePermission protobuf
					permJSON, err := json.Marshal(permData)
					if err != nil {
						continue
					}
					rp := &rolepermissionpb.RolePermission{}
					if err := protojson.Unmarshal(permJSON, rp); err == nil {
						role.RolePermissions = append(role.RolePermissions, rp)
					}
				}
			}
		}

		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &rolepb.GetRoleListPageDataResponse{
		RoleList: roles,
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

// GetRoleItemPageData retrieves a single role with enhanced item page data using CTE
func (r *PostgresRoleRepository) GetRoleItemPageData(
	ctx context.Context,
	req *rolepb.GetRoleItemPageDataRequest,
) (*rolepb.GetRoleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role item page data request is required")
	}
	if req.RoleId == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	// CTE Query - Single round-trip with enriched role_permission data
	query := `
		WITH role_permissions_agg AS (
			SELECT
				rp.role_id,
				jsonb_agg(
					jsonb_build_object(
						'id', rp.id,
						'role_id', rp.role_id,
						'permission_id', rp.permission_id,
						'permission', jsonb_build_object(
							'id', p.id,
							'name', p.name,
							'description', p.description,
							'active', p.active
						),
						'active', rp.active
					) ORDER BY p.name
				) FILTER (WHERE rp.id IS NOT NULL) as permissions
			FROM role_permission rp
			JOIN permission p ON rp.permission_id = p.id
			WHERE rp.active = true AND p.active = true
			GROUP BY rp.role_id
		)
		SELECT
			r.id,
			r.workspace_id,
			r.name,
			r.description,
			r.color,
			r.active,
			r.date_created,
			r.date_modified
			COALESCE(rpa.permissions, '[]'::jsonb) as role_permissions
		FROM role r
		LEFT JOIN role_permissions_agg rpa ON r.id = rpa.role_id
		WHERE r.id = $1 AND r.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.RoleId)

	var (
		id                   string
		workspaceId          *string
		name                 string
		description          string
		color                string
		active               bool
		dateCreated          time.Time
		dateModified         time.Time
		rolePermissionsJSON  []byte
	)

	err := row.Scan(
		&id,
		&workspaceId,
		&name,
		&description,
		&color,
		&active,
		&dateCreated,
		&dateModified,
		&rolePermissionsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role with ID '%s' not found", req.RoleId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query role item page data: %w", err)
	}

	role := &rolepb.Role{
		Id:          id,
		Name:        name,
		Description: description,
		Color:       color,
		Active:      active,
	}

	if workspaceId != nil {
		role.WorkspaceId = workspaceId
	}

	// Handle nullable timestamp fields

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		role.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		role.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		role.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		role.DateModifiedString = &dmStr
	}

	// Parse role_permissions JSONB array
	if len(rolePermissionsJSON) > 0 && string(rolePermissionsJSON) != "[]" {
		var permissionsData []map[string]interface{}
		if err := json.Unmarshal(rolePermissionsJSON, &permissionsData); err == nil {
			for _, permData := range permissionsData {
				// Convert map to RolePermission protobuf
				permJSON, err := json.Marshal(permData)
				if err != nil {
					continue
				}
				rp := &rolepermissionpb.RolePermission{}
				if err := protojson.Unmarshal(permJSON, rp); err == nil {
					role.RolePermissions = append(role.RolePermissions, rp)
				}
			}
		}
	}

	return &rolepb.GetRoleItemPageDataResponse{
		Role:    role,
		Success: true,
	}, nil
}


// NewRoleRepository creates a new PostgreSQL role repository (old-style constructor)
func NewRoleRepository(db *sql.DB, tableName string) rolepb.RoleDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresRoleRepository(dbOps, tableName)
}
