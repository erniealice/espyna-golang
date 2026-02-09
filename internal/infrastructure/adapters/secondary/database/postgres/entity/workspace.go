//go:build postgres

package entity

import (
	"time"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "workspace", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres workspace repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
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
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresWorkspaceRepository creates a new PostgreSQL workspace repository
func NewPostgresWorkspaceRepository(dbOps interfaces.DatabaseOperation, tableName string) workspacepb.WorkspaceDomainServiceServer {
	if tableName == "" {
		tableName = "workspace" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresWorkspaceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateWorkspace creates a new workspace using common PostgreSQL operations
func (r *PostgresWorkspaceRepository) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("workspace data is required")
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
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := protojson.Unmarshal(resultJSON, workspace); err != nil {
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
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := protojson.Unmarshal(resultJSON, workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &workspacepb.ReadWorkspaceResponse{
		Data: []*workspacepb.Workspace{workspace},
	}, nil
}

// UpdateWorkspace updates a workspace using common PostgreSQL operations
func (r *PostgresWorkspaceRepository) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
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
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	workspace := &workspacepb.Workspace{}
	if err := protojson.Unmarshal(resultJSON, workspace); err != nil {
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

// ListWorkspaces lists workspaces using common PostgreSQL operations
func (r *PostgresWorkspaceRepository) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var workspaces []*workspacepb.Workspace
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		workspace := &workspacepb.Workspace{}
		if err := protojson.Unmarshal(resultJSON, workspace); err != nil {
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

	// CTE Query - Single round-trip with filtering and pagination
	query := `
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
			WHERE ($1::text IS NULL OR $1::text = '' OR
				   w.name ILIKE $1 OR
				   w.description ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
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

	row := r.db.QueryRowContext(ctx, query, req.WorkspaceId)

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


// NewWorkspaceRepository creates a new PostgreSQL workspace repository (old-style constructor)
func NewWorkspaceRepository(db *sql.DB, tableName string) workspacepb.WorkspaceDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresWorkspaceRepository(dbOps, tableName)
}
