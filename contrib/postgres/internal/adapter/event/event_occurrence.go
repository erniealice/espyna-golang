//go:build postgresql

package event

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventoccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
)

// eventOccurrenceSortableSQLCols lists the SQL column names that are safe to
// sort by in GetEventOccurrenceListPageData. The query uses direct ORDER BY
// interpolation so this guard is critical.
var eventOccurrenceSortableSQLCols = []string{
	"eo.date_created",
	"eo.date_modified",
	"eo.start_date_time_utc",
	"eo.end_date_time_utc",
	"eo.event_id",
}

// eventOccurrenceViewToSQLColMap translates view-facing sort column keys to SQL
// column names. Columns absent from the map pass through unchanged.
var eventOccurrenceViewToSQLColMap = map[string]string{
	"date_created":        "eo.date_created",
	"date_modified":       "eo.date_modified",
	"start_date_time_utc": "eo.start_date_time_utc",
	"end_date_time_utc":   "eo.end_date_time_utc",
	"event_id":            "eo.event_id",
}

// PostgresEventOccurrenceRepository implements read-only event occurrence operations using PostgreSQL.
// This entity is populated by the background recurrence expansion job (cyta), not by user CRUD.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_occurrence_workspace_start ON event_occurrence(workspace_id, start_date_time_utc) - Primary calendar range query
//   - CREATE INDEX idx_event_occurrence_workspace_end ON event_occurrence(workspace_id, end_date_time_utc) - Range query end bound
//   - CREATE INDEX idx_event_occurrence_event_id ON event_occurrence(event_id) - Parent event filter
//   - CREATE INDEX idx_event_occurrence_active ON event_occurrence(active) WHERE active = true - Filter active records
type PostgresEventOccurrenceRepository struct {
	eventoccurrencepb.UnimplementedEventOccurrenceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventOccurrence, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_occurrence repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventOccurrenceRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventOccurrenceRepository creates a new PostgreSQL event occurrence repository
func NewPostgresEventOccurrenceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventoccurrencepb.EventOccurrenceDomainServiceServer {
	if tableName == "" {
		tableName = "event_occurrence" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventOccurrenceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// ListEventOccurrences lists event occurrences using common PostgreSQL operations
func (r *PostgresEventOccurrenceRepository) ListEventOccurrences(ctx context.Context, req *eventoccurrencepb.ListEventOccurrencesRequest) (*eventoccurrencepb.ListEventOccurrencesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event occurrences: %w", err)
	}

	var eventOccurrences []*eventoccurrencepb.EventOccurrence
	for _, result := range listResult.Data {
		eo, err := mapRowToEventOccurrence(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}
		eventOccurrences = append(eventOccurrences, eo)
	}

	return &eventoccurrencepb.ListEventOccurrencesResponse{
		Data:    eventOccurrences,
		Success: true,
	}, nil
}

// GetEventOccurrenceListPageData retrieves paginated event occurrence list data with CTE.
// Optimized for calendar range queries: workspace_id + start/end time window.
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventOccurrenceRepository) GetEventOccurrenceListPageData(
	ctx context.Context,
	req *eventoccurrencepb.GetEventOccurrenceListPageDataRequest,
) (*eventoccurrencepb.GetEventOccurrenceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Build search condition — search on event_id
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
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort — start_date_time_utc ASC is natural for calendar rendering.
	// Translate view-facing column key to SQL column name via ColMap.
	sortColKey := "eo.start_date_time_utc"
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
	}
	if mapped, ok := eventOccurrenceViewToSQLColMap[sortColKey]; ok {
		sortColKey = mapped
	}

	// A2 sort guard: reject any column not in the whitelist via core.BuildOrderBy.
	sortFragment, err := postgresCore.BuildOrderBy(
		eventOccurrenceSortableSQLCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: sortColKey, Direction: func() commonpb.SortDirection {
			if req.Sort != nil && len(req.Sort.Fields) > 0 {
				return req.Sort.Fields[0].Direction
			}
			return commonpb.SortDirection_ASC
		}()}}},
		"eo.start_date_time_utc ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for event_occurrence: %w", err)
	}

	// CTE Query — calendar range pattern with event_id search
	query := `
		WITH enriched AS (
			SELECT
				eo.id,
				eo.event_id,
				eo.start_date_time_utc,
				eo.end_date_time_utc,
				eo.is_exception,
				eo.is_cancelled,
				eo.exception_event_id,
				eo.workspace_id,
				eo.active,
				eo.date_created,
				eo.date_modified
			FROM event_occurrence eo
			WHERE eo.active = true
			  AND eo.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   eo.event_id ILIKE $2)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + sortFragment + `
		LIMIT $3 OFFSET $4;
	`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query event occurrence list page data: %w", err)
	}
	defer rows.Close()

	var eventOccurrences []*eventoccurrencepb.EventOccurrence
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			eventId          string
			startDateTimeUtc int64
			endDateTimeUtc   int64
			isException      bool
			isCancelled      bool
			exceptionEventId sql.NullString
			workspaceId      string
			active           bool
			dateCreated      time.Time
			dateModified     time.Time
			total            int64
		)

		err := rows.Scan(
			&id,
			&eventId,
			&startDateTimeUtc,
			&endDateTimeUtc,
			&isException,
			&isCancelled,
			&exceptionEventId,
			&workspaceId,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event occurrence row: %w", err)
		}

		totalCount = total

		eo := &eventoccurrencepb.EventOccurrence{
			Id:               id,
			EventId:          eventId,
			StartDateTimeUtc: startDateTimeUtc,
			EndDateTimeUtc:   endDateTimeUtc,
			IsException:      isException,
			IsCancelled:      isCancelled,
			WorkspaceId:      workspaceId,
			Active:           active,
		}

		if exceptionEventId.Valid {
			eo.ExceptionEventId = &exceptionEventId.String
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			eo.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			eo.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			eo.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			eo.DateModifiedString = &dmStr
		}

		eventOccurrences = append(eventOccurrences, eo)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event occurrence rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventoccurrencepb.GetEventOccurrenceListPageDataResponse{
		EventOccurrenceList: eventOccurrences,
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

// GetEventOccurrenceItemPageData retrieves a single event occurrence with enhanced item page data
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresEventOccurrenceRepository) GetEventOccurrenceItemPageData(
	ctx context.Context,
	req *eventoccurrencepb.GetEventOccurrenceItemPageDataRequest,
) (*eventoccurrencepb.GetEventOccurrenceItemPageDataResponse, error) {
	if req == nil || req.EventOccurrenceId == "" {
		return nil, fmt.Errorf("event occurrence ID required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Simple query for single event occurrence item
	query := `
		SELECT
			eo.id,
			eo.event_id,
			eo.start_date_time_utc,
			eo.end_date_time_utc,
			eo.is_exception,
			eo.is_cancelled,
			eo.exception_event_id,
			eo.workspace_id,
			eo.active,
			eo.date_created,
			eo.date_modified
		FROM event_occurrence eo
		WHERE eo.id = $1 AND eo.workspace_id = $2 AND eo.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.EventOccurrenceId, workspaceID)

	var (
		id               string
		eventId          string
		startDateTimeUtc int64
		endDateTimeUtc   int64
		isException      bool
		isCancelled      bool
		exceptionEventId sql.NullString
		workspaceId      string
		active           bool
		dateCreated      time.Time
		dateModified     time.Time
	)

	err := row.Scan(
		&id,
		&eventId,
		&startDateTimeUtc,
		&endDateTimeUtc,
		&isException,
		&isCancelled,
		&exceptionEventId,
		&workspaceId,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event occurrence with ID '%s' not found", req.EventOccurrenceId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event occurrence item page data: %w", err)
	}

	eo := &eventoccurrencepb.EventOccurrence{
		Id:               id,
		EventId:          eventId,
		StartDateTimeUtc: startDateTimeUtc,
		EndDateTimeUtc:   endDateTimeUtc,
		IsException:      isException,
		IsCancelled:      isCancelled,
		WorkspaceId:      workspaceId,
		Active:           active,
	}

	if exceptionEventId.Valid {
		eo.ExceptionEventId = &exceptionEventId.String
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		eo.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		eo.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		eo.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		eo.DateModifiedString = &dmStr
	}

	return &eventoccurrencepb.GetEventOccurrenceItemPageDataResponse{
		EventOccurrence: eo,
		Success:         true,
	}, nil
}

// mapRowToEventOccurrence converts a generic map row to an EventOccurrence protobuf.
// Used by ListEventOccurrences which goes through the generic dbOps.List path.
func mapRowToEventOccurrence(row map[string]any) (*eventoccurrencepb.EventOccurrence, error) {
	eo := &eventoccurrencepb.EventOccurrence{}

	if v, ok := row["id"].(string); ok {
		eo.Id = v
	}
	if v, ok := row["event_id"].(string); ok {
		eo.EventId = v
	}
	if v, ok := row["start_date_time_utc"].(int64); ok {
		eo.StartDateTimeUtc = v
	}
	if v, ok := row["end_date_time_utc"].(int64); ok {
		eo.EndDateTimeUtc = v
	}
	if v, ok := row["is_exception"].(bool); ok {
		eo.IsException = v
	}
	if v, ok := row["is_cancelled"].(bool); ok {
		eo.IsCancelled = v
	}
	if v, ok := row["exception_event_id"].(string); ok && v != "" {
		eo.ExceptionEventId = &v
	}
	if v, ok := row["workspace_id"].(string); ok {
		eo.WorkspaceId = v
	}
	if v, ok := row["active"].(bool); ok {
		eo.Active = v
	}

	// Audit timestamps
	if v, ok := row["date_created"].(time.Time); ok && !v.IsZero() {
		ts := v.UnixMilli()
		eo.DateCreated = &ts
		dcStr := v.Format(time.RFC3339)
		eo.DateCreatedString = &dcStr
	}
	if v, ok := row["date_modified"].(time.Time); ok && !v.IsZero() {
		ts := v.UnixMilli()
		eo.DateModified = &ts
		dmStr := v.Format(time.RFC3339)
		eo.DateModifiedString = &dmStr
	}

	return eo, nil
}

// NewEventOccurrenceRepository creates a new PostgreSQL event_occurrence repository (old-style constructor)
func NewEventOccurrenceRepository(db *sql.DB, tableName string) eventoccurrencepb.EventOccurrenceDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventOccurrenceRepository(dbOps, tableName)
}
