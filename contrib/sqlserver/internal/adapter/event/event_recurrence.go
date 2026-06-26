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
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EventRecurrence, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event_recurrence repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventRecurrenceRepository(dbOps, tableName), nil
	})
}

// SQLServerEventRecurrenceRepository implements event_recurrence CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerEventRecurrenceRepository struct {
	eventrecurrencepb.UnimplementedEventRecurrenceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerEventRecurrenceRepository creates a new SQL Server event_recurrence repository.
func NewSQLServerEventRecurrenceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventrecurrencepb.EventRecurrenceDomainServiceServer {
	if tableName == "" {
		tableName = "event_recurrence"
	}
	return &SQLServerEventRecurrenceRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerEventRecurrenceRepository) CreateEventRecurrence(ctx context.Context, req *eventrecurrencepb.CreateEventRecurrenceRequest) (*eventrecurrencepb.CreateEventRecurrenceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_recurrence data is required")
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
		return nil, fmt.Errorf("failed to create event_recurrence: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventrecurrencepb.CreateEventRecurrenceResponse{Data: []*eventrecurrencepb.EventRecurrence{obj}}, nil
}

func (r *SQLServerEventRecurrenceRepository) ReadEventRecurrence(ctx context.Context, req *eventrecurrencepb.ReadEventRecurrenceRequest) (*eventrecurrencepb.ReadEventRecurrenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_recurrence ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_recurrence: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventrecurrencepb.ReadEventRecurrenceResponse{Data: []*eventrecurrencepb.EventRecurrence{obj}}, nil
}

func (r *SQLServerEventRecurrenceRepository) UpdateEventRecurrence(ctx context.Context, req *eventrecurrencepb.UpdateEventRecurrenceRequest) (*eventrecurrencepb.UpdateEventRecurrenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_recurrence ID is required")
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
		return nil, fmt.Errorf("failed to update event_recurrence: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventrecurrencepb.EventRecurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventrecurrencepb.UpdateEventRecurrenceResponse{Data: []*eventrecurrencepb.EventRecurrence{obj}}, nil
}

func (r *SQLServerEventRecurrenceRepository) DeleteEventRecurrence(ctx context.Context, req *eventrecurrencepb.DeleteEventRecurrenceRequest) (*eventrecurrencepb.DeleteEventRecurrenceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_recurrence ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete event_recurrence: %w", err)
	}
	return &eventrecurrencepb.DeleteEventRecurrenceResponse{Success: true}, nil
}

func (r *SQLServerEventRecurrenceRepository) ListEventRecurrences(ctx context.Context, req *eventrecurrencepb.ListEventRecurrencesRequest) (*eventrecurrencepb.ListEventRecurrencesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_recurrences: %w", err)
	}
	var items []*eventrecurrencepb.EventRecurrence
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &eventrecurrencepb.EventRecurrence{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &eventrecurrencepb.ListEventRecurrencesResponse{Data: items}, nil
}

func (r *SQLServerEventRecurrenceRepository) GetEventRecurrenceListPageData(ctx context.Context, req *eventrecurrencepb.GetEventRecurrenceListPageDataRequest) (*eventrecurrencepb.GetEventRecurrenceListPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventRecurrenceListPageData not yet implemented")
}

func (r *SQLServerEventRecurrenceRepository) GetEventRecurrenceItemPageData(ctx context.Context, req *eventrecurrencepb.GetEventRecurrenceItemPageDataRequest) (*eventrecurrencepb.GetEventRecurrenceItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetEventRecurrenceItemPageData not yet implemented")
}
