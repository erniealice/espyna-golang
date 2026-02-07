//go:build mock_db

package payment

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// MockPaymentProfileRepository implements payment.PaymentProfileRepository using stateful mock data
type MockPaymentProfileRepository struct {
	paymentprofilepb.UnimplementedPaymentProfileDomainServiceServer
	businessType    string
	paymentProfiles map[string]*paymentprofilepb.PaymentProfile // Persistent in-memory store
	mutex           sync.RWMutex                                // Thread-safe concurrent access
	initialized     bool                                        // Prevent double initialization
	processor       *listdata.ListDataProcessor                 // List data processing
}

// PaymentProfileRepositoryOption allows configuration of payment profile repository behavior
type PaymentProfileRepositoryOption func(*MockPaymentProfileRepository)

// WithPaymentProfileTestOptimizations enables test-specific optimizations
func WithPaymentProfileTestOptimizations(enabled bool) PaymentProfileRepositoryOption {
	return func(r *MockPaymentProfileRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPaymentProfileRepository creates a new mock payment profile repository
func NewMockPaymentProfileRepository(businessType string, options ...PaymentProfileRepositoryOption) paymentprofilepb.PaymentProfileDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPaymentProfileRepository{
		businessType:    businessType,
		paymentProfiles: make(map[string]*paymentprofilepb.PaymentProfile),
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
func (r *MockPaymentProfileRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPaymentProfiles, err := datamock.LoadPaymentProfiles(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial payment profiles: %w", err)
	}

	// Convert and store each payment profile
	for _, rawPaymentProfile := range rawPaymentProfiles {
		if paymentProfile, err := r.mapToProtobufPaymentProfile(rawPaymentProfile); err == nil {
			r.paymentProfiles[paymentProfile.Id] = paymentProfile
		}
	}

	r.initialized = true
	return nil
}

// CreatePaymentProfile creates a new payment profile with stateful storage
func (r *MockPaymentProfileRepository) CreatePaymentProfile(ctx context.Context, req *paymentprofilepb.CreatePaymentProfileRequest) (*paymentprofilepb.CreatePaymentProfileResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create payment profile request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("payment profile data is required")
	}
	if req.Data.ClientId == "" {
		return nil, fmt.Errorf("payment profile client ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	paymentProfileID := fmt.Sprintf("payment-profile-%d-%d", now.UnixNano(), len(r.paymentProfiles))

	// Create new payment profile with proper timestamps and defaults
	newPaymentProfile := &paymentprofilepb.PaymentProfile{
		Id:                 paymentProfileID,
		ClientId:           req.Data.ClientId,
		PaymentMethodId:    req.Data.PaymentMethodId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.paymentProfiles[paymentProfileID] = newPaymentProfile

	return &paymentprofilepb.CreatePaymentProfileResponse{
		Data:    []*paymentprofilepb.PaymentProfile{newPaymentProfile},
		Success: true,
	}, nil
}

// ReadPaymentProfile retrieves a payment profile by ID from stateful storage
func (r *MockPaymentProfileRepository) ReadPaymentProfile(ctx context.Context, req *paymentprofilepb.ReadPaymentProfileRequest) (*paymentprofilepb.ReadPaymentProfileResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read payment profile request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated payment profiles)
	if paymentProfile, exists := r.paymentProfiles[req.Data.Id]; exists {
		return &paymentprofilepb.ReadPaymentProfileResponse{
			Data:    []*paymentprofilepb.PaymentProfile{paymentProfile},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("payment profile with ID '%s' not found", req.Data.Id)
}

// UpdatePaymentProfile updates an existing payment profile in stateful storage
func (r *MockPaymentProfileRepository) UpdatePaymentProfile(ctx context.Context, req *paymentprofilepb.UpdatePaymentProfileRequest) (*paymentprofilepb.UpdatePaymentProfileResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update payment profile request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify payment profile exists
	existingPaymentProfile, exists := r.paymentProfiles[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("payment profile with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPaymentProfile := &paymentprofilepb.PaymentProfile{
		Id:                 req.Data.Id,
		ClientId:           req.Data.ClientId,
		PaymentMethodId:    req.Data.PaymentMethodId,
		DateCreated:        existingPaymentProfile.DateCreated,       // Preserve original
		DateCreatedString:  existingPaymentProfile.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],                  // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.paymentProfiles[req.Data.Id] = updatedPaymentProfile

	return &paymentprofilepb.UpdatePaymentProfileResponse{
		Data:    []*paymentprofilepb.PaymentProfile{updatedPaymentProfile},
		Success: true,
	}, nil
}

// DeletePaymentProfile deletes a payment profile from stateful storage
func (r *MockPaymentProfileRepository) DeletePaymentProfile(ctx context.Context, req *paymentprofilepb.DeletePaymentProfileRequest) (*paymentprofilepb.DeletePaymentProfileResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete payment profile request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment profile ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify payment profile exists before deletion
	if _, exists := r.paymentProfiles[req.Data.Id]; !exists {
		return nil, fmt.Errorf("payment profile with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.paymentProfiles, req.Data.Id)

	return &paymentprofilepb.DeletePaymentProfileResponse{
		Success: true,
	}, nil
}

// ListPaymentProfiles retrieves all payment profiles from stateful storage
func (r *MockPaymentProfileRepository) ListPaymentProfiles(ctx context.Context, req *paymentprofilepb.ListPaymentProfilesRequest) (*paymentprofilepb.ListPaymentProfilesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of payment profiles
	paymentProfiles := make([]*paymentprofilepb.PaymentProfile, 0, len(r.paymentProfiles))
	for _, paymentProfile := range r.paymentProfiles {
		paymentProfiles = append(paymentProfiles, paymentProfile)
	}

	return &paymentprofilepb.ListPaymentProfilesResponse{
		Data:    paymentProfiles,
		Success: true,
	}, nil
}

// GetPaymentProfileListPageData retrieves payment profiles with advanced filtering, sorting, searching, and pagination
func (r *MockPaymentProfileRepository) GetPaymentProfileListPageData(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileListPageDataRequest,
) (*paymentprofilepb.GetPaymentProfileListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment profile list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of payment profiles
	paymentProfiles := make([]*paymentprofilepb.PaymentProfile, 0, len(r.paymentProfiles))
	for _, paymentProfile := range r.paymentProfiles {
		paymentProfiles = append(paymentProfiles, paymentProfile)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		paymentProfiles,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process payment profile list data: %w", err)
	}

	// Convert processed items back to payment profile protobuf format
	processedPaymentProfiles := make([]*paymentprofilepb.PaymentProfile, len(result.Items))
	for i, item := range result.Items {
		if paymentProfile, ok := item.(*paymentprofilepb.PaymentProfile); ok {
			processedPaymentProfiles[i] = paymentProfile
		} else {
			return nil, fmt.Errorf("failed to convert item to payment profile type")
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

	return &paymentprofilepb.GetPaymentProfileListPageDataResponse{
		PaymentProfileList: processedPaymentProfiles,
		Pagination:         result.PaginationResponse,
		SearchResults:      searchResults,
		Success:            true,
	}, nil
}

// GetPaymentProfileItemPageData retrieves a single payment profile with enhanced item page data
func (r *MockPaymentProfileRepository) GetPaymentProfileItemPageData(
	ctx context.Context,
	req *paymentprofilepb.GetPaymentProfileItemPageDataRequest,
) (*paymentprofilepb.GetPaymentProfileItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment profile item page data request is required")
	}
	if req.PaymentProfileId == "" {
		return nil, fmt.Errorf("payment profile ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	paymentProfile, exists := r.paymentProfiles[req.PaymentProfileId]
	if !exists {
		return nil, fmt.Errorf("payment profile with ID '%s' not found", req.PaymentProfileId)
	}

	// In a real implementation, you might:
	// 1. Load related data (client details, payment method details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &paymentprofilepb.GetPaymentProfileItemPageDataResponse{
		PaymentProfile: paymentProfile,
		Success:        true,
	}, nil
}

// mapToProtobufPaymentProfile converts raw mock data to protobuf PaymentProfile
func (r *MockPaymentProfileRepository) mapToProtobufPaymentProfile(rawPaymentProfile map[string]any) (*paymentprofilepb.PaymentProfile, error) {
	paymentProfile := &paymentprofilepb.PaymentProfile{}

	if id, ok := rawPaymentProfile["id"].(string); ok {
		paymentProfile.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	// Map ClientId - try different field names for compatibility
	if clientId, ok := rawPaymentProfile["clientId"].(string); ok {
		paymentProfile.ClientId = clientId
	} else if userId, ok := rawPaymentProfile["userId"].(string); ok {
		// Map userId to clientId for mock data compatibility
		paymentProfile.ClientId = userId
	}

	// Map PaymentMethodId - try different field names for compatibility
	if paymentMethodId, ok := rawPaymentProfile["paymentMethodId"].(string); ok {
		paymentProfile.PaymentMethodId = paymentMethodId
	} else if defaultPaymentMethodId, ok := rawPaymentProfile["defaultPaymentMethodId"].(string); ok {
		// Map defaultPaymentMethodId to PaymentMethodId for mock data compatibility
		paymentProfile.PaymentMethodId = defaultPaymentMethodId
	}

	if dateCreated, ok := rawPaymentProfile["dateCreated"].(string); ok {
		paymentProfile.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			paymentProfile.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPaymentProfile["dateModified"].(string); ok {
		paymentProfile.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			paymentProfile.DateModified = &timestamp
		}
	}

	if active, ok := rawPaymentProfile["active"].(bool); ok {
		paymentProfile.Active = active
	} else {
		// Default to active if not specified
		paymentProfile.Active = true
	}

	return paymentProfile, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPaymentProfileRepository) parseTimestamp(timestampStr string) (int64, error) {
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

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

// NewPaymentProfileRepository creates a new payment profile repository - Provider interface compatibility
func NewPaymentProfileRepository(businessType string) paymentprofilepb.PaymentProfileDomainServiceServer {
	return NewMockPaymentProfileRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "payment_profile", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPaymentProfileRepository(businessType), nil
	})
}
