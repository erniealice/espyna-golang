//go:build postgresql

package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresEventTagAssignmentRepository implements event_tag_assignment CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_event_tag_assignment_active ON event_tag_assignment(active) WHERE active = true
//   - CREATE INDEX idx_event_tag_assignment_event_id ON event_tag_assignment(event_id)
//   - CREATE INDEX idx_event_tag_assignment_event_tag_id ON event_tag_assignment(event_tag_id)
//   - CREATE UNIQUE INDEX uq_event_tag_assignment_pair ON event_tag_assignment(event_id, event_tag_id) WHERE active = true
type PostgresEventTagAssignmentRepository struct {
	eventtagassignmentpb.UnimplementedEventTagAssignmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EventTagAssignment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres event_tag_assignment repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEventTagAssignmentRepository(dbOps, tableName), nil
	})
}

// NewPostgresEventTagAssignmentRepository creates a new PostgreSQL event_tag_assignment repository
func NewPostgresEventTagAssignmentRepository(dbOps interfaces.DatabaseOperation, tableName string) eventtagassignmentpb.EventTagAssignmentDomainServiceServer {
	if tableName == "" {
		tableName = "event_tag_assignment"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresEventTagAssignmentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEventTagAssignment creates a new event_tag_assignment
func (r *PostgresEventTagAssignmentRepository) CreateEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
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

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	assignment := &eventtagassignmentpb.EventTagAssignment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, assignment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventtagassignmentpb.CreateEventTagAssignmentResponse{
		Data: []*eventtagassignmentpb.EventTagAssignment{assignment},
	}, nil
}

// ReadEventTagAssignment retrieves an event_tag_assignment
func (r *PostgresEventTagAssignmentRepository) ReadEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.ReadEventTagAssignmentRequest) (*eventtagassignmentpb.ReadEventTagAssignmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_tag_assignment: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	assignment := &eventtagassignmentpb.EventTagAssignment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, assignment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &eventtagassignmentpb.ReadEventTagAssignmentResponse{
		Data: []*eventtagassignmentpb.EventTagAssignment{assignment},
	}, nil
}

// DeleteEventTagAssignment deletes an event_tag_assignment (soft delete)
func (r *PostgresEventTagAssignmentRepository) DeleteEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.DeleteEventTagAssignmentRequest) (*eventtagassignmentpb.DeleteEventTagAssignmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event_tag_assignment: %w", err)
	}

	return &eventtagassignmentpb.DeleteEventTagAssignmentResponse{
		Success: true,
	}, nil
}

// ListEventTagAssignments lists event_tag_assignments using common PostgreSQL operations
func (r *PostgresEventTagAssignmentRepository) ListEventTagAssignments(ctx context.Context, req *eventtagassignmentpb.ListEventTagAssignmentsRequest) (*eventtagassignmentpb.ListEventTagAssignmentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_tag_assignments: %w", err)
	}

	var assignments []*eventtagassignmentpb.EventTagAssignment
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		assignment := &eventtagassignmentpb.EventTagAssignment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, assignment); err != nil {
			continue
		}
		assignments = append(assignments, assignment)
	}

	return &eventtagassignmentpb.ListEventTagAssignmentsResponse{
		Data: assignments,
	}, nil
}

// NewEventTagAssignmentRepository creates a new PostgreSQL event_tag_assignment repository (old-style constructor)
func NewEventTagAssignmentRepository(db *sql.DB, tableName string) eventtagassignmentpb.EventTagAssignmentDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEventTagAssignmentRepository(dbOps, tableName)
}
