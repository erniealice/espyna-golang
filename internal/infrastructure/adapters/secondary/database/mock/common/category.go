//go:build mock_db

package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "category", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockCategoryRepository(businessType), nil
	})
}

// MockCategoryRepository implements categorypb.CategoryServiceServer using stateful mock data
type MockCategoryRepository struct {
	categorypb.UnimplementedCategoryDomainServiceServer
	businessType string
	categories   map[string]*categorypb.Category // Persistent in-memory store
	mutex        sync.RWMutex                    // Thread-safe concurrent access
	initialized  bool                            // Prevent double initialization
}

// CategoryRepositoryOption allows configuration of repository behavior
type CategoryRepositoryOption func(*MockCategoryRepository)

// WithCategoryTestOptimizations enables test-specific optimizations
func WithCategoryTestOptimizations(enabled bool) CategoryRepositoryOption {
	return func(r *MockCategoryRepository) {
		// Test optimizations placeholder
	}
}

// NewMockCategoryRepository creates a new mock category repository
func NewMockCategoryRepository(businessType string, options ...CategoryRepositoryOption) categorypb.CategoryDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockCategoryRepository{
		businessType: businessType,
		categories:   make(map[string]*categorypb.Category),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if !repo.initialized {
		repo.initializeMockData()
		repo.initialized = true
	}

	return repo
}

// InitializeMockData loads mock data from copya package (public interface method)
func (r *MockCategoryRepository) InitializeMockData() {
	r.initializeMockData()
}

// initializeMockData loads mock data from copya package (internal implementation)
func (r *MockCategoryRepository) initializeMockData() {
	// Use datamock.LoadBusinessTypeModule instead of direct file reading
	rawCategories, err := datamock.LoadBusinessTypeModule(r.businessType, "category")
	if err != nil {
		// Silently fail and use empty dataset if loading fails
		return
	}

	// Convert raw data to protobuf models
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, rawCat := range rawCategories {
		if cat, err := r.mapToProtobufCategory(rawCat); err == nil {
			if cat.Id != "" {
				r.categories[cat.Id] = cat
			}
		}
	}
}

// mapToProtobufCategory converts raw mock data to protobuf Category
func (r *MockCategoryRepository) mapToProtobufCategory(rawCat map[string]any) (*categorypb.Category, error) {
	cat := &categorypb.Category{}

	// Map required fields
	if id, ok := rawCat["id"].(string); ok {
		cat.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawCat["name"].(string); ok {
		cat.Name = name
	}

	if description, ok := rawCat["description"].(string); ok {
		cat.Description = description
	}

	if code, ok := rawCat["code"].(string); ok {
		cat.Code = code
	}

	if module, ok := rawCat["module"].(string); ok {
		cat.Module = module
	}

	// Map parent_id if present
	if parentId, ok := rawCat["parent_id"].(string); ok {
		cat.ParentId = &parentId
	}

	// Map active status
	if active, ok := rawCat["active"].(bool); ok {
		cat.Active = active
	} else {
		cat.Active = true // Default to active
	}

	// Map display_order if present
	if displayOrder, ok := rawCat["display_order"].(int32); ok {
		cat.DisplayOrder = &displayOrder
	}

	return cat, nil
}

// CreateCategory creates a new category
func (r *MockCategoryRepository) CreateCategory(ctx context.Context, req *categorypb.CreateCategoryRequest) (*categorypb.CreateCategoryResponse, error) {
	if req == nil || req.Data == nil {
		return &categorypb.CreateCategoryResponse{
			Success: false,
			Error:   &categorypb.Error{Code: "INVALID_REQUEST", Message: "Request data is required"},
		}, nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate ID if not provided
	if req.Data.Id == "" {
		req.Data.Id = fmt.Sprintf("cat_%d", time.Now().UnixNano())
	}

	// Set timestamps
	now := time.Now()
	nowUnix := now.Unix()
	nowString := now.Format(time.RFC3339)
	req.Data.DateCreated = &nowUnix
	req.Data.DateCreatedString = &nowString
	req.Data.DateModified = &nowUnix
	req.Data.DateModifiedString = &nowString
	req.Data.Active = true

	// Store in memory
	r.categories[req.Data.Id] = req.Data

	return &categorypb.CreateCategoryResponse{
		Data:    []*categorypb.Category{req.Data},
		Success: true,
	}, nil
}

// ReadCategory retrieves a category by ID
func (r *MockCategoryRepository) ReadCategory(ctx context.Context, req *categorypb.ReadCategoryRequest) (*categorypb.ReadCategoryResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return &categorypb.ReadCategoryResponse{
			Success: false,
			Error:   &categorypb.Error{Code: "INVALID_REQUEST", Message: "Category ID is required"},
		}, nil
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	category, exists := r.categories[req.Data.Id]
	if !exists {
		return &categorypb.ReadCategoryResponse{
			Success: false,
			Error:   &categorypb.Error{Code: "NOT_FOUND", Message: fmt.Sprintf("Category with ID '%s' not found", req.Data.Id)},
		}, nil
	}

	return &categorypb.ReadCategoryResponse{
		Data:    []*categorypb.Category{category},
		Success: true,
	}, nil
}

// UpdateCategory updates an existing category
func (r *MockCategoryRepository) UpdateCategory(ctx context.Context, req *categorypb.UpdateCategoryRequest) (*categorypb.UpdateCategoryResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return &categorypb.UpdateCategoryResponse{
			Success: false,
			Error:   &categorypb.Error{Code: "INVALID_REQUEST", Message: "Category ID is required"},
		}, nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, exists := r.categories[req.Data.Id]
	if !exists {
		return &categorypb.UpdateCategoryResponse{
			Success: false,
			Error:   &categorypb.Error{Code: "NOT_FOUND", Message: fmt.Sprintf("Category with ID '%s' not found", req.Data.Id)},
		}, nil
	}

	// Update timestamps
	now := time.Now()
	nowUnix := now.Unix()
	nowString := now.Format(time.RFC3339)
	req.Data.DateModified = &nowUnix
	req.Data.DateModifiedString = &nowString

	// Preserve creation timestamps
	req.Data.DateCreated = existing.DateCreated
	req.Data.DateCreatedString = existing.DateCreatedString

	// Update in memory
	r.categories[req.Data.Id] = req.Data

	return &categorypb.UpdateCategoryResponse{
		Data:    []*categorypb.Category{req.Data},
		Success: true,
	}, nil
}

// DeleteCategory deletes a category
func (r *MockCategoryRepository) DeleteCategory(ctx context.Context, req *categorypb.DeleteCategoryRequest) (*categorypb.DeleteCategoryResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return &categorypb.DeleteCategoryResponse{
			Success: false,
			Error:   &categorypb.Error{Code: "INVALID_REQUEST", Message: "Category ID is required"},
		}, nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	category, exists := r.categories[req.Data.Id]
	if !exists {
		return &categorypb.DeleteCategoryResponse{
			Success: false,
			Error:   &categorypb.Error{Code: "NOT_FOUND", Message: fmt.Sprintf("Category with ID '%s' not found", req.Data.Id)},
		}, nil
	}

	// Perform soft delete
	category.Active = false
	now := time.Now()
	nowUnix := now.Unix()
	nowString := now.Format(time.RFC3339)
	category.DateModified = &nowUnix
	category.DateModifiedString = &nowString

	return &categorypb.DeleteCategoryResponse{
		Data:    []*categorypb.Category{category},
		Success: true,
	}, nil
}

// ListCategories lists all categories
func (r *MockCategoryRepository) ListCategories(ctx context.Context, req *categorypb.ListCategoriesRequest) (*categorypb.ListCategoriesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var categories []*categorypb.Category
	for _, cat := range r.categories {
		if cat.Active { // Only return active categories
			categories = append(categories, cat)
		}
	}

	// Apply limit if specified via pagination
	if req != nil && req.Pagination != nil && req.Pagination.Limit > 0 {
		limit := int(req.Pagination.Limit)
		if len(categories) > limit {
			categories = categories[:limit]
		}
	}

	return &categorypb.ListCategoriesResponse{
		Data:    categories,
		Success: true,
	}, nil
}

// NewCategoryRepository creates a new mock category repository (legacy constructor)
func NewCategoryRepository(businessType string) categorypb.CategoryDomainServiceServer {
	return NewMockCategoryRepository(businessType)
}
