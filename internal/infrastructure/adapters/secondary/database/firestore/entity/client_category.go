//go:build firestore

package entity

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	firestoreCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

func init() {
	registry.RegisterRepositoryFactory("firestore", "client_category", func(conn any, collectionName string) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore client_category repository requires *firestore.Client, got %T", conn)
		}
		dbOps := firestoreCore.NewFirestoreOperations(client)
		return NewFirestoreClientCategoryRepository(dbOps, collectionName), nil
	})
}

// FirestoreClientCategoryRepository implements client_category CRUD operations using Firestore
type FirestoreClientCategoryRepository struct {
	clientcategorypb.UnimplementedClientCategoryDomainServiceServer
	dbOps          interfaces.DatabaseOperation
	collectionName string
	mapper         *operations.ProtobufMapper
}

// NewFirestoreClientCategoryRepository creates a new Firestore client_category repository
func NewFirestoreClientCategoryRepository(dbOps interfaces.DatabaseOperation, collectionName string) clientcategorypb.ClientCategoryDomainServiceServer {
	if collectionName == "" {
		collectionName = "client_category" // default fallback
	}
	return &FirestoreClientCategoryRepository{
		dbOps:          dbOps,
		collectionName: collectionName,
		mapper:         operations.NewProtobufMapper(),
	}
}

// CreateClientCategory creates a new client_category using common Firestore operations
func (r *FirestoreClientCategoryRepository) CreateClientCategory(ctx context.Context, req *clientcategorypb.CreateClientCategoryRequest) (*clientcategorypb.CreateClientCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client_category data is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.collectionName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create client_category: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	clientCategory := &clientcategorypb.ClientCategory{}
	convertedClientCategory, err := operations.ConvertMapToProtobuf(result, clientCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientcategorypb.CreateClientCategoryResponse{
		Data: []*clientcategorypb.ClientCategory{convertedClientCategory},
	}, nil
}

// ReadClientCategory retrieves a client_category using common Firestore operations
func (r *FirestoreClientCategoryRepository) ReadClientCategory(ctx context.Context, req *clientcategorypb.ReadClientCategoryRequest) (*clientcategorypb.ReadClientCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_category ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client_category: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	clientCategory := &clientcategorypb.ClientCategory{}
	convertedClientCategory, err := operations.ConvertMapToProtobuf(result, clientCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientcategorypb.ReadClientCategoryResponse{
		Data: []*clientcategorypb.ClientCategory{convertedClientCategory},
	}, nil
}

// UpdateClientCategory updates a client_category using common Firestore operations
func (r *FirestoreClientCategoryRepository) UpdateClientCategory(ctx context.Context, req *clientcategorypb.UpdateClientCategoryRequest) (*clientcategorypb.UpdateClientCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_category ID is required")
	}

	// Convert protobuf to map using efficient ProtobufMapper
	data, err := r.mapper.ConvertProtobufToMap(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.collectionName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update client_category: %w", err)
	}

	// Convert result back to protobuf using clean ConvertMapToProtobuf
	clientCategory := &clientcategorypb.ClientCategory{}
	convertedClientCategory, err := operations.ConvertMapToProtobuf(result, clientCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientcategorypb.UpdateClientCategoryResponse{
		Data: []*clientcategorypb.ClientCategory{convertedClientCategory},
	}, nil
}

// DeleteClientCategory deletes a client_category using common Firestore operations
func (r *FirestoreClientCategoryRepository) DeleteClientCategory(ctx context.Context, req *clientcategorypb.DeleteClientCategoryRequest) (*clientcategorypb.DeleteClientCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_category ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.collectionName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete client_category: %w", err)
	}

	return &clientcategorypb.DeleteClientCategoryResponse{
		Success: true,
	}, nil
}

// ListClientCategories lists client_categories using common Firestore operations
func (r *FirestoreClientCategoryRepository) ListClientCategories(ctx context.Context, req *clientcategorypb.ListClientCategoriesRequest) (*clientcategorypb.ListClientCategoriesResponse, error) {
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
		return nil, fmt.Errorf("failed to list client_categories: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	clientCategories, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *clientcategorypb.ClientCategory {
		return &clientcategorypb.ClientCategory{}
	})

	// Log any conversion errors for debugging (optional, can be removed for production)
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice for proper JSON marshaling
	if clientCategories == nil {
		clientCategories = make([]*clientcategorypb.ClientCategory, 0)
	}

	return &clientcategorypb.ListClientCategoriesResponse{
		Data: clientCategories,
	}, nil
}

// GetClientCategoryListPageData retrieves client_categories with advanced filtering, sorting, searching, and pagination
func (r *FirestoreClientCategoryRepository) GetClientCategoryListPageData(
	ctx context.Context,
	req *clientcategorypb.GetClientCategoryListPageDataRequest,
) (*clientcategorypb.GetClientCategoryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client_category list page data request is required")
	}

	// Build ListParams from request
	listParams := &interfaces.ListParams{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// List documents using common operations with proper filter support
	listResult, err := r.dbOps.List(ctx, r.collectionName, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list client_categories: %w", err)
	}

	// Convert results to protobuf slice using ConvertSliceToProtobuf
	clientCategories, conversionErrs := operations.ConvertSliceToProtobuf(listResult.Data, func() *clientcategorypb.ClientCategory {
		return &clientcategorypb.ClientCategory{}
	})

	// Log any conversion errors for debugging
	for i, err := range conversionErrs {
		fmt.Printf("Failed to convert document %d: %v\n", i, err)
	}

	// Ensure we always return a non-nil slice
	if clientCategories == nil {
		clientCategories = make([]*clientcategorypb.ClientCategory, 0)
	}

	// Calculate pagination metadata
	totalCount := int64(len(listResult.Data))
	totalPages := int32(1)
	limit := int32(50)
	page := int32(1)

	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if req.Pagination != nil {
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
		}
	}

	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &clientcategorypb.GetClientCategoryListPageDataResponse{
		ClientCategoryList: clientCategories,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetClientCategoryItemPageData retrieves a single client_category with enhanced item page data
func (r *FirestoreClientCategoryRepository) GetClientCategoryItemPageData(
	ctx context.Context,
	req *clientcategorypb.GetClientCategoryItemPageDataRequest,
) (*clientcategorypb.GetClientCategoryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client_category item page data request is required")
	}
	if req.ClientCategoryId == "" {
		return nil, fmt.Errorf("client_category ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.collectionName, req.ClientCategoryId)
	if err != nil {
		return nil, fmt.Errorf("failed to read client_category: %w", err)
	}

	// Convert result to protobuf using clean ConvertMapToProtobuf
	clientCategory := &clientcategorypb.ClientCategory{}
	convertedClientCategory, err := operations.ConvertMapToProtobuf(result, clientCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf: %w", err)
	}

	return &clientcategorypb.GetClientCategoryItemPageDataResponse{
		ClientCategory: convertedClientCategory,
		Success:        true,
	}, nil
}
