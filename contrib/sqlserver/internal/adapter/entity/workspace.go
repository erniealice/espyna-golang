//go:build sqlserver

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Workspace, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver workspace repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerWorkspaceRepository(dbOps, tableName), nil
	})
}

// SQLServerWorkspaceRepository implements workspace CRUD operations using SQL Server.
type SQLServerWorkspaceRepository struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerWorkspaceRepository creates a new SQL Server workspace repository.
func NewSQLServerWorkspaceRepository(dbOps interfaces.DatabaseOperation, tableName string) workspacepb.WorkspaceDomainServiceServer {
	if tableName == "" {
		tableName = "workspace"
	}
	return &SQLServerWorkspaceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateWorkspace creates a new workspace using common SQL Server operations.
func (r *SQLServerWorkspaceRepository) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace data is required")
	}

	// EmitUnpopulated preserves false booleans (e.g. private=false) that would otherwise
	// disappear from the JSON payload and become NULL on insert.
	jsonData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspacepb.CreateWorkspaceResponse{Data: []*workspacepb.Workspace{workspace}}, nil
}

// ReadWorkspace retrieves a workspace using common SQL Server operations.
func (r *SQLServerWorkspaceRepository) ReadWorkspace(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspacepb.ReadWorkspaceResponse{Data: []*workspacepb.Workspace{workspace}, Success: true}, nil
}

// UpdateWorkspace updates a workspace using common SQL Server operations.
func (r *SQLServerWorkspaceRepository) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	jsonData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspacepb.UpdateWorkspaceResponse{Data: []*workspacepb.Workspace{workspace}}, nil
}

// DeleteWorkspace deletes a workspace using common SQL Server operations (soft delete).
func (r *SQLServerWorkspaceRepository) DeleteWorkspace(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete workspace: %w", err)
	}

	return &workspacepb.DeleteWorkspaceResponse{Success: true}, nil
}

var workspaceSortableSQLCols = []string{
	"id", "active", "name", "description", "private", "status",
	"date_created", "date_modified",
}

var workspaceSortSpec = espynahttp.SortSpec{AllowedCols: workspaceSortableSQLCols}

// ListWorkspaces lists workspaces using common SQL Server operations.
func (r *SQLServerWorkspaceRepository) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
	if err := espynahttp.ValidateSortColumns(workspaceSortSpec, req.GetSort(), "workspace"); err != nil {
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
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	var workspaces []*workspacepb.Workspace
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		workspace := &workspacepb.Workspace{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
			continue
		}
		workspaces = append(workspaces, workspace)
	}

	return &workspacepb.ListWorkspacesResponse{Data: workspaces}, nil
}

// GetWorkspaceListPageData retrieves workspaces with filtering, sorting, and pagination.
//
// SQL Server translation notes:
//   - $N → @pN.
//   - ILIKE → LIKE.
//   - LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER () retained (SQL Server 2017+).
//   - Workspace has no workspace_id predicate (it IS the workspace root entity).
func (r *SQLServerWorkspaceRepository) GetWorkspaceListPageData(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace list page data request is required")
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

	orderByClause, err := sqlserverCore.BuildOrderBy(workspaceSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	searchFields := []string{"w.name", "w.description"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)

	whereSQL := ""
	if len(filterClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := append(filterArgs, offset, limit)

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				w.id,
				w.name,
				w.description,
				w.private,
				w.workflow_template_id,
				w.active,
				w.date_created,
				w.date_modified
			FROM workspace w
			%s
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, orderByClause, offsetIdx, limitIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace list page data: %w", err)
	}
	defer rows.Close()

	var workspaces []*workspacepb.Workspace
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			name               string
			description        *string
			private            bool
			workflowTemplateID *string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&private,
			&workflowTemplateID,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace row: %w", err)
		}

		totalCount = total

		workspace := &workspacepb.Workspace{
			Id:      id,
			Name:    name,
			Private: private,
			Active:  active,
		}

		if description != nil {
			workspace.Description = *description
		}
		if workflowTemplateID != nil {
			workspace.WorkflowTemplateId = workflowTemplateID
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			workspace.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			workspace.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			workspace.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			workspace.DateModifiedString = &dmStr
		}

		workspaces = append(workspaces, workspace)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workspace rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &workspacepb.GetWorkspaceListPageDataResponse{
		WorkspaceList: workspaces,
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

// GetWorkspaceItemPageData retrieves a single workspace with enhanced item page data.
//
// SQL Server translation: TOP 1 replaces LIMIT 1; @p1 replaces $1.
func (r *SQLServerWorkspaceRepository) GetWorkspaceItemPageData(
	ctx context.Context,
	req *workspacepb.GetWorkspaceItemPageDataRequest,
) (*workspacepb.GetWorkspaceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace item page data request is required")
	}
	if req.WorkspaceId == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	query := `
		SELECT TOP 1
			w.id,
			w.name,
			w.description,
			w.private,
			w.workflow_template_id,
			w.active,
			w.date_created,
			w.date_modified
		FROM workspace w
		WHERE w.id = @p1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.WorkspaceId)

	var (
		id                 string
		name               string
		description        *string
		private            bool
		workflowTemplateID *string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&private,
		&workflowTemplateID,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace with ID '%s' not found", req.WorkspaceId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace item page data: %w", err)
	}

	workspace := &workspacepb.Workspace{
		Id:      id,
		Name:    name,
		Private: private,
		Active:  active,
	}

	if description != nil {
		workspace.Description = *description
	}
	if workflowTemplateID != nil {
		workspace.WorkflowTemplateId = workflowTemplateID
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		workspace.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		workspace.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		workspace.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		workspace.DateModifiedString = &dmStr
	}

	return &workspacepb.GetWorkspaceItemPageDataResponse{Workspace: workspace, Success: true}, nil
}

// SwitchWorkspace updates the session row's workspace_id and workspace_user_id.
//
// SQL Server translation:
//   - "session" → [session] (reserved word in T-SQL).
//   - $N → @pN.
//   - active = true → active = 1.
//   - LIMIT 1 → TOP 1.
func (r *SQLServerWorkspaceRepository) SwitchWorkspace(ctx context.Context, req *workspacepb.SwitchWorkspaceRequest) (*workspacepb.SwitchWorkspaceResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	userID := consumer.GetUserIDFromContext(ctx)
	if userID == "" {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "unauthorized"}}, nil
	}

	// Check workspace_user exists for this user + target workspace.
	var wsUserID string
	err := exec.QueryRowContext(ctx,
		`SELECT TOP 1 wu.id FROM workspace_user wu
		 WHERE wu.user_id = @p1 AND wu.workspace_id = @p2 AND wu.active = 1`,
		userID, req.WorkspaceId,
	).Scan(&wsUserID)
	if err != nil {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "no access to workspace"}}, nil
	}

	// Get workspace name.
	var wsName string
	_ = exec.QueryRowContext(ctx,
		`SELECT TOP 1 name FROM workspace WHERE id = @p1 AND active = 1`,
		req.WorkspaceId,
	).Scan(&wsName)

	// Update session — check RowsAffected; 0 means session token did not match active row.
	// [session] required because SESSION is a T-SQL reserved word.
	res, err := exec.ExecContext(ctx,
		`UPDATE [session] SET workspace_id = @p1, workspace_user_id = @p2
		 WHERE token = @p3 AND active = 1`,
		req.WorkspaceId, wsUserID, req.SessionToken,
	)
	if err != nil {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "failed to update session"}}, nil
	}
	if affected, raErr := res.RowsAffected(); raErr != nil {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "failed to update session"}}, nil
	} else if affected == 0 {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "session not found"}}, nil
	}

	return &workspacepb.SwitchWorkspaceResponse{
		Success:         true,
		WorkspaceUserId: &wsUserID,
		WorkspaceName:   &wsName,
	}, nil
}

// ListUserWorkspaces returns all workspaces accessible to a user.
//
// SQL Server translation: @p1 replaces $1; active = 1 replaces active = true.
func (r *SQLServerWorkspaceRepository) ListUserWorkspaces(ctx context.Context, req *workspacepb.ListUserWorkspacesRequest) (*workspacepb.ListUserWorkspacesResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	rows, err := exec.QueryContext(ctx,
		`SELECT w.id, w.name, wu.id AS workspace_user_id
		 FROM workspace w
		 JOIN workspace_user wu ON wu.workspace_id = w.id
		 WHERE wu.user_id = @p1 AND wu.active = 1 AND w.active = 1
		 ORDER BY w.name`,
		req.UserId,
	)
	if err != nil {
		return &workspacepb.ListUserWorkspacesResponse{Success: false}, nil
	}
	defer rows.Close()

	currentWsID := consumer.GetWorkspaceIDFromContext(ctx)
	var workspaces []*workspacepb.UserWorkspace
	for rows.Next() {
		var ws workspacepb.UserWorkspace
		if err := rows.Scan(&ws.WorkspaceId, &ws.WorkspaceName, &ws.WorkspaceUserId); err != nil {
			continue
		}
		ws.IsCurrent = ws.WorkspaceId == currentWsID
		workspaces = append(workspaces, &ws)
	}

	return &workspacepb.ListUserWorkspacesResponse{Workspaces: workspaces, Success: true}, nil
}

// NewWorkspaceRepository creates a new SQL Server workspace repository (old-style constructor).
func NewWorkspaceRepository(db *sql.DB, tableName string) workspacepb.WorkspaceDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerWorkspaceRepository(dbOps, tableName)
}
