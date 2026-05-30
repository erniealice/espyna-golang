//go:build postgresql

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresEventTagRepository implements event tag CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_tag_active ON event_tag(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_event_tag_workspace_id ON event_tag(workspace_id) - Multi-tenant scoping
//   - CREATE INDEX idx_event_tag_date_created ON event_tag(date_created DESC) - Default sorting
type PostgresEventTagRepository struct {
	eventtagpb.UnimplementedEventTagDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventTag, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_tag repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventTagRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventTagRepository creates a new PostgreSQL event tag repository
func NewPostgresEventTagRepository(dbOps interfaces.DatabaseOperation, tableName string) eventtagpb.EventTagDomainServiceServer {
	if tableName == "" {
		tableName = "event_tag" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventTagRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventTag creates a new event tag using common PostgreSQL operations
func (r *PostgresEventTagRepository) CreateEventTag(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event tag data is required")
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
		return nil, fmt.Errorf("failed to create event tag: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventTag := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventTag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventtagpb.CreateEventTagResponse{
		Data: []*eventtagpb.EventTag{eventTag},
	}, nil
}

// ReadEventTag retrieves an event tag using common PostgreSQL operations
func (r *PostgresEventTagRepository) ReadEventTag(ctx context.Context, req *eventtagpb.ReadEventTagRequest) (*eventtagpb.ReadEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event tag ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event tag: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventTag := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventTag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventtagpb.ReadEventTagResponse{
		Data: []*eventtagpb.EventTag{eventTag},
	}, nil
}

// UpdateEventTag updates an event tag using common PostgreSQL operations
func (r *PostgresEventTagRepository) UpdateEventTag(ctx context.Context, req *eventtagpb.UpdateEventTagRequest) (*eventtagpb.UpdateEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event tag ID is required")
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
		return nil, fmt.Errorf("failed to update event tag: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventTag := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventTag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventtagpb.UpdateEventTagResponse{
		Data: []*eventtagpb.EventTag{eventTag},
	}, nil
}

// DeleteEventTag deletes an event tag using common PostgreSQL operations (soft delete)
func (r *PostgresEventTagRepository) DeleteEventTag(ctx context.Context, req *eventtagpb.DeleteEventTagRequest) (*eventtagpb.DeleteEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event tag ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event tag: %w", err)
	}

	return &eventtagpb.DeleteEventTagResponse{
		Success: true,
	}, nil
}

var eventTagSortableSQLCols = []string{
	"id", "active", "name", "description", "color", "workspace_id",
	"date_created", "date_modified",
}

var eventTagSortSpec = espynahttp.SortSpec{AllowedCols: eventTagSortableSQLCols}

// ListEventTags lists event tags using common PostgreSQL operations
func (r *PostgresEventTagRepository) ListEventTags(ctx context.Context, req *eventtagpb.ListEventTagsRequest) (*eventtagpb.ListEventTagsResponse, error) {
	if err := espynahttp.ValidateSortColumns(eventTagSortSpec, req.GetSort(), "event_tag"); err != nil {
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
		return nil, fmt.Errorf("failed to list event tags: %w", err)
	}

	var eventTags []*eventtagpb.EventTag
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		eventTag := &eventtagpb.EventTag{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventTag); err != nil {
			continue
		}
		eventTags = append(eventTags, eventTag)
	}

	return &eventtagpb.ListEventTagsResponse{
		Data: eventTags,
	}, nil
}

// GetEventTagListPageData retrieves paginated event tag list data with CTE.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *PostgresEventTagRepository) GetEventTagListPageData(
	ctx context.Context,
	req *eventtagpb.GetEventTagListPageDataRequest,
) (*eventtagpb.GetEventTagListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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
	orderByClause, err := postgresCore.BuildOrderBy(eventTagSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				et.id,
				et.workspace_id,
				et.name,
				et.description,
				et.color,
				et.active,
				et.date_created,
				et.date_modified
			FROM event_tag et
			WHERE et.active = true
			  AND et.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   et.name ILIKE $2 OR
				   et.description ILIKE $2)
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
		return nil, fmt.Errorf("failed to query event tag list page data: %w", err)
	}
	defer rows.Close()

	var eventTags []*eventtagpb.EventTag
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			workspaceId  string
			name         string
			description  sql.NullString
			color        sql.NullString
			active       bool
			dateCreated  time.Time
			dateModified time.Time
			total        int64
		)

		err := rows.Scan(
			&id,
			&workspaceId,
			&name,
			&description,
			&color,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event tag row: %w", err)
		}

		totalCount = total

		eventTag := &eventtagpb.EventTag{
			Id:          id,
			WorkspaceId: workspaceId,
			Name:        name,
			Active:      active,
		}

		if description.Valid {
			eventTag.Description = description.String
		}
		if color.Valid {
			eventTag.Color = color.String
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			eventTag.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			eventTag.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			eventTag.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			eventTag.DateModifiedString = &dmStr
		}

		eventTags = append(eventTags, eventTag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event tag rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventtagpb.GetEventTagListPageDataResponse{
		EventTagList: eventTags,
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

// GetEventTagItemPageData retrieves a single event tag with enhanced item page data.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *PostgresEventTagRepository) GetEventTagItemPageData(
	ctx context.Context,
	req *eventtagpb.GetEventTagItemPageDataRequest,
) (*eventtagpb.GetEventTagItemPageDataResponse, error) {
	if req == nil || req.EventTagId == "" {
		return nil, fmt.Errorf("event tag ID required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	query := `
		SELECT
			et.id,
			et.workspace_id,
			et.name,
			et.description,
			et.color,
			et.active,
			et.date_created,
			et.date_modified
		FROM event_tag et
		WHERE et.id = $1 AND et.workspace_id = $2 AND et.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.EventTagId, workspaceID)

	var (
		id           string
		workspaceId  string
		name         string
		description  sql.NullString
		color        sql.NullString
		active       bool
		dateCreated  time.Time
		dateModified time.Time
	)

	err := row.Scan(
		&id,
		&workspaceId,
		&name,
		&description,
		&color,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event tag with ID '%s' not found", req.EventTagId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event tag item page data: %w", err)
	}

	eventTag := &eventtagpb.EventTag{
		Id:          id,
		WorkspaceId: workspaceId,
		Name:        name,
		Active:      active,
	}

	if description.Valid {
		eventTag.Description = description.String
	}
	if color.Valid {
		eventTag.Color = color.String
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		eventTag.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		eventTag.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		eventTag.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		eventTag.DateModifiedString = &dmStr
	}

	return &eventtagpb.GetEventTagItemPageDataResponse{
		EventTag: eventTag,
		Success:  true,
	}, nil
}

// NewEventTagRepository creates a new PostgreSQL event_tag repository (old-style constructor)
func NewEventTagRepository(db *sql.DB, tableName string) eventtagpb.EventTagDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventTagRepository(dbOps, tableName)
}
