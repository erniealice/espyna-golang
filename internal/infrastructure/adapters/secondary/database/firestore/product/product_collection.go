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

	productcollectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_collection"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "product_collection", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore product_collection repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreProductCollectionRepository(dbOps, collectionName), nil
	})
}

// FirestoreProductCollectionRepository implements product collection CRUD operations using Firestore
type FirestoreProductCollectionRepository struct {
	productcollectionpb.UnimplementedProductCollectionDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreProductCollectionRepository creates a new Firestore product collection repository
func NewFirestoreProductCollectionRepository(dbOps interfaces.DatabaseOperation, collectionName string) productcollectionpb.ProductCollectionDomainServiceServer {
	if collectionName == "" {
		collectionName = "product_collection" // default fallback
	}
	return &FirestoreProductCollectionRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateProductCollection creates a new product collection using common Firestore operations
func (r *FirestoreProductCollectionRepository) CreateProductCollection(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) (*productcollectionpb.CreateProductCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product collection data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create product collection: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	productCollection := &productcollectionpb.ProductCollection{}
	convertedProductCollection, err := operations.ConvertMapToProtobuf(result, productCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productcollectionpb.CreateProductCollectionResponse{
		Data: []*productcollectionpb.ProductCollection{convertedProductCollection},
	}, nil
}

// ReadProductCollection retrieves a product collection using common Firestore operations
func (r *FirestoreProductCollectionRepository) ReadProductCollection(ctx context.Context, req *productcollectionpb.ReadProductCollectionRequest) (*productcollectionpb.ReadProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product collection: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	productCollection := &productcollectionpb.ProductCollection{}
	convertedProductCollection, err := operations.ConvertMapToProtobuf(result, productCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productcollectionpb.ReadProductCollectionResponse{
		Data: []*productcollectionpb.ProductCollection{convertedProductCollection},
	}, nil
}

// UpdateProductCollection updates a product collection using common Firestore operations
func (r *FirestoreProductCollectionRepository) UpdateProductCollection(ctx context.Context, req *productcollectionpb.UpdateProductCollectionRequest) (*productcollectionpb.UpdateProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product collection: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	productCollection := &productcollectionpb.ProductCollection{}
	convertedProductCollection, err := operations.ConvertMapToProtobuf(result, productCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productcollectionpb.UpdateProductCollectionResponse{
		Data: []*productcollectionpb.ProductCollection{convertedProductCollection},
	}, nil
}

// DeleteProductCollection deletes a product collection using common Firestore operations
func (r *FirestoreProductCollectionRepository) DeleteProductCollection(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) (*productcollectionpb.DeleteProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product collection: %w", err)
	}

	return &productcollectionpb.DeleteProductCollectionResponse{
		Success: true,
	}, nil
}

// ListProductCollections lists product collections using common Firestore operations
func (r *FirestoreProductCollectionRepository) ListProductCollections(ctx context.Context, req *productcollectionpb.ListProductCollectionsRequest) (*productcollectionpb.ListProductCollectionsResponse, error) {
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
		return nil, fmt.Errorf("failed to list product collections: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	productCollections, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *productcollectionpb.ProductCollection {
		return &productcollectionpb.ProductCollection{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if productCollections == nil {
		productCollections = make([]*productcollectionpb.ProductCollection, 0)
	}

	return &productcollectionpb.ListProductCollectionsResponse{
		Data: productCollections,
	}, nil
}
