//go:build firestore

package event

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "event_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore event_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreEventAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreEventAttributeRepository implements event attribute CRUD operations using Firestore
type FirestoreEventAttributeRepository struct {
	eventattributepb.UnimplementedEventAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreEventAttributeRepository creates a new Firestore event attribute repository
func NewFirestoreEventAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) eventattributepb.EventAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "event_attribute" // default fallback
	}
	return &FirestoreEventAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateEventAttribute creates a new event attribute using common Firestore operations
func (r *FirestoreEventAttributeRepository) CreateEventAttribute(ctx context.Context, req *eventattributepb.CreateEventAttributeRequest) (*eventattributepb.CreateEventAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create event attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	eventAttribute := &eventattributepb.EventAttribute{}
	convertedEventAttribute, err := operations.ConvertMapToProtobuf(result, eventAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventattributepb.CreateEventAttributeResponse{
		Data: []*eventattributepb.EventAttribute{convertedEventAttribute},
	}, nil
}

// ReadEventAttribute retrieves a event attribute using common Firestore operations
func (r *FirestoreEventAttributeRepository) ReadEventAttribute(ctx context.Context, req *eventattributepb.ReadEventAttributeRequest) (*eventattributepb.ReadEventAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	eventAttribute := &eventattributepb.EventAttribute{}
	convertedEventAttribute, err := operations.ConvertMapToProtobuf(result, eventAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventattributepb.ReadEventAttributeResponse{
		Data: []*eventattributepb.EventAttribute{convertedEventAttribute},
	}, nil
}

// UpdateEventAttribute updates a event attribute using common Firestore operations
func (r *FirestoreEventAttributeRepository) UpdateEventAttribute(ctx context.Context, req *eventattributepb.UpdateEventAttributeRequest) (*eventattributepb.UpdateEventAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update event attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	eventAttribute := &eventattributepb.EventAttribute{}
	convertedEventAttribute, err := operations.ConvertMapToProtobuf(result, eventAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventattributepb.UpdateEventAttributeResponse{
		Data: []*eventattributepb.EventAttribute{convertedEventAttribute},
	}, nil
}

// DeleteEventAttribute deletes a event attribute using common Firestore operations
func (r *FirestoreEventAttributeRepository) DeleteEventAttribute(ctx context.Context, req *eventattributepb.DeleteEventAttributeRequest) (*eventattributepb.DeleteEventAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event attribute: %w", err)
	}

	return &eventattributepb.DeleteEventAttributeResponse{
		Success: true,
	}, nil
}

// ListEventAttributes lists event attributes using common Firestore operations
func (r *FirestoreEventAttributeRepository) ListEventAttributes(ctx context.Context, req *eventattributepb.ListEventAttributesRequest) (*eventattributepb.ListEventAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list event attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	eventAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *eventattributepb.EventAttribute {
		return &eventattributepb.EventAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if eventAttributes == nil {
		eventAttributes = make([]*eventattributepb.EventAttribute, 0)
	}

	return &eventattributepb.ListEventAttributesResponse{
		Data: eventAttributes,
	}, nil
}