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
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// MockDelegateRepository implements entity.DelegateRepository using stateful mock data
type MockDelegateRepository struct {
	delegatepb.UnimplementedDelegateDomainServiceServer
	businessType string
	delegates    map[string]*delegatepb.Delegate // Persistent in-memory store
	mutex        sync.RWMutex                    // Thread-safe concurrent access
	initialized  bool                            // Prevent double initialization
	processor    *listdata.ListDataProcessor     // List data processing
}

// DelegateRepositoryOption allows configuration of repository behavior
type DelegateRepositoryOption func(*MockDelegateRepository)

// WithDelegateTestOptimizations enables test-specific optimizations
func WithDelegateTestOptimizations(enabled bool) DelegateRepositoryOption {
	return func(r *MockDelegateRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockDelegateRepository creates a new mock delegate repository
func NewMockDelegateRepository(businessType string, options ...DelegateRepositoryOption) delegatepb.DelegateDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockDelegateRepository{
		businessType: businessType,
		delegates:    make(map[string]*delegatepb.Delegate),
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
func (r *MockDelegateRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawDelegates, err := datamock.LoadDelegates(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial delegates: %w", err)
	}

	// Convert and store each delegate
	for _, rawDelegate := range rawDelegates {
		if delegate, err := r.mapToProtobufDelegate(rawDelegate); err == nil {
			r.delegates[delegate.Id] = delegate
		}
	}

	r.initialized = true
	return nil
}

// CreateDelegate creates a new delegate with stateful storage
func (r *MockDelegateRepository) CreateDelegate(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create delegate request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("delegate data is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Use the provided ID or generate a unique ID with timestamp
	delegateID := req.Data.Id
	if delegateID == "" {
		now := time.Now()
		delegateID = fmt.Sprintf("delegate-%d-%d", now.UnixNano(), len(r.delegates))
	}

	// Create new delegate copying all fields from request
	newDelegate := &delegatepb.Delegate{
		Id:                 delegateID,
		UserId:             req.Data.UserId,
		DateCreated:        req.Data.DateCreated,
		DateCreatedString:  req.Data.DateCreatedString,
		DateModified:       req.Data.DateModified,
		DateModifiedString: req.Data.DateModifiedString,
		Active:             req.Data.Active,
	}

	// Copy User object if provided (this is the key fix)
	if req.Data.User != nil {
		newDelegate.User = &userpb.User{
			Id:                 req.Data.User.Id,
			FirstName:          req.Data.User.FirstName,
			LastName:           req.Data.User.LastName,
			EmailAddress:       req.Data.User.EmailAddress,
			DateCreated:        req.Data.User.DateCreated,
			DateCreatedString:  req.Data.User.DateCreatedString,
			DateModified:       req.Data.User.DateModified,
			DateModifiedString: req.Data.User.DateModifiedString,
			Active:             req.Data.User.Active,
		}
	}

	// Store in persistent map
	r.delegates[delegateID] = newDelegate

	return &delegatepb.CreateDelegateResponse{
		Data:    []*delegatepb.Delegate{newDelegate},
		Success: true,
	}, nil
}

// ReadDelegate retrieves a delegate by ID from stateful storage
func (r *MockDelegateRepository) ReadDelegate(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read delegate request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated delegates)
	if delegate, exists := r.delegates[req.Data.Id]; exists {
		return &delegatepb.ReadDelegateResponse{
			Data:    []*delegatepb.Delegate{delegate},
			Success: true,
		}, nil
	}

	// Return empty result for missing entity (no error)
	return &delegatepb.ReadDelegateResponse{
		Data:    []*delegatepb.Delegate{},
		Success: true,
	}, nil
}

// UpdateDelegate updates an existing delegate in stateful storage
func (r *MockDelegateRepository) UpdateDelegate(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update delegate request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify delegate exists
	existingDelegate, exists := r.delegates[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("delegate with ID '%s' does not exist", req.Data.Id)
	}

	// Update preserving use case timestamps and nested objects
	updatedDelegate := &delegatepb.Delegate{
		Id:                 req.Data.Id,
		UserId:             req.Data.UserId,
		Active:             req.Data.Active,
		// Preserve creation timestamps
		DateCreated:        existingDelegate.DateCreated,
		DateCreatedString:  existingDelegate.DateCreatedString,
		// Use timestamps from use case (correct millisecond units)
		DateModified:       req.Data.DateModified,
		DateModifiedString: req.Data.DateModifiedString,
	}

	// Copy nested User object if provided
	if req.Data.User != nil {
		updatedDelegate.User = &userpb.User{
			Id:                 req.Data.User.Id,
			FirstName:          req.Data.User.FirstName,
			LastName:           req.Data.User.LastName,
			EmailAddress:       req.Data.User.EmailAddress,
			DateCreated:        req.Data.User.DateCreated,
			DateCreatedString:  req.Data.User.DateCreatedString,
			DateModified:       req.Data.User.DateModified,
			DateModifiedString: req.Data.User.DateModifiedString,
			Active:             req.Data.User.Active,
		}
	} else {
		updatedDelegate.User = existingDelegate.User
	}

	// Update in persistent store
	r.delegates[req.Data.Id] = updatedDelegate

	return &delegatepb.UpdateDelegateResponse{
		Data:    []*delegatepb.Delegate{updatedDelegate},
		Success: true,
	}, nil
}

// DeleteDelegate deletes a delegate from stateful storage
func (r *MockDelegateRepository) DeleteDelegate(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete delegate request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify delegate exists before deletion
	if _, exists := r.delegates[req.Data.Id]; !exists {
		return nil, fmt.Errorf("delegate with ID '%s' does not exist", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.delegates, req.Data.Id)

	return &delegatepb.DeleteDelegateResponse{
		Success: true,
	}, nil
}

// ListDelegates retrieves all delegates from stateful storage
func (r *MockDelegateRepository) ListDelegates(ctx context.Context, req *delegatepb.ListDelegatesRequest) (*delegatepb.ListDelegatesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of delegates
	delegates := make([]*delegatepb.Delegate, 0, len(r.delegates))
	for _, delegate := range r.delegates {
		delegates = append(delegates, delegate)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		delegates,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process delegate list data: %w", err)
	}

	// Convert processed items back to delegate protobuf format
	processedDelegates := make([]*delegatepb.Delegate, len(result.Items))
	for i, item := range result.Items {
		if delegate, ok := item.(*delegatepb.Delegate); ok {
			processedDelegates[i] = delegate
		} else {
			return nil, fmt.Errorf("failed to convert item to delegate type")
		}
	}

	return &delegatepb.ListDelegatesResponse{
		Data:    processedDelegates,
		Success: true,
	}, nil
}

// mapToProtobufDelegate converts raw mock data to protobuf Delegate
func (r *MockDelegateRepository) mapToProtobufDelegate(raw map[string]any) (*delegatepb.Delegate, error) {
	delegate := &delegatepb.Delegate{}
	if id, ok := raw["id"].(string); ok {
		delegate.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}
	if userId, ok := raw["userId"].(string); ok {
		delegate.UserId = userId
	} else {
		return nil, fmt.Errorf("missing or invalid userId field")
	}

	// Create User object from mock data fields
	if name, hasName := raw["name"].(string); hasName {
		if email, hasEmail := raw["email"].(string); hasEmail {
			// Parse full name into first and last name
			nameParts := strings.Fields(strings.TrimSpace(name))
			firstName := nameParts[0]
			lastName := ""
			if len(nameParts) > 1 {
				lastName = strings.Join(nameParts[1:], " ")
			}

			delegate.User = &userpb.User{
				Id:           delegate.UserId,
				FirstName:    firstName,
				LastName:     lastName,
				EmailAddress: email,
				Active:       true,
			}

			// Set user timestamps if available
			if dateCreated, ok := raw["dateCreated"].(string); ok {
				delegate.User.DateCreatedString = &dateCreated
				if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
					delegate.User.DateCreated = &timestamp
				}
			}

			if dateModified, ok := raw["dateModified"].(string); ok {
				delegate.User.DateModifiedString = &dateModified
				if timestamp, err := r.parseTimestamp(dateModified); err == nil {
					delegate.User.DateModified = &timestamp
				}
			}
		}
	}
	if dateCreated, ok := raw["dateCreated"].(string); ok {
		delegate.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			delegate.DateCreated = &timestamp
		}
	}
	if dateModified, ok := raw["dateModified"].(string); ok {
		delegate.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			delegate.DateModified = &timestamp
		}
	}
	if active, ok := raw["active"].(bool); ok {
		delegate.Active = active
	}
	return delegate, nil
}

func (r *MockDelegateRepository) parseTimestamp(timestampStr string) (int64, error) {
	// Direct timestamp parsing (already in milliseconds)
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}
	// RFC3339 parsing converted to milliseconds
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil  // Consistent milliseconds
	}
	formats := []string{"2006-01-02T15:04:05Z", "2006-01-02 15:04:05", "2006-01-02T15:04:05.000Z"}
	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil  // Consistent milliseconds
		}
	}
	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// GetDelegateListPageData retrieves delegates with advanced filtering, sorting, searching, and pagination
func (r *MockDelegateRepository) GetDelegateListPageData(
	ctx context.Context,
	req *delegatepb.GetDelegateListPageDataRequest,
) (*delegatepb.GetDelegateListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get delegate list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of delegates
	delegates := make([]*delegatepb.Delegate, 0, len(r.delegates))
	for _, delegate := range r.delegates {
		delegates = append(delegates, delegate)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		delegates,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process delegate list data: %w", err)
	}

	// Convert processed items back to delegate protobuf format
	processedDelegates := make([]*delegatepb.Delegate, len(result.Items))
	for i, item := range result.Items {
		if delegate, ok := item.(*delegatepb.Delegate); ok {
			processedDelegates[i] = delegate
		} else {
			return nil, fmt.Errorf("failed to convert item to delegate type")
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

	return &delegatepb.GetDelegateListPageDataResponse{
		DelegateList:  processedDelegates,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetDelegateItemPageData retrieves a single delegate with enhanced item page data
func (r *MockDelegateRepository) GetDelegateItemPageData(
	ctx context.Context,
	req *delegatepb.GetDelegateItemPageDataRequest,
) (*delegatepb.GetDelegateItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get delegate item page data request is required")
	}
	if req.DelegateId == "" {
		return nil, fmt.Errorf("delegate ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	delegate, exists := r.delegates[req.DelegateId]
	if !exists {
		return nil, fmt.Errorf("delegate with ID '%s' not found", req.DelegateId)
	}

	// In a real implementation, you might:
	// 1. Load related data (user details, client relationships)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &delegatepb.GetDelegateItemPageDataResponse{
		Delegate: delegate,
		Success:  true,
	}, nil
}

// NewDelegateRepository creates a new delegate repository - Provider interface compatibility
func NewDelegateRepository(businessType string) delegatepb.DelegateDomainServiceServer {
	return NewMockDelegateRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "delegate", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockDelegateRepository(businessType), nil
	})
}
