package event

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firestoreCore "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", entityid.EventTagAssignment, func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore event_tag_assignment repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreEventTagAssignmentRepository(dbOps, collectionName), nil
	})
}

// FirestoreEventTagAssignmentRepository implements event_tag_assignment CRUD using Firestore
type FirestoreEventTagAssignmentRepository struct {
	eventtagassignmentpb.UnimplementedEventTagAssignmentDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreEventTagAssignmentRepository creates a new Firestore event_tag_assignment repository
func NewFirestoreEventTagAssignmentRepository(dbOps interfaces.DatabaseOperation, collectionName string) eventtagassignmentpb.EventTagAssignmentDomainServiceServer {
	if collectionName == "" {
		collectionName = "event_tag_assignment"
	}
	return &FirestoreEventTagAssignmentRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateEventTagAssignment creates a new event_tag_assignment
func (r *FirestoreEventTagAssignmentRepository) CreateEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_tag_assignment data is required")
	}

	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create event_tag_assignment: %w", err)
	}

	assignment := &eventtagassignmentpb.EventTagAssignment{}
	converted, err := operations.ConvertMapToProtobuf(result, assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventtagassignmentpb.CreateEventTagAssignmentResponse{
		Data: []*eventtagassignmentpb.EventTagAssignment{converted},
	}, nil
}

// ReadEventTagAssignment retrieves an event_tag_assignment
func (r *FirestoreEventTagAssignmentRepository) ReadEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.ReadEventTagAssignmentRequest) (*eventtagassignmentpb.ReadEventTagAssignmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_tag_assignment: %w", err)
	}

	assignment := &eventtagassignmentpb.EventTagAssignment{}
	converted, err := operations.ConvertMapToProtobuf(result, assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventtagassignmentpb.ReadEventTagAssignmentResponse{
		Data: []*eventtagassignmentpb.EventTagAssignment{converted},
	}, nil
}

// DeleteEventTagAssignment deletes an event_tag_assignment (soft delete)
func (r *FirestoreEventTagAssignmentRepository) DeleteEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.DeleteEventTagAssignmentRequest) (*eventtagassignmentpb.DeleteEventTagAssignmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required")
	}

	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event_tag_assignment: %w", err)
	}

	return &eventtagassignmentpb.DeleteEventTagAssignmentResponse{
		Success: true,
	}, nil
}

// ListEventTagAssignments lists event_tag_assignments
func (r *FirestoreEventTagAssignmentRepository) ListEventTagAssignments(ctx context.Context, req *eventtagassignmentpb.ListEventTagAssignmentsRequest) (*eventtagassignmentpb.ListEventTagAssignmentsResponse, error) {
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_tag_assignments: %w", err)
	}

	assignments, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *eventtagassignmentpb.EventTagAssignment {
		return &eventtagassignmentpb.EventTagAssignment{}
	})

	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	if assignments == nil {
		assignments = make([]*eventtagassignmentpb.EventTagAssignment, 0)
	}

	return &eventtagassignmentpb.ListEventTagAssignmentsResponse{
		Data: assignments,
	}, nil
}
