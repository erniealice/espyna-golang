//go:build mock_db

package event

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// MockEventTagAssignmentRepository implements event_tag_assignment operations using stateful mock data
type MockEventTagAssignmentRepository struct {
	eventtagassignmentpb.UnimplementedEventTagAssignmentDomainServiceServer
	businessType string
	assignments  map[string]*eventtagassignmentpb.EventTagAssignment
	mutex        sync.RWMutex
	initialized  bool
}

// NewMockEventTagAssignmentRepository creates a new mock event_tag_assignment repository
func NewMockEventTagAssignmentRepository(businessType string) eventtagassignmentpb.EventTagAssignmentDomainServiceServer {
	if businessType == "" {
		businessType = "education"
	}
	return &MockEventTagAssignmentRepository{
		businessType: businessType,
		assignments:  make(map[string]*eventtagassignmentpb.EventTagAssignment),
	}
}

// NewEventTagAssignmentRepository creates a new mock event_tag_assignment repository - Provider interface compatibility
func NewEventTagAssignmentRepository(businessType string) eventtagassignmentpb.EventTagAssignmentDomainServiceServer {
	return NewMockEventTagAssignmentRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", entityid.EventTagAssignment, func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockEventTagAssignmentRepository(businessType), nil
	})
}

// CreateEventTagAssignment creates a new event_tag_assignment
func (r *MockEventTagAssignmentRepository) CreateEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("event_tag_assignment data is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if req.Data.Id == "" {
		req.Data.Id = fmt.Sprintf("event_tag_assignment_%d", time.Now().UnixNano())
	}

	now := time.Now()
	req.Data.DateCreated = &[]int64{now.Unix()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.Unix()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	r.assignments[req.Data.Id] = req.Data

	return &eventtagassignmentpb.CreateEventTagAssignmentResponse{
		Data:    []*eventtagassignmentpb.EventTagAssignment{req.Data},
		Success: true,
	}, nil
}

// ReadEventTagAssignment retrieves an event_tag_assignment by ID
func (r *MockEventTagAssignmentRepository) ReadEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.ReadEventTagAssignmentRequest) (*eventtagassignmentpb.ReadEventTagAssignmentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	assignment, exists := r.assignments[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("event_tag_assignment not found: %s", req.Data.Id)
	}

	return &eventtagassignmentpb.ReadEventTagAssignmentResponse{
		Data:    []*eventtagassignmentpb.EventTagAssignment{assignment},
		Success: true,
	}, nil
}

// DeleteEventTagAssignment deletes an event_tag_assignment
func (r *MockEventTagAssignmentRepository) DeleteEventTagAssignment(ctx context.Context, req *eventtagassignmentpb.DeleteEventTagAssignmentRequest) (*eventtagassignmentpb.DeleteEventTagAssignmentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag_assignment ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.assignments[req.Data.Id]; !exists {
		return nil, fmt.Errorf("event_tag_assignment not found: %s", req.Data.Id)
	}

	delete(r.assignments, req.Data.Id)

	return &eventtagassignmentpb.DeleteEventTagAssignmentResponse{
		Success: true,
	}, nil
}

// ListEventTagAssignments retrieves all event_tag_assignments
func (r *MockEventTagAssignmentRepository) ListEventTagAssignments(ctx context.Context, req *eventtagassignmentpb.ListEventTagAssignmentsRequest) (*eventtagassignmentpb.ListEventTagAssignmentsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	assignments := make([]*eventtagassignmentpb.EventTagAssignment, 0, len(r.assignments))
	for _, assignment := range r.assignments {
		assignments = append(assignments, assignment)
	}

	return &eventtagassignmentpb.ListEventTagAssignmentsResponse{
		Data:    assignments,
		Success: true,
	}, nil
}
