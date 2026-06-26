//go:build postgresql

package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Workflow, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres workflow repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresWorkflowRepository(dbOps, tableName), nil
	})
}

// PostgresWorkflowRepository implements workflow CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_workflow_active ON workflow(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_workflow_workspace_id ON workflow(workspace_id) - Multi-tenant scoping
//   - CREATE INDEX idx_workflow_status ON workflow(status) - Filter by status
//   - CREATE INDEX idx_workflow_workflow_template_id ON workflow(workflow_template_id) - FK lookup
//   - CREATE INDEX idx_workflow_date_created ON workflow(date_created DESC) - Default sorting
type PostgresWorkflowRepository struct {
	workflowpb.UnimplementedWorkflowDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresWorkflowRepository creates a new PostgreSQL workflow repository
func NewPostgresWorkflowRepository(dbOps interfaces.DatabaseOperation, tableName string) workflowpb.WorkflowDomainServiceServer {
	if tableName == "" {
		tableName = "workflow" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresWorkflowRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateWorkflow creates a new workflow using common PostgreSQL operations
func (r *PostgresWorkflowRepository) CreateWorkflow(ctx context.Context, req *workflowpb.CreateWorkflowRequest) (*workflowpb.CreateWorkflowResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workflow data is required")
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
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workflow := &workflowpb.Workflow{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workflowpb.CreateWorkflowResponse{
		Data:    []*workflowpb.Workflow{workflow},
		Success: true,
	}, nil
}

// ReadWorkflow retrieves a workflow using common PostgreSQL operations
func (r *PostgresWorkflowRepository) ReadWorkflow(ctx context.Context, req *workflowpb.ReadWorkflowRequest) (*workflowpb.ReadWorkflowResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workflow := &workflowpb.Workflow{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workflowpb.ReadWorkflowResponse{
		Data:    []*workflowpb.Workflow{workflow},
		Success: true,
	}, nil
}

// UpdateWorkflow updates a workflow using common PostgreSQL operations
func (r *PostgresWorkflowRepository) UpdateWorkflow(ctx context.Context, req *workflowpb.UpdateWorkflowRequest) (*workflowpb.UpdateWorkflowResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required")
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
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workflow := &workflowpb.Workflow{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workflowpb.UpdateWorkflowResponse{
		Data:    []*workflowpb.Workflow{workflow},
		Success: true,
	}, nil
}

// DeleteWorkflow deletes a workflow using common PostgreSQL operations (soft delete)
func (r *PostgresWorkflowRepository) DeleteWorkflow(ctx context.Context, req *workflowpb.DeleteWorkflowRequest) (*workflowpb.DeleteWorkflowResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete workflow: %w", err)
	}

	return &workflowpb.DeleteWorkflowResponse{
		Success: true,
	}, nil
}

// ListWorkflows lists workflows using common PostgreSQL operations
func (r *PostgresWorkflowRepository) ListWorkflows(ctx context.Context, req *workflowpb.ListWorkflowsRequest) (*workflowpb.ListWorkflowsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	var workflows []*workflowpb.Workflow
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		workflow := &workflowpb.Workflow{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, workflow); err != nil {
			continue
		}
		workflows = append(workflows, workflow)
	}

	if workflows == nil {
		workflows = make([]*workflowpb.Workflow, 0)
	}

	return &workflowpb.ListWorkflowsResponse{
		Data:    workflows,
		Success: true,
	}, nil
}

var workflowSortableSQLCols = []string{
	"id", "name", "status", "workspace_id", "date_created", "date_modified", "version",
}

// GetWorkflowListPageData retrieves paginated workflow list data with CTE.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *PostgresWorkflowRepository) GetWorkflowListPageData(
	ctx context.Context,
	req *workflowpb.GetWorkflowListPageDataRequest,
) (*workflowpb.GetWorkflowListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). Route the
	// caller-supplied sort column through core.BuildOrderBy so an unknown column
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(workflowSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				w.id,
				w.name,
				w.description,
				w.status,
				w.workspace_id,
				w.active,
				w.version,
				w.date_created,
				w.date_modified
			FROM workflow w
			WHERE w.active = true
			  AND w.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   w.name ILIKE $2 OR
				   w.description ILIKE $2 OR
				   w.status ILIKE $2)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s
		LIMIT $3 OFFSET $4;
	`, orderByClause)

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow list page data: %w", err)
	}
	defer rows.Close()

	var workflows []*workflowpb.Workflow
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			name         string
			description  sql.NullString
			status       string
			workspaceId  sql.NullString
			active       bool
			version      sql.NullInt32
			dateCreated  time.Time
			dateModified time.Time
			total        int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&status,
			&workspaceId,
			&active,
			&version,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow row: %w", err)
		}

		totalCount = total

		workflow := &workflowpb.Workflow{
			Id:     id,
			Name:   name,
			Status: status,
			Active: active,
		}

		if description.Valid {
			workflow.Description = &description.String
		}
		if workspaceId.Valid {
			workflow.WorkspaceId = &workspaceId.String
		}
		if version.Valid {
			workflow.Version = &version.Int32
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			workflow.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			workflow.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			workflow.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			workflow.DateModifiedString = &dmStr
		}

		workflows = append(workflows, workflow)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workflow rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &workflowpb.GetWorkflowListPageDataResponse{
		WorkflowList: workflows,
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

// GetWorkflowItemPageData retrieves a single workflow with enhanced item page data.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *PostgresWorkflowRepository) GetWorkflowItemPageData(
	ctx context.Context,
	req *workflowpb.GetWorkflowItemPageDataRequest,
) (*workflowpb.GetWorkflowItemPageDataResponse, error) {
	if req == nil || req.WorkflowId == "" {
		return nil, fmt.Errorf("workflow ID required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		SELECT
			w.id,
			w.name,
			w.description,
			w.status,
			w.workspace_id,
			w.active,
			w.version,
			w.date_created,
			w.date_modified
		FROM workflow w
		WHERE w.id = $1 AND w.workspace_id = $2 AND w.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.WorkflowId, workspaceID)

	var (
		id           string
		name         string
		description  sql.NullString
		status       string
		workspaceId  sql.NullString
		active       bool
		version      sql.NullInt32
		dateCreated  time.Time
		dateModified time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&status,
		&workspaceId,
		&active,
		&version,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow with ID '%s' not found", req.WorkflowId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow item page data: %w", err)
	}

	workflow := &workflowpb.Workflow{
		Id:     id,
		Name:   name,
		Status: status,
		Active: active,
	}

	if description.Valid {
		workflow.Description = &description.String
	}
	if workspaceId.Valid {
		workflow.WorkspaceId = &workspaceId.String
	}
	if version.Valid {
		workflow.Version = &version.Int32
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		workflow.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		workflow.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		workflow.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		workflow.DateModifiedString = &dmStr
	}

	return &workflowpb.GetWorkflowItemPageDataResponse{
		Workflow: workflow,
		Success:  true,
	}, nil
}

// NewWorkflowRepository creates a new PostgreSQL workflow repository (old-style constructor)
func NewWorkflowRepository(db *sql.DB, tableName string) workflowpb.WorkflowDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresWorkflowRepository(dbOps, tableName)
}
