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
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", entityid.EventTag, func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore event_tag repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreEventTagRepository(dbOps, collectionName), nil
	})
}

// FirestoreEventTagRepository implements event_tag CRUD operations using Firestore
type FirestoreEventTagRepository struct {
	eventtagpb.UnimplementedEventTagDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreEventTagRepository creates a new Firestore event_tag repository
func NewFirestoreEventTagRepository(dbOps interfaces.DatabaseOperation, collectionName string) eventtagpb.EventTagDomainServiceServer {
	if collectionName == "" {
		collectionName = "event_tag"
	}
	return &FirestoreEventTagRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateEventTag creates a new event_tag using common Firestore operations
func (r *FirestoreEventTagRepository) CreateEventTag(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("event_tag data is required")
	}

	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create event_tag: %w", err)
	}

	eventTag := &eventtagpb.EventTag{}
	converted, err := operations.ConvertMapToProtobuf(result, eventTag)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventtagpb.CreateEventTagResponse{
		Data: []*eventtagpb.EventTag{converted},
	}, nil
}

// ReadEventTag retrieves an event_tag using common Firestore operations
func (r *FirestoreEventTagRepository) ReadEventTag(ctx context.Context, req *eventtagpb.ReadEventTagRequest) (*eventtagpb.ReadEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read event_tag: %w", err)
	}

	eventTag := &eventtagpb.EventTag{}
	converted, err := operations.ConvertMapToProtobuf(result, eventTag)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventtagpb.ReadEventTagResponse{
		Data: []*eventtagpb.EventTag{converted},
	}, nil
}

// UpdateEventTag updates an event_tag using common Firestore operations
func (r *FirestoreEventTagRepository) UpdateEventTag(ctx context.Context, req *eventtagpb.UpdateEventTagRequest) (*eventtagpb.UpdateEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}

	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update event_tag: %w", err)
	}

	eventTag := &eventtagpb.EventTag{}
	converted, err := operations.ConvertMapToProtobuf(result, eventTag)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &eventtagpb.UpdateEventTagResponse{
		Data: []*eventtagpb.EventTag{converted},
	}, nil
}

// DeleteEventTag deletes an event_tag using common Firestore operations (soft delete)
func (r *FirestoreEventTagRepository) DeleteEventTag(ctx context.Context, req *eventtagpb.DeleteEventTagRequest) (*eventtagpb.DeleteEventTagResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("event_tag ID is required")
	}

	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete event_tag: %w", err)
	}

	return &eventtagpb.DeleteEventTagResponse{
		Success: true,
	}, nil
}

// ListEventTags lists event_tags using common Firestore operations
func (r *FirestoreEventTagRepository) ListEventTags(ctx context.Context, req *eventtagpb.ListEventTagsRequest) (*eventtagpb.ListEventTagsResponse, error) {
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list event_tags: %w", err)
	}

	eventTags, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *eventtagpb.EventTag {
		return &eventtagpb.EventTag{}
	})

	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	if eventTags == nil {
		eventTags = make([]*eventtagpb.EventTag, 0)
	}

	return &eventtagpb.ListEventTagsResponse{
		Data: eventTags,
	}, nil
}
