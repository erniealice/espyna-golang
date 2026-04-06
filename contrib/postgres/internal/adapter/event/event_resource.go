package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresEventResourceRepository implements event resource CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_resource_active ON event_resource(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_event_resource_event_id ON event_resource(event_id) - Filter by event
//   - CREATE INDEX idx_event_resource_resource_id ON event_resource(resource_id) - Filter by resource
//   - CREATE INDEX idx_event_resource_workspace_id ON event_resource(workspace_id) - Tenant scope
//   - CREATE INDEX idx_event_resource_date_created ON event_resource(date_created DESC) - Default sorting
type PostgresEventResourceRepository struct {
	eventresourcepb.UnimplementedEventResourceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventResource, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_resource repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventResourceRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventResourceRepository creates a new PostgreSQL event resource repository
func NewPostgresEventResourceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventresourcepb.EventResourceDomainServiceServer {
	if tableName == "" {
		tableName = "event_resource" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventResourceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventResource creates a new event resource using common PostgreSQL operations
func (r *PostgresEventResourceRepository) CreateEventResource(ctx context.Context, req *eventresourcepb.CreateEventResourceRequest) (*eventresourcepb.CreateEventResourceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event resource data is required")
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
		return nil, fmt.Errorf("failed to create event resource: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventResource := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventresourcepb.CreateEventResourceResponse{
		Data: []*eventresourcepb.EventResource{eventResource},
	}, nil
}

// ReadEventResource retrieves an event resource using common PostgreSQL operations
func (r *PostgresEventResourceRepository) ReadEventResource(ctx context.Context, req *eventresourcepb.ReadEventResourceRequest) (*eventresourcepb.ReadEventResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event resource ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event resource: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventResource := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventresourcepb.ReadEventResourceResponse{
		Data: []*eventresourcepb.EventResource{eventResource},
	}, nil
}

// UpdateEventResource updates an event resource using common PostgreSQL operations
func (r *PostgresEventResourceRepository) UpdateEventResource(ctx context.Context, req *eventresourcepb.UpdateEventResourceRequest) (*eventresourcepb.UpdateEventResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event resource ID is required")
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
		return nil, fmt.Errorf("failed to update event resource: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventResource := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventresourcepb.UpdateEventResourceResponse{
		Data: []*eventresourcepb.EventResource{eventResource},
	}, nil
}

// DeleteEventResource deletes an event resource using common PostgreSQL operations
func (r *PostgresEventResourceRepository) DeleteEventResource(ctx context.Context, req *eventresourcepb.DeleteEventResourceRequest) (*eventresourcepb.DeleteEventResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event resource ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event resource: %w", err)
	}

	return &eventresourcepb.DeleteEventResourceResponse{
		Success: true,
	}, nil
}

// ListEventResources lists event resources using common PostgreSQL operations
func (r *PostgresEventResourceRepository) ListEventResources(ctx context.Context, req *eventresourcepb.ListEventResourcesRequest) (*eventresourcepb.ListEventResourcesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event resources: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var eventResources []*eventresourcepb.EventResource
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		eventResource := &eventresourcepb.EventResource{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventResource); err != nil {
			// Log error and continue with next item
			continue
		}
		eventResources = append(eventResources, eventResource)
	}

	return &eventresourcepb.ListEventResourcesResponse{
		Data: eventResources,
	}, nil
}

// GetEventResourceListPageData retrieves paginated event resource list data with CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventResourceRepository) GetEventResourceListPageData(
	ctx context.Context,
	req *eventresourcepb.GetEventResourceListPageDataRequest,
) (*eventresourcepb.GetEventResourceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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

	// CTE Query - Junction table pattern with event and generic resource FK filtering
	query := `
		WITH enriched AS (
			SELECT
				er.id,
				er.event_id,
				er.resource_id,
				er.resource_type,
				er.status,
				er.name,
				er.workspace_id,
				er.active,
				er.date_created,
				er.date_modified
			FROM event_resource er
			WHERE er.active = true
			  AND er.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   er.name ILIKE $2 OR
				   er.resource_id ILIKE $2 OR
				   er.event_id ILIKE $2)
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
		return nil, fmt.Errorf("failed to query event resource list page data: %w", err)
	}
	defer rows.Close()

	var eventResources []*eventresourcepb.EventResource
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			eventId      string
			resourceId   string
			resourceType int32
			status       int32
			name         sql.NullString
			workspaceId  string
			active       bool
			dateCreated  time.Time
			dateModified time.Time
			total        int64
		)

		err := rows.Scan(
			&id,
			&eventId,
			&resourceId,
			&resourceType,
			&status,
			&name,
			&workspaceId,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event resource row: %w", err)
		}

		totalCount = total

		eventResource := &eventresourcepb.EventResource{
			Id:           id,
			EventId:      eventId,
			ResourceId:   resourceId,
			ResourceType: eventresourcepb.ResourceType(resourceType),
			Status:       eventresourcepb.ResourceStatus(status),
			WorkspaceId:  workspaceId,
			Active:       active,
		}

		if name.Valid {
			eventResource.Name = &name.String
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			eventResource.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			eventResource.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			eventResource.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			eventResource.DateModifiedString = &dmStr
		}

		eventResources = append(eventResources, eventResource)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event resource rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventresourcepb.GetEventResourceListPageDataResponse{
		EventResourceList: eventResources,
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

// GetEventResourceItemPageData retrieves a single event resource with enhanced item page data
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventResourceRepository) GetEventResourceItemPageData(
	ctx context.Context,
	req *eventresourcepb.GetEventResourceItemPageDataRequest,
) (*eventresourcepb.GetEventResourceItemPageDataResponse, error) {
	if req == nil || req.EventResourceId == "" {
		return nil, fmt.Errorf("event resource ID required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Simple query for single event resource item
	query := `
		SELECT
			er.id,
			er.event_id,
			er.resource_id,
			er.resource_type,
			er.status,
			er.name,
			er.workspace_id,
			er.active,
			er.date_created,
			er.date_modified
		FROM event_resource er
		WHERE er.id = $1 AND er.workspace_id = $2 AND er.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.EventResourceId, workspaceID)

	var (
		id           string
		eventId      string
		resourceId   string
		resourceType int32
		status       int32
		name         sql.NullString
		workspaceId  string
		active       bool
		dateCreated  time.Time
		dateModified time.Time
	)

	err := row.Scan(
		&id,
		&eventId,
		&resourceId,
		&resourceType,
		&status,
		&name,
		&workspaceId,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event resource with ID '%s' not found", req.EventResourceId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event resource item page data: %w", err)
	}

	eventResource := &eventresourcepb.EventResource{
		Id:           id,
		EventId:      eventId,
		ResourceId:   resourceId,
		ResourceType: eventresourcepb.ResourceType(resourceType),
		Status:       eventresourcepb.ResourceStatus(status),
		WorkspaceId:  workspaceId,
		Active:       active,
	}

	if name.Valid {
		eventResource.Name = &name.String
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		eventResource.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		eventResource.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		eventResource.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		eventResource.DateModifiedString = &dmStr
	}

	return &eventresourcepb.GetEventResourceItemPageDataResponse{
		EventResource: eventResource,
		Success:       true,
	}, nil
}

// NewEventResourceRepository creates a new PostgreSQL event_resource repository (old-style constructor)
func NewEventResourceRepository(db *sql.DB, tableName string) eventresourcepb.EventResourceDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventResourceRepository(dbOps, tableName)
}
