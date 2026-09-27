//go:build mysql

package event

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.EventAttendee, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql event_attendee repository requires *sql.DB, got %T", conn)
		}
		return NewMySQLEventAttendeeRepository(mysqlCore.NewWorkspaceAwareOperations(db), tableName), nil
	})
}

type MySQLEventAttendeeRepository struct {
	eventattendeepb.UnimplementedEventAttendeeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewMySQLEventAttendeeRepository(dbOps interfaces.DatabaseOperation, tableName string) eventattendeepb.EventAttendeeDomainServiceServer {
	if tableName == "" {
		tableName = "event_attendee"
	}
	return &MySQLEventAttendeeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLEventAttendeeRepository) CreateEventAttendee(ctx context.Context, req *eventattendeepb.CreateEventAttendeeRequest) (*eventattendeepb.CreateEventAttendeeResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("event_attendee data or ID is required")
	}
	b, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("encode data: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("decode data: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("create event_attendee: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventattendeepb.CreateEventAttendeeResponse{Data: []*eventattendeepb.EventAttendee{obj}}, nil
}

func (r *MySQLEventAttendeeRepository) ReadEventAttendee(ctx context.Context, req *eventattendeepb.ReadEventAttendeeRequest) (*eventattendeepb.ReadEventAttendeeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_attendee data or ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("read event_attendee: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventattendeepb.ReadEventAttendeeResponse{Data: []*eventattendeepb.EventAttendee{obj}}, nil
}

func (r *MySQLEventAttendeeRepository) UpdateEventAttendee(ctx context.Context, req *eventattendeepb.UpdateEventAttendeeRequest) (*eventattendeepb.UpdateEventAttendeeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_attendee data or ID is required")
	}
	b, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("encode data: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("decode data: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("update event_attendee: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventattendeepb.UpdateEventAttendeeResponse{Data: []*eventattendeepb.EventAttendee{obj}}, nil
}

func (r *MySQLEventAttendeeRepository) DeleteEventAttendee(ctx context.Context, req *eventattendeepb.DeleteEventAttendeeRequest) (*eventattendeepb.DeleteEventAttendeeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_attendee data or ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("delete event_attendee: %w", err)
	}
	return &eventattendeepb.DeleteEventAttendeeResponse{Success: true}, nil
}

func (r *MySQLEventAttendeeRepository) ListEventAttendees(ctx context.Context, req *eventattendeepb.ListEventAttendeesRequest) (*eventattendeepb.ListEventAttendeesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("list event_attendee: %w", err)
	}
	items := make([]*eventattendeepb.EventAttendee, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, fmt.Errorf("marshal list row: %w", err)
		}
		obj := &eventattendeepb.EventAttendee{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, fmt.Errorf("decode list row: %w", err)
		}
		items = append(items, obj)
	}
	return &eventattendeepb.ListEventAttendeesResponse{Data: items}, nil
}

func (r *MySQLEventAttendeeRepository) GetEventAttendeeListPageData(ctx context.Context, req *eventattendeepb.GetEventAttendeeListPageDataRequest) (*eventattendeepb.GetEventAttendeeListPageDataResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Sort = req.Sort
		params.Search = req.Search
		params.Pagination = req.Pagination
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, err
	}
	items := make([]*eventattendeepb.EventAttendee, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, err
		}
		obj := &eventattendeepb.EventAttendee{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, err
		}
		items = append(items, obj)
	}
	return &eventattendeepb.GetEventAttendeeListPageDataResponse{EventAttendeeList: items, Pagination: rows.Pagination}, nil
}

func (r *MySQLEventAttendeeRepository) GetEventAttendeeItemPageData(ctx context.Context, req *eventattendeepb.GetEventAttendeeItemPageDataRequest) (*eventattendeepb.GetEventAttendeeItemPageDataResponse, error) {
	if req == nil || req.EventAttendeeId == "" {
		return nil, fmt.Errorf("event_attendee ID is required")
	}
	row, err := r.dbOps.Read(ctx, r.tableName, req.EventAttendeeId)
	if err != nil {
		return nil, fmt.Errorf("read event_attendee: %w", err)
	}
	b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
	if err != nil {
		return nil, err
	}
	obj := &eventattendeepb.EventAttendee{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
		return nil, err
	}
	return &eventattendeepb.GetEventAttendeeItemPageDataResponse{EventAttendee: obj}, nil
}
