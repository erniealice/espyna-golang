//go:build postgresql

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresEventAttributeRepository implements event_attribute CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_attribute_active ON event_attribute(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_event_attribute_event_id ON event_attribute(event_id) - FK lookup on event_id
//   - CREATE INDEX idx_event_attribute_attribute_id ON event_attribute(attribute_id) - FK lookup on attribute_id
//   - CREATE INDEX idx_event_attribute_date_created ON event_attribute(date_created DESC) - Default sorting
type PostgresEventAttributeRepository struct {
	eventattributepb.UnimplementedEventAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresEventAttributeRepository creates a new PostgreSQL event attribute repository
func NewPostgresEventAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) eventattributepb.EventAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "event_attribute" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventAttribute creates a new event attribute using common PostgreSQL operations
func (r *PostgresEventAttributeRepository) CreateEventAttribute(ctx context.Context, req *eventattributepb.CreateEventAttributeRequest) (*eventattributepb.CreateEventAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event attribute data is required")
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
		return nil, fmt.Errorf("failed to create event attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventAttribute := &eventattributepb.EventAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventattributepb.CreateEventAttributeResponse{
		Data:    []*eventattributepb.EventAttribute{eventAttribute},
		Success: true,
	}, nil
}

// ReadEventAttribute retrieves an event attribute using common PostgreSQL operations
func (r *PostgresEventAttributeRepository) ReadEventAttribute(ctx context.Context, req *eventattributepb.ReadEventAttributeRequest) (*eventattributepb.ReadEventAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventAttribute := &eventattributepb.EventAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventattributepb.ReadEventAttributeResponse{
		Data:    []*eventattributepb.EventAttribute{eventAttribute},
		Success: true,
	}, nil
}

// UpdateEventAttribute updates an event attribute using common PostgreSQL operations
func (r *PostgresEventAttributeRepository) UpdateEventAttribute(ctx context.Context, req *eventattributepb.UpdateEventAttributeRequest) (*eventattributepb.UpdateEventAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attribute ID is required")
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
		return nil, fmt.Errorf("failed to update event attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	eventAttribute := &eventattributepb.EventAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventattributepb.UpdateEventAttributeResponse{
		Data:    []*eventattributepb.EventAttribute{eventAttribute},
		Success: true,
	}, nil
}

// DeleteEventAttribute deletes an event attribute using common PostgreSQL operations (soft delete)
func (r *PostgresEventAttributeRepository) DeleteEventAttribute(ctx context.Context, req *eventattributepb.DeleteEventAttributeRequest) (*eventattributepb.DeleteEventAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attribute ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event attribute: %w", err)
	}

	return &eventattributepb.DeleteEventAttributeResponse{
		Success: true,
	}, nil
}

// ListEventAttributes lists event attributes using common PostgreSQL operations
func (r *PostgresEventAttributeRepository) ListEventAttributes(ctx context.Context, req *eventattributepb.ListEventAttributesRequest) (*eventattributepb.ListEventAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event attributes: %w", err)
	}

	var eventAttributes []*eventattributepb.EventAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		eventAttribute := &eventattributepb.EventAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, eventAttribute); err != nil {
			continue
		}
		eventAttributes = append(eventAttributes, eventAttribute)
	}

	if eventAttributes == nil {
		eventAttributes = make([]*eventattributepb.EventAttribute, 0)
	}

	return &eventattributepb.ListEventAttributesResponse{
		Data:    eventAttributes,
		Success: true,
	}, nil
}

var eventAttributeSortableSQLCols = []string{
	"id", "event_id", "attribute_id", "value", "date_created", "date_modified",
}

// GetEventAttributeListPageData retrieves paginated event attribute list data with CTE.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *PostgresEventAttributeRepository) GetEventAttributeListPageData(
	ctx context.Context,
	req *eventattributepb.GetEventAttributeListPageDataRequest,
) (*eventattributepb.GetEventAttributeListPageDataResponse, error) {
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
	orderByClause, err := postgresCore.BuildOrderBy(eventAttributeSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				ea.id,
				ea.event_id,
				ea.attribute_id,
				ea.value,
				ea.active,
				ea.date_created,
				ea.date_modified
			FROM event_attribute ea
			WHERE ea.active = true
			  AND ea.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
				   ea.event_id ILIKE $2 OR
				   ea.attribute_id ILIKE $2 OR
				   ea.value ILIKE $2)
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
		return nil, fmt.Errorf("failed to query event attribute list page data: %w", err)
	}
	defer rows.Close()

	var eventAttributes []*eventattributepb.EventAttribute
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			eventID      string
			attributeID  string
			value        string
			active       bool
			dateCreated  time.Time
			dateModified time.Time
			total        int64
		)

		err := rows.Scan(
			&id,
			&eventID,
			&attributeID,
			&value,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event attribute row: %w", err)
		}

		totalCount = total

		eventAttribute := &eventattributepb.EventAttribute{
			Id:          id,
			EventId:     eventID,
			AttributeId: attributeID,
			Value:       value,
			Active:      active,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			eventAttribute.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			eventAttribute.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			eventAttribute.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			eventAttribute.DateModifiedString = &dmStr
		}

		eventAttributes = append(eventAttributes, eventAttribute)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event attribute rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &eventattributepb.GetEventAttributeListPageDataResponse{
		EventAttributeList: eventAttributes,
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

// GetEventAttributeItemPageData retrieves a single event attribute with enhanced item page data.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
func (r *PostgresEventAttributeRepository) GetEventAttributeItemPageData(
	ctx context.Context,
	req *eventattributepb.GetEventAttributeItemPageDataRequest,
) (*eventattributepb.GetEventAttributeItemPageDataResponse, error) {
	if req == nil || req.EventAttributeId == "" {
		return nil, fmt.Errorf("event attribute ID required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	query := `
		SELECT
			ea.id,
			ea.event_id,
			ea.attribute_id,
			ea.value,
			ea.active,
			ea.date_created,
			ea.date_modified
		FROM event_attribute ea
		WHERE ea.id = $1 AND ea.workspace_id = $2 AND ea.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.EventAttributeId, workspaceID)

	var (
		id           string
		eventID      string
		attributeID  string
		value        string
		active       bool
		dateCreated  time.Time
		dateModified time.Time
	)

	err := row.Scan(
		&id,
		&eventID,
		&attributeID,
		&value,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event attribute with ID '%s' not found", req.EventAttributeId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event attribute item page data: %w", err)
	}

	eventAttribute := &eventattributepb.EventAttribute{
		Id:          id,
		EventId:     eventID,
		AttributeId: attributeID,
		Value:       value,
		Active:      active,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		eventAttribute.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		eventAttribute.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		eventAttribute.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		eventAttribute.DateModifiedString = &dmStr
	}

	return &eventattributepb.GetEventAttributeItemPageDataResponse{
		EventAttribute: eventAttribute,
		Success:        true,
	}, nil
}

// NewEventAttributeRepository creates a new PostgreSQL event_attribute repository (old-style constructor)
func NewEventAttributeRepository(db *sql.DB, tableName string) eventattributepb.EventAttributeDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventAttributeRepository(dbOps, tableName)
}
