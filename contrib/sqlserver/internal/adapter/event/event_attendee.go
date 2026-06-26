//go:build sqlserver

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EventAttendee, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event_attendee repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventAttendeeRepository(dbOps, tableName), nil
	})
}

// SQLServerEventAttendeeRepository implements event_attendee CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerEventAttendeeRepository struct {
	eventattendeepb.UnimplementedEventAttendeeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerEventAttendeeRepository creates a new SQL Server event_attendee repository.
func NewSQLServerEventAttendeeRepository(dbOps interfaces.DatabaseOperation, tableName string) eventattendeepb.EventAttendeeDomainServiceServer {
	if tableName == "" {
		tableName = "event_attendee"
	}
	return &SQLServerEventAttendeeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerEventAttendeeRepository) CreateEventAttendee(ctx context.Context, req *eventattendeepb.CreateEventAttendeeRequest) (*eventattendeepb.CreateEventAttendeeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_attendee data is required")
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
		return nil, fmt.Errorf("failed to create event_attendee: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventattendeepb.CreateEventAttendeeResponse{Data: []*eventattendeepb.EventAttendee{obj}}, nil
}

func (r *SQLServerEventAttendeeRepository) ReadEventAttendee(ctx context.Context, req *eventattendeepb.ReadEventAttendeeRequest) (*eventattendeepb.ReadEventAttendeeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_attendee ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_attendee: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventattendeepb.ReadEventAttendeeResponse{Data: []*eventattendeepb.EventAttendee{obj}}, nil
}

func (r *SQLServerEventAttendeeRepository) UpdateEventAttendee(ctx context.Context, req *eventattendeepb.UpdateEventAttendeeRequest) (*eventattendeepb.UpdateEventAttendeeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_attendee ID is required")
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
		return nil, fmt.Errorf("failed to update event_attendee: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventattendeepb.UpdateEventAttendeeResponse{Data: []*eventattendeepb.EventAttendee{obj}}, nil
}

func (r *SQLServerEventAttendeeRepository) DeleteEventAttendee(ctx context.Context, req *eventattendeepb.DeleteEventAttendeeRequest) (*eventattendeepb.DeleteEventAttendeeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_attendee ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete event_attendee: %w", err)
	}
	return &eventattendeepb.DeleteEventAttendeeResponse{Success: true}, nil
}

func (r *SQLServerEventAttendeeRepository) ListEventAttendees(ctx context.Context, req *eventattendeepb.ListEventAttendeesRequest) (*eventattendeepb.ListEventAttendeesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_attendees: %w", err)
	}
	var items []*eventattendeepb.EventAttendee
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &eventattendeepb.EventAttendee{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &eventattendeepb.ListEventAttendeesResponse{Data: items}, nil
}

func (r *SQLServerEventAttendeeRepository) GetEventAttendeeListPageData(ctx context.Context, req *eventattendeepb.GetEventAttendeeListPageDataRequest) (*eventattendeepb.GetEventAttendeeListPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventAttendeeListPageData not yet implemented")
}

func (r *SQLServerEventAttendeeRepository) GetEventAttendeeItemPageData(ctx context.Context, req *eventattendeepb.GetEventAttendeeItemPageDataRequest) (*eventattendeepb.GetEventAttendeeItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventAttendeeItemPageData not yet implemented")
}
