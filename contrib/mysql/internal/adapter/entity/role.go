//go:build mysql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Role, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql role repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLRoleRepository(dbOps, tableName), nil
	})
}

// MySQLRoleRepository implements role CRUD operations using MySQL 8.0+.
type MySQLRoleRepository struct {
	rolepb.UnimplementedRoleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLRoleRepository creates a new MySQL role repository.
func NewMySQLRoleRepository(dbOps interfaces.DatabaseOperation, tableName string) rolepb.RoleDomainServiceServer {
	if tableName == "" {
		tableName = "role"
	}
	return &MySQLRoleRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateRole creates a new role using common MySQL operations.
func (r *MySQLRoleRepository) CreateRole(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("role data is required")
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
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	role := &rolepb.Role{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepb.CreateRoleResponse{
		Data: []*rolepb.Role{role},
	}, nil
}

// ReadRole retrieves a role using common MySQL operations.
func (r *MySQLRoleRepository) ReadRole(ctx context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read role: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	role := &rolepb.Role{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepb.ReadRoleResponse{
		Data:    []*rolepb.Role{role},
		Success: true,
	}, nil
}

// UpdateRole updates a role using common MySQL operations.
func (r *MySQLRoleRepository) UpdateRole(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
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
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	role := &rolepb.Role{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &rolepb.UpdateRoleResponse{
		Data: []*rolepb.Role{role},
	}, nil
}

// DeleteRole deletes a role using common MySQL operations.
func (r *MySQLRoleRepository) DeleteRole(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete role: %w", err)
	}

	return &rolepb.DeleteRoleResponse{
		Success: true,
	}, nil
}

var roleSortableSQLCols = []string{
	"id", "active", "name", "description", "color", "workspace_id",
	"date_created", "date_modified",
}

var roleSortSpec = espynahttp.SortSpec{AllowedCols: roleSortableSQLCols}

// ListRoles lists roles using common MySQL operations.
func (r *MySQLRoleRepository) ListRoles(ctx context.Context, req *rolepb.ListRolesRequest) (*rolepb.ListRolesResponse, error) {
	if err := espynahttp.ValidateSortColumns(roleSortSpec, req.GetSort(), "role"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	var roles []*rolepb.Role
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		role := &rolepb.Role{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, role); err != nil {
			continue
		}
		roles = append(roles, role)
	}

	return &rolepb.ListRolesResponse{
		Data: roles,
	}, nil
}

// GetRoleListPageData retrieves roles with aggregated permissions using MySQL CTE.
//
// Dialect translation from postgres gold standard:
//   - $1/$2/$3/$4 → ? (MySQL positional placeholders)
//   - jsonb_agg(...) FILTER (WHERE ...) → JSON_ARRAYAGG(... via WHERE in inner CTE)
//   - jsonb_build_object → JSON_OBJECT
//   - COALESCE(..., '[]'::jsonb) → COALESCE(..., JSON_ARRAY())
//   - ILIKE → LIKE
//   - ARRAY[]::integer[] → JSON array (applicable_principal_types stored as JSON)
//   - EXTRACT(EPOCH FROM ...) * 1000)::bigint → UNIX_TIMESTAMP(...) * 1000
//   - TO_CHAR(... AT TIME ZONE 'UTC', ...) → DATE_FORMAT(CONVERT_TZ(...), ...)
//   - pq.Array scan → JSON_ARRAYAGG scan (read as []byte, parse JSON)
//   - active = true → active = 1
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLRoleRepository) GetRoleListPageData(
	ctx context.Context,
	req *rolepb.GetRoleListPageDataRequest,
) (*rolepb.GetRoleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role list page data request is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

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

	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Dialect translation:
	//   - jsonb_agg ... FILTER (WHERE rp.id IS NOT NULL) →
	//     filter via WHERE in the role_permissions_agg subquery (already done by INNER JOIN)
	//   - jsonb_build_object → JSON_OBJECT
	//   - EXTRACT(EPOCH FROM ...) * 1000)::bigint → UNIX_TIMESTAMP(...) * 1000
	//   - TO_CHAR(... AT TIME ZONE 'UTC', ...) → DATE_FORMAT(CONVERT_TZ(...))
	//   - COALESCE(..., '[]'::jsonb) → COALESCE(..., JSON_ARRAY())
	//   - ARRAY[]::integer[] column → stored as JSON in MySQL; scan as []byte
	//   - ILIKE → LIKE; $N → ?
	query := `
		WITH role_permissions_agg AS (
			SELECT
				rp.role_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', rp.id,
						'role_id', rp.role_id,
						'permission_id', rp.permission_id,
						'permission', JSON_OBJECT(
							'id', p.id,
							'name', p.name,
							'permission_code', p.permission_code,
							'permission_type', p.permission_type,
							'description', p.description,
							'active', p.active
						),
						'active', rp.active,
						'dateCreated', UNIX_TIMESTAMP(rp.date_created) * 1000,
						'dateCreatedString', DATE_FORMAT(CONVERT_TZ(rp.date_created, '+00:00', '+00:00'), '%Y-%m-%dT%H:%i:%sZ')
					)
				) AS permissions
			FROM role_permission rp
			JOIN permission p ON rp.permission_id = p.id
			WHERE rp.active = 1 AND p.active = 1
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
				r.date_modified,
				COALESCE(rpa.permissions, JSON_ARRAY()) AS role_permissions,
				COALESCE(r.applicable_principal_types, JSON_ARRAY()) AS applicable_principal_types
			FROM role r
			LEFT JOIN role_permissions_agg rpa ON r.id = rpa.role_id
			WHERE r.workspace_id = ?
			  AND (? = '' OR
				   r.name LIKE ? OR
				   r.description LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT ? OFFSET ?;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query role list page data: %w", err)
	}
	defer rows.Close()

	var roles []*rolepb.Role
	var totalCount int64

	for rows.Next() {
		var (
			id                      string
			workspaceId             *string
			name                    string
			description             string
			color                   string
			active                  bool
			dateCreated             time.Time
			dateModified            time.Time
			rolePermissionsJSON     []byte
			applicablePrincipalJSON []byte
			total                   int64
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
			&applicablePrincipalJSON,
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

		if len(rolePermissionsJSON) > 0 && string(rolePermissionsJSON) != "[]" {
			var permissionsData []map[string]interface{}
			if err := json.Unmarshal(rolePermissionsJSON, &permissionsData); err == nil {
				for _, permData := range permissionsData {
					permJSON, err := json.Marshal(permData)
					if err != nil {
						continue
					}
					rp := &rolepermissionpb.RolePermission{}
					if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(permJSON, rp); err == nil {
						role.RolePermissions = append(role.RolePermissions, rp)
					}
				}
			}
		}

		// applicable_principal_types stored as JSON array in MySQL; parse as []int64.
		if len(applicablePrincipalJSON) > 0 && string(applicablePrincipalJSON) != "[]" {
			var ints []int64
			if err := json.Unmarshal(applicablePrincipalJSON, &ints); err == nil {
				for _, v := range ints {
					role.ApplicablePrincipalTypes = append(role.ApplicablePrincipalTypes, principaltypepb.PrincipalType(v))
				}
			}
		}

		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}

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

// GetRoleItemPageData retrieves a single role with aggregated permissions.
//
// Dialect translation: jsonb_agg → JSON_ARRAYAGG; jsonb_build_object → JSON_OBJECT;
// FILTER (WHERE rp.id IS NOT NULL) → WHERE in CTE; $N → ?; active = true → active = 1.
func (r *MySQLRoleRepository) GetRoleItemPageData(
	ctx context.Context,
	req *rolepb.GetRoleItemPageDataRequest,
) (*rolepb.GetRoleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role item page data request is required")
	}
	if req.RoleId == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	query := `
		WITH role_permissions_agg AS (
			SELECT
				rp.role_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', rp.id,
						'role_id', rp.role_id,
						'permission_id', rp.permission_id,
						'permission', JSON_OBJECT(
							'id', p.id,
							'name', p.name,
							'permission_code', p.permission_code,
							'permission_type', p.permission_type,
							'description', p.description,
							'active', p.active
						),
						'active', rp.active,
						'dateCreated', UNIX_TIMESTAMP(rp.date_created) * 1000,
						'dateCreatedString', DATE_FORMAT(CONVERT_TZ(rp.date_created, '+00:00', '+00:00'), '%Y-%m-%dT%H:%i:%sZ')
					)
				) AS permissions
			FROM role_permission rp
			JOIN permission p ON rp.permission_id = p.id
			WHERE rp.active = 1 AND p.active = 1
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
			r.date_modified,
			COALESCE(rpa.permissions, JSON_ARRAY()) AS role_permissions
		FROM role r
		LEFT JOIN role_permissions_agg rpa ON r.id = rpa.role_id
		WHERE r.id = ? AND r.workspace_id = ?
		LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.RoleId, workspaceID)

	var (
		id                  string
		workspaceId         *string
		name                string
		description         string
		color               string
		active              bool
		dateCreated         time.Time
		dateModified        time.Time
		rolePermissionsJSON []byte
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

	if len(rolePermissionsJSON) > 0 && string(rolePermissionsJSON) != "[]" {
		var permissionsData []map[string]interface{}
		if err := json.Unmarshal(rolePermissionsJSON, &permissionsData); err == nil {
			for _, permData := range permissionsData {
				permJSON, err := json.Marshal(permData)
				if err != nil {
					continue
				}
				rp := &rolepermissionpb.RolePermission{}
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(permJSON, rp); err == nil {
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

// NewRoleRepository creates a new MySQL role repository (old-style constructor).
func NewRoleRepository(db *sql.DB, tableName string) rolepb.RoleDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLRoleRepository(dbOps, tableName)
}
