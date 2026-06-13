//go:build sqlserver

package event

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
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

// SQLServerEventOccurrenceRepository implements read-only event occurrence operations using SQL Server.
// This entity is populated by the background recurrence expansion job (cyta), not by user CRUD.
type SQLServerEventOccurrenceRepository struct {
	eventoccurrencepb.UnimplementedEventOccurrenceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EventOccurrence, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event_occurrence repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventOccurrenceRepository(dbOps, tableName), nil
	})
}

// NewSQLServerEventOccurrenceRepository creates a new SQL Server event occurrence repository.
func NewSQLServerEventOccurrenceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventoccurrencepb.EventOccurrenceDomainServiceServer {
	if tableName == "" {
		tableName = "event_occurrence"
	}
	return &SQLServerEventOccurrenceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// ListEventOccurrences lists event occurrences using common SQL Server operations.
func (r *SQLServerEventOccurrenceRepository) ListEventOccurrences(ctx context.Context, req *eventoccurrencepb.ListEventOccurrencesRequest) (*eventoccurrencepb.ListEventOccurrencesResponse, error) {
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
		eo, err := mapRowToEventOccurrenceSS(result)
		if err != nil {
			continue
		}
		eventOccurrences = append(eventOccurrences, eo)
	}

	return &eventoccurrencepb.ListEventOccurrencesResponse{
		Data:    eventOccurrences,
		Success: true,
	}, nil
}

// GetEventOccurrenceListPageData retrieves paginated event occurrence list data.
//
// SQL Server differences vs postgres:
//   - $N → @pN
//   - active = true → active = 1
//   - ILIKE → LIKE
//   - LIMIT/OFFSET → OFFSET/FETCH NEXT
//   - Uses WorkspaceAwareOperations executor for CTE
func (r *SQLServerEventOccurrenceRepository) GetEventOccurrenceListPageData(
	ctx context.Context,
	req *eventoccurrencepb.GetEventOccurrenceListPageDataRequest,
) (*eventoccurrencepb.GetEventOccurrenceListPageDataResponse, error) {
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

	sortColKey := "eo.start_date_time_utc"
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
	}
	if mapped, ok := eventOccurrenceViewToSQLColMap[sortColKey]; ok {
		sortColKey = mapped
	}

	sortDir := commonpb.SortDirection_ASC
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortDir = req.Sort.Fields[0].Direction
	}

	sortFragment, err := sqlserverCore.BuildOrderBy(
		eventOccurrenceSortableSQLCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: sortColKey, Direction: sortDir}}},
		"eo.start_date_time_utc ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for event_occurrence: %w", err)
	}

	// p1=workspace_id, p2=searchPattern, p3=limit, p4=offset
	query := fmt.Sprintf(`
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
			WHERE eo.active = 1
			  AND eo.workspace_id = @p1
			  AND (@p2 = '' OR eo.event_id LIKE @p2)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY;
	`, sortFragment)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, searchPattern, offset, limit)
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

		if err := rows.Scan(
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
		); err != nil {
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

// GetEventOccurrenceItemPageData retrieves a single event occurrence.
//
// SQL Server differences vs postgres: @pN placeholders; active = 1; TOP 1 instead of LIMIT 1.
func (r *SQLServerEventOccurrenceRepository) GetEventOccurrenceItemPageData(
	ctx context.Context,
	req *eventoccurrencepb.GetEventOccurrenceItemPageDataRequest,
) (*eventoccurrencepb.GetEventOccurrenceItemPageDataResponse, error) {
	if req == nil || req.EventOccurrenceId == "" {
		return nil, fmt.Errorf("event occurrence ID required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		SELECT TOP 1
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
		WHERE eo.id = @p1 AND eo.workspace_id = @p2 AND eo.active = 1
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.EventOccurrenceId, workspaceID)

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

// mapRowToEventOccurrenceSS converts a generic map row to an EventOccurrence protobuf.
// Used by ListEventOccurrences which goes through the generic dbOps.List path.
func mapRowToEventOccurrenceSS(row map[string]any) (*eventoccurrencepb.EventOccurrence, error) {
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
