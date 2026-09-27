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
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.EventTag, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql event_tag repository requires *sql.DB, got %T", conn)
		}
		return NewMySQLEventTagRepository(mysqlCore.NewWorkspaceAwareOperations(db), tableName), nil
	})
}

type MySQLEventTagRepository struct {
	eventtagpb.UnimplementedEventTagDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewMySQLEventTagRepository(dbOps interfaces.DatabaseOperation, tableName string) eventtagpb.EventTagDomainServiceServer {
	if tableName == "" {
		tableName = "event_tag"
	}
	return &MySQLEventTagRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLEventTagRepository) CreateEventTag(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("event_tag data or ID is required")
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
		return nil, fmt.Errorf("create event_tag: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventtagpb.CreateEventTagResponse{Data: []*eventtagpb.EventTag{obj}}, nil
}

func (r *MySQLEventTagRepository) ReadEventTag(ctx context.Context, req *eventtagpb.ReadEventTagRequest) (*eventtagpb.ReadEventTagResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag data or ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("read event_tag: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventtagpb.ReadEventTagResponse{Data: []*eventtagpb.EventTag{obj}}, nil
}

func (r *MySQLEventTagRepository) UpdateEventTag(ctx context.Context, req *eventtagpb.UpdateEventTagRequest) (*eventtagpb.UpdateEventTagResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag data or ID is required")
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
		return nil, fmt.Errorf("update event_tag: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventtagpb.UpdateEventTagResponse{Data: []*eventtagpb.EventTag{obj}}, nil
}

func (r *MySQLEventTagRepository) DeleteEventTag(ctx context.Context, req *eventtagpb.DeleteEventTagRequest) (*eventtagpb.DeleteEventTagResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag data or ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("delete event_tag: %w", err)
	}
	return &eventtagpb.DeleteEventTagResponse{Success: true}, nil
}

func (r *MySQLEventTagRepository) ListEventTags(ctx context.Context, req *eventtagpb.ListEventTagsRequest) (*eventtagpb.ListEventTagsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("list event_tag: %w", err)
	}
	items := make([]*eventtagpb.EventTag, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, fmt.Errorf("marshal list row: %w", err)
		}
		obj := &eventtagpb.EventTag{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, fmt.Errorf("decode list row: %w", err)
		}
		items = append(items, obj)
	}
	return &eventtagpb.ListEventTagsResponse{Data: items}, nil
}

func (r *MySQLEventTagRepository) GetEventTagListPageData(ctx context.Context, req *eventtagpb.GetEventTagListPageDataRequest) (*eventtagpb.GetEventTagListPageDataResponse, error) {
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
	items := make([]*eventtagpb.EventTag, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, err
		}
		obj := &eventtagpb.EventTag{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, err
		}
		items = append(items, obj)
	}
	return &eventtagpb.GetEventTagListPageDataResponse{EventTagList: items, Pagination: rows.Pagination}, nil
}

func (r *MySQLEventTagRepository) GetEventTagItemPageData(ctx context.Context, req *eventtagpb.GetEventTagItemPageDataRequest) (*eventtagpb.GetEventTagItemPageDataResponse, error) {
	if req == nil || req.EventTagId == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}
	row, err := r.dbOps.Read(ctx, r.tableName, req.EventTagId)
	if err != nil {
		return nil, fmt.Errorf("read event_tag: %w", err)
	}
	b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
	if err != nil {
		return nil, err
	}
	obj := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
		return nil, err
	}
	return &eventtagpb.GetEventTagItemPageDataResponse{EventTag: obj}, nil
}
