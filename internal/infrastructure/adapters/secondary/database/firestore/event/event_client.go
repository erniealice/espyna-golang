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
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "event_client", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore event_client repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreEventClientRepository(dbOps, collectionName), nil
	})
}

// FirestoreEventClientRepository implements event_client CRUD operations using Firestore
type FirestoreEventClientRepository struct {
	eventclientpb.UnimplementedEventClientDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreEventClientRepository creates a new Firestore event_client repository
func NewFirestoreEventClientRepository(dbOps interfaces.DatabaseOperation, collectionName string) eventclientpb.EventClientDomainServiceServer {
	if collectionName == "" {
		collectionName = "event_client" // default fallback
	}
	return &FirestoreEventClientRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateEventClient creates a new event_client using common Firestore operations
func (r *FirestoreEventClientRepository) CreateEventClient(ctx context.Context, req *eventclientpb.CreateEventClientRequest) (*eventclientpb.CreateEventClientResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_client data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create event_client: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	eventClient := &eventclientpb.EventClient{}
	convertedEventClient, err := operations.ConvertMapToProtobuf(result, eventClient)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventclientpb.CreateEventClientResponse{
		Data: []*eventclientpb.EventClient{convertedEventClient},
	}, nil
}

// ReadEventClient retrieves an event_client using common Firestore operations
func (r *FirestoreEventClientRepository) ReadEventClient(ctx context.Context, req *eventclientpb.ReadEventClientRequest) (*eventclientpb.ReadEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_client ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_client: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	eventClient := &eventclientpb.EventClient{}
	convertedEventClient, err := operations.ConvertMapToProtobuf(result, eventClient)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventclientpb.ReadEventClientResponse{
		Data: []*eventclientpb.EventClient{convertedEventClient},
	}, nil
}

// UpdateEventClient updates an event_client using common Firestore operations
func (r *FirestoreEventClientRepository) UpdateEventClient(ctx context.Context, req *eventclientpb.UpdateEventClientRequest) (*eventclientpb.UpdateEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_client ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update event_client: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	eventClient := &eventclientpb.EventClient{}
	convertedEventClient, err := operations.ConvertMapToProtobuf(result, eventClient)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventclientpb.UpdateEventClientResponse{
		Data: []*eventclientpb.EventClient{convertedEventClient},
	}, nil
}

// DeleteEventClient deletes an event_client using common Firestore operations
func (r *FirestoreEventClientRepository) DeleteEventClient(ctx context.Context, req *eventclientpb.DeleteEventClientRequest) (*eventclientpb.DeleteEventClientResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_client ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event_client: %w", err)
	}

	return &eventclientpb.DeleteEventClientResponse{
		Success: true,
	}, nil
}

// ListEventClients lists event_clients using common Firestore operations
func (r *FirestoreEventClientRepository) ListEventClients(ctx context.Context, req *eventclientpb.ListEventClientsRequest) (*eventclientpb.ListEventClientsResponse, error) {
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
		return nil, fmt.Errorf("failed to list event_clients: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	eventClients, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *eventclientpb.EventClient {
		return &eventclientpb.EventClient{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if eventClients == nil {
		eventClients = make([]*eventclientpb.EventClient, 0)
	}

	return &eventclientpb.ListEventClientsResponse{
		Data: eventClients,
	}, nil
}
