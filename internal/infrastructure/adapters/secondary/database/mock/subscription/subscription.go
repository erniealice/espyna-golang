//go:build mock_db

package subscription

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "subscription", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockSubscriptionRepository(businessType), nil
	})
}

// MockSubscriptionRepository implements subscription.SubscriptionRepository using stateful mock data
type MockSubscriptionRepository struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	businessType  string
	subscriptions map[string]*subscriptionpb.Subscription // Persistent in-memory store
	mutex         sync.RWMutex                            // Thread-safe concurrent access
	initialized   bool                                    // Prevent double initialization
	processor     *listdata.ListDataProcessor             // List data processing utilities
}

// SubscriptionRepositoryOption allows configuration of repository behavior
type SubscriptionRepositoryOption func(*MockSubscriptionRepository)

// WithSubscriptionTestOptimizations enables test-specific optimizations
func WithSubscriptionTestOptimizations(enabled bool) SubscriptionRepositoryOption {
	return func(r *MockSubscriptionRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockSubscriptionRepository creates a new mock subscription repository
func NewMockSubscriptionRepository(businessType string, options ...SubscriptionRepositoryOption) subscriptionpb.SubscriptionDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockSubscriptionRepository{
		businessType:  businessType,
		subscriptions: make(map[string]*subscriptionpb.Subscription),
		processor:     listdata.NewListDataProcessor(),
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
func (r *MockSubscriptionRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawSubscriptions, err := datamock.LoadBusinessTypeModule(r.businessType, "subscription")
	if err != nil {
		return fmt.Errorf("failed to load initial subscriptions: %w", err)
	}

	// Convert and store each subscription
	for _, rawSubscription := range rawSubscriptions {
		if subscription, err := r.mapToProtobufSubscription(rawSubscription); err == nil {
			r.subscriptions[subscription.Id] = subscription
		}
	}

	r.initialized = true
	return nil
}

// CreateSubscription creates a new subscription with stateful storage
func (r *MockSubscriptionRepository) CreateSubscription(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create subscription request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("subscription data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("subscription name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	subscriptionID := fmt.Sprintf("subscription-%d-%d", now.UnixNano(), len(r.subscriptions))

	// Create new subscription with proper timestamps and defaults
	newSubscription := &subscriptionpb.Subscription{
		Id:                 subscriptionID,
		Name:               req.Data.Name,
		PricePlanId:        req.Data.PricePlanId,
		ClientId:           req.Data.ClientId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.subscriptions[subscriptionID] = newSubscription

	return &subscriptionpb.CreateSubscriptionResponse{
		Data:    []*subscriptionpb.Subscription{newSubscription},
		Success: true,
	}, nil
}

// ReadSubscription retrieves a subscription by ID from stateful storage
func (r *MockSubscriptionRepository) ReadSubscription(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read subscription request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated subscriptions)
	if subscription, exists := r.subscriptions[req.Data.Id]; exists {
		return &subscriptionpb.ReadSubscriptionResponse{
			Data:    []*subscriptionpb.Subscription{subscription},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("subscription with ID '%s' not found", req.Data.Id)
}

// UpdateSubscription updates an existing subscription in stateful storage
func (r *MockSubscriptionRepository) UpdateSubscription(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update subscription request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify subscription exists
	existingSubscription, exists := r.subscriptions[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("subscription with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedSubscription := &subscriptionpb.Subscription{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		PricePlanId:        req.Data.PricePlanId,
		ClientId:           req.Data.ClientId,
		DateCreated:        existingSubscription.DateCreated,       // Preserve original
		DateCreatedString:  existingSubscription.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],                // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.subscriptions[req.Data.Id] = updatedSubscription

	return &subscriptionpb.UpdateSubscriptionResponse{
		Data:    []*subscriptionpb.Subscription{updatedSubscription},
		Success: true,
	}, nil
}

// DeleteSubscription deletes a subscription from stateful storage
func (r *MockSubscriptionRepository) DeleteSubscription(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete subscription request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify subscription exists before deletion
	if _, exists := r.subscriptions[req.Data.Id]; !exists {
		return nil, fmt.Errorf("subscription with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.subscriptions, req.Data.Id)

	return &subscriptionpb.DeleteSubscriptionResponse{
		Success: true,
	}, nil
}

// ListSubscriptions retrieves all subscriptions from stateful storage
func (r *MockSubscriptionRepository) ListSubscriptions(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of subscriptions
	subscriptions := make([]*subscriptionpb.Subscription, 0, len(r.subscriptions))
	for _, subscription := range r.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	return &subscriptionpb.ListSubscriptionsResponse{
		Data:    subscriptions,
		Success: true,
	}, nil
}

// GetSubscriptionListPageData retrieves subscriptions with advanced filtering, sorting, searching, and pagination
func (r *MockSubscriptionRepository) GetSubscriptionListPageData(
	ctx context.Context,
	req *subscriptionpb.GetSubscriptionListPageDataRequest,
) (*subscriptionpb.GetSubscriptionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get subscription list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of subscriptions
	subscriptions := make([]*subscriptionpb.Subscription, 0, len(r.subscriptions))
	for _, subscription := range r.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		subscriptions,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process subscription list data: %w", err)
	}

	// Convert processed items back to subscription protobuf format
	processedSubscriptions := make([]*subscriptionpb.Subscription, len(result.Items))
	for i, item := range result.Items {
		if subscription, ok := item.(*subscriptionpb.Subscription); ok {
			processedSubscriptions[i] = subscription
		} else {
			return nil, fmt.Errorf("failed to convert item to subscription type")
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

	return &subscriptionpb.GetSubscriptionListPageDataResponse{
		SubscriptionList: processedSubscriptions,
		Pagination:       result.PaginationResponse,
		SearchResults:    searchResults,
		Success:          true,
	}, nil
}

// GetSubscriptionItemPageData retrieves a single subscription with enhanced item page data
func (r *MockSubscriptionRepository) GetSubscriptionItemPageData(
	ctx context.Context,
	req *subscriptionpb.GetSubscriptionItemPageDataRequest,
) (*subscriptionpb.GetSubscriptionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get subscription item page data request is required")
	}
	if req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	subscription, exists := r.subscriptions[req.SubscriptionId]
	if !exists {
		return nil, fmt.Errorf("subscription with ID '%s' not found", req.SubscriptionId)
	}

	// In a real implementation, you might:
	// 1. Load related data (plan details, client details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &subscriptionpb.GetSubscriptionItemPageDataResponse{
		Subscription: subscription,
		Success:      true,
	}, nil
}

// mapToProtobufSubscription converts raw mock data to protobuf Subscription
func (r *MockSubscriptionRepository) mapToProtobufSubscription(rawSubscription map[string]any) (*subscriptionpb.Subscription, error) {
	subscription := &subscriptionpb.Subscription{}

	// Map required fields
	if id, ok := rawSubscription["id"].(string); ok {
		subscription.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawSubscription["name"].(string); ok {
		subscription.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if pricePlanId, ok := rawSubscription["pricePlanId"].(string); ok {
		subscription.PricePlanId = pricePlanId
	}

	if clientId, ok := rawSubscription["clientId"].(string); ok {
		subscription.ClientId = clientId
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawSubscription["dateCreated"].(string); ok {
		subscription.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			subscription.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawSubscription["dateModified"].(string); ok {
		subscription.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			subscription.DateModified = &timestamp
		}
	}

	if active, ok := rawSubscription["active"].(bool); ok {
		subscription.Active = active
	}

	return subscription, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockSubscriptionRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewSubscriptionRepository creates a new mock subscription repository (registry constructor)
func NewSubscriptionRepository(data map[string]*subscriptionpb.Subscription) subscriptionpb.SubscriptionDomainServiceServer {
	repo := &MockSubscriptionRepository{
		businessType:  "education", // Default business type
		subscriptions: data,
		mutex:         sync.RWMutex{},
		processor:     listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.subscriptions = make(map[string]*subscriptionpb.Subscription)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
