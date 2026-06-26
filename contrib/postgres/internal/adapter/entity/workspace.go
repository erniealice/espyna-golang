//go:build postgresql

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
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Workspace, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres workspace repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresWorkspaceRepository(dbOps, tableName), nil
	})
}

// PostgresWorkspaceRepository implements workspace CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_workspace_active ON workspace(active) WHERE active = true - Filter active workspaces
//   - CREATE INDEX idx_workspace_name ON workspace(name) - Search on name field
//   - CREATE INDEX idx_workspace_name_trgm ON workspace USING gin(name gin_trgm_ops) - Fuzzy search support
//   - CREATE INDEX idx_workspace_date_created ON workspace(date_created DESC) - Default sorting
type PostgresWorkspaceRepository struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresWorkspaceRepository creates a new PostgreSQL workspace repository
func NewPostgresWorkspaceRepository(dbOps interfaces.DatabaseOperation, tableName string) workspacepb.WorkspaceDomainServiceServer {
	if tableName == "" {
		tableName = "workspace" // default fallback
	}

	return &PostgresWorkspaceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateWorkspace creates a new workspace using common PostgreSQL operations
func (r *PostgresWorkspaceRepository) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace data is required")
	}

	// Emit unpopulated fields so false booleans are preserved instead of
	// disappearing from the JSON payload and becoming NULL on insert.
	jsonData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(req.Data)
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
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// ReadWorkspace retrieves a workspace using common PostgreSQL operations
func (r *PostgresWorkspaceRepository) ReadWorkspace(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// UpdateWorkspace updates a workspace using common PostgreSQL operations
func (r *PostgresWorkspaceRepository) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Emit unpopulated fields so false booleans are preserved instead of
	// disappearing from the JSON payload and becoming NULL on update.
	jsonData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(req.Data)
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
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// DeleteWorkspace deletes a workspace using common PostgreSQL operations
func (r *PostgresWorkspaceRepository) DeleteWorkspace(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
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

// ListWorkspaces lists workspaces using common PostgreSQL operations.
func (r *PostgresWorkspaceRepository) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
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

	// Convert results to protobuf slice using protojson
	var workspaces []*workspacepb.Workspace
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		workspace := &workspacepb.Workspace{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workspace); err != nil {
			// Log error and continue with next item
			continue
		}
		workspaces = append(workspaces, workspace)
	}

	return &workspacepb.ListWorkspacesResponse{
		Data: workspaces,
	}, nil
}

// GetWorkspaceListPageData retrieves workspaces with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresWorkspaceRepository) GetWorkspaceListPageData(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace list page data request is required")
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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). Mirrors the
	// supplier.GetSupplierListPageData exemplar: route the caller-supplied sort
	// column through core.BuildOrderBy so an unknown column errors instead of
	// being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(workspaceSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	// Build filter/search WHERE clauses (start at $1)
	searchFields := []string{"w.name", "w.description"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)

	whereSQL := ""
	if len(filterClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(filterClauses, " AND ")
	}

	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	queryArgs := append(filterArgs, limit, offset)

	// CTE Query - Single round-trip with filtering and pagination
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
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s
		LIMIT $%d OFFSET $%d;
	`, whereSQL, orderByClause, limitIdx, offsetIdx)

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

	// Calculate pagination metadata
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

// GetWorkspaceItemPageData retrieves a single workspace with enhanced item page data
func (r *PostgresWorkspaceRepository) GetWorkspaceItemPageData(
	ctx context.Context,
	req *workspacepb.GetWorkspaceItemPageDataRequest,
) (*workspacepb.GetWorkspaceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace item page data request is required")
	}
	if req.WorkspaceId == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Simple query for single workspace item
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
		WHERE w.id = $1
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
func (r *PostgresWorkspaceRepository) SwitchWorkspace(ctx context.Context, req *workspacepb.SwitchWorkspaceRequest) (*workspacepb.SwitchWorkspaceResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	// 1. Validate: get user_id from context
	userID := identity.Must(ctx).UserID
	if userID == "" {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "unauthorized"}}, nil
	}

	// 2. Check workspace_user exists for this user + target workspace
	var wsUserID string
	err := exec.QueryRowContext(ctx,
		`SELECT wu.id FROM workspace_user wu
		 WHERE wu.user_id = $1 AND wu.workspace_id = $2 AND wu.active = true
		 LIMIT 1`,
		userID, req.WorkspaceId,
	).Scan(&wsUserID)
	if err != nil {
		return &workspacepb.SwitchWorkspaceResponse{Success: false, Error: &commonpb.Error{Message: "no access to workspace"}}, nil
	}

	// 3. Get workspace name
	var wsName string
	_ = exec.QueryRowContext(ctx,
		`SELECT name FROM workspace WHERE id = $1 AND active = true`,
		req.WorkspaceId,
	).Scan(&wsName)

	// 4. Update session (A6 — check RowsAffected; 0 means the session token did
	// not match an active row, so the switch silently no-opped before this fix).
	res, err := exec.ExecContext(ctx,
		`UPDATE "session" SET workspace_id = $1, workspace_user_id = $2
		 WHERE token = $3 AND active = true`,
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
func (r *PostgresWorkspaceRepository) ListUserWorkspaces(ctx context.Context, req *workspacepb.ListUserWorkspacesRequest) (*workspacepb.ListUserWorkspacesResponse, error) {
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	rows, err := exec.QueryContext(ctx,
		`SELECT w.id, w.name, wu.id AS workspace_user_id
		 FROM workspace w
		 JOIN workspace_user wu ON wu.workspace_id = w.id
		 WHERE wu.user_id = $1 AND wu.active = true AND w.active = true
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

// NewWorkspaceRepository creates a new PostgreSQL workspace repository (old-style constructor)
func NewWorkspaceRepository(db *sql.DB, tableName string) workspacepb.WorkspaceDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresWorkspaceRepository(dbOps, tableName)
}
