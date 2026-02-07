//go:build mock_db

package entity

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// MockClientRepository implements entity.ClientRepository using stateful mock data
type MockClientRepository struct {
	clientpb.UnimplementedClientDomainServiceServer
	businessType string
	clients      map[string]*clientpb.Client // Persistent in-memory store
	mutex        sync.RWMutex                // Thread-safe concurrent access
	initialized  bool                        // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing utilities
}

// ClientRepositoryOption allows configuration of repository behavior
type ClientRepositoryOption func(*MockClientRepository)

// WithClientTestOptimizations enables test-specific optimizations
func WithClientTestOptimizations(enabled bool) ClientRepositoryOption {
	return func(r *MockClientRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockClientRepository creates a new mock client repository
func NewMockClientRepository(businessType string, options ...ClientRepositoryOption) clientpb.ClientDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockClientRepository{
		businessType: businessType,
		clients:      make(map[string]*clientpb.Client),
		processor:    listdata.NewListDataProcessor(),
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
func (r *MockClientRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawClients, err := datamock.LoadClients(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial clients: %w", err)
	}

	// Convert and store each client
	for _, rawClient := range rawClients {
		if client, err := r.mapToProtobufClient(rawClient); err == nil {
			r.clients[client.Id] = client
		}
	}

	r.initialized = true
	return nil
}

// CreateClient creates a new client with stateful storage
func (r *MockClientRepository) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create client request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("client data is required")
	}
	if req.Data.UserId == "" {
		return nil, fmt.Errorf("client userId is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	clientID := fmt.Sprintf("client-%d-%d", now.UnixNano(), len(r.clients))

	// Create new client with proper timestamps and defaults
	newClient := &clientpb.Client{
		Id:                 clientID,
		UserId:             req.Data.UserId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
		InternalId:         req.Data.InternalId,
	}

	// Store in persistent map
	r.clients[clientID] = newClient

	return &clientpb.CreateClientResponse{
		Data:    []*clientpb.Client{newClient},
		Success: true,
	}, nil
}

// ReadClient retrieves a client by ID from stateful storage
func (r *MockClientRepository) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read client request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated clients)
	if client, exists := r.clients[req.Data.Id]; exists {
		return &clientpb.ReadClientResponse{
			Data:    []*clientpb.Client{client},
			Success: true,
		}, nil
	}

	return &clientpb.ReadClientResponse{
		Data:    []*clientpb.Client{},
		Success: false,
	}, nil
}

// UpdateClient updates an existing client in stateful storage
func (r *MockClientRepository) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update client request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify client exists
	existingClient, exists := r.clients[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("client with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedClient := &clientpb.Client{
		Id:                 req.Data.Id,
		UserId:             req.Data.UserId,
		User:               req.Data.User,                    // Include User field from request
		DateCreated:        existingClient.DateCreated,       // Preserve original
		DateCreatedString:  existingClient.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],     // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
		InternalId:         req.Data.InternalId,
	}

	// Update in persistent store
	r.clients[req.Data.Id] = updatedClient

	return &clientpb.UpdateClientResponse{
		Data:    []*clientpb.Client{updatedClient},
		Success: true,
	}, nil
}

// DeleteClient deletes a client from stateful storage
func (r *MockClientRepository) DeleteClient(ctx context.Context, req *clientpb.DeleteClientRequest) (*clientpb.DeleteClientResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete client request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify client exists before deletion
	if _, exists := r.clients[req.Data.Id]; !exists {
		return nil, fmt.Errorf("client with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.clients, req.Data.Id)

	return &clientpb.DeleteClientResponse{
		Success: true,
	}, nil
}

// ListClients retrieves all clients from stateful storage
func (r *MockClientRepository) ListClients(ctx context.Context, req *clientpb.ListClientsRequest) (*clientpb.ListClientsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of clients
	clients := make([]*clientpb.Client, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		clients,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process client list data: %w", err)
	}

	// Convert processed items back to client protobuf format
	processedClients := make([]*clientpb.Client, len(result.Items))
	for i, item := range result.Items {
		if client, ok := item.(*clientpb.Client); ok {
			processedClients[i] = client
		} else {
			return nil, fmt.Errorf("failed to convert item to client type")
		}
	}

	return &clientpb.ListClientsResponse{
		Data:    processedClients,
		Success: true,
	}, nil
}

// GetClientListPageData retrieves clients with advanced filtering, sorting, searching, and pagination
func (r *MockClientRepository) GetClientListPageData(
	ctx context.Context,
	req *clientpb.GetClientListPageDataRequest,
) (*clientpb.GetClientListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of clients
	clients := make([]*clientpb.Client, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		clients,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process client list data: %w", err)
	}

	// Convert processed items back to client protobuf format
	processedClients := make([]*clientpb.Client, len(result.Items))
	for i, item := range result.Items {
		if client, ok := item.(*clientpb.Client); ok {
			processedClients[i] = client
		} else {
			return nil, fmt.Errorf("failed to convert item to client type")
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

	return &clientpb.GetClientListPageDataResponse{
		ClientList:    processedClients,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetClientItemPageData retrieves a single client with enhanced item page data
func (r *MockClientRepository) GetClientItemPageData(
	ctx context.Context,
	req *clientpb.GetClientItemPageDataRequest,
) (*clientpb.GetClientItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client item page data request is required")
	}
	if req.ClientId == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	client, exists := r.clients[req.ClientId]
	if !exists {
		return nil, fmt.Errorf("client with ID '%s' not found", req.ClientId)
	}

	// In a real implementation, you might:
	// 1. Load related data (group details, user details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &clientpb.GetClientItemPageDataResponse{
		Client:  client,
		Success: true,
	}, nil
}

// mapToProtobufClient converts raw mock data to protobuf Client
func (r *MockClientRepository) mapToProtobufClient(rawClient map[string]any) (*clientpb.Client, error) {
	client := &clientpb.Client{}

	// Map required fields
	if id, ok := rawClient["id"].(string); ok {
		client.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if userId, ok := rawClient["userId"].(string); ok {
		client.UserId = userId
	} else {
		return nil, fmt.Errorf("missing or invalid userId field")
	}

	// Create User object from mock data fields
	if name, hasName := rawClient["name"].(string); hasName {
		if email, hasEmail := rawClient["email"].(string); hasEmail {
			// Parse full name into first and last name
			nameParts := strings.Fields(strings.TrimSpace(name))
			firstName := nameParts[0]
			lastName := ""
			if len(nameParts) > 1 {
				lastName = strings.Join(nameParts[1:], " ")
			}

			client.User = &userpb.User{
				Id:           client.UserId,
				FirstName:    firstName,
				LastName:     lastName,
				EmailAddress: email,
				Active:       true,
			}

			// Set user timestamps if available
			if dateCreated, ok := rawClient["dateCreated"].(string); ok {
				client.User.DateCreatedString = &dateCreated
				if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
					client.User.DateCreated = &timestamp
				}
			}

			if dateModified, ok := rawClient["dateModified"].(string); ok {
				client.User.DateModifiedString = &dateModified
				if timestamp, err := r.parseTimestamp(dateModified); err == nil {
					client.User.DateModified = &timestamp
				}
			}
		}
	}

	// Map optional fields
	if internalId, ok := rawClient["internalId"].(string); ok {
		client.InternalId = internalId
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawClient["dateCreated"].(string); ok {
		client.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			client.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawClient["dateModified"].(string); ok {
		client.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			client.DateModified = &timestamp
		}
	}

	if active, ok := rawClient["active"].(bool); ok {
		client.Active = active
	}

	return client, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockClientRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewClientRepository creates a new client repository - Provider interface compatibility
func NewClientRepository(businessType string) clientpb.ClientDomainServiceServer {
	return NewMockClientRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "client", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockClientRepository(businessType), nil
	})
}
