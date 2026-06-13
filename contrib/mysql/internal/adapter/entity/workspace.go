//go:build mysql

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
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Workspace, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql workspace repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLWorkspaceRepository(dbOps, tableName), nil
	})
}

// MySQLWorkspaceRepository implements workspace CRUD operations using MySQL 8.0+.
type MySQLWorkspaceRepository struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLWorkspaceRepository creates a new MySQL workspace repository.
func NewMySQLWorkspaceRepository(dbOps interfaces.DatabaseOperation, tableName string) workspacepb.WorkspaceDomainServiceServer {
	if tableName == "" {
		tableName = "workspace"
	}
	return &MySQLWorkspaceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateWorkspace creates a new workspace using common MySQL operations.
func (r *MySQLWorkspaceRepository) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace data is required")
	}

	// EmitUnpopulated: false booleans are preserved instead of disappearing from the JSON payload.
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspacepb.CreateWorkspaceResponse{
		Data: []*workspacepb.Workspace{workspace},
	}, nil
}

// ReadWorkspace retrieves a workspace using common MySQL operations.
func (r *MySQLWorkspaceRepository) ReadWorkspace(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspacepb.ReadWorkspaceResponse{
		Data:    []*workspacepb.Workspace{workspace},
		Success: true,
	}, nil
}

// UpdateWorkspace updates a workspace using common MySQL operations.
func (r *MySQLWorkspaceRepository) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspacepb.UpdateWorkspaceResponse{
		Data: []*workspacepb.Workspace{workspace},
	}, nil
}

// DeleteWorkspace deletes a workspace using common MySQL operations (soft delete).
func (r *MySQLWorkspaceRepository) DeleteWorkspace(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete workspace: %w", err)
	}

	return &workspacepb.DeleteWorkspaceResponse{
		Success: true,
	}, nil
}

var workspaceSortableSQLCols = []string{
	"id", "active", "name", "description", "private", "status",
	"date_created", "date_modified",
}

var workspaceSortSpec = espynahttp.SortSpec{AllowedCols: workspaceSortableSQLCols}

// ListWorkspaces lists workspaces using common MySQL operations.
func (r *MySQLWorkspaceRepository) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		workspace := &workspacepb.Workspace{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
			continue
		}
		workspaces = append(workspaces, workspace)
	}

	return &workspacepb.ListWorkspacesResponse{
		Data: workspaces,
	}, nil
}

// GetWorkspaceListPageData retrieves workspaces with filtering, sorting, searching, and pagination.
//
// Dialect translation from postgres gold standard:
//   - $1/$2,... → ? (MySQL positional placeholders, args in same left-to-right order)
//   - ILIKE → LIKE
//   - COUNT(*) OVER () stays — MySQL 8.0+ supports window functions
//   - mysqlCore.BuildOrderBy uses backtick quoting
func (r *MySQLWorkspaceRepository) GetWorkspaceListPageData(
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

	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Build filter/search WHERE clauses (start at idx 1 — no leading workspace_id binding for workspace table).
	searchFields := []string{"w.name", "w.description"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)

	whereSQL := ""
	if len(filterClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(filterClauses, " AND ")
	}

	queryArgs := append(filterArgs, limit, offset)

	// CTE query — MySQL 8.0+ supports CTEs and COUNT(*) OVER ().
	// Dialect: ILIKE → LIKE; $N → ?; LIMIT/OFFSET use positional ?
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
		ORDER BY %s %s
		LIMIT ? OFFSET ?;
	`, whereSQL, sortField, sortOrder)

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

	hasNext := page < totalPages
	hasPrev := page > 1

	return &workspacepb.GetWorkspaceListPageDataResponse{
		WorkspaceList: workspaces,
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

// GetWorkspaceItemPageData retrieves a single workspace with enhanced item page data.
//
// Dialect translation: $1 → ?; no dialect-specific syntax differences for this simple SELECT.
func (r *MySQLWorkspaceRepository) GetWorkspaceItemPageData(
	ctx context.Context,
	req *workspacepb.GetWorkspaceItemPageDataRequest,
) (*workspacepb.GetWorkspaceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace item page data request is required")
	}
	if req.WorkspaceId == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Dialect: $1 → ?
	query := `
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
		WHERE w.id = ?
		LIMIT 1;
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

	return &workspacepb.GetWorkspaceItemPageDataResponse{
		Workspace: workspace,
		Success:   true,
	}, nil
}

// SwitchWorkspace updates the session row's workspace_id and workspace_user_id.
//
// Dialect translation from postgres gold standard:
//   - "session" → `session` (reserved word in MySQL)
//   - $1/$2/$3 → ? (positional)
//   - active = true → active = 1
func (r *MySQLWorkspaceRepository) SwitchWorkspace(ctx context.Context, req *workspacepb.SwitchWorkspaceRequest) (*workspacepb.SwitchWorkspaceResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	userID := identity.Must(ctx).UserID
	if userID == "" {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "unauthorized"}}, nil
	}

	// Dialect: $1/$2 → ?; active = true → active = 1
	var wsUserID string
	err := exec.QueryRowContext(ctx,
		`SELECT wu.id FROM workspace_user wu
		 WHERE wu.user_id = ? AND wu.workspace_id = ? AND wu.active = 1
		 LIMIT 1`,
		userID, req.WorkspaceId,
	).Scan(&wsUserID)
	if err != nil {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "no access to workspace"}}, nil
	}

	var wsName string
	_ = exec.QueryRowContext(ctx,
		`SELECT name FROM workspace WHERE id = ? AND active = 1`,
		req.WorkspaceId,
	).Scan(&wsName)

	// Dialect: "session" → `session`; $N → ?; active = true → active = 1
	res, err := exec.ExecContext(ctx,
		"UPDATE `session` SET workspace_id = ?, workspace_user_id = ? WHERE token = ? AND active = 1",
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
// Dialect translation: $1 → ?; active = true → active = 1.
func (r *MySQLWorkspaceRepository) ListUserWorkspaces(ctx context.Context, req *workspacepb.ListUserWorkspacesRequest) (*workspacepb.ListUserWorkspacesResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	// Dialect: $1 → ?; active = true → active = 1
	rows, err := exec.QueryContext(ctx,
		`SELECT w.id, w.name, wu.id AS workspace_user_id
		 FROM workspace w
		 JOIN workspace_user wu ON wu.workspace_id = w.id
		 WHERE wu.user_id = ? AND wu.active = 1 AND w.active = 1
		 ORDER BY w.name`,
		req.UserId,
	)
	if err != nil {
		return &workspacepb.ListUserWorkspacesResponse{Success: false}, nil
	}
	defer rows.Close()

	currentWsID := identity.Must(ctx).WorkspaceID
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

// NewWorkspaceRepository creates a new MySQL workspace repository (old-style constructor).
func NewWorkspaceRepository(db *sql.DB, tableName string) workspacepb.WorkspaceDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLWorkspaceRepository(dbOps, tableName)
}
