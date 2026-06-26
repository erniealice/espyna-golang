//go:build mysql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.WorkspaceUser, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql workspace_user repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLWorkspaceUserRepository(dbOps, tableName), nil
	})
}

// MySQLWorkspaceUserRepository implements workspace user CRUD operations using MySQL 8.0+.
type MySQLWorkspaceUserRepository struct {
	workspaceuserpb.UnimplementedWorkspaceUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLWorkspaceUserRepository creates a new MySQL workspace user repository.
func NewMySQLWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, tableName string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	if tableName == "" {
		tableName = "workspace_user"
	}
	return &MySQLWorkspaceUserRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateWorkspaceUser creates a new workspace user using common MySQL operations.
func (r *MySQLWorkspaceUserRepository) CreateWorkspaceUser(ctx context.Context, req *workspaceuserpb.CreateWorkspaceUserRequest) (*workspaceuserpb.CreateWorkspaceUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace user data is required")
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
		return nil, fmt.Errorf("failed to create workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUser := &workspaceuserpb.WorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserpb.CreateWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{workspaceUser},
	}, nil
}

// ReadWorkspaceUser retrieves a workspace user using common MySQL operations.
func (r *MySQLWorkspaceUserRepository) ReadWorkspaceUser(ctx context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUser := &workspaceuserpb.WorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserpb.ReadWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{workspaceUser},
	}, nil
}

// UpdateWorkspaceUser updates a workspace user using common MySQL operations.
func (r *MySQLWorkspaceUserRepository) UpdateWorkspaceUser(ctx context.Context, req *workspaceuserpb.UpdateWorkspaceUserRequest) (*workspaceuserpb.UpdateWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
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
		return nil, fmt.Errorf("failed to update workspace user: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspaceUser := &workspaceuserpb.WorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspaceUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspaceuserpb.UpdateWorkspaceUserResponse{
		Data: []*workspaceuserpb.WorkspaceUser{workspaceUser},
	}, nil
}

// DeleteWorkspaceUser deletes a workspace user using common MySQL operations.
func (r *MySQLWorkspaceUserRepository) DeleteWorkspaceUser(ctx context.Context, req *workspaceuserpb.DeleteWorkspaceUserRequest) (*workspaceuserpb.DeleteWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete workspace user: %w", err)
	}

	return &workspaceuserpb.DeleteWorkspaceUserResponse{
		Success: true,
	}, nil
}

// ListWorkspaceUsers lists workspace users with joined user data.
//
// Dialect translation: "user" → `user`; $1 cast removed; active = true → active = 1.
func (r *MySQLWorkspaceUserRepository) ListWorkspaceUsers(ctx context.Context, req *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	// Dialect: "user" → `user`; $1::text = '' OR wu.workspace_id = $1::text → ? = '' OR wu.workspace_id = ?
	query := `
		SELECT
			wu.id, wu.workspace_id, wu.user_id, wu.active,
			wu.date_created, wu.date_modified,
			u.id, u.first_name, u.last_name, u.email_address, u.mobile_number, u.active
		FROM workspace_user wu
		LEFT JOIN ` + "`user`" + ` u ON wu.user_id = u.id
		WHERE wu.active = 1
		  AND (? = '' OR wu.workspace_id = ?)
		ORDER BY wu.date_created DESC
	`

	wsID := identity.Must(ctx).WorkspaceID
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, wsID, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace users: %w", err)
	}
	defer rows.Close()

	var workspaceUsers []*workspaceuserpb.WorkspaceUser
	for rows.Next() {
		var (
			id, workspaceId, userId string
			active                  bool
			dateCreated             time.Time
			dateModified            time.Time
			userIdValue             *string
			userFirstName           *string
			userLastName            *string
			userEmailAddress        *string
			userPhoneNumber         *string
			userActive              *bool
		)

		if err := rows.Scan(
			&id, &workspaceId, &userId, &active,
			&dateCreated, &dateModified,
			&userIdValue, &userFirstName, &userLastName, &userEmailAddress, &userPhoneNumber, &userActive,
		); err != nil {
			continue
		}

		workspaceUser := &workspaceuserpb.WorkspaceUser{
			Id:          id,
			WorkspaceId: workspaceId,
			UserId:      userId,
			Active:      active,
		}

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

		if userIdValue != nil {
			workspaceUser.User = &userpb.User{
				Id:           *userIdValue,
				FirstName:    derefStr(userFirstName),
				LastName:     derefStr(userLastName),
				EmailAddress: derefStr(userEmailAddress),
				MobileNumber: derefStr(userPhoneNumber),
				Active:       userActive != nil && *userActive,
			}
		}

		workspaceUsers = append(workspaceUsers, workspaceUser)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workspace user rows: %w", err)
	}

	return &workspaceuserpb.ListWorkspaceUsersResponse{
		Data: workspaceUsers,
	}, nil
}

// derefStr safely dereferences a string pointer, returning "" if nil.
func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// workspaceUserSortAllowlist maps external sort field names to safe SQL column references.
var workspaceUserSortAllowlist = map[string]string{
	"date_created":    "wu.date_created",
	"date_modified":   "wu.date_modified",
	"u.first_name":    "u.first_name",
	"u.last_name":     "u.last_name",
	"u.email_address": "u.email_address",
}

// GetWorkspaceUserListPageData retrieves workspace users with filtering, sorting, searching, and pagination.
//
// Dialect translation from postgres gold standard:
//   - $1/$2,... → ? (MySQL positional placeholders, same arg order)
//   - "user" → `user` (backtick-quoted reserved word)
//   - jsonb_agg → JSON_ARRAYAGG
//   - jsonb_build_object → JSON_OBJECT
//   - COALESCE(..., '[]'::jsonb) → COALESCE(..., JSON_ARRAY())
//   - EXTRACT(EPOCH FROM ...) * 1000)::bigint → UNIX_TIMESTAMP(...) * 1000
//   - TO_CHAR(...) → DATE_FORMAT(CONVERT_TZ(...))
//   - active = true → active = 1; u.active = true → u.active = 1
//   - COUNT(*) OVER() stays — MySQL 8.0+ window functions
//
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *MySQLWorkspaceUserRepository) GetWorkspaceUserListPageData(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserListPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user list page data request is required")
	}

	var workspaceID string
	if req.Filters != nil && len(req.Filters.Filters) > 0 {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "workspace_id" {
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

	sortCol := "wu.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		f := req.Sort.Fields[0]
		if col, ok := workspaceUserSortAllowlist[f.Field]; ok {
			sortCol = col
		}
		if f.Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Strip workspace_id filter to avoid double-binding.
	var filteredReqFilters *commonpb.FilterRequest
	if req.Filters != nil {
		var nonWorkspaceFilters []*commonpb.TypedFilter
		for _, f := range req.Filters.Filters {
			if f.Field != "workspace_id" {
				nonWorkspaceFilters = append(nonWorkspaceFilters, f)
			}
		}
		if len(nonWorkspaceFilters) > 0 {
			filteredReqFilters = &commonpb.FilterRequest{Filters: nonWorkspaceFilters}
		}
	}

	searchFields := []string{"u.first_name", "u.last_name", "u.email_address"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(filteredReqFilters, req.Search, searchFields, 2)

	hardWhere := "wu.active = 1 AND wu.workspace_id = ?"
	extraWhere := ""
	if len(filterClauses) > 0 {
		extraWhere = " AND " + strings.Join(filterClauses, " AND ")
	}

	allArgs := []any{workspaceID}
	allArgs = append(allArgs, filterArgs...)
	allArgs = append(allArgs, limit, offset)

	// Dialect: "user" → `user`; jsonb_agg → JSON_ARRAYAGG; jsonb_build_object → JSON_OBJECT;
	// EXTRACT/TO_CHAR → UNIX_TIMESTAMP/DATE_FORMAT; COALESCE('[]'::jsonb) → JSON_ARRAY();
	// active = true → active = 1; COUNT(*) OVER() stays.
	query := fmt.Sprintf(`
		WITH user_roles_agg AS (
			SELECT
				wur.workspace_user_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', wur.id,
						'workspace_user_id', wur.workspace_user_id,
						'role_id', wur.role_id,
						'role', JSON_OBJECT(
							'id', r.id,
							'name', r.name,
							'description', r.description,
							'color', r.color,
							'active', r.active
						),
						'date_created', UNIX_TIMESTAMP(wur.date_created) * 1000,
						'date_created_string', DATE_FORMAT(CONVERT_TZ(wur.date_created, '+00:00', '+00:00'), '%%Y-%%m-%%dT%%H:%%i:%%sZ'),
						'date_modified', UNIX_TIMESTAMP(wur.date_modified) * 1000,
						'active', wur.active
					)
				) AS roles
			FROM workspace_user_role wur
			JOIN role r ON wur.role_id = r.id
			WHERE wur.active = 1 AND r.active = 1
			GROUP BY wur.workspace_user_id
		)
		SELECT
			wu.id,
			wu.workspace_id,
			wu.user_id,
			wu.active,
			wu.date_created,
			wu.date_modified,
			u.id AS user_id_value,
			u.first_name AS user_first_name,
			u.last_name AS user_last_name,
			u.email_address AS user_email_address,
			u.mobile_number AS user_phone_number,
			u.active AS user_active,
			COALESCE(ura.roles, JSON_ARRAY()) AS workspace_user_roles,
			COUNT(*) OVER() AS total_count
		FROM workspace_user wu
		LEFT JOIN `+"`user`"+` u ON wu.user_id = u.id AND u.active = 1
		LEFT JOIN user_roles_agg ura ON wu.id = ura.workspace_user_id
		WHERE %s%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, hardWhere, extraWhere, sortCol, sortOrder)

	exec2 := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec2.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace user list page data: %w", err)
	}
	defer rows.Close()

	var workspaceUsers []*workspaceuserpb.WorkspaceUser
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			workspaceId  string
			userId       string
			active       bool
			dateCreated  time.Time
			dateModified time.Time
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

		if userIdValue != nil {
			workspaceUser.User = &userpb.User{
				Id:           *userIdValue,
				FirstName:    derefStr(userFirstName),
				LastName:     derefStr(userLastName),
				EmailAddress: derefStr(userEmailAddress),
				MobileNumber: derefStr(userPhoneNumber),
				Active:       userActive != nil && *userActive,
			}
		}

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

		if len(workspaceUserRolesJSON) > 0 {
			var rolesData []map[string]interface{}
			if err := json.Unmarshal(workspaceUserRolesJSON, &rolesData); err == nil {
				for _, roleData := range rolesData {
					roleJSON, err := json.Marshal(roleData)
					if err != nil {
						continue
					}
					wur := &workspaceuserrolepb.WorkspaceUserRole{}
					if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(roleJSON, wur); err != nil {
						log.Printf("Failed to unmarshal workspace_user_role JSON: %v (json: %s)", err, string(roleJSON))
						continue
					}
					workspaceUser.WorkspaceUserRoles = append(workspaceUser.WorkspaceUserRoles, wur)
				}
			}
		}

		workspaceUsers = append(workspaceUsers, workspaceUser)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workspace user rows: %w", err)
	}

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

// GetWorkspaceUserItemPageData retrieves a single workspace user with enriched user data and aggregated roles.
//
// Dialect translation:
//   - "user" → `user`; jsonb_agg → JSON_ARRAYAGG; jsonb_build_object → JSON_OBJECT;
//   - COALESCE('[]'::jsonb) → JSON_ARRAY(); $N → ?; active = true → active = 1;
//   - $2::text = ” OR wu.workspace_id = $2::text → ? = ” OR wu.workspace_id = ?
func (r *MySQLWorkspaceUserRepository) GetWorkspaceUserItemPageData(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user item page data request is required")
	}
	if req.WorkspaceUserId == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	// Dialect: "user" → `user`; JSONB → JSON; active = true → active = 1; $N → ?
	query := `
		WITH user_roles_agg AS (
			SELECT
				wur.workspace_user_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', wur.id,
						'workspace_user_id', wur.workspace_user_id,
						'role_id', wur.role_id,
						'role', JSON_OBJECT(
							'id', r.id,
							'name', r.name,
							'description', r.description,
							'color', r.color,
							'active', r.active
						),
						'date_created', UNIX_TIMESTAMP(wur.date_created) * 1000,
						'date_created_string', DATE_FORMAT(CONVERT_TZ(wur.date_created, '+00:00', '+00:00'), '%Y-%m-%dT%H:%i:%sZ'),
						'date_modified', UNIX_TIMESTAMP(wur.date_modified) * 1000,
						'active', wur.active
					)
				) AS roles
			FROM workspace_user_role wur
			JOIN role r ON wur.role_id = r.id
			WHERE wur.active = 1 AND r.active = 1
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
				u.id AS user_id_value,
				u.first_name AS user_first_name,
				u.last_name AS user_last_name,
				u.email_address AS user_email_address,
				u.mobile_number AS user_phone_number,
				u.active AS user_active,
				COALESCE(ura.roles, JSON_ARRAY()) AS workspace_user_roles
			FROM workspace_user wu
			LEFT JOIN ` + "`user`" + ` u ON wu.user_id = u.id AND u.active = 1
			LEFT JOIN user_roles_agg ura ON wu.id = ura.workspace_user_id
			WHERE wu.id = ? AND wu.active = 1
			  AND (? = '' OR wu.workspace_id = ?)
		)
		SELECT * FROM enriched LIMIT 1;
	`

	wsID := identity.Must(ctx).WorkspaceID
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.WorkspaceUserId, wsID, wsID)

	var (
		id           string
		workspaceId  string
		userId       string
		active       bool
		dateCreated  time.Time
		dateModified time.Time
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

	if len(workspaceUserRolesJSON) > 0 {
		var rolesData []map[string]interface{}
		if err := json.Unmarshal(workspaceUserRolesJSON, &rolesData); err == nil {
			for _, roleData := range rolesData {
				roleJSON, err := json.Marshal(roleData)
				if err != nil {
					continue
				}
				wur := &workspaceuserrolepb.WorkspaceUserRole{}
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(roleJSON, wur); err != nil {
					log.Printf("Failed to unmarshal workspace_user_role JSON: %v (json: %s)", err, string(roleJSON))
					continue
				}
				workspaceUser.WorkspaceUserRoles = append(workspaceUser.WorkspaceUserRoles, wur)
			}
		}
	}

	return &workspaceuserpb.GetWorkspaceUserItemPageDataResponse{
		WorkspaceUser: workspaceUser,
		Success:       true,
	}, nil
}

// NewWorkspaceUserRepository creates a new MySQL workspace_user repository (old-style constructor).
func NewWorkspaceUserRepository(db *sql.DB, tableName string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLWorkspaceUserRepository(dbOps, tableName)
}
