//go:build mock_db

package event

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// MockEventRepository implements event.EventRepository using stateful mock data
type MockEventRepository struct {
	eventpb.UnimplementedEventDomainServiceServer
	businessType string
	events       map[string]*eventpb.Event   // Persistent in-memory store
	mutex        sync.RWMutex                // Thread-safe concurrent access
	initialized  bool                        // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing utilities
}

// RepositoryOption allows configuration of repository behavior
type RepositoryOption func(*MockEventRepository)

// WithTestOptimizations enables test-specific optimizations
func WithTestOptimizations(enabled bool) RepositoryOption {
	return func(r *MockEventRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockEventRepository creates a new mock event repository
func NewMockEventRepository(businessType string, options ...RepositoryOption) eventpb.EventDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockEventRepository{
		businessType: businessType,
		events:       make(map[string]*eventpb.Event),
		processor:    listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if err := repo.loadInitialData(); err != nil {
		// Log error but don't fail - allows graceful degradation
		fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockEventRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawEvents, err := datamock.LoadEvents(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial events: %w", err)
	}

	// Convert and store each event
	for _, rawEvent := range rawEvents {
		if event, err := r.mapToProtobufEvent(rawEvent); err == nil {
			r.events[event.Id] = event
		}
	}

	r.initialized = true
	return nil
}

// CreateEvent creates a new event with stateful storage
func (r *MockEventRepository) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create event request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("event data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("event name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	eventID := fmt.Sprintf("event-%d-%d", now.UnixNano(), len(r.events))

	// Create new event with proper timestamps and defaults
	newEvent := &eventpb.Event{
		Id:                 eventID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        &[]int64{now.Unix()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.Unix()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
		StartDateTimeUtc:   req.Data.StartDateTimeUtc,
		EndDateTimeUtc:     req.Data.EndDateTimeUtc,
		Timezone:           req.Data.Timezone,
	}

	// Store in persistent map
	r.events[eventID] = newEvent

	return &eventpb.CreateEventResponse{
		Data:    []*eventpb.Event{newEvent},
		Success: true,
	}, nil
}

// ReadEvent retrieves an event by ID from stateful storage
func (r *MockEventRepository) ReadEvent(ctx context.Context, req *eventpb.ReadEventRequest) (*eventpb.ReadEventResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read event request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated events)
	if event, exists := r.events[req.Data.Id]; exists {
		return &eventpb.ReadEventResponse{
			Data:    []*eventpb.Event{event},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("event with ID '%s' not found", req.Data.Id)
}

// UpdateEvent updates an existing event in stateful storage
func (r *MockEventRepository) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.UpdateEventResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update event request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify event exists
	existingEvent, exists := r.events[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("event with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedEvent := &eventpb.Event{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        existingEvent.DateCreated,       // Preserve original
		DateCreatedString:  existingEvent.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.Unix()}[0],         // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
		StartDateTimeUtc:   req.Data.StartDateTimeUtc,
		EndDateTimeUtc:     req.Data.EndDateTimeUtc,
		Timezone:           req.Data.Timezone,
	}

	// Update in persistent store
	r.events[req.Data.Id] = updatedEvent

	return &eventpb.UpdateEventResponse{
		Data:    []*eventpb.Event{updatedEvent},
		Success: true,
	}, nil
}

// DeleteEvent deletes an event from stateful storage
func (r *MockEventRepository) DeleteEvent(ctx context.Context, req *eventpb.DeleteEventRequest) (*eventpb.DeleteEventResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete event request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify event exists before deletion
	if _, exists := r.events[req.Data.Id]; !exists {
		return nil, fmt.Errorf("event with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.events, req.Data.Id)

	return &eventpb.DeleteEventResponse{
		Success: true,
	}, nil
}

// ListEvents retrieves all events from stateful storage
func (r *MockEventRepository) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of events
	events := make([]*eventpb.Event, 0, len(r.events))
	for _, event := range r.events {
		events = append(events, event)
	}

	return &eventpb.ListEventsResponse{
		Data:    events,
		Success: true,
	}, nil
}

// mapToProtobufEvent converts raw mock data to protobuf Event
func (r *MockEventRepository) mapToProtobufEvent(rawEvent map[string]any) (*eventpb.Event, error) {
	event := &eventpb.Event{}

	// Map required fields
	if id, ok := rawEvent["id"].(string); ok {
		event.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawEvent["name"].(string); ok {
		event.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if description, ok := rawEvent["description"].(string); ok {
		event.Description = &description
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawEvent["dateCreated"].(string); ok {
		event.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			event.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawEvent["dateModified"].(string); ok {
		event.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			event.DateModified = &timestamp
		}
	}

	if startDateTime, ok := rawEvent["startDateTimeUtc"].(string); ok {
		if timestamp, err := r.parseTimestamp(startDateTime); err == nil {
			event.StartDateTimeUtc = timestamp
		}
	}

	if endDateTime, ok := rawEvent["endDateTimeUtc"].(string); ok {
		if timestamp, err := r.parseTimestamp(endDateTime); err == nil {
			event.EndDateTimeUtc = timestamp
		}
	}

	// Map optional string timestamp fields for protobuf compatibility
	if startDateTimeUtcString, ok := rawEvent["startDateTimeUtcString"].(string); ok {
		event.StartDateTimeUtcString = &startDateTimeUtcString
	}

	if endDateTimeUtcString, ok := rawEvent["endDateTimeUtcString"].(string); ok {
		event.EndDateTimeUtcString = &endDateTimeUtcString
	}

	if timezone, ok := rawEvent["timezone"].(string); ok {
		event.Timezone = timezone
	}

	if active, ok := rawEvent["active"].(bool); ok {
		event.Active = active
	}

	return event, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockEventRepository) parseTimestamp(timestampStr string) (int64, error) {
	// Try parsing as Unix timestamp first
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	// Try parsing as RFC3339 format
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.Unix(), nil
	}

	// Try parsing as other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.Unix(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// GetEventListPageData retrieves events with advanced filtering, sorting, searching, and pagination
func (r *MockEventRepository) GetEventListPageData(
	ctx context.Context,
	req *eventpb.GetEventListPageDataRequest,
) (*eventpb.GetEventListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of events
	events := make([]*eventpb.Event, 0, len(r.events))
	for _, event := range r.events {
		events = append(events, event)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		events,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process event list data: %w", err)
	}

	// Convert processed items back to event protobuf format
	processedEvents := make([]*eventpb.Event, len(result.Items))
	for i, item := range result.Items {
		if event, ok := item.(*eventpb.Event); ok {
			processedEvents[i] = event
		} else {
			return nil, fmt.Errorf("failed to convert item to event type")
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &eventpb.GetEventListPageDataResponse{
		EventList:     processedEvents,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetEventItemPageData retrieves a single event with enhanced item page data
func (r *MockEventRepository) GetEventItemPageData(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) (*eventpb.GetEventItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get event item page data request is required")
	}
	if req.EventId == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	event, exists := r.events[req.EventId]
	if !exists {
		return nil, fmt.Errorf("event with ID '%s' not found", req.EventId)
	}

	// In a real implementation, you might:
	// 1. Load related data (client relationships, venue details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption with timezone handling
	// 4. Add audit logging

	return &eventpb.GetEventItemPageDataResponse{
		Event:   event,
		Success: true,
	}, nil
}

// NewEventRepository creates a new event repository - Provider interface compatibility
func NewEventRepository(businessType string) eventpb.EventDomainServiceServer {
	return NewMockEventRepository(businessType)
}

// NewEventClientRepository creates a new event client repository - Provider interface compatibility
func NewEventClientRepository(businessType string) eventclientpb.EventClientDomainServiceServer {
	repo := &MockEventClientRepository{
		businessType: businessType,
		eventClients: make(map[string]*eventclientpb.EventClient),
		initialized:  false,
	}

	// Initialize with mock data
	if err := repo.initializeMockData(); err != nil {
		// Log error but return functioning repository
		fmt.Printf("⚠️ Failed to load initial event client data: %v\n", err)
	}

	return repo
}

// MockEventClientRepository implements event client operations using stateful mock data
type MockEventClientRepository struct {
	eventclientpb.UnimplementedEventClientDomainServiceServer
	businessType string
	eventClients map[string]*eventclientpb.EventClient
	mutex        sync.RWMutex
	initialized  bool
}

// initializeMockData loads initial event client data from copya package
func (r *MockEventClientRepository) initializeMockData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil
	}

	// Load event clients from copya using business type module
	rawEventClients, err := datamock.LoadBusinessTypeModule(r.businessType, "event-client")
	if err != nil {
		return fmt.Errorf("failed to load initial event clients: %w", err)
	}

	// Convert and store each event client
	for _, rawEventClient := range rawEventClients {
		if eventClient, err := r.mapToProtobufEventClient(rawEventClient); err == nil {
			r.eventClients[eventClient.Id] = eventClient
		}
	}

	r.initialized = true
	return nil
}

// mapToProtobufEventClient converts raw mock data to protobuf EventClient
func (r *MockEventClientRepository) mapToProtobufEventClient(rawEventClient map[string]any) (*eventclientpb.EventClient, error) {
	eventClient := &eventclientpb.EventClient{}

	// Map required fields
	if id, ok := rawEventClient["id"].(string); ok {
		eventClient.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if eventId, ok := rawEventClient["eventId"].(string); ok {
		eventClient.EventId = eventId
	} else {
		return nil, fmt.Errorf("missing or invalid eventId field")
	}

	if clientId, ok := rawEventClient["clientId"].(string); ok {
		eventClient.ClientId = clientId
	} else {
		return nil, fmt.Errorf("missing or invalid clientId field")
	}

	// Map optional fields
	if active, ok := rawEventClient["active"].(bool); ok {
		eventClient.Active = active
	}

	// Map timestamp fields
	if dateCreated, ok := rawEventClient["dateCreated"].(string); ok {
		if timestamp, err := strconv.ParseInt(dateCreated, 10, 64); err == nil {
			eventClient.DateCreated = &timestamp
		}
	}

	if dateCreatedString, ok := rawEventClient["dateCreatedString"].(string); ok {
		eventClient.DateCreatedString = &dateCreatedString
	}

	if dateModified, ok := rawEventClient["dateModified"].(string); ok {
		if timestamp, err := strconv.ParseInt(dateModified, 10, 64); err == nil {
			eventClient.DateModified = &timestamp
		}
	}

	if dateModifiedString, ok := rawEventClient["dateModifiedString"].(string); ok {
		eventClient.DateModifiedString = &dateModifiedString
	}

	return eventClient, nil
}

// Implement basic CRUD operations for EventClient
func (r *MockEventClientRepository) CreateEventClient(ctx context.Context, req *eventclientpb.CreateEventClientRequest) (*eventclientpb.CreateEventClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event client data is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate ID if not provided
	if req.Data.Id == "" {
		req.Data.Id = fmt.Sprintf("event_client_%d", time.Now().UnixNano())
	}

	// Set default active status if not specified
	if !req.Data.Active {
		req.Data.Active = true
	}

	// Set timestamps (protobuf fields are pointers)
	now := time.Now()
	req.Data.DateCreated = &[]int64{now.Unix()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]

	// Store the event client
	r.eventClients[req.Data.Id] = req.Data

	return &eventclientpb.CreateEventClientResponse{
		Data:    []*eventclientpb.EventClient{req.Data},
		Success: true,
	}, nil
}

func (r *MockEventClientRepository) ReadEventClient(ctx context.Context, req *eventclientpb.ReadEventClientRequest) (*eventclientpb.ReadEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event client ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	eventClient, exists := r.eventClients[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("event client not found: %s", req.Data.Id)
	}

	return &eventclientpb.ReadEventClientResponse{
		Data:    []*eventclientpb.EventClient{eventClient},
		Success: true,
	}, nil
}

func (r *MockEventClientRepository) UpdateEventClient(ctx context.Context, req *eventclientpb.UpdateEventClientRequest) (*eventclientpb.UpdateEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event client ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.eventClients[req.Data.Id]; !exists {
		return nil, fmt.Errorf("event client not found: %s", req.Data.Id)
	}

	// Update timestamps (protobuf fields are pointers)
	now := time.Now()
	req.Data.DateModified = &[]int64{now.Unix()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	r.eventClients[req.Data.Id] = req.Data

	return &eventclientpb.UpdateEventClientResponse{
		Data:    []*eventclientpb.EventClient{req.Data},
		Success: true,
	}, nil
}

func (r *MockEventClientRepository) DeleteEventClient(ctx context.Context, req *eventclientpb.DeleteEventClientRequest) (*eventclientpb.DeleteEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event client ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.eventClients[req.Data.Id]; !exists {
		return nil, fmt.Errorf("event client not found: %s", req.Data.Id)
	}

	delete(r.eventClients, req.Data.Id)

	return &eventclientpb.DeleteEventClientResponse{
		Success: true,
	}, nil
}

func (r *MockEventClientRepository) ListEventClients(ctx context.Context, req *eventclientpb.ListEventClientsRequest) (*eventclientpb.ListEventClientsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var eventClients []*eventclientpb.EventClient
	for _, eventClient := range r.eventClients {
		eventClients = append(eventClients, eventClient)
	}

	return &eventclientpb.ListEventClientsResponse{
		Data:    eventClients,
		Success: true,
	}, nil
}

func init() {
	registry.RegisterRepositoryFactory("mock", "event", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockEventRepository(businessType), nil
	})
}
