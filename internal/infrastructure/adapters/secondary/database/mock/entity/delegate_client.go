//go:build mock_db

package entity

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
)

// MockDelegateClientRepository implements entity.DelegateClientRepository using stateful mock data
type MockDelegateClientRepository struct {
	delegateclientpb.UnimplementedDelegateClientDomainServiceServer
	businessType    string
	delegateClients map[string]*delegateclientpb.DelegateClient // Persistent in-memory store
	mutex           sync.RWMutex                                // Thread-safe concurrent access
	initialized     bool                                        // Prevent double initialization
	processor       *listdata.ListDataProcessor                 // List data processing
}

// DelegateClientRepositoryOption allows configuration of repository behavior
type DelegateClientRepositoryOption func(*MockDelegateClientRepository)

// WithDelegateClientTestOptimizations enables test-specific optimizations
func WithDelegateClientTestOptimizations(enabled bool) DelegateClientRepositoryOption {
	return func(r *MockDelegateClientRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockDelegateClientRepository creates a new mock delegate client repository
func NewMockDelegateClientRepository(businessType string, options ...DelegateClientRepositoryOption) delegateclientpb.DelegateClientDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockDelegateClientRepository{
		businessType:    businessType,
		delegateClients: make(map[string]*delegateclientpb.DelegateClient),
		processor:       listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if err := repo.loadInitialData(); err != nil {
		// Log error but don't fail - allows graceful degradation
		fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockDelegateClientRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawDelegateClients, err := datamock.LoadDelegateClients(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial delegate clients: %w", err)
	}

	// Convert and store each delegate client
	for _, rawDelegateClient := range rawDelegateClients {
		if delegateClient, err := r.mapToProtobufDelegateClient(rawDelegateClient); err == nil {
			r.delegateClients[delegateClient.Id] = delegateClient
		}
	}

	r.initialized = true
	return nil
}

// CreateDelegateClient creates a new delegate client relationship with stateful storage
func (r *MockDelegateClientRepository) CreateDelegateClient(ctx context.Context, req *delegateclientpb.CreateDelegateClientRequest) (*delegateclientpb.CreateDelegateClientResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create delegate client request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("delegate client data is required")
	}
	if req.Data.DelegateId == "" {
		return nil, fmt.Errorf("delegate client delegateId is required")
	}
	if req.Data.ClientId == "" {
		return nil, fmt.Errorf("delegate client clientId is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	delegateClientID := fmt.Sprintf("delegate-client-%d-%d", now.UnixNano(), len(r.delegateClients))

	// Create new delegate client with proper timestamps and defaults
	newDelegateClient := &delegateclientpb.DelegateClient{
		Id:                 delegateClientID,
		DelegateId:         req.Data.DelegateId,
		ClientId:           req.Data.ClientId,
		Client:             req.Data.Client,
		DateCreated:        &[]int64{now.Unix()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.Unix()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.delegateClients[delegateClientID] = newDelegateClient

	return &delegateclientpb.CreateDelegateClientResponse{
		Data:    []*delegateclientpb.DelegateClient{newDelegateClient},
		Success: true,
	}, nil
}

// ReadDelegateClient retrieves a delegate client relationship by ID from stateful storage
func (r *MockDelegateClientRepository) ReadDelegateClient(ctx context.Context, req *delegateclientpb.ReadDelegateClientRequest) (*delegateclientpb.ReadDelegateClientResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read delegate client request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated delegate clients)
	if delegateClient, exists := r.delegateClients[req.Data.Id]; exists {
		return &delegateclientpb.ReadDelegateClientResponse{
			Data:    []*delegateclientpb.DelegateClient{delegateClient},
			Success: true,
		}, nil
	}

	// Return empty result for missing entity (no error)
	return &delegateclientpb.ReadDelegateClientResponse{
		Data:    []*delegateclientpb.DelegateClient{},
		Success: true,
	}, nil
}

// UpdateDelegateClient updates an existing delegate client relationship in stateful storage
func (r *MockDelegateClientRepository) UpdateDelegateClient(ctx context.Context, req *delegateclientpb.UpdateDelegateClientRequest) (*delegateclientpb.UpdateDelegateClientResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update delegate client request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify delegate client exists
	existingDelegateClient, exists := r.delegateClients[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("delegate client with ID '%s' not found", req.Data.Id)
	}

	// Update preserving use case timestamps (correct millisecond units)
	updatedDelegateClient := &delegateclientpb.DelegateClient{
		Id:                 req.Data.Id,
		DelegateId:         req.Data.DelegateId,
		ClientId:           req.Data.ClientId,
		Client:             req.Data.Client,
		DateCreated:        existingDelegateClient.DateCreated,       // Preserve original
		DateCreatedString:  existingDelegateClient.DateCreatedString, // Preserve original
		DateModified:       req.Data.DateModified,                    // Use case sets UnixMilli()
		DateModifiedString: req.Data.DateModifiedString,              // Use case sets RFC3339
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.delegateClients[req.Data.Id] = updatedDelegateClient

	return &delegateclientpb.UpdateDelegateClientResponse{
		Data:    []*delegateclientpb.DelegateClient{updatedDelegateClient},
		Success: true,
	}, nil
}

// DeleteDelegateClient deletes a delegate client relationship from stateful storage
func (r *MockDelegateClientRepository) DeleteDelegateClient(ctx context.Context, req *delegateclientpb.DeleteDelegateClientRequest) (*delegateclientpb.DeleteDelegateClientResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete delegate client request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate client ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify delegate client exists before deletion
	if _, exists := r.delegateClients[req.Data.Id]; !exists {
		return nil, fmt.Errorf("delegate client with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.delegateClients, req.Data.Id)

	return &delegateclientpb.DeleteDelegateClientResponse{
		Success: true,
	}, nil
}

// ListDelegateClients retrieves all delegate client relationships from stateful storage
func (r *MockDelegateClientRepository) ListDelegateClients(ctx context.Context, req *delegateclientpb.ListDelegateClientsRequest) (*delegateclientpb.ListDelegateClientsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of delegate clients
	delegateClients := make([]*delegateclientpb.DelegateClient, 0, len(r.delegateClients))
	for _, delegateClient := range r.delegateClients {
		delegateClients = append(delegateClients, delegateClient)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		delegateClients,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process delegate client list data: %w", err)
	}

	// Convert processed items back to delegate client protobuf format
	processedDelegateClients := make([]*delegateclientpb.DelegateClient, len(result.Items))
	for i, item := range result.Items {
		if delegateClient, ok := item.(*delegateclientpb.DelegateClient); ok {
			processedDelegateClients[i] = delegateClient
		} else {
			return nil, fmt.Errorf("failed to convert item to delegate client type")
		}
	}

	return &delegateclientpb.ListDelegateClientsResponse{
		Data:    processedDelegateClients,
		Success: true,
	}, nil
}

// mapToProtobufDelegateClient converts raw mock data to protobuf DelegateClient
func (r *MockDelegateClientRepository) mapToProtobufDelegateClient(rawDelegateClient map[string]any) (*delegateclientpb.DelegateClient, error) {
	delegateClient := &delegateclientpb.DelegateClient{}

	// Map required fields
	if id, ok := rawDelegateClient["id"].(string); ok {
		delegateClient.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if delegateId, ok := rawDelegateClient["delegateId"].(string); ok {
		delegateClient.DelegateId = delegateId
	} else {
		return nil, fmt.Errorf("missing or invalid delegateId field")
	}

	if clientId, ok := rawDelegateClient["clientId"].(string); ok {
		delegateClient.ClientId = clientId
	} else {
		return nil, fmt.Errorf("missing or invalid clientId field")
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawDelegateClient["dateCreated"].(string); ok {
		delegateClient.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			delegateClient.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawDelegateClient["dateModified"].(string); ok {
		delegateClient.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			delegateClient.DateModified = &timestamp
		}
	}

	if active, ok := rawDelegateClient["active"].(bool); ok {
		delegateClient.Active = active
	}

	return delegateClient, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockDelegateClientRepository) parseTimestamp(timestampStr string) (int64, error) {
	// Try parsing as Unix timestamp first
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	// Try parsing as RFC3339 format
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.Unix(), nil
	}

	// Try parsing as other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.Unix(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// GetDelegateClientListPageData retrieves delegate clients with advanced filtering, sorting, searching, and pagination
func (r *MockDelegateClientRepository) GetDelegateClientListPageData(
	ctx context.Context,
	req *delegateclientpb.GetDelegateClientListPageDataRequest,
) (*delegateclientpb.GetDelegateClientListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get delegate client list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of delegate clients
	delegateClients := make([]*delegateclientpb.DelegateClient, 0, len(r.delegateClients))
	for _, delegateClient := range r.delegateClients {
		delegateClients = append(delegateClients, delegateClient)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		delegateClients,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process delegate client list data: %w", err)
	}

	// Convert processed items back to delegate client protobuf format
	processedDelegateClients := make([]*delegateclientpb.DelegateClient, len(result.Items))
	for i, item := range result.Items {
		if delegateClient, ok := item.(*delegateclientpb.DelegateClient); ok {
			processedDelegateClients[i] = delegateClient
		} else {
			return nil, fmt.Errorf("failed to convert item to delegate client type")
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

	return &delegateclientpb.GetDelegateClientListPageDataResponse{
		DelegateClientList: processedDelegateClients,
		Pagination:         result.PaginationResponse,
		SearchResults:      searchResults,
		Success:            true,
	}, nil
}

// GetDelegateClientItemPageData retrieves a single delegate client with enhanced item page data
func (r *MockDelegateClientRepository) GetDelegateClientItemPageData(
	ctx context.Context,
	req *delegateclientpb.GetDelegateClientItemPageDataRequest,
) (*delegateclientpb.GetDelegateClientItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get delegate client item page data request is required")
	}
	if req.DelegateClientId == "" {
		return nil, fmt.Errorf("delegate client ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	delegateClient, exists := r.delegateClients[req.DelegateClientId]
	if !exists {
		return nil, fmt.Errorf("delegate client with ID '%s' not found", req.DelegateClientId)
	}

	// In a real implementation, you might:
	// 1. Load related data (delegate details, client details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &delegateclientpb.GetDelegateClientItemPageDataResponse{
		DelegateClient: delegateClient,
		Success:        true,
	}, nil
}

// NewDelegateClientRepository creates a new delegate client repository - Provider interface compatibility
func NewDelegateClientRepository(businessType string) delegateclientpb.DelegateClientDomainServiceServer {
	return NewMockDelegateClientRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "delegate_client", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockDelegateClientRepository(businessType), nil
	})
}
