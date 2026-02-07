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

	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "product_plan", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore product_plan repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreProductPlanRepository(dbOps, collectionName), nil
	})
}

// FirestoreProductPlanRepository implements product plan CRUD operations using Firestore
type FirestoreProductPlanRepository struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreProductPlanRepository creates a new Firestore product plan repository
func NewFirestoreProductPlanRepository(dbOps interfaces.DatabaseOperation, collectionName string) productplanpb.ProductPlanDomainServiceServer {
	if collectionName == "" {
		collectionName = "product_plan" // default fallback
	}
	return &FirestoreProductPlanRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateProductPlan creates a new product plan using common Firestore operations
func (r *FirestoreProductPlanRepository) CreateProductPlan(ctx context.Context, req *productplanpb.CreateProductPlanRequest) (*productplanpb.CreateProductPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product plan data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create product plan: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	productPlan := &productplanpb.ProductPlan{}
	convertedProductPlan, err := operations.ConvertMapToProtobuf(result, productPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productplanpb.CreateProductPlanResponse{
		Data: []*productplanpb.ProductPlan{convertedProductPlan},
	}, nil
}

// ReadProductPlan retrieves a product plan using common Firestore operations
func (r *FirestoreProductPlanRepository) ReadProductPlan(ctx context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product plan: %w", err)
	}

	// Convert result to protobuf using efficient ProtobufMapper
	productPlan := &productplanpb.ProductPlan{}
	convertedProductPlan, err := operations.ConvertMapToProtobuf(result, productPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productplanpb.ReadProductPlanResponse{
		Data: []*productplanpb.ProductPlan{convertedProductPlan},
	}, nil
}

// UpdateProductPlan updates a product plan using common Firestore operations
func (r *FirestoreProductPlanRepository) UpdateProductPlan(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) (*productplanpb.UpdateProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	// Convert protobuf to map using ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product plan: %w", err)
	}

	// Convert result back to protobuf using ProtobufMapper
	productPlan := &productplanpb.ProductPlan{}
	convertedProductPlan, err := operations.ConvertMapToProtobuf(result, productPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &productplanpb.UpdateProductPlanResponse{
		Data: []*productplanpb.ProductPlan{convertedProductPlan},
	}, nil
}

// DeleteProductPlan deletes a product plan using common Firestore operations
func (r *FirestoreProductPlanRepository) DeleteProductPlan(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) (*productplanpb.DeleteProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product plan: %w", err)
	}

	return &productplanpb.DeleteProductPlanResponse{
		Success: true,
	}, nil
}

// ListProductPlans lists product plans using common Firestore operations
func (r *FirestoreProductPlanRepository) ListProductPlans(ctx context.Context, req *productplanpb.ListProductPlansRequest) (*productplanpb.ListProductPlansResponse, error) {
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
		return nil, fmt.Errorf("failed to list product plans: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	productPlans, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *productplanpb.ProductPlan {
		return &productplanpb.ProductPlan{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if productPlans == nil {
		productPlans = make([]*productplanpb.ProductPlan, 0)
	}

	return &productplanpb.ListProductPlansResponse{
		Data: productPlans,
	}, nil
}
