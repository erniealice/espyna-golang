//go:build postgresql

package event

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
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresEventAttendeeRepository implements event attendee CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_attendee_active ON event_attendee(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_event_attendee_event_id ON event_attendee(event_id) - Filter by event
//   - CREATE INDEX idx_event_attendee_client_id ON event_attendee(client_id) - Filter by client
//   - CREATE INDEX idx_event_attendee_workspace_user_id ON event_attendee(workspace_user_id) - Filter by workspace user
//   - CREATE INDEX idx_event_attendee_date_created ON event_attendee(date_created DESC) - Default sorting
type PostgresEventAttendeeRepository struct {
	eventattendeepb.UnimplementedEventAttendeeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventAttendee, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_attendee repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventAttendeeRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventAttendeeRepository creates a new PostgreSQL event attendee repository
func NewPostgresEventAttendeeRepository(dbOps interfaces.DatabaseOperation, tableName string) eventattendeepb.EventAttendeeDomainServiceServer {
	if tableName == "" {
		tableName = "event_attendee" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventAttendeeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventAttendee creates a new event attendee using common PostgreSQL operations
func (r *PostgresEventAttendeeRepository) CreateEventAttendee(ctx context.Context, req *eventattendeepb.CreateEventAttendeeRequest) (*eventattendeepb.CreateEventAttendeeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event attendee data is required")
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
		return nil, fmt.Errorf("failed to create event attendee: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventAttendee := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttendee); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventattendeepb.CreateEventAttendeeResponse{
		Data: []*eventattendeepb.EventAttendee{eventAttendee},
	}, nil
}

// ReadEventAttendee retrieves an event attendee using common PostgreSQL operations
func (r *PostgresEventAttendeeRepository) ReadEventAttendee(ctx context.Context, req *eventattendeepb.ReadEventAttendeeRequest) (*eventattendeepb.ReadEventAttendeeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attendee ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event attendee: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventAttendee := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttendee); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventattendeepb.ReadEventAttendeeResponse{
		Data: []*eventattendeepb.EventAttendee{eventAttendee},
	}, nil
}

// UpdateEventAttendee updates an event attendee using common PostgreSQL operations
func (r *PostgresEventAttendeeRepository) UpdateEventAttendee(ctx context.Context, req *eventattendeepb.UpdateEventAttendeeRequest) (*eventattendeepb.UpdateEventAttendeeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attendee ID is required")
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
		return nil, fmt.Errorf("failed to update event attendee: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventAttendee := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttendee); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventattendeepb.UpdateEventAttendeeResponse{
		Data: []*eventattendeepb.EventAttendee{eventAttendee},
	}, nil
}

// DeleteEventAttendee deletes an event attendee using common PostgreSQL operations
func (r *PostgresEventAttendeeRepository) DeleteEventAttendee(ctx context.Context, req *eventattendeepb.DeleteEventAttendeeRequest) (*eventattendeepb.DeleteEventAttendeeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attendee ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event attendee: %w", err)
	}

	return &eventattendeepb.DeleteEventAttendeeResponse{
		Success: true,
	}, nil
}

// ListEventAttendees lists event attendees using common PostgreSQL operations
func (r *PostgresEventAttendeeRepository) ListEventAttendees(ctx context.Context, req *eventattendeepb.ListEventAttendeesRequest) (*eventattendeepb.ListEventAttendeesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event attendees: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var eventAttendees []*eventattendeepb.EventAttendee
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		eventAttendee := &eventattendeepb.EventAttendee{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttendee); err != nil {
			// Log error and continue with next item
			continue
		}
		eventAttendees = append(eventAttendees, eventAttendee)
	}

	return &eventattendeepb.ListEventAttendeesResponse{
		Data: eventAttendees,
	}, nil
}

var eventAttendeeSortableSQLCols = []string{
	"id", "event_id", "client_id", "workspace_user_id", "role", "status",
	"is_organizer", "display_name", "workspace_id", "active",
	"date_created", "date_modified",
}

// GetEventAttendeeListPageData retrieves paginated event attendee list data with CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventAttendeeRepository) GetEventAttendeeListPageData(
	ctx context.Context,
	req *eventattendeepb.GetEventAttendeeListPageDataRequest,
) (*eventattendeepb.GetEventAttendeeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := identity.Must(ctx).WorkspaceID

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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). Route the
	// caller-supplied sort column through core.BuildOrderBy so an unknown column
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(eventAttendeeSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	// CTE Query — join table pattern with event FK and optional client/workspace_user FK filtering
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				ea.id,
				ea.event_id,
				ea.client_id,
				ea.workspace_user_id,
				ea.role,
				ea.status,
				ea.is_organizer,
				ea.display_name,
				ea.workspace_id,
				ea.active,
				ea.date_created,
				ea.date_modified
			FROM event_attendee ea
			WHERE ea.active = true
			  AND ea.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   ea.display_name ILIKE $2 OR
				   ea.event_id ILIKE $2 OR
				   ea.client_id ILIKE $2)
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
		return nil, fmt.Errorf("failed to query event attendee list page data: %w", err)
	}
	defer rows.Close()

	var eventAttendees []*eventattendeepb.EventAttendee
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			eventId         string
			clientId        sql.NullString
			workspaceUserId sql.NullString
			role            int32
			status          int32
			isOrganizer     bool
			displayName     sql.NullString
			workspaceId     string
			active          bool
			dateCreated     time.Time
			dateModified    time.Time
			total           int64
		)

		err := rows.Scan(
			&id,
			&eventId,
			&clientId,
			&workspaceUserId,
			&role,
			&status,
			&isOrganizer,
			&displayName,
			&workspaceId,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event attendee row: %w", err)
		}

		totalCount = total

		eventAttendee := &eventattendeepb.EventAttendee{
			Id:          id,
			EventId:     eventId,
			WorkspaceId: workspaceId,
			Role:        eventattendeepb.AttendeeRole(role),
			Status:      eventattendeepb.AttendeeStatus(status),
			IsOrganizer: isOrganizer,
			Active:      active,
		}

		if clientId.Valid {
			eventAttendee.ClientId = &clientId.String
		}
		if workspaceUserId.Valid {
			eventAttendee.WorkspaceUserId = &workspaceUserId.String
		}
		if displayName.Valid {
			eventAttendee.DisplayName = &displayName.String
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			eventAttendee.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			eventAttendee.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			eventAttendee.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			eventAttendee.DateModifiedString = &dmStr
		}

		eventAttendees = append(eventAttendees, eventAttendee)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event attendee rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventattendeepb.GetEventAttendeeListPageDataResponse{
		EventAttendeeList: eventAttendees,
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

// GetEventAttendeeItemPageData retrieves a single event attendee with enhanced item page data
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventAttendeeRepository) GetEventAttendeeItemPageData(
	ctx context.Context,
	req *eventattendeepb.GetEventAttendeeItemPageDataRequest,
) (*eventattendeepb.GetEventAttendeeItemPageDataResponse, error) {
	if req == nil || req.EventAttendeeId == "" {
		return nil, fmt.Errorf("event attendee ID required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := identity.Must(ctx).WorkspaceID

	// Simple query for single event attendee item
	query := `
		SELECT
			ea.id,
			ea.event_id,
			ea.client_id,
			ea.workspace_user_id,
			ea.role,
			ea.status,
			ea.is_organizer,
			ea.display_name,
			ea.workspace_id,
			ea.active,
			ea.date_created,
			ea.date_modified
		FROM event_attendee ea
		WHERE ea.id = $1 AND ea.workspace_id = $2 AND ea.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.EventAttendeeId, workspaceID)

	var (
		id              string
		eventId         string
		clientId        sql.NullString
		workspaceUserId sql.NullString
		role            int32
		status          int32
		isOrganizer     bool
		displayName     sql.NullString
		workspaceId     string
		active          bool
		dateCreated     time.Time
		dateModified    time.Time
	)

	err := row.Scan(
		&id,
		&eventId,
		&clientId,
		&workspaceUserId,
		&role,
		&status,
		&isOrganizer,
		&displayName,
		&workspaceId,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event attendee with ID '%s' not found", req.EventAttendeeId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event attendee item page data: %w", err)
	}

	eventAttendee := &eventattendeepb.EventAttendee{
		Id:          id,
		EventId:     eventId,
		WorkspaceId: workspaceId,
		Role:        eventattendeepb.AttendeeRole(role),
		Status:      eventattendeepb.AttendeeStatus(status),
		IsOrganizer: isOrganizer,
		Active:      active,
	}

	if clientId.Valid {
		eventAttendee.ClientId = &clientId.String
	}
	if workspaceUserId.Valid {
		eventAttendee.WorkspaceUserId = &workspaceUserId.String
	}
	if displayName.Valid {
		eventAttendee.DisplayName = &displayName.String
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		eventAttendee.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		eventAttendee.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		eventAttendee.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		eventAttendee.DateModifiedString = &dmStr
	}

	return &eventattendeepb.GetEventAttendeeItemPageDataResponse{
		EventAttendee: eventAttendee,
		Success:       true,
	}, nil
}

// NewEventAttendeeRepository creates a new PostgreSQL event_attendee repository (old-style constructor)
func NewEventAttendeeRepository(db *sql.DB, tableName string) eventattendeepb.EventAttendeeDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventAttendeeRepository(dbOps, tableName)
}
