//go:build postgresql

package event

import (
	"time"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// PostgresEventRepository implements event CRUD operations using PostgreSQL
type PostgresEventRepository struct {
	eventpb.UnimplementedEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "event", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresEventRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventRepository creates a new PostgreSQL event repository
func NewPostgresEventRepository(dbOps interfaces.DatabaseOperation, tableName string) eventpb.EventDomainServiceServer {
	if tableName == "" {
		tableName = "event" // default fallback
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEvent creates a new event using common PostgreSQL operations
func (r *PostgresEventRepository) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event data is required")
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
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := protojson.Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.CreateEventResponse{
		Data: []*eventpb.Event{event},
	}, nil
}

// ReadEvent retrieves an event using common PostgreSQL operations
func (r *PostgresEventRepository) ReadEvent(ctx context.Context, req *eventpb.ReadEventRequest) (*eventpb.ReadEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := protojson.Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.ReadEventResponse{
		Data: []*eventpb.Event{event},
	}, nil
}

// UpdateEvent updates an event using common PostgreSQL operations
func (r *PostgresEventRepository) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.UpdateEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
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
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := protojson.Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.UpdateEventResponse{
		Data: []*eventpb.Event{event},
	}, nil
}

// DeleteEvent deletes an event using common PostgreSQL operations
func (r *PostgresEventRepository) DeleteEvent(ctx context.Context, req *eventpb.DeleteEventRequest) (*eventpb.DeleteEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event: %w", err)
	}

	return &eventpb.DeleteEventResponse{
		Success: true,
	}, nil
}

// ListEvents lists events using common PostgreSQL operations
func (r *PostgresEventRepository) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var events []*eventpb.Event
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		event := &eventpb.Event{}
		if err := protojson.Unmarshal(resultJSON, event); err != nil {
			// Log error and continue with next item
			continue
		}
		events = append(events, event)
	}

	return &eventpb.ListEventsResponse{
		Data: events,
	}, nil
}

// GetEventListPageData retrieves events with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresEventRepository) GetEventListPageData(
	ctx context.Context,
	req *eventpb.GetEventListPageDataRequest,
) (*eventpb.GetEventListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event list page data request is required")
	}

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

	sortField := "start_date_time_utc"
	sortOrder := "ASC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				e.id,
				e.name,
				e.description,
				e.start_date_time_utc,
				e.end_date_time_utc,
				e.active,
				e.date_created,
				e.date_modified
			FROM event e
			WHERE e.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   e.name ILIKE $1 OR
				   e.description ILIKE $1)
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
		return nil, fmt.Errorf("failed to query event list page data: %w", err)
	}
	defer rows.Close()

	var events []*eventpb.Event
	var totalCount int64

	for rows.Next() {
		var (
			id                  string
			name                string
			description         *string
			startDateTimeUTC    *string
			endDateTimeUTC      *string
			active              bool
			dateCreated         time.Time
			dateModified        time.Time
			total               int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&startDateTimeUTC,
			&endDateTimeUTC,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		totalCount = total

		event := &eventpb.Event{
			Id:     id,
			Name:   name,
			Active: active,
		}

		if description != nil {
			event.Description = description
		}

		if startDateTimeUTC != nil && *startDateTimeUTC != "" {
			if ts, err := operations.ParseTimestamp(*startDateTimeUTC); err == nil {
				event.StartDateTimeUtc = ts
			}
		}

		if endDateTimeUTC != nil && *endDateTimeUTC != "" {
			if ts, err := operations.ParseTimestamp(*endDateTimeUTC); err == nil {
				event.EndDateTimeUtc = ts
			}
		}

		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		event.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		event.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		event.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		event.DateModifiedString = &dmStr
	}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventpb.GetEventListPageDataResponse{
		EventList: events,
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

// GetEventItemPageData retrieves a single event with enhanced item page data using CTE
func (r *PostgresEventRepository) GetEventItemPageData(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) (*eventpb.GetEventItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event item page data request is required")
	}
	if req.EventId == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	query := `
		SELECT
			id,
			name,
			description,
			start_date_time_utc,
			end_date_time_utc,
			active,
			date_created,
			date_modified
		FROM event
		WHERE id = $1 AND active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.EventId)

	var (
		id                 string
		name               string
		description        *string
		startDateTimeUTC   *string
		endDateTimeUTC     *string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&startDateTimeUTC,
		&endDateTimeUTC,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event with ID '%s' not found", req.EventId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event item page data: %w", err)
	}

	event := &eventpb.Event{
		Id:     id,
		Name:   name,
		Active: active,
	}

	if description != nil {
		event.Description = description
	}

			if startDateTimeUTC != nil && *startDateTimeUTC != "" {
				if ts, err := operations.ParseTimestamp(*startDateTimeUTC); err == nil {
					event.StartDateTimeUtc = ts
				}
			}
	
			if endDateTimeUTC != nil && *endDateTimeUTC != "" {
				if ts, err := operations.ParseTimestamp(*endDateTimeUTC); err == nil {
					event.EndDateTimeUtc = ts
				}
			}
	
			if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		event.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		event.DateCreatedString = &dcStr
	}
			if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		event.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		event.DateModifiedString = &dmStr
	}
	return &eventpb.GetEventItemPageDataResponse{
		Event:   event,
		Success: true,
	}, nil
}


// NewEventRepository creates a new PostgreSQL event repository (old-style constructor)
func NewEventRepository(db *sql.DB, tableName string) eventpb.EventDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresEventRepository(dbOps, tableName)
}
