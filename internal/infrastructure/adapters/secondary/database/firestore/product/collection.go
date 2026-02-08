//go:build firestore

package product

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "collection", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore collection repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreCollectionRepository(dbOps, collectionName), nil
	})
}

// FirestoreCollectionRepository implements collection CRUD operations using Firestore
type FirestoreCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreCollectionRepository creates a new Firestore collection repository
func NewFirestoreCollectionRepository(dbOps interfaces.DatabaseOperation, collectionName string) collectionpb.CollectionDomainServiceServer {
	if collectionName == "" {
		collectionName = "collection" // default fallback
	}
	return &FirestoreCollectionRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateCollection creates a new collection using common Firestore operations
func (r *FirestoreCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	collection := &collectionpb.Collection{}
	convertedCollection, err := operations.ConvertMapToProtobuf(result, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionpb.CreateCollectionResponse{
		Data: []*collectionpb.Collection{convertedCollection},
	}, nil
}

// ReadCollection retrieves a collection using common Firestore operations
func (r *FirestoreCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	collection := &collectionpb.Collection{}
	convertedCollection, err := operations.ConvertMapToProtobuf(result, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionpb.ReadCollectionResponse{
		Data: []*collectionpb.Collection{convertedCollection},
	}, nil
}

// UpdateCollection updates a collection using common Firestore operations
func (r *FirestoreCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	collection := &collectionpb.Collection{}
	convertedCollection, err := operations.ConvertMapToProtobuf(result, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionpb.UpdateCollectionResponse{
		Data: []*collectionpb.Collection{convertedCollection},
	}, nil
}

// DeleteCollection deletes a collection using common Firestore operations
func (r *FirestoreCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &collectionpb.DeleteCollectionResponse{
		Success: true,
	}, nil
}

// ListCollections lists collections using common Firestore operations
func (r *FirestoreCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
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
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	collections, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *collectionpb.Collection {
		return &collectionpb.Collection{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if collections == nil {
		collections = make([]*collectionpb.Collection, 0)
	}

	return &collectionpb.ListCollectionsResponse{
		Data: collections,
	}, nil
}
