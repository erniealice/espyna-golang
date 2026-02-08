//go:build firestore

package common

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "category", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore category repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreCategoryRepository(dbOps, collectionName), nil
	})
}

// FirestoreCategoryRepository implements category CRUD operations using Firestore
type FirestoreCategoryRepository struct {
	commonpb.UnimplementedCategoryDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreCategoryRepository creates a new Firestore category repository
func NewFirestoreCategoryRepository(dbOps interfaces.DatabaseOperation, collectionName string) commonpb.CategoryDomainServiceServer {
	if collectionName == "" {
		collectionName = "category" // default fallback
	}
	return &FirestoreCategoryRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateCategory creates a new category using common Firestore operations
func (r *FirestoreCategoryRepository) CreateCategory(ctx context.Context, req *commonpb.CreateCategoryRequest) (*commonpb.CreateCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("category data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	category := &commonpb.Category{}
	convertedCategory, err := operations.ConvertMapToProtobuf(result, category)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &commonpb.CreateCategoryResponse{
		Data:    []*commonpb.Category{convertedCategory},
		Success: true,
	}, nil
}

// ReadCategory retrieves a category using common Firestore operations
func (r *FirestoreCategoryRepository) ReadCategory(ctx context.Context, req *commonpb.ReadCategoryRequest) (*commonpb.ReadCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("category ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read category: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	category := &commonpb.Category{}
	convertedCategory, err := operations.ConvertMapToProtobuf(result, category)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &commonpb.ReadCategoryResponse{
		Data:    []*commonpb.Category{convertedCategory},
		Success: true,
	}, nil
}

// UpdateCategory updates a category using common Firestore operations
func (r *FirestoreCategoryRepository) UpdateCategory(ctx context.Context, req *commonpb.UpdateCategoryRequest) (*commonpb.UpdateCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("category ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	category := &commonpb.Category{}
	convertedCategory, err := operations.ConvertMapToProtobuf(result, category)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &commonpb.UpdateCategoryResponse{
		Data:    []*commonpb.Category{convertedCategory},
		Success: true,
	}, nil
}

// DeleteCategory deletes a category using common Firestore operations
func (r *FirestoreCategoryRepository) DeleteCategory(ctx context.Context, req *commonpb.DeleteCategoryRequest) (*commonpb.DeleteCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("category ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete category: %w", err)
	}

	return &commonpb.DeleteCategoryResponse{
		Success: true,
	}, nil
}

// ListCategories lists categories using common Firestore operations with filter support
func (r *FirestoreCategoryRepository) ListCategories(ctx context.Context, req *commonpb.ListCategoriesRequest) (*commonpb.ListCategoriesResponse, error) {
	// Log the collection name being queried
	fmt.Printf("📋 ListCategories: Querying Firestore collection '%s'\n", r.collectionName)

	// Build ListParams from request - pass filters directly to dbOps.List
	listParams := &interfaces.ListParams{
		Filters:    req.Filters,
		Pagination: req.Pagination,
	}

	fmt.Printf("📋 ListCategories: Filters applied: %+v\n", req.Filters)

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		fmt.Printf("❌ ListCategories: Failed to query collection '%s': %v\n", r.collectionName, err)
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	// Use listResult.Data instead of results
	results := listResult.Data

	fmt.Printf("✅ ListCategories: Retrieved %d documents from collection '%s'\n", len(results), r.collectionName)

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	categories, conversionErrs := operations.ConvertSliceToProtobuf(results, func() *commonpb.Category {
		return &commonpb.Category{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// CRITICAL FIX: Ensure we always return a non-nil slice for proper JSON marshaling
	// This guarantees the "data" field is always included in the JSON response
	if categories == nil {
		categories = make([]*commonpb.Category, 0)
	}

	return &commonpb.ListCategoriesResponse{
		Data:    categories,
		Success: true,
	}, nil
}
