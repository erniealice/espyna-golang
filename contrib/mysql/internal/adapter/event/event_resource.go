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
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.EventResource, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql event_resource repository requires *sql.DB, got %T", conn)
		}
		return NewMySQLEventResourceRepository(mysqlCore.NewWorkspaceAwareOperations(db), tableName), nil
	})
}

type MySQLEventResourceRepository struct {
	eventresourcepb.UnimplementedEventResourceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewMySQLEventResourceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventresourcepb.EventResourceDomainServiceServer {
	if tableName == "" {
		tableName = "event_resource"
	}
	return &MySQLEventResourceRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLEventResourceRepository) CreateEventResource(ctx context.Context, req *eventresourcepb.CreateEventResourceRequest) (*eventresourcepb.CreateEventResourceResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("event_resource data or ID is required")
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
		return nil, fmt.Errorf("create event_resource: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventresourcepb.CreateEventResourceResponse{Data: []*eventresourcepb.EventResource{obj}}, nil
}

func (r *MySQLEventResourceRepository) ReadEventResource(ctx context.Context, req *eventresourcepb.ReadEventResourceRequest) (*eventresourcepb.ReadEventResourceResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_resource data or ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("read event_resource: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventresourcepb.ReadEventResourceResponse{Data: []*eventresourcepb.EventResource{obj}}, nil
}

func (r *MySQLEventResourceRepository) UpdateEventResource(ctx context.Context, req *eventresourcepb.UpdateEventResourceRequest) (*eventresourcepb.UpdateEventResourceResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_resource data or ID is required")
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
		return nil, fmt.Errorf("update event_resource: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventresourcepb.UpdateEventResourceResponse{Data: []*eventresourcepb.EventResource{obj}}, nil
}

func (r *MySQLEventResourceRepository) DeleteEventResource(ctx context.Context, req *eventresourcepb.DeleteEventResourceRequest) (*eventresourcepb.DeleteEventResourceResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_resource data or ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("delete event_resource: %w", err)
	}
	return &eventresourcepb.DeleteEventResourceResponse{Success: true}, nil
}

func (r *MySQLEventResourceRepository) ListEventResources(ctx context.Context, req *eventresourcepb.ListEventResourcesRequest) (*eventresourcepb.ListEventResourcesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("list event_resource: %w", err)
	}
	items := make([]*eventresourcepb.EventResource, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, fmt.Errorf("marshal list row: %w", err)
		}
		obj := &eventresourcepb.EventResource{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, fmt.Errorf("decode list row: %w", err)
		}
		items = append(items, obj)
	}
	return &eventresourcepb.ListEventResourcesResponse{Data: items}, nil
}

func (r *MySQLEventResourceRepository) GetEventResourceListPageData(ctx context.Context, req *eventresourcepb.GetEventResourceListPageDataRequest) (*eventresourcepb.GetEventResourceListPageDataResponse, error) {
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
	items := make([]*eventresourcepb.EventResource, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, err
		}
		obj := &eventresourcepb.EventResource{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, err
		}
		items = append(items, obj)
	}
	return &eventresourcepb.GetEventResourceListPageDataResponse{EventResourceList: items, Pagination: rows.Pagination}, nil
}

func (r *MySQLEventResourceRepository) GetEventResourceItemPageData(ctx context.Context, req *eventresourcepb.GetEventResourceItemPageDataRequest) (*eventresourcepb.GetEventResourceItemPageDataResponse, error) {
	if req == nil || req.EventResourceId == "" {
		return nil, fmt.Errorf("event_resource ID is required")
	}
	row, err := r.dbOps.Read(ctx, r.tableName, req.EventResourceId)
	if err != nil {
		return nil, fmt.Errorf("read event_resource: %w", err)
	}
	b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
	if err != nil {
		return nil, err
	}
	obj := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
		return nil, err
	}
	return &eventresourcepb.GetEventResourceItemPageDataResponse{EventResource: obj}, nil
}
