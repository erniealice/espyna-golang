//go:build mock_db

package subscription

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
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "license_history", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockLicenseHistoryRepository(businessType), nil
	})
}

// MockLicenseHistoryRepository implements license_history.LicenseHistoryRepository using stateful mock data
type MockLicenseHistoryRepository struct {
	licensehistorypb.UnimplementedLicenseHistoryDomainServiceServer
	businessType    string
	licenseHistories map[string]*licensehistorypb.LicenseHistory // Persistent in-memory store
	mutex           sync.RWMutex                                 // Thread-safe concurrent access
	initialized     bool                                         // Prevent double initialization
	processor       *listdata.ListDataProcessor                  // List data processing utilities
}

// LicenseHistoryRepositoryOption allows configuration of repository behavior
type LicenseHistoryRepositoryOption func(*MockLicenseHistoryRepository)

// WithLicenseHistoryTestOptimizations enables test-specific optimizations
func WithLicenseHistoryTestOptimizations(enabled bool) LicenseHistoryRepositoryOption {
	return func(r *MockLicenseHistoryRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockLicenseHistoryRepository creates a new mock license history repository
func NewMockLicenseHistoryRepository(businessType string, options ...LicenseHistoryRepositoryOption) licensehistorypb.LicenseHistoryDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockLicenseHistoryRepository{
		businessType:    businessType,
		licenseHistories: make(map[string]*licensehistorypb.LicenseHistory),
		processor:       listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if err := repo.loadInitialData(); err != nil {
		// Log error but don't fail - allows graceful degradation
		fmt.Printf("Warning: Failed to load initial license history mock data: %v\n", err)
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockLicenseHistoryRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawHistories, err := datamock.LoadBusinessTypeModule(r.businessType, "license_history")
	if err != nil {
		// License history data may not exist yet, this is not a critical error
		r.initialized = true
		return nil
	}

	// Convert and store each license history
	for _, rawHistory := range rawHistories {
		if history, err := r.mapToProtobufLicenseHistory(rawHistory); err == nil {
			r.licenseHistories[history.Id] = history
		}
	}

	r.initialized = true
	return nil
}

// CreateLicenseHistory creates a new license history record with stateful storage
func (r *MockLicenseHistoryRepository) CreateLicenseHistory(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) (*licensehistorypb.CreateLicenseHistoryResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create license history request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("license history data is required")
	}
	if req.Data.LicenseId == "" {
		return nil, fmt.Errorf("license_id is required")
	}
	if req.Data.PerformedBy == "" {
		return nil, fmt.Errorf("performed_by is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	historyID := req.Data.Id
	if historyID == "" {
		historyID = fmt.Sprintf("license-history-%d-%d", now.UnixNano(), len(r.licenseHistories))
	}

	// Create new license history with proper timestamps and defaults
	newHistory := &licensehistorypb.LicenseHistory{
		Id:                   historyID,
		LicenseId:            req.Data.LicenseId,
		Action:               req.Data.Action,
		AssigneeId:           req.Data.AssigneeId,
		AssigneeType:         req.Data.AssigneeType,
		AssigneeName:         req.Data.AssigneeName,
		PreviousAssigneeId:   req.Data.PreviousAssigneeId,
		PreviousAssigneeType: req.Data.PreviousAssigneeType,
		PreviousAssigneeName: req.Data.PreviousAssigneeName,
		PerformedBy:          req.Data.PerformedBy,
		Reason:               req.Data.Reason,
		Notes:                req.Data.Notes,
		LicenseStatusBefore:  req.Data.LicenseStatusBefore,
		LicenseStatusAfter:   req.Data.LicenseStatusAfter,
		DateCreated:          now.UnixMilli(),
		DateCreatedString:    now.Format(time.RFC3339),
		Active:               true,
	}

	// Store in persistent map
	r.licenseHistories[historyID] = newHistory

	return &licensehistorypb.CreateLicenseHistoryResponse{
		Data:    []*licensehistorypb.LicenseHistory{newHistory},
		Success: true,
	}, nil
}

// ReadLicenseHistory retrieves a license history by ID from stateful storage
func (r *MockLicenseHistoryRepository) ReadLicenseHistory(ctx context.Context, req *licensehistorypb.ReadLicenseHistoryRequest) (*licensehistorypb.ReadLicenseHistoryResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read license history request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("license history ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	if history, exists := r.licenseHistories[req.Data.Id]; exists {
		return &licensehistorypb.ReadLicenseHistoryResponse{
			Data:    []*licensehistorypb.LicenseHistory{history},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("license history with ID '%s' not found", req.Data.Id)
}

// ListLicenseHistory retrieves all license history records from stateful storage
func (r *MockLicenseHistoryRepository) ListLicenseHistory(ctx context.Context, req *licensehistorypb.ListLicenseHistoryRequest) (*licensehistorypb.ListLicenseHistoryResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of license histories
	histories := make([]*licensehistorypb.LicenseHistory, 0, len(r.licenseHistories))
	for _, history := range r.licenseHistories {
		// Filter by license_id if provided
		if req != nil && req.LicenseId != nil && *req.LicenseId != "" {
			if history.LicenseId != *req.LicenseId {
				continue
			}
		}
		histories = append(histories, history)
	}

	return &licensehistorypb.ListLicenseHistoryResponse{
		Data:    histories,
		Success: true,
	}, nil
}

// GetLicenseHistoryListPageData retrieves license histories with advanced filtering, sorting, searching, and pagination
func (r *MockLicenseHistoryRepository) GetLicenseHistoryListPageData(
	ctx context.Context,
	req *licensehistorypb.GetLicenseHistoryListPageDataRequest,
) (*licensehistorypb.GetLicenseHistoryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get license history list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of license histories
	histories := make([]*licensehistorypb.LicenseHistory, 0, len(r.licenseHistories))
	for _, history := range r.licenseHistories {
		// Filter by license_id if provided
		if req.LicenseId != nil && *req.LicenseId != "" {
			if history.LicenseId != *req.LicenseId {
				continue
			}
		}
		histories = append(histories, history)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		histories,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process license history list data: %w", err)
	}

	// Convert processed items back to license history protobuf format
	processedHistories := make([]*licensehistorypb.LicenseHistory, len(result.Items))
	for i, item := range result.Items {
		if history, ok := item.(*licensehistorypb.LicenseHistory); ok {
			processedHistories[i] = history
		} else {
			return nil, fmt.Errorf("failed to convert item to license history type")
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

	return &licensehistorypb.GetLicenseHistoryListPageDataResponse{
		LicenseHistoryList: processedHistories,
		Pagination:         result.PaginationResponse,
		SearchResults:      searchResults,
		Success:            true,
	}, nil
}

// mapToProtobufLicenseHistory converts raw mock data to protobuf LicenseHistory
func (r *MockLicenseHistoryRepository) mapToProtobufLicenseHistory(rawHistory map[string]any) (*licensehistorypb.LicenseHistory, error) {
	history := &licensehistorypb.LicenseHistory{}

	// Map required fields
	if id, ok := rawHistory["id"].(string); ok {
		history.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if licenseId, ok := rawHistory["licenseId"].(string); ok {
		history.LicenseId = licenseId
	}

	if performedBy, ok := rawHistory["performedBy"].(string); ok {
		history.PerformedBy = performedBy
	}

	// Handle optional string fields
	if assigneeId, ok := rawHistory["assigneeId"].(string); ok {
		history.AssigneeId = &assigneeId
	}

	if assigneeType, ok := rawHistory["assigneeType"].(string); ok {
		history.AssigneeType = &assigneeType
	}

	if assigneeName, ok := rawHistory["assigneeName"].(string); ok {
		history.AssigneeName = &assigneeName
	}

	if previousAssigneeId, ok := rawHistory["previousAssigneeId"].(string); ok {
		history.PreviousAssigneeId = &previousAssigneeId
	}

	if previousAssigneeType, ok := rawHistory["previousAssigneeType"].(string); ok {
		history.PreviousAssigneeType = &previousAssigneeType
	}

	if previousAssigneeName, ok := rawHistory["previousAssigneeName"].(string); ok {
		history.PreviousAssigneeName = &previousAssigneeName
	}

	if reason, ok := rawHistory["reason"].(string); ok {
		history.Reason = &reason
	}

	if notes, ok := rawHistory["notes"].(string); ok {
		history.Notes = &notes
	}

	// Handle date fields
	if dateCreated, ok := rawHistory["dateCreated"].(string); ok {
		history.DateCreatedString = dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			history.DateCreated = timestamp
		}
	}

	if active, ok := rawHistory["active"].(bool); ok {
		history.Active = active
	}

	return history, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockLicenseHistoryRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewLicenseHistoryRepository creates a new mock license history repository (registry constructor)
func NewLicenseHistoryRepository(data map[string]*licensehistorypb.LicenseHistory) licensehistorypb.LicenseHistoryDomainServiceServer {
	repo := &MockLicenseHistoryRepository{
		businessType:    "education", // Default business type
		licenseHistories: data,
		mutex:           sync.RWMutex{},
		processor:       listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.licenseHistories = make(map[string]*licensehistorypb.LicenseHistory)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
