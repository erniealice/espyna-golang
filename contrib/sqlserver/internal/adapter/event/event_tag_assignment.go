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
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EventTagAssignment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver event_tag_assignment repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEventTagAssignmentRepository(dbOps, tableName), nil
	})
}

// SQLServerEventTagAssignmentRepository implements event_tag_assignment CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE; active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
type SQLServerEventTagAssignmentRepository struct {
	eventtagassignmentpb.UnimplementedEventTagAssignmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerEventTagAssignmentRepository creates a new SQL Server event_tag_assignment repository.
func NewSQLServerEventTagAssignmentRepository(dbOps interfaces.DatabaseOperation, tableName string) eventtagassignmentpb.EventTagAssignmentDomainServiceServer {
	if tableName == "" {
		tableName = "event_tag_assignment"
	}
	return &SQLServerEventTagAssignmentRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerEventTagAssignmentRepository) CreateEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_tag_assignment data is required")
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
		return nil, fmt.Errorf("failed to create event_tag_assignment: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventtagassignmentpb.EventTagAssignment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventtagassignmentpb.CreateEventTagAssignmentResponse{Data: []*eventtagassignmentpb.EventTagAssignment{obj}}, nil
}

func (r *SQLServerEventTagAssignmentRepository) ReadEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.ReadEventTagAssignmentRequest) (*eventtagassignmentpb.ReadEventTagAssignmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_tag_assignment: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	obj := &eventtagassignmentpb.EventTagAssignment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &eventtagassignmentpb.ReadEventTagAssignmentResponse{Data: []*eventtagassignmentpb.EventTagAssignment{obj}}, nil
}

func (r *SQLServerEventTagAssignmentRepository) DeleteEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.DeleteEventTagAssignmentRequest) (*eventtagassignmentpb.DeleteEventTagAssignmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete event_tag_assignment: %w", err)
	}
	return &eventtagassignmentpb.DeleteEventTagAssignmentResponse{Success: true}, nil
}

func (r *SQLServerEventTagAssignmentRepository) ListEventTagAssignments(ctx context.Context, req *eventtagassignmentpb.ListEventTagAssignmentsRequest) (*eventtagassignmentpb.ListEventTagAssignmentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_tag_assignments: %w", err)
	}
	var items []*eventtagassignmentpb.EventTagAssignment
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		obj := &eventtagassignmentpb.EventTagAssignment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
			continue
		}
		items = append(items, obj)
	}
	return &eventtagassignmentpb.ListEventTagAssignmentsResponse{Data: items}, nil
}
