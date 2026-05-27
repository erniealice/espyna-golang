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
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EventResource, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event_resource repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventResourceRepository(dbOps, tableName), nil
	})
}

// SQLServerEventResourceRepository implements event_resource CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerEventResourceRepository struct {
	eventresourcepb.UnimplementedEventResourceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerEventResourceRepository creates a new SQL Server event_resource repository.
func NewSQLServerEventResourceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventresourcepb.EventResourceDomainServiceServer {
	if tableName == "" {
		tableName = "event_resource"
	}
	return &SQLServerEventResourceRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerEventResourceRepository) CreateEventResource(ctx context.Context, req *eventresourcepb.CreateEventResourceRequest) (*eventresourcepb.CreateEventResourceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_resource data is required")
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
		return nil, fmt.Errorf("failed to create event_resource: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventresourcepb.CreateEventResourceResponse{Data: []*eventresourcepb.EventResource{obj}}, nil
}

func (r *SQLServerEventResourceRepository) ReadEventResource(ctx context.Context, req *eventresourcepb.ReadEventResourceRequest) (*eventresourcepb.ReadEventResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_resource ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_resource: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventresourcepb.ReadEventResourceResponse{Data: []*eventresourcepb.EventResource{obj}}, nil
}

func (r *SQLServerEventResourceRepository) UpdateEventResource(ctx context.Context, req *eventresourcepb.UpdateEventResourceRequest) (*eventresourcepb.UpdateEventResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_resource ID is required")
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
		return nil, fmt.Errorf("failed to update event_resource: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventresourcepb.EventResource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventresourcepb.UpdateEventResourceResponse{Data: []*eventresourcepb.EventResource{obj}}, nil
}

func (r *SQLServerEventResourceRepository) DeleteEventResource(ctx context.Context, req *eventresourcepb.DeleteEventResourceRequest) (*eventresourcepb.DeleteEventResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_resource ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete event_resource: %w", err)
	}
	return &eventresourcepb.DeleteEventResourceResponse{Success: true}, nil
}

func (r *SQLServerEventResourceRepository) ListEventResources(ctx context.Context, req *eventresourcepb.ListEventResourcesRequest) (*eventresourcepb.ListEventResourcesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_resources: %w", err)
	}
	var items []*eventresourcepb.EventResource
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &eventresourcepb.EventResource{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &eventresourcepb.ListEventResourcesResponse{Data: items}, nil
}

func (r *SQLServerEventResourceRepository) GetEventResourceListPageData(ctx context.Context, req *eventresourcepb.GetEventResourceListPageDataRequest) (*eventresourcepb.GetEventResourceListPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventResourceListPageData not yet implemented")
}

func (r *SQLServerEventResourceRepository) GetEventResourceItemPageData(ctx context.Context, req *eventresourcepb.GetEventResourceItemPageDataRequest) (*eventresourcepb.GetEventResourceItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventResourceItemPageData not yet implemented")
}
