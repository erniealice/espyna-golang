//go:build mysql

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/operations"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	"google.golang.org/protobuf/encoding/protojson"
)

// eventSortableSQLCols lists SQL column names safe to sort by in GetEventListPageData.
var eventSortableSQLCols = []string{
	"e.date_created",
	"e.date_modified",
	"e.name",
	"e.start_date_time_utc",
	"e.end_date_time_utc",
}

// eventViewToSQLColMap translates view-facing sort column keys to SQL column names.
var eventViewToSQLColMap = map[string]string{
	"date_created":        "e.date_created",
	"date_modified":       "e.date_modified",
	"name":                "e.name",
	"start_date_time_utc": "e.start_date_time_utc",
	"end_date_time_utc":   "e.end_date_time_utc",
}

// MySQLEventRepository implements event CRUD operations using MySQL 8.0+.
type MySQLEventRepository struct {
	eventpb.UnimplementedEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Event, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql event repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLEventRepository(dbOps, tableName), nil
	})
}

// NewMySQLEventRepository creates a new MySQL event repository.
func NewMySQLEventRepository(dbOps interfaces.DatabaseOperation, tableName string) eventpb.EventDomainServiceServer {
	if tableName == "" {
		tableName = "event"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEvent creates a new event.
func (r *MySQLEventRepository) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event data is required")
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
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "start_date_time_utc", "end_date_time_utc")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.CreateEventResponse{
		Data: []*eventpb.Event{event},
	}, nil
}

// ReadEvent retrieves an event.
func (r *MySQLEventRepository) ReadEvent(ctx context.Context, req *eventpb.ReadEventRequest) (*eventpb.ReadEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "start_date_time_utc", "end_date_time_utc")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.ReadEventResponse{
		Data: []*eventpb.Event{event},
	}, nil
}

// UpdateEvent updates an event.
func (r *MySQLEventRepository) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.UpdateEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
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
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "start_date_time_utc", "end_date_time_utc")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.UpdateEventResponse{
		Data: []*eventpb.Event{event},
	}, nil
}

// DeleteEvent deletes an event (soft delete).
func (r *MySQLEventRepository) DeleteEvent(ctx context.Context, req *eventpb.DeleteEventRequest) (*eventpb.DeleteEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event: %w", err)
	}

	return &eventpb.DeleteEventResponse{
		Success: true,
	}, nil
}

// ListEvents lists events.
func (r *MySQLEventRepository) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []*eventpb.Event
	for _, result := range listResult.Data {
		mysqlCore.ConvertMillisToDateStr(result, "start_date_time_utc", "end_date_time_utc")
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		event := &eventpb.Event{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
			continue
		}
		events = append(events, event)
	}

	return &eventpb.ListEventsResponse{
		Data: events,
	}, nil
}

// GetEventListPageData retrieves events with advanced filtering, sorting, searching,
// and pagination.
//
// Dialect translation from postgres gold standard:
//   - $1::text IS NULL OR ... → (? = ” OR ...)
//   - $2, $3 (LIMIT/OFFSET) → ?, ? (same positional order)
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1
//   - COUNT(*) stays (CTE counted pattern)
//   - Sort column whitelist guard remains; ORDER BY is interpolated after whitelist check
//
// CRITICAL: workspace_id isolation enforced by WorkspaceAwareOperations.
func (r *MySQLEventRepository) GetEventListPageData(
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

	// Translate view-facing sort key to SQL column.
	if mapped, ok := eventViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	// Whitelist guard: reject unsorted column not in allowlist.
	allowed := false
	for _, c := range eventSortableSQLCols {
		if c == sortField {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortField, "event", eventSortableSQLCols)
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	// Dialect: ILIKE → LIKE, $N → ?, active = true → active = 1
	query := fmt.Sprintf(`
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
			WHERE e.active = 1
			  AND (? = '' OR e.workspace_id = ?)
			  AND (? = '' OR
				   e.name LIKE ? OR
				   e.description LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT ? OFFSET ?;
	`, sortField, sortOrder)

	rows, err := r.db.QueryContext(ctx, query,
		workspaceID, workspaceID,
		searchPattern, searchPattern, searchPattern,
		limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query event list page data: %w", err)
	}
	defer rows.Close()

	var events []*eventpb.Event
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			name             string
			description      *string
			startDateTimeUTC *string
			endDateTimeUTC   *string
			active           bool
			dateCreated      time.Time
			dateModified     time.Time
			total            int64
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

// GetEventItemPageData retrieves a single event with enhanced item page data.
//
// Dialect: $1 → ?, active = true → active = 1, workspace_id predicate.
// CRITICAL: workspace_id isolation enforced by WorkspaceAwareOperations on CRUD;
// item-page query adds explicit workspace_id predicate for direct SQL path.
func (r *MySQLEventRepository) GetEventItemPageData(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) (*eventpb.GetEventItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event item page data request is required")
	}
	if req.EventId == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	// Dialect: $1 → ?, $2 → ?, active = true → active = 1
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
		WHERE id = ? AND active = 1 AND workspace_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, req.EventId, workspaceID)

	var (
		id               string
		name             string
		description      *string
		startDateTimeUTC *string
		endDateTimeUTC   *string
		active           bool
		dateCreated      time.Time
		dateModified     time.Time
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

// NewEventRepository creates a new MySQL event repository (old-style constructor).
func NewEventRepository(db *sql.DB, tableName string) eventpb.EventDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLEventRepository(dbOps, tableName)
}
