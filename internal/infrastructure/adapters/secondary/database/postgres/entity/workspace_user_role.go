//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"time"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "workspace_user_role", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres workspace_user_role repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresWorkspaceUserRoleRepository(dbOps, tableName), nil
	})
}

// PostgresWorkspaceUserRoleRepository implements workspace user role CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_workspace_user_role_active ON workspace_user_role(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_workspace_user_role_workspace_user_id ON workspace_user_role(workspace_user_id) - Filter by workspace user
//   - CREATE INDEX idx_workspace_user_role_role_id ON workspace_user_role(role_id) - Filter by role
//   - CREATE INDEX idx_workspace_user_role_date_created ON workspace_user_role(date_created DESC) - Default sorting
type PostgresWorkspaceUserRoleRepository struct {
	workspaceuserrolepb.UnimplementedWorkspaceUserRoleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresWorkspaceUserRoleRepository creates a new PostgreSQL workspace user role repository
func NewPostgresWorkspaceUserRoleRepository(dbOps interfaces.DatabaseOperation, tableName string) workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer {
	if tableName == "" {
		tableName = "workspace_user_role" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresWorkspaceUserRoleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateWorkspaceUserRole creates a new workspace user role using common PostgreSQL operations
func (r *PostgresWorkspaceUserRoleRepository) CreateWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.CreateWorkspaceUserRoleRequest) (*workspaceuserrolepb.CreateWorkspaceUserRoleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace user role data is required")
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
		return nil, fmt.Errorf("failed to create workspace user role: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{}
	if err := protojson.Unmarshal(resultJSON, workspaceUserRole); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserrolepb.CreateWorkspaceUserRoleResponse{
		Data: []*workspaceuserrolepb.WorkspaceUserRole{workspaceUserRole},
	}, nil
}

// ReadWorkspaceUserRole retrieves a workspace user role using common PostgreSQL operations
func (r *PostgresWorkspaceUserRoleRepository) ReadWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.ReadWorkspaceUserRoleRequest) (*workspaceuserrolepb.ReadWorkspaceUserRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace user role: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{}
	if err := protojson.Unmarshal(resultJSON, workspaceUserRole); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserrolepb.ReadWorkspaceUserRoleResponse{
		Data: []*workspaceuserrolepb.WorkspaceUserRole{workspaceUserRole},
	}, nil
}

// UpdateWorkspaceUserRole updates a workspace user role using common PostgreSQL operations
func (r *PostgresWorkspaceUserRoleRepository) UpdateWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.UpdateWorkspaceUserRoleRequest) (*workspaceuserrolepb.UpdateWorkspaceUserRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
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
		return nil, fmt.Errorf("failed to update workspace user role: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{}
	if err := protojson.Unmarshal(resultJSON, workspaceUserRole); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserrolepb.UpdateWorkspaceUserRoleResponse{
		Data: []*workspaceuserrolepb.WorkspaceUserRole{workspaceUserRole},
	}, nil
}

// DeleteWorkspaceUserRole deletes a workspace user role using common PostgreSQL operations
func (r *PostgresWorkspaceUserRoleRepository) DeleteWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workspace user role: %w", err)
	}

	return &workspaceuserrolepb.DeleteWorkspaceUserRoleResponse{
		Success: true,
	}, nil
}

// ListWorkspaceUserRoles lists workspace user roles using common PostgreSQL operations
func (r *PostgresWorkspaceUserRoleRepository) ListWorkspaceUserRoles(ctx context.Context, req *workspaceuserrolepb.ListWorkspaceUserRolesRequest) (*workspaceuserrolepb.ListWorkspaceUserRolesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace user roles: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var workspaceUserRoles []*workspaceuserrolepb.WorkspaceUserRole
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		workspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{}
		if err := protojson.Unmarshal(resultJSON, workspaceUserRole); err != nil {
			// Log error and continue with next item
			continue
		}
		workspaceUserRoles = append(workspaceUserRoles, workspaceUserRole)
	}

	return &workspaceuserrolepb.ListWorkspaceUserRolesResponse{
		Data: workspaceUserRoles,
	}, nil
}

// GetWorkspaceUserRoleListPageData retrieves paginated workspace user role list data with CTE
func (r *PostgresWorkspaceUserRoleRepository) GetWorkspaceUserRoleListPageData(ctx context.Context, req *workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest) (*workspaceuserrolepb.GetWorkspaceUserRoleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}
	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}
	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `WITH enriched AS (SELECT id, workspace_user_id, role_id, active, date_created, date_modified FROM workspace_user_role WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR workspace_user_id ILIKE $1 OR role_id ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var workspaceUserRoles []*workspaceuserrolepb.WorkspaceUserRole
	var totalCount int64
	for rows.Next() {
		var id, workspaceUserId, roleId string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &workspaceUserId, &roleId, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		workspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{Id: id, WorkspaceUserId: workspaceUserId, RoleId: roleId, Active: active}
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		workspaceUserRole.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		workspaceUserRole.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		workspaceUserRole.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		workspaceUserRole.DateModifiedString = &dmStr
	}
		workspaceUserRoles = append(workspaceUserRoles, workspaceUserRole)
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataResponse{WorkspaceUserRoleList: workspaceUserRoles, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetWorkspaceUserRoleItemPageData retrieves workspace user role item page data
func (r *PostgresWorkspaceUserRoleRepository) GetWorkspaceUserRoleItemPageData(ctx context.Context, req *workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataRequest) (*workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse, error) {
	if req == nil || req.WorkspaceUserRoleId == "" {
		return nil, fmt.Errorf("workspace user role ID required")
	}
	query := `SELECT id, workspace_user_id, role_id, active, date_created, date_modified FROM workspace_user_role WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.WorkspaceUserRoleId)
	var id, workspaceUserId, roleId string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &workspaceUserId, &roleId, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace user role not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	workspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{Id: id, WorkspaceUserId: workspaceUserId, RoleId: roleId, Active: active}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		workspaceUserRole.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		workspaceUserRole.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		workspaceUserRole.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		workspaceUserRole.DateModifiedString = &dmStr
	}
	return &workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse{WorkspaceUserRole: workspaceUserRole, Success: true}, nil
}


// NewWorkspaceUserRoleRepository creates a new PostgreSQL workspace_user_role repository (old-style constructor)
func NewWorkspaceUserRoleRepository(db *sql.DB, tableName string) workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresWorkspaceUserRoleRepository(dbOps, tableName)
}
