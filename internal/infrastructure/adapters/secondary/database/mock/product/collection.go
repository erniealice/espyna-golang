//go:build mock_db

package product

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// MockCollectionRepository implements product.CollectionRepository using stateful mock data
type MockCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	businessType    string
	collections     map[string]*collectionpb.Collection // Persistent in-memory store
	mutex           sync.RWMutex                        // Thread-safe concurrent access
	initialized     bool                                // Prevent double initialization
	skipInitialData bool                                // Option to skip loading baseline data
	processor       *listdata.ListDataProcessor         // List data processor for filtering, sorting, searching, and pagination
}

// CollectionRepositoryOption allows configuration of repository behavior
type CollectionRepositoryOption func(*MockCollectionRepository)

// WithCollectionTestOptimizations enables test-specific optimizations
func WithCollectionTestOptimizations(enabled bool) CollectionRepositoryOption {
	return func(r *MockCollectionRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// WithoutInitialData prevents the repository from loading the baseline mock data.
func WithoutInitialData() CollectionRepositoryOption {
	return func(r *MockCollectionRepository) {
		r.skipInitialData = true
	}
}

// NewMockCollectionRepository creates a new mock collection repository
func NewMockCollectionRepository(businessType string, options ...CollectionRepositoryOption) collectionpb.CollectionDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockCollectionRepository{
		businessType: businessType,
		collections:  make(map[string]*collectionpb.Collection),
		processor:    listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once, unless skipped by an option.
	if !repo.skipInitialData {
		if err := repo.loadInitialData(); err != nil {
			// Log error but don't fail - allows graceful degradation
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockCollectionRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawCollections, err := datamock.LoadBusinessTypeModule(r.businessType, "collection")
	if err != nil {
		return fmt.Errorf("failed to load initial collections: %w", err)
	}

	// Convert and store each collection
	for _, rawCollection := range rawCollections {
		if collection, err := r.mapToProtobufCollection(rawCollection); err == nil {
			r.collections[collection.Id] = collection
		}
	}

	r.initialized = true
	return nil
}

// CreateCollection creates a new collection with stateful storage
func (r *MockCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create collection request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("collection data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	collectionID := req.Data.Id
	if collectionID == "" {
		// Generate unique ID with timestamp
		now := time.Now()
		collectionID = fmt.Sprintf("collection-%d-%d", now.UnixNano(), len(r.collections))
	}

	// Create new collection with proper timestamps and defaults
	newCollection := &collectionpb.Collection{
		Id:                 collectionID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.collections[collectionID] = newCollection

	return &collectionpb.CreateCollectionResponse{
		Data:    []*collectionpb.Collection{newCollection},
		Success: true,
	}, nil
}

// ReadCollection retrieves a collection by ID from stateful storage
func (r *MockCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read collection request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated collections)
	if collection, exists := r.collections[req.Data.Id]; exists {
		return &collectionpb.ReadCollectionResponse{
			Data:    []*collectionpb.Collection{collection},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("collection with ID '%s' not found", req.Data.Id)
}

// UpdateCollection updates an existing collection in stateful storage
func (r *MockCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update collection request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify collection exists
	existingCollection, exists := r.collections[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("collection with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedCollection := &collectionpb.Collection{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        existingCollection.DateCreated,       // Preserve original
		DateCreatedString:  existingCollection.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],              // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.collections[req.Data.Id] = updatedCollection

	return &collectionpb.UpdateCollectionResponse{
		Data:    []*collectionpb.Collection{updatedCollection},
		Success: true,
	}, nil
}

// DeleteCollection deletes a collection from stateful storage
func (r *MockCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete collection request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify collection exists before deletion
	if _, exists := r.collections[req.Data.Id]; !exists {
		return nil, fmt.Errorf("collection with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.collections, req.Data.Id)

	return &collectionpb.DeleteCollectionResponse{
		Success: true,
	}, nil
}

// ListCollections retrieves all collections from stateful storage
func (r *MockCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of collections
	collections := make([]*collectionpb.Collection, 0, len(r.collections))
	for _, collection := range r.collections {
		collections = append(collections, collection)
	}

	return &collectionpb.ListCollectionsResponse{
		Data:    collections,
		Success: true,
	}, nil
}

// mapToProtobufCollection converts raw mock data to protobuf Collection
func (r *MockCollectionRepository) mapToProtobufCollection(rawCollection map[string]any) (*collectionpb.Collection, error) {
	collection := &collectionpb.Collection{}

	// Map required fields
	if id, ok := rawCollection["id"].(string); ok {
		collection.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawCollection["name"].(string); ok {
		collection.Name = name
	}

	if description, ok := rawCollection["description"].(string); ok {
		collection.Description = description
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawCollection["dateCreated"].(string); ok {
		collection.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			collection.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawCollection["dateModified"].(string); ok {
		collection.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			collection.DateModified = &timestamp
		}
	}

	if active, ok := rawCollection["active"].(bool); ok {
		collection.Active = active
	}

	return collection, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockCollectionRepository) parseTimestamp(timestampStr string) (int64, error) {
	// Try parsing as Unix timestamp first
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	// Try parsing as RFC3339 format
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try parsing as other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// GetCollectionListPageData retrieves collections with advanced filtering, sorting, searching, and pagination
func (r *MockCollectionRepository) GetCollectionListPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionListPageDataRequest,
) (*collectionpb.GetCollectionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of collections
	collections := make([]*collectionpb.Collection, 0, len(r.collections))
	for _, collection := range r.collections {
		collections = append(collections, collection)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		collections,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process collection list data: %w", err)
	}

	// Convert processed items back to collection protobuf format
	processedCollections := make([]*collectionpb.Collection, len(result.Items))
	for i, item := range result.Items {
		if collection, ok := item.(*collectionpb.Collection); ok {
			processedCollections[i] = collection
		} else {
			return nil, fmt.Errorf("failed to convert item to collection type")
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &collectionpb.GetCollectionListPageDataResponse{
		CollectionList: processedCollections,
		Pagination:     result.PaginationResponse,
		SearchResults:  searchResults,
		Success:        true,
	}, nil
}

// GetCollectionItemPageData retrieves a single collection with enhanced item page data
func (r *MockCollectionRepository) GetCollectionItemPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionItemPageDataRequest,
) (*collectionpb.GetCollectionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection item page data request is required")
	}
	if req.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	collection, exists := r.collections[req.CollectionId]
	if !exists {
		return nil, fmt.Errorf("collection with ID '%s' not found", req.CollectionId)
	}

	// In a real implementation, you might:
	// 1. Load related data (product details, category details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &collectionpb.GetCollectionItemPageDataResponse{
		Collection: collection,
		Success:    true,
	}, nil
}

// NewCollectionRepository creates a new collection repository - Provider interface compatibility
func NewCollectionRepository(businessType string) collectionpb.CollectionDomainServiceServer {
	return NewMockCollectionRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "collection", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockCollectionRepository(businessType), nil
	})
}
