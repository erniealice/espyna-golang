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
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EventClient, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event_client repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventClientRepository(dbOps, tableName), nil
	})
}

// SQLServerEventClientRepository implements event_client CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerEventClientRepository struct {
	eventclientpb.UnimplementedEventClientDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerEventClientRepository creates a new SQL Server event_client repository.
func NewSQLServerEventClientRepository(dbOps interfaces.DatabaseOperation, tableName string) eventclientpb.EventClientDomainServiceServer {
	if tableName == "" {
		tableName = "event_client"
	}
	return &SQLServerEventClientRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerEventClientRepository) CreateEventClient(ctx context.Context, req *eventclientpb.CreateEventClientRequest) (*eventclientpb.CreateEventClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_client data is required")
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
		return nil, fmt.Errorf("failed to create event_client: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventclientpb.EventClient{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventclientpb.CreateEventClientResponse{Data: []*eventclientpb.EventClient{obj}}, nil
}

func (r *SQLServerEventClientRepository) ReadEventClient(ctx context.Context, req *eventclientpb.ReadEventClientRequest) (*eventclientpb.ReadEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_client ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_client: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventclientpb.EventClient{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventclientpb.ReadEventClientResponse{Data: []*eventclientpb.EventClient{obj}}, nil
}

func (r *SQLServerEventClientRepository) UpdateEventClient(ctx context.Context, req *eventclientpb.UpdateEventClientRequest) (*eventclientpb.UpdateEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_client ID is required")
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
		return nil, fmt.Errorf("failed to update event_client: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventclientpb.EventClient{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventclientpb.UpdateEventClientResponse{Data: []*eventclientpb.EventClient{obj}}, nil
}

func (r *SQLServerEventClientRepository) DeleteEventClient(ctx context.Context, req *eventclientpb.DeleteEventClientRequest) (*eventclientpb.DeleteEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_client ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete event_client: %w", err)
	}
	return &eventclientpb.DeleteEventClientResponse{Success: true}, nil
}

func (r *SQLServerEventClientRepository) ListEventClients(ctx context.Context, req *eventclientpb.ListEventClientsRequest) (*eventclientpb.ListEventClientsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_clients: %w", err)
	}
	var items []*eventclientpb.EventClient
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &eventclientpb.EventClient{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &eventclientpb.ListEventClientsResponse{Data: items}, nil
}

func (r *SQLServerEventClientRepository) GetEventClientListPageData(ctx context.Context, req *eventclientpb.GetEventClientListPageDataRequest) (*eventclientpb.GetEventClientListPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventClientListPageData not yet implemented")
}

func (r *SQLServerEventClientRepository) GetEventClientItemPageData(ctx context.Context, req *eventclientpb.GetEventClientItemPageDataRequest) (*eventclientpb.GetEventClientItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventClientItemPageData not yet implemented")
}
