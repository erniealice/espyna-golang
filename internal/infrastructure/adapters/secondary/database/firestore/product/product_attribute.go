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

	productattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/product_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "product_attribute", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore product_attribute repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreProductAttributeRepository(dbOps, collectionName), nil
	})
}

// FirestoreProductAttributeRepository implements product attribute CRUD operations using Firestore
type FirestoreProductAttributeRepository struct {
	productattributepb.UnimplementedProductAttributeDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreProductAttributeRepository creates a new Firestore product attribute repository
func NewFirestoreProductAttributeRepository(dbOps interfaces.DatabaseOperation, collectionName string) productattributepb.ProductAttributeDomainServiceServer {
	if collectionName == "" {
		collectionName = "product_attribute" // default fallback
	}
	return &FirestoreProductAttributeRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateProductAttribute creates a new product attribute using common Firestore operations
func (r *FirestoreProductAttributeRepository) CreateProductAttribute(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) (*productattributepb.CreateProductAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product attribute data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create product attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	productAttribute := &productattributepb.ProductAttribute{}
	convertedProductAttribute, err := operations.ConvertMapToProtobuf(result, productAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productattributepb.CreateProductAttributeResponse{
		Data: []*productattributepb.ProductAttribute{convertedProductAttribute},
	}, nil
}

// ReadProductAttribute retrieves a product attribute using common Firestore operations
func (r *FirestoreProductAttributeRepository) ReadProductAttribute(ctx context.Context, req *productattributepb.ReadProductAttributeRequest) (*productattributepb.ReadProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	// Use the proper primary ID from the protobuf model
	// This follows Firestore best practices using document IDs
	docID := req.Data.Id

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to read product attribute: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	productAttribute := &productattributepb.ProductAttribute{}
	convertedProductAttribute, err := operations.ConvertMapToProtobuf(result, productAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productattributepb.ReadProductAttributeResponse{
		Data: []*productattributepb.ProductAttribute{convertedProductAttribute},
	}, nil
}

// UpdateProductAttribute updates a product attribute using common Firestore operations
func (r *FirestoreProductAttributeRepository) UpdateProductAttribute(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) (*productattributepb.UpdateProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Use the proper primary ID from the protobuf model
	docID := req.Data.Id

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, docID, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product attribute: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	productAttribute := &productattributepb.ProductAttribute{}
	convertedProductAttribute, err := operations.ConvertMapToProtobuf(result, productAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productattributepb.UpdateProductAttributeResponse{
		Data: []*productattributepb.ProductAttribute{convertedProductAttribute},
	}, nil
}

// DeleteProductAttribute deletes a product attribute using common Firestore operations
func (r *FirestoreProductAttributeRepository) DeleteProductAttribute(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) (*productattributepb.DeleteProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	// Use the proper primary ID from the protobuf model
	docID := req.Data.Id

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product attribute: %w", err)
	}

	return &productattributepb.DeleteProductAttributeResponse{
		Success: true,
	}, nil
}

// ListProductAttributes lists product attributes using common Firestore operations
func (r *FirestoreProductAttributeRepository) ListProductAttributes(ctx context.Context, req *productattributepb.ListProductAttributesRequest) (*productattributepb.ListProductAttributesResponse, error) {
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
		return nil, fmt.Errorf("failed to list product attributes: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	productAttributes, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *productattributepb.ProductAttribute {
		return &productattributepb.ProductAttribute{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if productAttributes == nil {
		productAttributes = make([]*productattributepb.ProductAttribute, 0)
	}

	return &productattributepb.ListProductAttributesResponse{
		Data: productAttributes,
	}, nil
}
