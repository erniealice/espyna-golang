//go:build postgres

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// PostgresEventClientRepository implements event client CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_client_active ON event_client(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_event_client_event_id ON event_client(event_id) - Filter by event
//   - CREATE INDEX idx_event_client_client_id ON event_client(client_id) - Filter by client
//   - CREATE INDEX idx_event_client_date_created ON event_client(date_created DESC) - Default sorting
type PostgresEventClientRepository struct {
	eventclientpb.UnimplementedEventClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "event_client", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_client repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresEventClientRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventClientRepository creates a new PostgreSQL event client repository
func NewPostgresEventClientRepository(dbOps interfaces.DatabaseOperation, tableName string) eventclientpb.EventClientDomainServiceServer {
	if tableName == "" {
		tableName = "event_client" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventClientRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventClient creates a new event client using common PostgreSQL operations
func (r *PostgresEventClientRepository) CreateEventClient(ctx context.Context, req *eventclientpb.CreateEventClientRequest) (*eventclientpb.CreateEventClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event client data is required")
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
		return nil, fmt.Errorf("failed to create event client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventClient := &eventclientpb.EventClient{}
	if err := protojson.Unmarshal(resultJSON, eventClient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventclientpb.CreateEventClientResponse{
		Data: []*eventclientpb.EventClient{eventClient},
	}, nil
}

// ReadEventClient retrieves an event client using common PostgreSQL operations
func (r *PostgresEventClientRepository) ReadEventClient(ctx context.Context, req *eventclientpb.ReadEventClientRequest) (*eventclientpb.ReadEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event client ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event client: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventClient := &eventclientpb.EventClient{}
	if err := protojson.Unmarshal(resultJSON, eventClient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventclientpb.ReadEventClientResponse{
		Data: []*eventclientpb.EventClient{eventClient},
	}, nil
}

// UpdateEventClient updates an event client using common PostgreSQL operations
func (r *PostgresEventClientRepository) UpdateEventClient(ctx context.Context, req *eventclientpb.UpdateEventClientRequest) (*eventclientpb.UpdateEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event client ID is required")
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
		return nil, fmt.Errorf("failed to update event client: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventClient := &eventclientpb.EventClient{}
	if err := protojson.Unmarshal(resultJSON, eventClient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventclientpb.UpdateEventClientResponse{
		Data: []*eventclientpb.EventClient{eventClient},
	}, nil
}

// DeleteEventClient deletes an event client using common PostgreSQL operations
func (r *PostgresEventClientRepository) DeleteEventClient(ctx context.Context, req *eventclientpb.DeleteEventClientRequest) (*eventclientpb.DeleteEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event client ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event client: %w", err)
	}

	return &eventclientpb.DeleteEventClientResponse{
		Success: true,
	}, nil
}

// ListEventClients lists event clients using common PostgreSQL operations
func (r *PostgresEventClientRepository) ListEventClients(ctx context.Context, req *eventclientpb.ListEventClientsRequest) (*eventclientpb.ListEventClientsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list event clients: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var eventClients []*eventclientpb.EventClient
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		eventClient := &eventclientpb.EventClient{}
		if err := protojson.Unmarshal(resultJSON, eventClient); err != nil {
			// Log error and continue with next item
			continue
		}
		eventClients = append(eventClients, eventClient)
	}

	return &eventclientpb.ListEventClientsResponse{
		Data: eventClients,
	}, nil
}

// GetEventClientListPageData retrieves paginated event client list data with CTE
func (r *PostgresEventClientRepository) GetEventClientListPageData(
	ctx context.Context,
	req *eventclientpb.GetEventClientListPageDataRequest,
) (*eventclientpb.GetEventClientListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
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

	// CTE Query - Junction table pattern with event and client FK filtering
	query := `
		WITH enriched AS (
			SELECT
				ec.id,
				ec.event_id,
				ec.client_id,
				ec.active,
				ec.date_created,
				ec.date_created_string,
				ec.date_modified,
				ec.date_modified_string
			FROM event_client ec
			WHERE ec.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   ec.event_id ILIKE $1 OR
				   ec.client_id ILIKE $1)
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
		return nil, fmt.Errorf("failed to query event client list page data: %w", err)
	}
	defer rows.Close()

	var eventClients []*eventclientpb.EventClient
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			eventId            string
			clientId           string
			active             bool
			dateCreated        *string
			dateCreatedString  *string
			dateModified       *string
			dateModifiedString *string
			total              int64
		)

		err := rows.Scan(
			&id,
			&eventId,
			&clientId,
			&active,
			&dateCreated,
			&dateCreatedString,
			&dateModified,
			&dateModifiedString,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event client row: %w", err)
		}

		totalCount = total

		eventClient := &eventclientpb.EventClient{
			Id:       id,
			EventId:  eventId,
			ClientId: clientId,
			Active:   active,
		}

		if dateCreatedString != nil {
			eventClient.DateCreatedString = dateCreatedString
		}
		if dateModifiedString != nil {
			eventClient.DateModifiedString = dateModifiedString
		}

		// Parse timestamps if provided
		if dateCreated != nil && *dateCreated != "" {
			if ts, err := parseEventClientTimestamp(*dateCreated); err == nil {
				eventClient.DateCreated = &ts
			}
		}
		if dateModified != nil && *dateModified != "" {
			if ts, err := parseEventClientTimestamp(*dateModified); err == nil {
				eventClient.DateModified = &ts
			}
		}

		eventClients = append(eventClients, eventClient)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event client rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventclientpb.GetEventClientListPageDataResponse{
		EventClientList: eventClients,
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

// GetEventClientItemPageData retrieves a single event client with enhanced item page data
func (r *PostgresEventClientRepository) GetEventClientItemPageData(
	ctx context.Context,
	req *eventclientpb.GetEventClientItemPageDataRequest,
) (*eventclientpb.GetEventClientItemPageDataResponse, error) {
	if req == nil || req.EventClientId == "" {
		return nil, fmt.Errorf("event client ID required")
	}

	// Simple query for single event client item
	query := `
		SELECT
			ec.id,
			ec.event_id,
			ec.client_id,
			ec.active,
			ec.date_created,
			ec.date_created_string,
			ec.date_modified,
			ec.date_modified_string
		FROM event_client ec
		WHERE ec.id = $1 AND ec.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.EventClientId)

	var (
		id                 string
		eventId            string
		clientId           string
		active             bool
		dateCreated        *string
		dateCreatedString  *string
		dateModified       *string
		dateModifiedString *string
	)

	err := row.Scan(
		&id,
		&eventId,
		&clientId,
		&active,
		&dateCreated,
		&dateCreatedString,
		&dateModified,
		&dateModifiedString,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event client with ID '%s' not found", req.EventClientId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event client item page data: %w", err)
	}

	eventClient := &eventclientpb.EventClient{
		Id:       id,
		EventId:  eventId,
		ClientId: clientId,
		Active:   active,
	}

	if dateCreatedString != nil {
		eventClient.DateCreatedString = dateCreatedString
	}
	if dateModifiedString != nil {
		eventClient.DateModifiedString = dateModifiedString
	}

	// Parse timestamps if provided
	if dateCreated != nil && *dateCreated != "" {
		if ts, err := parseEventClientTimestamp(*dateCreated); err == nil {
			eventClient.DateCreated = &ts
		}
	}
	if dateModified != nil && *dateModified != "" {
		if ts, err := parseEventClientTimestamp(*dateModified); err == nil {
			eventClient.DateModified = &ts
		}
	}

	return &eventclientpb.GetEventClientItemPageDataResponse{
		EventClient: eventClient,
		Success:     true,
	}, nil
}

// parseEventClientTimestamp converts string timestamp to Unix timestamp (milliseconds)
func parseEventClientTimestamp(timestampStr string) (int64, error) {
	// Try parsing as RFC3339 format first (most common)
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// NewEventClientRepository creates a new PostgreSQL event_client repository (old-style constructor)
func NewEventClientRepository(db *sql.DB, tableName string) eventclientpb.EventClientDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresEventClientRepository(dbOps, tableName)
}
