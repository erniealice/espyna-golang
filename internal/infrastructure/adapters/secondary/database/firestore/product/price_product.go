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

	priceproductpb "leapfor.xyz/esqyma/golang/v1/domain/product/price_product"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "price_product", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore price_product repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestorePriceProductRepository(dbOps, collectionName), nil
	})
}

// FirestorePriceProductRepository implements price product CRUD operations using Firestore
type FirestorePriceProductRepository struct {
	priceproductpb.UnimplementedPriceProductDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestorePriceProductRepository creates a new Firestore price product repository
func NewFirestorePriceProductRepository(dbOps interfaces.DatabaseOperation, collectionName string) priceproductpb.PriceProductDomainServiceServer {
	if collectionName == "" {
		collectionName = "price_product" // default fallback
	}
	return &FirestorePriceProductRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreatePriceProduct creates a new price product using common Firestore operations
func (r *FirestorePriceProductRepository) CreatePriceProduct(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) (*priceproductpb.CreatePriceProductResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price product data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price product: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	priceProduct := &priceproductpb.PriceProduct{}
	convertedPriceProduct, err := operations.ConvertMapToProtobuf(result, priceProduct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceproductpb.CreatePriceProductResponse{
		Data: []*priceproductpb.PriceProduct{convertedPriceProduct},
	}, nil
}

// ReadPriceProduct retrieves a price product using common Firestore operations
func (r *FirestorePriceProductRepository) ReadPriceProduct(ctx context.Context, req *priceproductpb.ReadPriceProductRequest) (*priceproductpb.ReadPriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price product: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	priceProduct := &priceproductpb.PriceProduct{}
	convertedPriceProduct, err := operations.ConvertMapToProtobuf(result, priceProduct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceproductpb.ReadPriceProductResponse{
		Data: []*priceproductpb.PriceProduct{convertedPriceProduct},
	}, nil
}

// UpdatePriceProduct updates a price product using common Firestore operations
func (r *FirestorePriceProductRepository) UpdatePriceProduct(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) (*priceproductpb.UpdatePriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price product: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	priceProduct := &priceproductpb.PriceProduct{}
	convertedPriceProduct, err := operations.ConvertMapToProtobuf(result, priceProduct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &priceproductpb.UpdatePriceProductResponse{
		Data: []*priceproductpb.PriceProduct{convertedPriceProduct},
	}, nil
}

// DeletePriceProduct deletes a price product using common Firestore operations
func (r *FirestorePriceProductRepository) DeletePriceProduct(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) (*priceproductpb.DeletePriceProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete price product: %w", err)
	}

	return &priceproductpb.DeletePriceProductResponse{
		Success: true,
	}, nil
}

// ListPriceProducts lists price products using common Firestore operations
func (r *FirestorePriceProductRepository) ListPriceProducts(ctx context.Context, req *priceproductpb.ListPriceProductsRequest) (*priceproductpb.ListPriceProductsResponse, error) {
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
		return nil, fmt.Errorf("failed to list price products: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	priceProducts, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *priceproductpb.PriceProduct {
		return &priceproductpb.PriceProduct{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if priceProducts == nil {
		priceProducts = make([]*priceproductpb.PriceProduct, 0)
	}

	return &priceproductpb.ListPriceProductsResponse{
		Data: priceProducts,
	}, nil
}
