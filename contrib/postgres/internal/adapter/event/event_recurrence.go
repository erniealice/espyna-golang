//go:build postgresql

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
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresEventRecurrenceRepository implements event recurrence CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_recurrence_active ON event_recurrence(active) WHERE active = true
//   - CREATE INDEX idx_event_recurrence_workspace_id ON event_recurrence(workspace_id)
//   - CREATE INDEX idx_event_recurrence_freq ON event_recurrence(freq)
//   - CREATE INDEX idx_event_recurrence_date_created ON event_recurrence(date_created DESC)
type PostgresEventRecurrenceRepository struct {
	eventrecurrencepb.UnimplementedEventRecurrenceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventRecurrence, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_recurrence repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventRecurrenceRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventRecurrenceRepository creates a new PostgreSQL event recurrence repository
func NewPostgresEventRecurrenceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventrecurrencepb.EventRecurrenceDomainServiceServer {
	if tableName == "" {
		tableName = "event_recurrence" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventRecurrenceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventRecurrence creates a new event recurrence using common PostgreSQL operations
func (r *PostgresEventRecurrenceRepository) CreateEventRecurrence(ctx context.Context, req *eventrecurrencepb.CreateEventRecurrenceRequest) (*eventrecurrencepb.CreateEventRecurrenceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event recurrence data is required")
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
		return nil, fmt.Errorf("failed to create event recurrence: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventRecurrence := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventRecurrence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventrecurrencepb.CreateEventRecurrenceResponse{
		Data: []*eventrecurrencepb.EventRecurrence{eventRecurrence},
	}, nil
}

// ReadEventRecurrence retrieves an event recurrence using common PostgreSQL operations
func (r *PostgresEventRecurrenceRepository) ReadEventRecurrence(ctx context.Context, req *eventrecurrencepb.ReadEventRecurrenceRequest) (*eventrecurrencepb.ReadEventRecurrenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event recurrence ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event recurrence: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventRecurrence := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventRecurrence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventrecurrencepb.ReadEventRecurrenceResponse{
		Data: []*eventrecurrencepb.EventRecurrence{eventRecurrence},
	}, nil
}

// UpdateEventRecurrence updates an event recurrence using common PostgreSQL operations
func (r *PostgresEventRecurrenceRepository) UpdateEventRecurrence(ctx context.Context, req *eventrecurrencepb.UpdateEventRecurrenceRequest) (*eventrecurrencepb.UpdateEventRecurrenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event recurrence ID is required")
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
		return nil, fmt.Errorf("failed to update event recurrence: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventRecurrence := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventRecurrence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventrecurrencepb.UpdateEventRecurrenceResponse{
		Data: []*eventrecurrencepb.EventRecurrence{eventRecurrence},
	}, nil
}

// DeleteEventRecurrence deletes an event recurrence using common PostgreSQL operations
func (r *PostgresEventRecurrenceRepository) DeleteEventRecurrence(ctx context.Context, req *eventrecurrencepb.DeleteEventRecurrenceRequest) (*eventrecurrencepb.DeleteEventRecurrenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event recurrence ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event recurrence: %w", err)
	}

	return &eventrecurrencepb.DeleteEventRecurrenceResponse{
		Success: true,
	}, nil
}

// ListEventRecurrences lists event recurrences using common PostgreSQL operations
func (r *PostgresEventRecurrenceRepository) ListEventRecurrences(ctx context.Context, req *eventrecurrencepb.ListEventRecurrencesRequest) (*eventrecurrencepb.ListEventRecurrencesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event recurrences: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var eventRecurrences []*eventrecurrencepb.EventRecurrence
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		eventRecurrence := &eventrecurrencepb.EventRecurrence{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventRecurrence); err != nil {
			// Log error and continue with next item
			continue
		}
		eventRecurrences = append(eventRecurrences, eventRecurrence)
	}

	return &eventrecurrencepb.ListEventRecurrencesResponse{
		Data: eventRecurrences,
	}, nil
}

// GetEventRecurrenceListPageData retrieves paginated event recurrence list data with CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventRecurrenceRepository) GetEventRecurrenceListPageData(
	ctx context.Context,
	req *eventrecurrencepb.GetEventRecurrenceListPageDataRequest,
) (*eventrecurrencepb.GetEventRecurrenceListPageDataResponse, error) {
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

	// CTE Query - standalone entity with search on name, description, rrule_string
	query := `
		WITH enriched AS (
			SELECT
				er.id,
				er.name,
				er.description,
				er.rrule_string,
				er.workspace_id,
				er.freq,
				er.interval,
				er.count,
				er.until_utc,
				er.by_day,
				er.by_month_day,
				er.exdate_string,
				er.active,
				er.date_created,
				er.date_modified
			FROM event_recurrence er
			WHERE er.active = true
			  AND er.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   er.name ILIKE $2 OR
				   er.description ILIKE $2 OR
				   er.rrule_string ILIKE $2)
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
		return nil, fmt.Errorf("failed to query event recurrence list page data: %w", err)
	}
	defer rows.Close()

	var eventRecurrences []*eventrecurrencepb.EventRecurrence
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			name         string
			description  sql.NullString
			rruleString  string
			workspaceId  string
			freq         int32
			interval     int32
			count        sql.NullInt32
			untilUtc     sql.NullInt64
			byDay        sql.NullString
			byMonthDay   sql.NullString
			exdateString sql.NullString
			active       bool
			dateCreated  time.Time
			dateModified time.Time
			total        int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&rruleString,
			&workspaceId,
			&freq,
			&interval,
			&count,
			&untilUtc,
			&byDay,
			&byMonthDay,
			&exdateString,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event recurrence row: %w", err)
		}

		totalCount = total

		eventRecurrence := &eventrecurrencepb.EventRecurrence{
			Id:          id,
			Name:        name,
			RruleString: rruleString,
			WorkspaceId: workspaceId,
			Freq:        eventrecurrencepb.RecurrenceFrequency(freq),
			Interval:    interval,
			Active:      active,
		}

		if description.Valid {
			eventRecurrence.Description = &description.String
		}
		if count.Valid {
			eventRecurrence.Count = &count.Int32
		}
		if untilUtc.Valid {
			eventRecurrence.UntilUtc = &untilUtc.Int64
		}
		if byDay.Valid {
			eventRecurrence.ByDay = &byDay.String
		}
		if byMonthDay.Valid {
			eventRecurrence.ByMonthDay = &byMonthDay.String
		}
		if exdateString.Valid {
			eventRecurrence.ExdateString = &exdateString.String
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			eventRecurrence.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			eventRecurrence.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			eventRecurrence.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			eventRecurrence.DateModifiedString = &dmStr
		}

		eventRecurrences = append(eventRecurrences, eventRecurrence)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event recurrence rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventrecurrencepb.GetEventRecurrenceListPageDataResponse{
		EventRecurrenceList: eventRecurrences,
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

// GetEventRecurrenceItemPageData retrieves a single event recurrence with enhanced item page data
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventRecurrenceRepository) GetEventRecurrenceItemPageData(
	ctx context.Context,
	req *eventrecurrencepb.GetEventRecurrenceItemPageDataRequest,
) (*eventrecurrencepb.GetEventRecurrenceItemPageDataResponse, error) {
	if req == nil || req.EventRecurrenceId == "" {
		return nil, fmt.Errorf("event recurrence ID required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Simple query for single event recurrence item
	query := `
		SELECT
			er.id,
			er.name,
			er.description,
			er.rrule_string,
			er.workspace_id,
			er.freq,
			er.interval,
			er.count,
			er.until_utc,
			er.by_day,
			er.by_month_day,
			er.exdate_string,
			er.active,
			er.date_created,
			er.date_modified
		FROM event_recurrence er
		WHERE er.id = $1 AND er.workspace_id = $2 AND er.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.EventRecurrenceId, workspaceID)

	var (
		id           string
		name         string
		description  sql.NullString
		rruleString  string
		workspaceId  string
		freq         int32
		interval     int32
		count        sql.NullInt32
		untilUtc     sql.NullInt64
		byDay        sql.NullString
		byMonthDay   sql.NullString
		exdateString sql.NullString
		active       bool
		dateCreated  time.Time
		dateModified time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&rruleString,
		&workspaceId,
		&freq,
		&interval,
		&count,
		&untilUtc,
		&byDay,
		&byMonthDay,
		&exdateString,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event recurrence with ID '%s' not found", req.EventRecurrenceId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event recurrence item page data: %w", err)
	}

	eventRecurrence := &eventrecurrencepb.EventRecurrence{
		Id:          id,
		Name:        name,
		RruleString: rruleString,
		WorkspaceId: workspaceId,
		Freq:        eventrecurrencepb.RecurrenceFrequency(freq),
		Interval:    interval,
		Active:      active,
	}

	if description.Valid {
		eventRecurrence.Description = &description.String
	}
	if count.Valid {
		eventRecurrence.Count = &count.Int32
	}
	if untilUtc.Valid {
		eventRecurrence.UntilUtc = &untilUtc.Int64
	}
	if byDay.Valid {
		eventRecurrence.ByDay = &byDay.String
	}
	if byMonthDay.Valid {
		eventRecurrence.ByMonthDay = &byMonthDay.String
	}
	if exdateString.Valid {
		eventRecurrence.ExdateString = &exdateString.String
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		eventRecurrence.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		eventRecurrence.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		eventRecurrence.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		eventRecurrence.DateModifiedString = &dmStr
	}

	return &eventrecurrencepb.GetEventRecurrenceItemPageDataResponse{
		EventRecurrence: eventRecurrence,
		Success:         true,
	}, nil
}

// NewEventRecurrenceRepository creates a new PostgreSQL event_recurrence repository (old-style constructor)
func NewEventRecurrenceRepository(db *sql.DB, tableName string) eventrecurrencepb.EventRecurrenceDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventRecurrenceRepository(dbOps, tableName)
}