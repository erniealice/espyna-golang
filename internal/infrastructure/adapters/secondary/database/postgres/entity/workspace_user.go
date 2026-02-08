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
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
	workspaceuserrolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user_role"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "workspace_user", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres workspace_user repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresWorkspaceUserRepository(dbOps, tableName), nil
	})
}

// PostgresWorkspaceUserRepository implements workspace user CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_workspace_user_workspace_id ON workspace_user(workspace_id) - Multi-tenancy filter (CRITICAL)
//   - CREATE INDEX idx_workspace_user_user_id ON workspace_user(user_id) - Foreign key relationship to user table
//   - CREATE INDEX idx_workspace_user_active ON workspace_user(active) - Filter active records
//   - CREATE INDEX idx_workspace_user_date_created ON workspace_user(date_created DESC) - Default sorting
//   - CREATE INDEX idx_workspace_user_role_workspace_user_id ON workspace_user_role(workspace_user_id) - Junction table lookup
//   - CREATE INDEX idx_workspace_user_role_role_id ON workspace_user_role(role_id) - Junction table foreign key
//   - CREATE INDEX idx_workspace_user_role_active ON workspace_user_role(active) - Filter active junction records
//   - CREATE INDEX idx_role_active ON role(active) - Filter active roles
//   - CREATE INDEX idx_user_first_name ON "user"(first_name) - Search performance on joined table
//   - CREATE INDEX idx_user_last_name ON "user"(last_name) - Search performance on joined table
//   - CREATE INDEX idx_user_email_address ON "user"(email_address) - Search performance on joined table
//
// TODO: Add comprehensive tests for GetWorkspaceUserListPageData:
//   - Test with no search query (list all active workspace users)
//   - Test with search query matching user first_name
//   - Test with search query matching user last_name
//   - Test with search query matching user email_address
//   - Test pagination (page 1, page 2, page size variations)
//   - Test sorting (by different fields, ASC and DESC)
//   - Test with no matching results
//   - Test with inactive workspace users (should be filtered out)
//   - Test with null user_id (LEFT JOIN behavior)
//   - Test with inactive user (should be filtered out via JOIN condition)
//   - Test workspace_id filtering (multi-tenancy requirement)
//   - Test with workspace users having multiple roles
//   - Test with workspace users having no roles (empty workspace_user_roles array)
//   - Test with inactive roles (should be filtered out of aggregation)
//
// TODO: Add comprehensive tests for GetWorkspaceUserItemPageData:
//   - Test with valid workspace user ID (with associated user and roles)
//   - Test with valid workspace user ID (without associated user - null user_id)
//   - Test with non-existent workspace user ID
//   - Test with inactive workspace user (should return not found)
//   - Test with workspace user having inactive user (user fields should be null)
//   - Test with workspace user having multiple roles
//   - Test with workspace user having no roles (empty workspace_user_roles array)
//   - Test with workspace user having inactive roles (should be filtered out)
//   - Test timestamp parsing for date_created and date_modified
//   - Test workspace_id filtering (multi-tenancy requirement)
type PostgresWorkspaceUserRepository struct {
	workspaceuserpb.UnimplementedWorkspaceUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresWorkspaceUserRepository creates a new PostgreSQL workspace user repository
func NewPostgresWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, tableName string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	if tableName == "" {
		tableName = "workspace_user" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresWorkspaceUserRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateWorkspaceUser creates a new workspace user using common PostgreSQL operations
func (r *PostgresWorkspaceUserRepository) CreateWorkspaceUser(ctx context.Context, req *workspaceuserpb.CreateWorkspaceUserRequest) (*workspaceuserpb.CreateWorkspaceUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace user data is required")
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
		return nil, fmt.Errorf("failed to create workspace user: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUser := &workspaceuserpb.WorkspaceUser{}
	if err := protojson.Unmarshal(resultJSON, workspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserpb.CreateWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{workspaceUser},
	}, nil
}

// ReadWorkspaceUser retrieves a workspace user using common PostgreSQL operations
func (r *PostgresWorkspaceUserRepository) ReadWorkspaceUser(ctx context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace user: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUser := &workspaceuserpb.WorkspaceUser{}
	if err := protojson.Unmarshal(resultJSON, workspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserpb.ReadWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{workspaceUser},
	}, nil
}

// UpdateWorkspaceUser updates a workspace user using common PostgreSQL operations
func (r *PostgresWorkspaceUserRepository) UpdateWorkspaceUser(ctx context.Context, req *workspaceuserpb.UpdateWorkspaceUserRequest) (*workspaceuserpb.UpdateWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
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
		return nil, fmt.Errorf("failed to update workspace user: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUser := &workspaceuserpb.WorkspaceUser{}
	if err := protojson.Unmarshal(resultJSON, workspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserpb.UpdateWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{workspaceUser},
	}, nil
}

// DeleteWorkspaceUser deletes a workspace user using common PostgreSQL operations
func (r *PostgresWorkspaceUserRepository) DeleteWorkspaceUser(ctx context.Context, req *workspaceuserpb.DeleteWorkspaceUserRequest) (*workspaceuserpb.DeleteWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workspace user: %w", err)
	}

	return &workspaceuserpb.DeleteWorkspaceUserResponse{
		Success: true,
	}, nil
}

// ListWorkspaceUsers lists workspace users using common PostgreSQL operations
func (r *PostgresWorkspaceUserRepository) ListWorkspaceUsers(ctx context.Context, req *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace users: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var workspaceUsers []*workspaceuserpb.WorkspaceUser
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		workspaceUser := &workspaceuserpb.WorkspaceUser{}
		if err := protojson.Unmarshal(resultJSON, workspaceUser); err != nil {
			// Log error and continue with next item
			continue
		}
		workspaceUsers = append(workspaceUsers, workspaceUser)
	}

	return &workspaceuserpb.ListWorkspaceUsersResponse{
		Data: workspaceUsers,
	}, nil
}

// GetWorkspaceUserListPageData retrieves workspace users with advanced filtering, sorting, searching, and pagination using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresWorkspaceUserRepository) GetWorkspaceUserListPageData(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserListPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user list page data request is required")
	}

	// Extract workspace_id from filters (REQUIRED for multi-tenancy)
	var workspaceID string
	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "workspace_id" {
				// Extract value from string filter (the most common case for workspace_id)
				if stringFilter := filter.GetStringFilter(); stringFilter != nil {
					workspaceID = stringFilter.Value
					break
				}
			}
		}
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id filter is required for multi-tenancy")
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

	// CTE Query - Single round-trip with enriched user data and aggregated workspace_user_role relationships
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on workspace_user.workspace_id (CRITICAL for multi-tenancy)
	// - INDEX RECOMMENDATION: Create index on workspace_user.user_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on workspace_user_role.workspace_user_id (junction table lookup)
	// - INDEX RECOMMENDATION: Create index on workspace_user_role.role_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on workspace_user_role.active (filter active junction records)
	// - INDEX RECOMMENDATION: Create index on role.active (filter active roles)
	// - INDEX RECOMMENDATION: Create index on user.first_name, user.last_name, user.email_address for search performance
	// - INDEX RECOMMENDATION: Create index on workspace_user.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on workspace_user.date_created for default sorting
	query := `
		WITH user_roles_agg AS (
			SELECT
				wur.workspace_user_id,
				jsonb_agg(
					jsonb_build_object(
						'id', wur.id,
						'workspace_user_id', wur.workspace_user_id,
						'role_id', wur.role_id,
						'role', jsonb_build_object(
							'id', r.id,
							'name', r.name,
							'permissions', r.permissions,
							'active', r.active
						),
						'date_created', EXTRACT(EPOCH FROM wur.date_created) * 1000,
						'date_modified', EXTRACT(EPOCH FROM wur.date_modified) * 1000,
						'active', wur.active
					)
				) as roles
			FROM workspace_user_role wur
			JOIN role r ON wur.role_id = r.id
			WHERE wur.active = true AND r.active = true
			GROUP BY wur.workspace_user_id
		),
		enriched AS (
			SELECT
				wu.id,
				wu.workspace_id,
				wu.user_id,
				wu.active,
				wu.date_created,
				wu.date_modified,
				-- User fields (1:1 relationship) - Direct fields for protobuf mapping
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.phone_number as user_phone_number,
				u.active as user_active,
				-- Workspace user roles (many-to-many via junction table)
				COALESCE(ura.roles, '[]'::jsonb) as workspace_user_roles
			FROM workspace_user wu
			LEFT JOIN "user" u ON wu.user_id = u.id AND u.active = true
			LEFT JOIN user_roles_agg ura ON wu.id = ura.workspace_user_id
			WHERE wu.active = true
			  AND wu.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   u.first_name ILIKE $2 OR
				   u.last_name ILIKE $2 OR
				   u.email_address ILIKE $2)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $3 OFFSET $4;
	`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace user list page data: %w", err)
	}
	defer rows.Close()

	var workspaceUsers []*workspaceuserpb.WorkspaceUser
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			workspaceId        string
			userId             string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			// User fields
			userIdValue      *string
			userFirstName    *string
			userLastName     *string
			userEmailAddress *string
			userPhoneNumber  *string
			userActive       *bool
			// Workspace user roles
			workspaceUserRolesJSON []byte
			total                  int64
		)

		err := rows.Scan(
			&id,
			&workspaceId,
			&userId,
			&active,
			&dateCreated,
			&dateModified,
			&userIdValue,
			&userFirstName,
			&userLastName,
			&userEmailAddress,
			&userPhoneNumber,
			&userActive,
			&workspaceUserRolesJSON,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace user row: %w", err)
		}

		totalCount = total

		workspaceUser := &workspaceuserpb.WorkspaceUser{
			Id:          id,
			WorkspaceId: workspaceId,
			UserId:      userId,
			Active:      active,
		}

		// Handle nullable timestamp fields

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		workspaceUser.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		workspaceUser.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		workspaceUser.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		workspaceUser.DateModifiedString = &dmStr
	}

		// Parse workspace_user_roles JSONB array
		if len(workspaceUserRolesJSON) > 0 {
			var rolesData []map[string]interface{}
			if err := json.Unmarshal(workspaceUserRolesJSON, &rolesData); err == nil {
				for _, roleData := range rolesData {
					// Convert map to WorkspaceUserRole protobuf
					roleJSON, err := json.Marshal(roleData)
					if err != nil {
						continue
					}
					wur := &workspaceuserrolepb.WorkspaceUserRole{}
					if err := protojson.Unmarshal(roleJSON, wur); err == nil {
						workspaceUser.WorkspaceUserRoles = append(workspaceUser.WorkspaceUserRoles, wur)
					}
				}
			}
		}

		workspaceUsers = append(workspaceUsers, workspaceUser)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workspace user rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &workspaceuserpb.GetWorkspaceUserListPageDataResponse{
		WorkspaceUserList: workspaceUsers,
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

// GetWorkspaceUserItemPageData retrieves a single workspace user with enhanced item page data using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresWorkspaceUserRepository) GetWorkspaceUserItemPageData(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user item page data request is required")
	}
	if req.WorkspaceUserId == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	// CTE Query - Single round-trip with enriched user data and aggregated workspace_user_role relationships
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on workspace_user.id (primary key - typically automatic)
	// - INDEX RECOMMENDATION: Create index on workspace_user.workspace_id (multi-tenancy filter)
	// - INDEX RECOMMENDATION: Create index on workspace_user.user_id (foreign key)
	// - INDEX RECOMMENDATION: Create index on workspace_user_role.workspace_user_id (junction table lookup)
	query := `
		WITH user_roles_agg AS (
			SELECT
				wur.workspace_user_id,
				jsonb_agg(
					jsonb_build_object(
						'id', wur.id,
						'workspace_user_id', wur.workspace_user_id,
						'role_id', wur.role_id,
						'role', jsonb_build_object(
							'id', r.id,
							'name', r.name,
							'permissions', r.permissions,
							'active', r.active
						),
						'date_created', EXTRACT(EPOCH FROM wur.date_created) * 1000,
						'date_modified', EXTRACT(EPOCH FROM wur.date_modified) * 1000,
						'active', wur.active
					)
				) as roles
			FROM workspace_user_role wur
			JOIN role r ON wur.role_id = r.id
			WHERE wur.active = true AND r.active = true
			GROUP BY wur.workspace_user_id
		),
		enriched AS (
			SELECT
				wu.id,
				wu.workspace_id,
				wu.user_id,
				wu.active,
				wu.date_created,
				wu.date_modified,
				-- User fields (1:1 relationship) - Direct fields for protobuf mapping
				u.id as user_id_value,
				u.first_name as user_first_name,
				u.last_name as user_last_name,
				u.email_address as user_email_address,
				u.phone_number as user_phone_number,
				u.active as user_active,
				-- Workspace user roles (many-to-many via junction table)
				COALESCE(ura.roles, '[]'::jsonb) as workspace_user_roles
			FROM workspace_user wu
			LEFT JOIN "user" u ON wu.user_id = u.id AND u.active = true
			LEFT JOIN user_roles_agg ura ON wu.id = ura.workspace_user_id
			WHERE wu.id = $1 AND wu.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.WorkspaceUserId)

	var (
		id                 string
		workspaceId        string
		userId             string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
		// User fields
		userIdValue      *string
		userFirstName    *string
		userLastName     *string
		userEmailAddress *string
		userPhoneNumber  *string
		userActive       *bool
		// Workspace user roles
		workspaceUserRolesJSON []byte
	)

	err := row.Scan(
		&id,
		&workspaceId,
		&userId,
		&active,
		&dateCreated,
		&dateModified,
		&userIdValue,
		&userFirstName,
		&userLastName,
		&userEmailAddress,
		&userPhoneNumber,
		&userActive,
		&workspaceUserRolesJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace user with ID '%s' not found", req.WorkspaceUserId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace user item page data: %w", err)
	}

	workspaceUser := &workspaceuserpb.WorkspaceUser{
		Id:          id,
		WorkspaceId: workspaceId,
		UserId:      userId,
		Active:      active,
	}

	// Handle nullable timestamp fields

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		workspaceUser.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		workspaceUser.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		workspaceUser.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		workspaceUser.DateModifiedString = &dmStr
	}

	// Parse workspace_user_roles JSONB array
	if len(workspaceUserRolesJSON) > 0 {
		var rolesData []map[string]interface{}
		if err := json.Unmarshal(workspaceUserRolesJSON, &rolesData); err == nil {
			for _, roleData := range rolesData {
				// Convert map to WorkspaceUserRole protobuf
				roleJSON, err := json.Marshal(roleData)
				if err != nil {
					continue
				}
				wur := &workspaceuserrolepb.WorkspaceUserRole{}
				if err := protojson.Unmarshal(roleJSON, wur); err == nil {
					workspaceUser.WorkspaceUserRoles = append(workspaceUser.WorkspaceUserRoles, wur)
				}
			}
		}
	}

	return &workspaceuserpb.GetWorkspaceUserItemPageDataResponse{
		WorkspaceUser: workspaceUser,
		Success:       true,
	}, nil
}


// NewWorkspaceUserRepository creates a new PostgreSQL workspace_user repository (old-style constructor)
func NewWorkspaceUserRepository(db *sql.DB, tableName string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresWorkspaceUserRepository(dbOps, tableName)
}
