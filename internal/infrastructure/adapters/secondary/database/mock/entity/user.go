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
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// MockUserRepository implements entity.UserRepository using stateful mock data
type MockUserRepository struct {
	userpb.UnimplementedUserDomainServiceServer
	businessType string
	users        map[string]*userpb.User // Persistent in-memory store
	mutex        sync.RWMutex            // Thread-safe concurrent access
	initialized  bool                    // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing utilities
}

// UserRepositoryOption allows configuration of repository behavior
type UserRepositoryOption func(*MockUserRepository)

// WithUserTestOptimizations enables test-specific optimizations
func WithUserTestOptimizations(enabled bool) UserRepositoryOption {
	return func(r *MockUserRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockUserRepository creates a new mock user repository
func NewMockUserRepository(businessType string, options ...UserRepositoryOption) userpb.UserDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockUserRepository{
		businessType: businessType,
		users:        make(map[string]*userpb.User),
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
func (r *MockUserRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawUsers, err := datamock.LoadUsers(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial users: %w", err)
	}

	// Convert and store each user
	for _, rawUser := range rawUsers {
		if user, err := r.mapToProtobufUser(rawUser); err == nil {
			r.users[user.Id] = user
		}
	}

	r.initialized = true
	return nil
}

// CreateUser creates a new user with stateful storage
func (r *MockUserRepository) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create user request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("user data is required")
	}
	if req.Data.FirstName == "" {
		return nil, fmt.Errorf("first name is required")
	}
	if req.Data.LastName == "" {
		return nil, fmt.Errorf("last name is required")
	}
	if req.Data.EmailAddress == "" {
		return nil, fmt.Errorf("email address is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	userID := fmt.Sprintf("user-%d-%d", now.UnixNano(), len(r.users))

	// Create new user with proper timestamps and defaults
	newUser := &userpb.User{
		Id:                 userID,
		FirstName:          req.Data.FirstName,
		LastName:           req.Data.LastName,
		EmailAddress:       req.Data.EmailAddress,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.users[userID] = newUser

	return &userpb.CreateUserResponse{
		Data:    []*userpb.User{newUser},
		Success: true,
	}, nil
}

// ReadUser retrieves a user by ID from stateful storage
func (r *MockUserRepository) ReadUser(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read user request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated users)
	if user, exists := r.users[req.Data.Id]; exists {
		return &userpb.ReadUserResponse{
			Data:    []*userpb.User{user},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("user with ID '%s' not found", req.Data.Id)
}

// UpdateUser updates an existing user in stateful storage
func (r *MockUserRepository) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update user request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify user exists
	existingUser, exists := r.users[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("user with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedUser := &userpb.User{
		Id:                 req.Data.Id,
		FirstName:          req.Data.FirstName,
		LastName:           req.Data.LastName,
		EmailAddress:       req.Data.EmailAddress,
		DateCreated:        existingUser.DateCreated,       // Preserve original
		DateCreatedString:  existingUser.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],        // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.users[req.Data.Id] = updatedUser

	return &userpb.UpdateUserResponse{
		Data:    []*userpb.User{updatedUser},
		Success: true,
	}, nil
}

// DeleteUser deletes a user from stateful storage
func (r *MockUserRepository) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete user request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("user ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify user exists before deletion
	if _, exists := r.users[req.Data.Id]; !exists {
		return nil, fmt.Errorf("user with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.users, req.Data.Id)

	return &userpb.DeleteUserResponse{
		Success: true,
	}, nil
}

// ListUsers retrieves all users from stateful storage
func (r *MockUserRepository) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of users
	items := make([]*userpb.User, 0, len(r.users))
	for _, user := range r.users {
		items = append(items, user)
	}

	// Process list data with processor
	result, err := r.processor.ProcessListRequest(
		items,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process list data: %w", err)
	}

	// Convert result.Items back to protobuf type
	processed := make([]*userpb.User, len(result.Items))
	for i, item := range result.Items {
		if typed, ok := item.(*userpb.User); ok {
			processed[i] = typed
		}
	}

	return &userpb.ListUsersResponse{
		Data:    processed,
		Success: true,
	}, nil
}

// mapToProtobufUser converts raw mock data to protobuf User
func (r *MockUserRepository) mapToProtobufUser(rawUser map[string]any) (*userpb.User, error) {
	user := &userpb.User{}

	// Map required fields
	if id, ok := rawUser["id"].(string); ok {
		user.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if firstName, ok := rawUser["firstName"].(string); ok {
		user.FirstName = firstName
	} else {
		return nil, fmt.Errorf("missing or invalid firstName field")
	}

	if lastName, ok := rawUser["lastName"].(string); ok {
		user.LastName = lastName
	} else {
		return nil, fmt.Errorf("missing or invalid lastName field")
	}

	if emailAddress, ok := rawUser["emailAddress"].(string); ok {
		user.EmailAddress = emailAddress
	} else {
		return nil, fmt.Errorf("missing or invalid emailAddress field")
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawUser["dateCreated"].(string); ok {
		user.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			user.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawUser["dateModified"].(string); ok {
		user.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			user.DateModified = &timestamp
		}
	}

	if active, ok := rawUser["active"].(bool); ok {
		user.Active = active
	}

	return user, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockUserRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// GetUserListPageData retrieves users with advanced filtering, sorting, searching, and pagination
func (r *MockUserRepository) GetUserListPageData(
	ctx context.Context,
	req *userpb.GetUserListPageDataRequest,
) (*userpb.GetUserListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get user list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of users
	users := make([]*userpb.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		users,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process user list data: %w", err)
	}

	// Convert processed items back to user protobuf format
	processedUsers := make([]*userpb.User, len(result.Items))
	for i, item := range result.Items {
		if user, ok := item.(*userpb.User); ok {
			processedUsers[i] = user
		} else {
			return nil, fmt.Errorf("failed to convert item to user type")
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

	return &userpb.GetUserListPageDataResponse{
		UserList:      processedUsers,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetUserItemPageData retrieves a single user with enhanced item page data
func (r *MockUserRepository) GetUserItemPageData(
	ctx context.Context,
	req *userpb.GetUserItemPageDataRequest,
) (*userpb.GetUserItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get user item page data request is required")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	user, exists := r.users[req.UserId]
	if !exists {
		return nil, fmt.Errorf("user with ID '%s' not found", req.UserId)
	}

	// In a real implementation, you might:
	// 1. Load related data (role details, workspace memberships)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &userpb.GetUserItemPageDataResponse{
		User:    user,
		Success: true,
	}, nil
}

// NewUserRepository creates a new user repository - Provider interface compatibility
func NewUserRepository(businessType string) userpb.UserDomainServiceServer {
	return NewMockUserRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "user", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockUserRepository(businessType), nil
	})
}
