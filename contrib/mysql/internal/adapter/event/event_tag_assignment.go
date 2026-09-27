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
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.EventTagAssignment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql event_tag_assignment repository requires *sql.DB, got %T", conn)
		}
		return NewMySQLEventTagAssignmentRepository(mysqlCore.NewWorkspaceAwareOperations(db), tableName), nil
	})
}

type MySQLEventTagAssignmentRepository struct {
	eventtagassignmentpb.UnimplementedEventTagAssignmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewMySQLEventTagAssignmentRepository(dbOps interfaces.DatabaseOperation, tableName string) eventtagassignmentpb.EventTagAssignmentDomainServiceServer {
	if tableName == "" {
		tableName = "event_tag_assignment"
	}
	return &MySQLEventTagAssignmentRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLEventTagAssignmentRepository) CreateEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("event_tag_assignment data or ID is required")
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
		return nil, fmt.Errorf("create event_tag_assignment: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventtagassignmentpb.EventTagAssignment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventtagassignmentpb.CreateEventTagAssignmentResponse{Data: []*eventtagassignmentpb.EventTagAssignment{obj}}, nil
}

func (r *MySQLEventTagAssignmentRepository) ReadEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.ReadEventTagAssignmentRequest) (*eventtagassignmentpb.ReadEventTagAssignmentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment data or ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("read event_tag_assignment: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	obj := &eventtagassignmentpb.EventTagAssignment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, obj); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return &eventtagassignmentpb.ReadEventTagAssignmentResponse{Data: []*eventtagassignmentpb.EventTagAssignment{obj}}, nil
}

func (r *MySQLEventTagAssignmentRepository) DeleteEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.DeleteEventTagAssignmentRequest) (*eventtagassignmentpb.DeleteEventTagAssignmentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment data or ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("delete event_tag_assignment: %w", err)
	}
	return &eventtagassignmentpb.DeleteEventTagAssignmentResponse{Success: true}, nil
}

func (r *MySQLEventTagAssignmentRepository) ListEventTagAssignments(ctx context.Context, req *eventtagassignmentpb.ListEventTagAssignmentsRequest) (*eventtagassignmentpb.ListEventTagAssignmentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("list event_tag_assignment: %w", err)
	}
	items := make([]*eventtagassignmentpb.EventTagAssignment, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, fmt.Errorf("marshal list row: %w", err)
		}
		obj := &eventtagassignmentpb.EventTagAssignment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, fmt.Errorf("decode list row: %w", err)
		}
		items = append(items, obj)
	}
	return &eventtagassignmentpb.ListEventTagAssignmentsResponse{Data: items}, nil
}
