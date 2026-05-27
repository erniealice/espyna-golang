//go:build sqlserver

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.WorkspaceUser, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver workspace_user repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerWorkspaceUserRepository(dbOps, tableName), nil
	})
}

// SQLServerWorkspaceUserRepository implements workspace user CRUD operations using SQL Server.
type SQLServerWorkspaceUserRepository struct {
	workspaceuserpb.UnimplementedWorkspaceUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerWorkspaceUserRepository creates a new SQL Server workspace user repository.
func NewSQLServerWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, tableName string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	if tableName == "" {
		tableName = "workspace_user"
	}
	return &SQLServerWorkspaceUserRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateWorkspaceUser creates a new workspace user using common SQL Server operations.
func (r *SQLServerWorkspaceUserRepository) CreateWorkspaceUser(ctx context.Context, req *workspaceuserpb.CreateWorkspaceUserRequest) (*workspaceuserpb.CreateWorkspaceUserResponse, error) {
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

	return &workspaceuserpb.CreateWorkspaceUserResponse{Data: []*workspaceuserpb.WorkspaceUser{workspaceUser}}, nil
}

// ReadWorkspaceUser retrieves a workspace user using common SQL Server operations.
func (r *SQLServerWorkspaceUserRepository) ReadWorkspaceUser(ctx context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
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

	return &workspaceuserpb.ReadWorkspaceUserResponse{Data: []*workspaceuserpb.WorkspaceUser{workspaceUser}}, nil
}

// UpdateWorkspaceUser updates a workspace user using common SQL Server operations.
func (r *SQLServerWorkspaceUserRepository) UpdateWorkspaceUser(ctx context.Context, req *workspaceuserpb.UpdateWorkspaceUserRequest) (*workspaceuserpb.UpdateWorkspaceUserResponse, error) {
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

	return &workspaceuserpb.UpdateWorkspaceUserResponse{Data: []*workspaceuserpb.WorkspaceUser{workspaceUser}}, nil
}

// DeleteWorkspaceUser deletes a workspace user using common SQL Server operations.
func (r *SQLServerWorkspaceUserRepository) DeleteWorkspaceUser(ctx context.Context, req *workspaceuserpb.DeleteWorkspaceUserRequest) (*workspaceuserpb.DeleteWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete workspace user: %w", err)
	}

	return &workspaceuserpb.DeleteWorkspaceUserResponse{Success: true}, nil
}

// ListWorkspaceUsers lists workspace users with joined user data.
//
// SQL Server translation:
//   - "user" → [user] (reserved word).
//   - $1 → @p1.
//   - active = true → active = 1.
//   - "($1::text = ” OR wu.workspace_id = $1::text)" → "(@p1 = ” OR wu.workspace_id = @p1)".
func (r *SQLServerWorkspaceUserRepository) ListWorkspaceUsers(ctx context.Context, req *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	query := `
		SELECT
			wu.id, wu.workspace_id, wu.user_id, wu.active,
			wu.date_created, wu.date_modified,
			u.id, u.first_name, u.last_name, u.email_address, u.mobile_number, u.active
		FROM workspace_user wu
		LEFT JOIN [user] u ON wu.user_id = u.id
		WHERE wu.active = 1
		  AND (@p1 = '' OR wu.workspace_id = @p1)
		ORDER BY wu.date_created DESC
	`

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, wsID)
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

	return &workspaceuserpb.ListWorkspaceUsersResponse{Data: workspaceUsers}, nil
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

// GetWorkspaceUserListPageData retrieves workspace users with filtering, sorting, and pagination.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server translation notes:
//   - "user" → [user].
//   - $N → @pN.
//   - ILIKE → LIKE.
//   - jsonb_agg(jsonb_build_object(...)) → FOR JSON PATH correlated subquery.
//   - wu.active = true → wu.active = 1; r.active = true → r.active = 1.
//   - LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER () retained (SQL Server 2017+).
func (r *SQLServerWorkspaceUserRepository) GetWorkspaceUserListPageData(
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
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(filteredReqFilters, req.Search, searchFields, 2)

	// Hard WHERE: always active + workspace_id.
	hardWhere := "wu.active = 1 AND wu.workspace_id = @p1"
	extraWhere := ""
	if len(filterClauses) > 0 {
		extraWhere = " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	allArgs := []any{workspaceID}
	allArgs = append(allArgs, filterArgs...)
	allArgs = append(allArgs, offset, limit)

	// FOR JSON PATH subquery replaces jsonb_agg(jsonb_build_object(...)).
	// Returns NULL when no active roles exist; Go maps NULL to empty slice.
	query := fmt.Sprintf(`
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
			(SELECT
				wur.id,
				wur.workspace_user_id,
				wur.role_id,
				wur.active,
				r.id AS [role.id],
				r.name AS [role.name],
				r.description AS [role.description],
				r.color AS [role.color],
				r.active AS [role.active]
			 FROM workspace_user_role wur
			 JOIN role r ON wur.role_id = r.id
			 WHERE wur.workspace_user_id = wu.id AND wur.active = 1 AND r.active = 1
			 FOR JSON PATH) AS workspace_user_roles,
			COUNT(*) OVER () AS total_count
		FROM workspace_user wu
		LEFT JOIN [user] u ON wu.user_id = u.id AND u.active = 1
		WHERE %s%s
		ORDER BY %s %s
		OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY
	`, hardWhere, extraWhere, sortCol, sortOrder, offsetIdx, limitIdx)

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

		// FOR JSON PATH returns NULL when no rows — treat as empty slice.
		rolesJSON := workspaceUserRolesJSON
		if len(rolesJSON) == 0 {
			rolesJSON = []byte("[]")
		}
		if len(rolesJSON) > 0 {
			var rolesData []map[string]interface{}
			if err := json.Unmarshal(rolesJSON, &rolesData); err == nil {
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

	return &workspaceuserpb.GetWorkspaceUserListPageDataResponse{
		WorkspaceUserList: workspaceUsers,
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

// GetWorkspaceUserItemPageData retrieves a single workspace user with enriched user and role data.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server translation: TOP 1 replaces LIMIT 1; [user] replaces "user";
// FOR JSON PATH replaces jsonb_agg; active = 1 replaces active = true.
func (r *SQLServerWorkspaceUserRepository) GetWorkspaceUserItemPageData(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user item page data request is required")
	}
	if req.WorkspaceUserId == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	query := `
		SELECT TOP 1
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
			(SELECT
				wur.id,
				wur.workspace_user_id,
				wur.role_id,
				wur.active,
				r.id AS [role.id],
				r.name AS [role.name],
				r.description AS [role.description],
				r.color AS [role.color],
				r.active AS [role.active]
			 FROM workspace_user_role wur
			 JOIN role r ON wur.role_id = r.id
			 WHERE wur.workspace_user_id = wu.id AND wur.active = 1 AND r.active = 1
			 FOR JSON PATH) AS workspace_user_roles
		FROM workspace_user wu
		LEFT JOIN [user] u ON wu.user_id = u.id AND u.active = 1
		WHERE wu.id = @p1 AND wu.active = 1
		  AND (@p2 = '' OR wu.workspace_id = @p2);
	`

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.WorkspaceUserId, wsID)

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

	rolesJSON := workspaceUserRolesJSON
	if len(rolesJSON) == 0 {
		rolesJSON = []byte("[]")
	}
	if len(rolesJSON) > 0 {
		var rolesData []map[string]interface{}
		if err := json.Unmarshal(rolesJSON, &rolesData); err == nil {
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

// NewWorkspaceUserRepository creates a new SQL Server workspace_user repository (old-style constructor).
func NewWorkspaceUserRepository(db *sql.DB, tableName string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerWorkspaceUserRepository(dbOps, tableName)
}
