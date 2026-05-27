//go:build sqlserver

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/operations"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	"google.golang.org/protobuf/encoding/protojson"
)

// eventSortableSQLCols lists the SQL column names that are safe to sort by in
// GetEventListPageData. Routed through core.BuildOrderBy (A2 guard).
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

// SQLServerEventRepository implements event CRUD operations using SQL Server.
type SQLServerEventRepository struct {
	eventpb.UnimplementedEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Event, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventRepository(dbOps, tableName), nil
	})
}

// NewSQLServerEventRepository creates a new SQL Server event repository.
func NewSQLServerEventRepository(dbOps interfaces.DatabaseOperation, tableName string) eventpb.EventDomainServiceServer {
	if tableName == "" {
		tableName = "event"
	}
	return &SQLServerEventRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// getExec extracts a DBExecutor from the dbOps wrapper.
func (r *SQLServerEventRepository) getExec(ctx context.Context) dbExecutor {
	return r.dbOps.(executorProvider).GetExecutor(ctx)
}

// CreateEvent creates a new event.
func (r *SQLServerEventRepository) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
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

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.CreateEventResponse{Data: []*eventpb.Event{event}}, nil
}

// ReadEvent retrieves an event.
func (r *SQLServerEventRepository) ReadEvent(ctx context.Context, req *eventpb.ReadEventRequest) (*eventpb.ReadEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.ReadEventResponse{Data: []*eventpb.Event{event}}, nil
}

// UpdateEvent updates an event.
func (r *SQLServerEventRepository) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.UpdateEventResponse, error) {
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

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	event := &eventpb.Event{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventpb.UpdateEventResponse{Data: []*eventpb.Event{event}}, nil
}

// DeleteEvent deletes an event (soft delete).
func (r *SQLServerEventRepository) DeleteEvent(ctx context.Context, req *eventpb.DeleteEventRequest) (*eventpb.DeleteEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete event: %w", err)
	}

	return &eventpb.DeleteEventResponse{Success: true}, nil
}

// ListEvents lists events.
func (r *SQLServerEventRepository) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		event := &eventpb.Event{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, event); err != nil {
			continue
		}
		events = append(events, event)
	}

	return &eventpb.ListEventsResponse{Data: events}, nil
}

// GetEventListPageData retrieves events with advanced filtering, sorting, searching, and pagination.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1.
//   - ILIKE → LIKE (default CI collation).
//   - Pagination: ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER() is retained — SQL Server 2017+ supports it.
//   - No FILTER (WHERE) needed — no conditional aggregates in this query.
func (r *SQLServerEventRepository) GetEventListPageData(
	ctx context.Context,
	req *eventpb.GetEventListPageDataRequest,
) (*eventpb.GetEventListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event list page data request is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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

	// Translate view-facing sort key to SQL column name.
	sortColKey := "e.start_date_time_utc"
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
	}
	if mapped, ok := eventViewToSQLColMap[sortColKey]; ok {
		sortColKey = mapped
	}

	// A2 sort guard via BuildOrderBy.
	sortFragment, err := sqlserverCore.BuildOrderBy(
		eventSortableSQLCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{
			Field: sortColKey,
			Direction: func() commonpb.SortDirection {
				if req.Sort != nil && len(req.Sort.Fields) > 0 {
					return req.Sort.Fields[0].Direction
				}
				return commonpb.SortDirection_ASC
			}(),
		}}},
		"e.start_date_time_utc ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for event: %w", err)
	}

	// @p1 = workspaceID. Filter/search start at @p2.
	searchFields := []string{"e.name", "e.description"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	whereSQL := "WHERE e.active = 1 AND e.workspace_id = @p1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := append([]any{workspaceID}, filterArgs...)
	queryArgs = append(queryArgs, offset, limit)

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
				e.date_modified,
				COUNT(*) OVER() AS total_count
			FROM event e
			%s
		)
		SELECT * FROM enriched
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, sortFragment, offsetIdx, limitIdx)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
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

		if err := rows.Scan(
			&id, &name, &description, &startDateTimeUTC, &endDateTimeUTC,
			&active, &dateCreated, &dateModified, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		totalCount = total

		event := &eventpb.Event{Id: id, Name: name, Active: active}

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
// SQL Server differences: @p1, active = 1.
func (r *SQLServerEventRepository) GetEventItemPageData(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) (*eventpb.GetEventItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event item page data request is required")
	}
	if req.EventId == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	const query = `
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
		WHERE id = @p1 AND active = 1
	`

	exec := r.getExec(ctx)
	row := exec.QueryRowContext(ctx, query, req.EventId)

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
		&id, &name, &description, &startDateTimeUTC, &endDateTimeUTC,
		&active, &dateCreated, &dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event with ID '%s' not found", req.EventId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event item page data: %w", err)
	}

	event := &eventpb.Event{Id: id, Name: name, Active: active}

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

// NewEventRepository creates a new SQL Server event repository (old-style constructor).
func NewEventRepository(db *sql.DB, tableName string) eventpb.EventDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerEventRepository(dbOps, tableName)
}
