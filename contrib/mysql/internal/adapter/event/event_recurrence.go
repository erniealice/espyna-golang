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
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.EventRecurrence, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql event_recurrence repository requires *sql.DB, got %T", conn)
		}
		return NewMySQLEventRecurrenceRepository(mysqlCore.NewWorkspaceAwareOperations(db), tableName), nil
	})
}

type MySQLEventRecurrenceRepository struct {
	eventrecurrencepb.UnimplementedEventRecurrenceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewMySQLEventRecurrenceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventrecurrencepb.EventRecurrenceDomainServiceServer {
	if tableName == "" {
		tableName = "event_recurrence"
	}
	return &MySQLEventRecurrenceRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLEventRecurrenceRepository) CreateEventRecurrence(ctx context.Context, req *eventrecurrencepb.CreateEventRecurrenceRequest) (*eventrecurrencepb.CreateEventRecurrenceResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("event_recurrence data or ID is required")
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
		return nil, fmt.Errorf("create event_recurrence: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventrecurrencepb.CreateEventRecurrenceResponse{Data: []*eventrecurrencepb.EventRecurrence{obj}}, nil
}

func (r *MySQLEventRecurrenceRepository) ReadEventRecurrence(ctx context.Context, req *eventrecurrencepb.ReadEventRecurrenceRequest) (*eventrecurrencepb.ReadEventRecurrenceResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_recurrence data or ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("read event_recurrence: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventrecurrencepb.ReadEventRecurrenceResponse{Data: []*eventrecurrencepb.EventRecurrence{obj}}, nil
}

func (r *MySQLEventRecurrenceRepository) UpdateEventRecurrence(ctx context.Context, req *eventrecurrencepb.UpdateEventRecurrenceRequest) (*eventrecurrencepb.UpdateEventRecurrenceResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_recurrence data or ID is required")
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
		return nil, fmt.Errorf("update event_recurrence: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventrecurrencepb.UpdateEventRecurrenceResponse{Data: []*eventrecurrencepb.EventRecurrence{obj}}, nil
}

func (r *MySQLEventRecurrenceRepository) DeleteEventRecurrence(ctx context.Context, req *eventrecurrencepb.DeleteEventRecurrenceRequest) (*eventrecurrencepb.DeleteEventRecurrenceResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_recurrence data or ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("delete event_recurrence: %w", err)
	}
	return &eventrecurrencepb.DeleteEventRecurrenceResponse{Success: true}, nil
}

func (r *MySQLEventRecurrenceRepository) ListEventRecurrences(ctx context.Context, req *eventrecurrencepb.ListEventRecurrencesRequest) (*eventrecurrencepb.ListEventRecurrencesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("list event_recurrence: %w", err)
	}
	items := make([]*eventrecurrencepb.EventRecurrence, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, fmt.Errorf("marshal list row: %w", err)
		}
		obj := &eventrecurrencepb.EventRecurrence{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, fmt.Errorf("decode list row: %w", err)
		}
		items = append(items, obj)
	}
	return &eventrecurrencepb.ListEventRecurrencesResponse{Data: items}, nil
}

func (r *MySQLEventRecurrenceRepository) GetEventRecurrenceListPageData(ctx context.Context, req *eventrecurrencepb.GetEventRecurrenceListPageDataRequest) (*eventrecurrencepb.GetEventRecurrenceListPageDataResponse, error) {
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
	items := make([]*eventrecurrencepb.EventRecurrence, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, err
		}
		obj := &eventrecurrencepb.EventRecurrence{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, err
		}
		items = append(items, obj)
	}
	return &eventrecurrencepb.GetEventRecurrenceListPageDataResponse{EventRecurrenceList: items, Pagination: rows.Pagination}, nil
}

func (r *MySQLEventRecurrenceRepository) GetEventRecurrenceItemPageData(ctx context.Context, req *eventrecurrencepb.GetEventRecurrenceItemPageDataRequest) (*eventrecurrencepb.GetEventRecurrenceItemPageDataResponse, error) {
	if req == nil || req.EventRecurrenceId == "" {
		return nil, fmt.Errorf("event_recurrence ID is required")
	}
	row, err := r.dbOps.Read(ctx, r.tableName, req.EventRecurrenceId)
	if err != nil {
		return nil, fmt.Errorf("read event_recurrence: %w", err)
	}
	b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
	if err != nil {
		return nil, err
	}
	obj := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
		return nil, err
	}
	return &eventrecurrencepb.GetEventRecurrenceItemPageDataResponse{EventRecurrence: obj}, nil
}
