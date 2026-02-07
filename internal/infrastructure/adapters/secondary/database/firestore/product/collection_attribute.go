//go:build firestore

package product

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	collectionattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "collection_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore collection_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreCollectionAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreCollectionAttributeRepository implements collection attribute CRUD operations using Firestore
type FirestoreCollectionAttributeRepository struct {
	collectionattributepb.UnimplementedCollectionAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreCollectionAttributeRepository creates a new Firestore collection attribute repository
func NewFirestoreCollectionAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) collectionattributepb.CollectionAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "collection_attribute" // default fallback
	}
	return &FirestoreCollectionAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateCollectionAttribute creates a new collection attribute using common Firestore operations
func (r *FirestoreCollectionAttributeRepository) CreateCollectionAttribute(ctx context.Context, req *collectionattributepb.CreateCollectionAttributeRequest) (*collectionattributepb.CreateCollectionAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	collectionAttribute := &collectionattributepb.CollectionAttribute{}
	convertedCollectionAttribute, err := operations.ConvertMapToProtobuf(result, collectionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionattributepb.CreateCollectionAttributeResponse{
		Data: []*collectionattributepb.CollectionAttribute{convertedCollectionAttribute},
	}, nil
}

// ReadCollectionAttribute retrieves a collection attribute using common Firestore operations
func (r *FirestoreCollectionAttributeRepository) ReadCollectionAttribute(ctx context.Context, req *collectionattributepb.ReadCollectionAttributeRequest) (*collectionattributepb.ReadCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
	}

	// Read document using common operations with the ID
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	collectionAttribute := &collectionattributepb.CollectionAttribute{}
	convertedCollectionAttribute, err := operations.ConvertMapToProtobuf(result, collectionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionattributepb.ReadCollectionAttributeResponse{
		Data: []*collectionattributepb.CollectionAttribute{convertedCollectionAttribute},
	}, nil
}

// UpdateCollectionAttribute updates a collection attribute using common Firestore operations
func (r *FirestoreCollectionAttributeRepository) UpdateCollectionAttribute(ctx context.Context, req *collectionattributepb.UpdateCollectionAttributeRequest) (*collectionattributepb.UpdateCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	collectionAttribute := &collectionattributepb.CollectionAttribute{}
	convertedCollectionAttribute, err := operations.ConvertMapToProtobuf(result, collectionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &collectionattributepb.UpdateCollectionAttributeResponse{
		Data: []*collectionattributepb.CollectionAttribute{convertedCollectionAttribute},
	}, nil
}

// DeleteCollectionAttribute deletes a collection attribute using common Firestore operations
func (r *FirestoreCollectionAttributeRepository) DeleteCollectionAttribute(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) (*collectionattributepb.DeleteCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection attribute: %w", err)
	}

	return &collectionattributepb.DeleteCollectionAttributeResponse{
		Success: true,
	}, nil
}

// ListCollectionAttributes lists collection attributes using common Firestore operations
func (r *FirestoreCollectionAttributeRepository) ListCollectionAttributes(ctx context.Context, req *collectionattributepb.ListCollectionAttributesRequest) (*collectionattributepb.ListCollectionAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list collection attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	collectionAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *collectionattributepb.CollectionAttribute {
		return &collectionattributepb.CollectionAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if collectionAttributes == nil {
		collectionAttributes = make([]*collectionattributepb.CollectionAttribute, 0)
	}

	return &collectionattributepb.ListCollectionAttributesResponse{
		Data: collectionAttributes,
	}, nil
}