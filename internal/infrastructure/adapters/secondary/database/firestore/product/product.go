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

	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "product", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore product repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreProductRepository(dbOps, collectionName), nil
	})
}

// FirestoreProductRepository implements product CRUD operations using Firestore
type FirestoreProductRepository struct {
	productpb.UnimplementedProductDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreProductRepository creates a new Firestore product repository
func NewFirestoreProductRepository(dbOps interfaces.DatabaseOperation, collectionName string) productpb.ProductDomainServiceServer {
	if collectionName == "" {
		collectionName = "product" // default fallback
	}
	return &FirestoreProductRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateProduct creates a new product using common Firestore operations
func (r *FirestoreProductRepository) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	product := &productpb.Product{}
	convertedProduct, err := operations.ConvertMapToProtobuf(result, product)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productpb.CreateProductResponse{
		Data: []*productpb.Product{convertedProduct},
	}, nil
}

// ReadProduct retrieves a product using common Firestore operations
func (r *FirestoreProductRepository) ReadProduct(ctx context.Context, req *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	product := &productpb.Product{}
	convertedProduct, err := operations.ConvertMapToProtobuf(result, product)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productpb.ReadProductResponse{
		Data: []*productpb.Product{convertedProduct},
	}, nil
}

// UpdateProduct updates a product using common Firestore operations
func (r *FirestoreProductRepository) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	product := &productpb.Product{}
	convertedProduct, err := operations.ConvertMapToProtobuf(result, product)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productpb.UpdateProductResponse{
		Data: []*productpb.Product{convertedProduct},
	}, nil
}

// DeleteProduct deletes a product using common Firestore operations
func (r *FirestoreProductRepository) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}

	return &productpb.DeleteProductResponse{
		Success: true,
	}, nil
}

// ListProducts lists products using common Firestore operations
func (r *FirestoreProductRepository) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
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
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	products, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *productpb.Product {
		return &productpb.Product{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if products == nil {
		products = make([]*productpb.Product, 0)
	}

	return &productpb.ListProductsResponse{
		Data: products,
	}, nil
}
