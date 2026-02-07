//go:build firestore

package event

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "event", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore event repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreEventRepository(dbOps, collectionName), nil
	})
}

// FirestoreEventRepository implements event CRUD operations using Firestore
type FirestoreEventRepository struct {
	eventpb.UnimplementedEventDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreEventRepository creates a new Firestore event repository
func NewFirestoreEventRepository(dbOps interfaces.DatabaseOperation, collectionName string) eventpb.EventDomainServiceServer {
	if collectionName == "" {
		collectionName = "event" // default fallback
	}
	return &FirestoreEventRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateEvent creates a new event using common Firestore operations
func (r *FirestoreEventRepository) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	event := &eventpb.Event{}
	convertedEvent, err := operations.ConvertMapToProtobuf(result, event)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventpb.CreateEventResponse{
		Data: []*eventpb.Event{convertedEvent},
	}, nil
}

// ReadEvent retrieves an event using common Firestore operations
func (r *FirestoreEventRepository) ReadEvent(ctx context.Context, req *eventpb.ReadEventRequest) (*eventpb.ReadEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	event := &eventpb.Event{}
	convertedEvent, err := operations.ConvertMapToProtobuf(result, event)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventpb.ReadEventResponse{
		Data: []*eventpb.Event{convertedEvent},
	}, nil
}

// UpdateEvent updates an event using common Firestore operations
func (r *FirestoreEventRepository) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.UpdateEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	event := &eventpb.Event{}
	convertedEvent, err := operations.ConvertMapToProtobuf(result, event)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventpb.UpdateEventResponse{
		Data: []*eventpb.Event{convertedEvent},
	}, nil
}

// DeleteEvent deletes an event using common Firestore operations
func (r *FirestoreEventRepository) DeleteEvent(ctx context.Context, req *eventpb.DeleteEventRequest) (*eventpb.DeleteEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event: %w", err)
	}

	return &eventpb.DeleteEventResponse{
		Success: true,
	}, nil
}

// ListEvents lists events using common Firestore operations
func (r *FirestoreEventRepository) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	events, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *eventpb.Event {
		return &eventpb.Event{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if events == nil {
		events = make([]*eventpb.Event, 0)
	}

	return &eventpb.ListEventsResponse{
		Data: events,
	}, nil
}
