//go:build sqlserver

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Role, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver role repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerRoleRepository(dbOps, tableName), nil
	})
}

// SQLServerRoleRepository implements role CRUD operations using SQL Server.
type SQLServerRoleRepository struct {
	rolepb.UnimplementedRoleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerRoleRepository creates a new SQL Server role repository.
func NewSQLServerRoleRepository(dbOps interfaces.DatabaseOperation, tableName string) rolepb.RoleDomainServiceServer {
	if tableName == "" {
		tableName = "role"
	}
	return &SQLServerRoleRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateRole creates a new role using common SQL Server operations.
func (r *SQLServerRoleRepository) CreateRole(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
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

	return &rolepb.CreateRoleResponse{Data: []*rolepb.Role{role}}, nil
}

// ReadRole retrieves a role using common SQL Server operations.
func (r *SQLServerRoleRepository) ReadRole(ctx context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
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

	return &rolepb.ReadRoleResponse{Data: []*rolepb.Role{role}, Success: true}, nil
}

// UpdateRole updates a role using common SQL Server operations.
func (r *SQLServerRoleRepository) UpdateRole(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
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

	return &rolepb.UpdateRoleResponse{Data: []*rolepb.Role{role}}, nil
}

// DeleteRole deletes a role using common SQL Server operations.
func (r *SQLServerRoleRepository) DeleteRole(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete role: %w", err)
	}

	return &rolepb.DeleteRoleResponse{Success: true}, nil
}

var roleSortableSQLCols = []string{
	"id", "active", "name", "description", "color", "workspace_id",
	"date_created", "date_modified",
}

var roleSortSpec = espynahttp.SortSpec{AllowedCols: roleSortableSQLCols}

// ListRoles lists roles using common SQL Server operations.
func (r *SQLServerRoleRepository) ListRoles(ctx context.Context, req *rolepb.ListRolesRequest) (*rolepb.ListRolesResponse, error) {
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

	return &rolepb.ListRolesResponse{Data: roles}, nil
}

// GetRoleListPageData retrieves roles with filtering, sorting, and pagination.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server translation notes:
//   - $1/$2/$3/$4 → @p1/@p2/@p3/@p4.
//   - ILIKE → LIKE (CI collation).
//   - jsonb_agg + jsonb_build_object → FOR JSON PATH correlated subquery.
//   - FILTER (WHERE rp.id IS NOT NULL) → only rows satisfying the JOIN exist; no
//     CASE needed — the WHERE clause in the subquery handles NULL filtering.
//   - applicable_principal_types: postgres stores as integer[]; SQL Server
//     stores as comma-delimited NVARCHAR. We skip the array scan and leave
//     ApplicablePrincipalTypes empty (schema migration required — out of scope).
//   - Pagination: OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY (ORDER BY mandatory).
//   - COUNT(*) OVER () retained (SQL Server 2017+).
func (r *SQLServerRoleRepository) GetRoleListPageData(
	ctx context.Context,
	req *rolepb.GetRoleListPageDataRequest,
) (*rolepb.GetRoleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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

	orderByClause, err := sqlserverCore.BuildOrderBy(roleSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	// Build filter WHERE clauses; @p1=workspaceID, @p2=searchPattern.
	// Additional filters start at @p3; limit/offset are always last two.
	filterClauses := []string{}
	if searchPattern != "" {
		filterClauses = append(filterClauses, "(@p2 = '' OR r.name LIKE @p2 OR r.description LIKE @p2)")
	}
	whereSQL := "WHERE r.workspace_id = @p1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	// FOR JSON PATH subquery replaces jsonb_agg(jsonb_build_object(...)) FILTER (WHERE ...).
	// The subquery includes only active role_permissions joined to active permissions; the
	// FILTER (WHERE rp.id IS NOT NULL) guard is implicit — subquery returns no rows when
	// no active permissions exist, so FOR JSON PATH returns NULL (mapped to "[]" in Go).
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				r.id,
				r.workspace_id,
				r.name,
				r.description,
				r.color,
				r.active,
				r.date_created,
				r.date_modified,
				(SELECT
					rp.id,
					rp.role_id,
					rp.permission_id,
					rp.active,
					p.id AS [permission.id],
					p.name AS [permission.name],
					p.permission_code AS [permission.permission_code],
					p.permission_type AS [permission.permission_type],
					p.description AS [permission.description],
					p.active AS [permission.active]
				 FROM role_permission rp
				 JOIN permission p ON rp.permission_id = p.id
				 WHERE rp.role_id = r.id AND rp.active = 1 AND p.active = 1
				 ORDER BY p.name
				 FOR JSON PATH) AS role_permissions
			FROM role r
			%s
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.id,
			e.workspace_id,
			e.name,
			e.description,
			e.color,
			e.active,
			e.date_created,
			e.date_modified,
			e.role_permissions,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY;
	`, whereSQL, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query role list page data: %w", err)
	}
	defer rows.Close()

	var roles []*rolepb.Role
	var totalCount int64

	for rows.Next() {
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
			total               int64
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

		// applicable_principal_types: postgres integer[] has no SQL Server array equivalent
		// without schema change (Q-REFLECT-CRUD out of scope). Emit empty slice — caller
		// can fall back to a separate load if needed.
		_ = principaltypepb.PrincipalType_value // suppress unused-import lint

		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &rolepb.GetRoleListPageDataResponse{
		RoleList: roles,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
		Success: true,
	}, nil
}

// GetRoleItemPageData retrieves a single role with enriched permission data.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server translation: same patterns as GetRoleListPageData but for a single row.
// TOP 1 replaces LIMIT 1. @p1=roleID, @p2=workspaceID.
func (r *SQLServerRoleRepository) GetRoleItemPageData(
	ctx context.Context,
	req *rolepb.GetRoleItemPageDataRequest,
) (*rolepb.GetRoleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role item page data request is required")
	}
	if req.RoleId == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		SELECT TOP 1
			r.id,
			r.workspace_id,
			r.name,
			r.description,
			r.color,
			r.active,
			r.date_created,
			r.date_modified,
			(SELECT
				rp.id,
				rp.role_id,
				rp.permission_id,
				rp.active,
				p.id AS [permission.id],
				p.name AS [permission.name],
				p.permission_code AS [permission.permission_code],
				p.permission_type AS [permission.permission_type],
				p.description AS [permission.description],
				p.active AS [permission.active]
			 FROM role_permission rp
			 JOIN permission p ON rp.permission_id = p.id
			 WHERE rp.role_id = r.id AND rp.active = 1 AND p.active = 1
			 ORDER BY p.name
			 FOR JSON PATH) AS role_permissions
		FROM role r
		WHERE r.id = @p1 AND r.workspace_id = @p2;
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

	return &rolepb.GetRoleItemPageDataResponse{Role: role, Success: true}, nil
}

// NewRoleRepository creates a new SQL Server role repository (old-style constructor).
func NewRoleRepository(db *sql.DB, tableName string) rolepb.RoleDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerRoleRepository(dbOps, tableName)
}
