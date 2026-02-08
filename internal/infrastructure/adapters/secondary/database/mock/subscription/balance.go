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
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "balance", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockBalanceRepository(businessType), nil
	})
}

// MockBalanceRepository implements subscription.BalanceRepository using stateful mock data
type MockBalanceRepository struct {
	balancepb.UnimplementedBalanceDomainServiceServer
	businessType string
	balances     map[string]*balancepb.Balance // Persistent in-memory store
	mutex        sync.RWMutex                  // Thread-safe concurrent access
	initialized  bool                          // Prevent double initialization
	processor    *listdata.ListDataProcessor   // List data processing utilities
}

// BalanceRepositoryOption allows configuration of repository behavior
type BalanceRepositoryOption func(*MockBalanceRepository)

// WithBalanceTestOptimizations enables test-specific optimizations
func WithBalanceTestOptimizations(enabled bool) BalanceRepositoryOption {
	return func(r *MockBalanceRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockBalanceRepository creates a new mock balance repository
func NewMockBalanceRepository(businessType string, options ...BalanceRepositoryOption) balancepb.BalanceDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockBalanceRepository{
		businessType: businessType,
		balances:     make(map[string]*balancepb.Balance),
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
func (r *MockBalanceRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawBalances, err := datamock.LoadBusinessTypeModule(r.businessType, "balance")
	if err != nil {
		return fmt.Errorf("failed to load initial balances: %w", err)
	}

	// Convert and store each balance
	for _, rawBalance := range rawBalances {
		if balance, err := r.mapToProtobufBalance(rawBalance); err == nil {
			r.balances[balance.Id] = balance
		}
	}

	r.initialized = true
	return nil
}

// CreateBalance creates a new balance with stateful storage
func (r *MockBalanceRepository) CreateBalance(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create balance request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("balance data is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	balanceID := fmt.Sprintf("balance-%d-%d", now.UnixNano(), len(r.balances))

	// Create new balance with proper timestamps and defaults
	newBalance := &balancepb.Balance{
		Id:                 balanceID,
		Amount:             req.Data.Amount,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.balances[balanceID] = newBalance

	return &balancepb.CreateBalanceResponse{
		Data:    []*balancepb.Balance{newBalance},
		Success: true,
	}, nil
}

// ReadBalance retrieves a balance by ID from stateful storage
func (r *MockBalanceRepository) ReadBalance(ctx context.Context, req *balancepb.ReadBalanceRequest) (*balancepb.ReadBalanceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read balance request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated balances)
	if balance, exists := r.balances[req.Data.Id]; exists {
		return &balancepb.ReadBalanceResponse{
			Data:    []*balancepb.Balance{balance},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("balance with ID '%s' not found", req.Data.Id)
}

// UpdateBalance updates an existing balance in stateful storage
func (r *MockBalanceRepository) UpdateBalance(ctx context.Context, req *balancepb.UpdateBalanceRequest) (*balancepb.UpdateBalanceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update balance request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify balance exists
	existingBalance, exists := r.balances[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("balance with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedBalance := &balancepb.Balance{
		Id:                 req.Data.Id,
		Amount:             req.Data.Amount,
		DateCreated:        existingBalance.DateCreated,       // Preserve original
		DateCreatedString:  existingBalance.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],           // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.balances[req.Data.Id] = updatedBalance

	return &balancepb.UpdateBalanceResponse{
		Data:    []*balancepb.Balance{updatedBalance},
		Success: true,
	}, nil
}

// DeleteBalance deletes a balance from stateful storage
func (r *MockBalanceRepository) DeleteBalance(ctx context.Context, req *balancepb.DeleteBalanceRequest) (*balancepb.DeleteBalanceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete balance request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify balance exists before deletion
	if _, exists := r.balances[req.Data.Id]; !exists {
		return nil, fmt.Errorf("balance with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.balances, req.Data.Id)

	return &balancepb.DeleteBalanceResponse{
		Success: true,
	}, nil
}

// ListBalances retrieves all balances from stateful storage
func (r *MockBalanceRepository) ListBalances(ctx context.Context, req *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of balances
	balances := make([]*balancepb.Balance, 0, len(r.balances))
	for _, balance := range r.balances {
		balances = append(balances, balance)
	}

	return &balancepb.ListBalancesResponse{
		Data:    balances,
		Success: true,
	}, nil
}

// GetBalanceListPageData retrieves balances with advanced filtering, sorting, searching, and pagination
func (r *MockBalanceRepository) GetBalanceListPageData(
	ctx context.Context,
	req *balancepb.GetBalanceListPageDataRequest,
) (*balancepb.GetBalanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get balance list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of balances
	balances := make([]*balancepb.Balance, 0, len(r.balances))
	for _, balance := range r.balances {
		balances = append(balances, balance)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		balances,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process balance list data: %w", err)
	}

	// Convert processed items back to balance protobuf format
	processedBalances := make([]*balancepb.Balance, len(result.Items))
	for i, item := range result.Items {
		if balance, ok := item.(*balancepb.Balance); ok {
			processedBalances[i] = balance
		} else {
			return nil, fmt.Errorf("failed to convert item to balance type")
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

	return &balancepb.GetBalanceListPageDataResponse{
		BalanceList:   processedBalances,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetBalanceItemPageData retrieves a single balance with enhanced item page data
func (r *MockBalanceRepository) GetBalanceItemPageData(
	ctx context.Context,
	req *balancepb.GetBalanceItemPageDataRequest,
) (*balancepb.GetBalanceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get balance item page data request is required")
	}
	if req.BalanceId == "" {
		return nil, fmt.Errorf("balance ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	balance, exists := r.balances[req.BalanceId]
	if !exists {
		return nil, fmt.Errorf("balance with ID '%s' not found", req.BalanceId)
	}

	// In a real implementation, you might:
	// 1. Load related data (client details, subscription details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &balancepb.GetBalanceItemPageDataResponse{
		Balance: balance,
		Success: true,
	}, nil
}

// mapToProtobufBalance converts raw mock data to protobuf Balance
func (r *MockBalanceRepository) mapToProtobufBalance(rawBalance map[string]any) (*balancepb.Balance, error) {
	balance := &balancepb.Balance{}

	// Map required fields
	if id, ok := rawBalance["id"].(string); ok {
		balance.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if amount, ok := rawBalance["amount"].(float64); ok {
		balance.Amount = amount
	} else {
		return nil, fmt.Errorf("missing or invalid amount field")
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawBalance["dateCreated"].(string); ok {
		balance.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			balance.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawBalance["dateModified"].(string); ok {
		balance.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			balance.DateModified = &timestamp
		}
	}

	if active, ok := rawBalance["active"].(bool); ok {
		balance.Active = active
	}

	return balance, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockBalanceRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewBalanceRepository creates a new mock balance repository (registry constructor)
func NewBalanceRepository(data map[string]*balancepb.Balance) balancepb.BalanceDomainServiceServer {
	repo := &MockBalanceRepository{
		businessType: "education", // Default business type
		balances:     data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.balances = make(map[string]*balancepb.Balance)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
