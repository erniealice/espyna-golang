//go:build mock_db

package event

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// MockEventTagRepository implements event_tag operations using stateful mock data
type MockEventTagRepository struct {
	eventtagpb.UnimplementedEventTagDomainServiceServer
	businessType string
	eventTags    map[string]*eventtagpb.EventTag
	mutex        sync.RWMutex
	initialized  bool
}

// NewMockEventTagRepository creates a new mock event_tag repository
func NewMockEventTagRepository(businessType string) eventtagpb.EventTagDomainServiceServer {
	if businessType == "" {
		businessType = "education"
	}
	return &MockEventTagRepository{
		businessType: businessType,
		eventTags:    make(map[string]*eventtagpb.EventTag),
	}
}

// NewEventTagRepository creates a new mock event_tag repository - Provider interface compatibility
func NewEventTagRepository(businessType string) eventtagpb.EventTagDomainServiceServer {
	return NewMockEventTagRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock_db", entityid.EventTag, func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockEventTagRepository(businessType), nil
	})
}

// CreateEventTag creates a new event_tag with stateful storage
func (r *MockEventTagRepository) CreateEventTag(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("event_tag data is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if req.Data.Id == "" {
		req.Data.Id = fmt.Sprintf("event_tag_%d", time.Now().UnixNano())
	}

	now := time.Now()
	req.Data.DateCreated = &[]int64{now.Unix()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.Unix()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	r.eventTags[req.Data.Id] = req.Data

	return &eventtagpb.CreateEventTagResponse{
		Data:    []*eventtagpb.EventTag{req.Data},
		Success: true,
	}, nil
}

// ReadEventTag retrieves an event_tag by ID
func (r *MockEventTagRepository) ReadEventTag(ctx context.Context, req *eventtagpb.ReadEventTagRequest) (*eventtagpb.ReadEventTagResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	eventTag, exists := r.eventTags[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("event_tag not found: %s", req.Data.Id)
	}

	return &eventtagpb.ReadEventTagResponse{
		Data:    []*eventtagpb.EventTag{eventTag},
		Success: true,
	}, nil
}

// UpdateEventTag updates an existing event_tag
func (r *MockEventTagRepository) UpdateEventTag(ctx context.Context, req *eventtagpb.UpdateEventTagRequest) (*eventtagpb.UpdateEventTagResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, exists := r.eventTags[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("event_tag not found: %s", req.Data.Id)
	}

	now := time.Now()
	req.Data.DateCreated = existing.DateCreated
	req.Data.DateCreatedString = existing.DateCreatedString
	req.Data.DateModified = &[]int64{now.Unix()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	r.eventTags[req.Data.Id] = req.Data

	return &eventtagpb.UpdateEventTagResponse{
		Data:    []*eventtagpb.EventTag{req.Data},
		Success: true,
	}, nil
}

// DeleteEventTag deletes an event_tag
func (r *MockEventTagRepository) DeleteEventTag(ctx context.Context, req *eventtagpb.DeleteEventTagRequest) (*eventtagpb.DeleteEventTagResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.eventTags[req.Data.Id]; !exists {
		return nil, fmt.Errorf("event_tag not found: %s", req.Data.Id)
	}

	delete(r.eventTags, req.Data.Id)

	return &eventtagpb.DeleteEventTagResponse{
		Success: true,
	}, nil
}

// ListEventTags retrieves all event_tags
func (r *MockEventTagRepository) ListEventTags(ctx context.Context, req *eventtagpb.ListEventTagsRequest) (*eventtagpb.ListEventTagsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	eventTags := make([]*eventtagpb.EventTag, 0, len(r.eventTags))
	for _, eventTag := range r.eventTags {
		eventTags = append(eventTags, eventTag)
	}

	return &eventtagpb.ListEventTagsResponse{
		Data:    eventTags,
		Success: true,
	}, nil
}

// GetEventTagListPageData retrieves event_tags for the list page
func (r *MockEventTagRepository) GetEventTagListPageData(
	ctx context.Context,
	req *eventtagpb.GetEventTagListPageDataRequest,
) (*eventtagpb.GetEventTagListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event_tag list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	eventTags := make([]*eventtagpb.EventTag, 0, len(r.eventTags))
	for _, eventTag := range r.eventTags {
		eventTags = append(eventTags, eventTag)
	}

	return &eventtagpb.GetEventTagListPageDataResponse{
		EventTagList: eventTags,
		Success:      true,
	}, nil
}

// GetEventTagItemPageData retrieves a single event_tag item page
func (r *MockEventTagRepository) GetEventTagItemPageData(
	ctx context.Context,
	req *eventtagpb.GetEventTagItemPageDataRequest,
) (*eventtagpb.GetEventTagItemPageDataResponse, error) {
	if req == nil || req.EventTagId == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	eventTag, exists := r.eventTags[req.EventTagId]
	if !exists {
		return nil, fmt.Errorf("event_tag with ID '%s' not found", req.EventTagId)
	}

	return &eventtagpb.GetEventTagItemPageDataResponse{
		EventTag: eventTag,
		Success:  true,
	}, nil
}
