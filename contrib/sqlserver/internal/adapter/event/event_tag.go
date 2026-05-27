//go:build sqlserver

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EventTag, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event_tag repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventTagRepository(dbOps, tableName), nil
	})
}

// SQLServerEventTagRepository implements event_tag CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerEventTagRepository struct {
	eventtagpb.UnimplementedEventTagDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerEventTagRepository creates a new SQL Server event_tag repository.
func NewSQLServerEventTagRepository(dbOps interfaces.DatabaseOperation, tableName string) eventtagpb.EventTagDomainServiceServer {
	if tableName == "" {
		tableName = "event_tag"
	}
	return &SQLServerEventTagRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerEventTagRepository) CreateEventTag(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_tag data is required")
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
		return nil, fmt.Errorf("failed to create event_tag: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventtagpb.CreateEventTagResponse{Data: []*eventtagpb.EventTag{obj}}, nil
}

func (r *SQLServerEventTagRepository) ReadEventTag(ctx context.Context, req *eventtagpb.ReadEventTagRequest) (*eventtagpb.ReadEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_tag: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventtagpb.ReadEventTagResponse{Data: []*eventtagpb.EventTag{obj}}, nil
}

func (r *SQLServerEventTagRepository) UpdateEventTag(ctx context.Context, req *eventtagpb.UpdateEventTagRequest) (*eventtagpb.UpdateEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required")
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
		return nil, fmt.Errorf("failed to update event_tag: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventtagpb.EventTag{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventtagpb.UpdateEventTagResponse{Data: []*eventtagpb.EventTag{obj}}, nil
}

func (r *SQLServerEventTagRepository) DeleteEventTag(ctx context.Context, req *eventtagpb.DeleteEventTagRequest) (*eventtagpb.DeleteEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete event_tag: %w", err)
	}
	return &eventtagpb.DeleteEventTagResponse{Success: true}, nil
}

func (r *SQLServerEventTagRepository) ListEventTags(ctx context.Context, req *eventtagpb.ListEventTagsRequest) (*eventtagpb.ListEventTagsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_tags: %w", err)
	}
	var items []*eventtagpb.EventTag
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &eventtagpb.EventTag{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &eventtagpb.ListEventTagsResponse{Data: items}, nil
}

func (r *SQLServerEventTagRepository) GetEventTagListPageData(ctx context.Context, req *eventtagpb.GetEventTagListPageDataRequest) (*eventtagpb.GetEventTagListPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventTagListPageData not yet implemented")
}

func (r *SQLServerEventTagRepository) GetEventTagItemPageData(ctx context.Context, req *eventtagpb.GetEventTagItemPageDataRequest) (*eventtagpb.GetEventTagItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventTagItemPageData not yet implemented")
}
