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
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// MockStaffRepository implements entity.StaffRepository using stateful mock data
type MockStaffRepository struct {
	staffpb.UnimplementedStaffDomainServiceServer
	businessType string
	staffs       map[string]*staffpb.Staff // Persistent in-memory store
	mutex        sync.RWMutex              // Thread-safe concurrent access
	initialized  bool                      // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing
}

// StaffRepositoryOption allows configuration of repository behavior
type StaffRepositoryOption func(*MockStaffRepository)

// WithStaffTestOptimizations enables test-specific optimizations
func WithStaffTestOptimizations(enabled bool) StaffRepositoryOption {
	return func(r *MockStaffRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockStaffRepository creates a new mock staff repository
func NewMockStaffRepository(businessType string, options ...StaffRepositoryOption) staffpb.StaffDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockStaffRepository{
		businessType: businessType,
		staffs:       make(map[string]*staffpb.Staff),
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
func (r *MockStaffRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawStaffs, err := datamock.LoadStaff(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial staff: %w", err)
	}

	// Convert and store each staff
	for _, rawStaff := range rawStaffs {
		if staff, err := r.mapToProtobuf(rawStaff); err == nil {
			r.staffs[staff.Id] = staff
		}
	}

	r.initialized = true
	return nil
}

// CreateStaff creates a new staff with stateful storage
func (r *MockStaffRepository) CreateStaff(ctx context.Context, req *staffpb.CreateStaffRequest) (*staffpb.CreateStaffResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create staff request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("staff data is required")
	}
	if req.Data.UserId == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	staffID := fmt.Sprintf("staff-%d-%d", now.UnixNano(), len(r.staffs))

	// Create new staff with proper timestamps and defaults
	newStaff := &staffpb.Staff{
		Id:                 staffID,
		UserId:             req.Data.UserId,
		User:               req.Data.User, // Preserve the User object from request
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}
	
	// If User object exists, ensure it has an ID
	if newStaff.User != nil && newStaff.User.Id == "" {
		userID := fmt.Sprintf("user-%d", now.UnixNano())
		newStaff.User.Id = userID
		newStaff.UserId = userID // Keep UserId in sync
	}

	// Store in persistent map
	r.staffs[staffID] = newStaff

	return &staffpb.CreateStaffResponse{
		Data:    []*staffpb.Staff{newStaff},
		Success: true,
	}, nil
}

// ReadStaff retrieves a staff by ID from stateful storage
func (r *MockStaffRepository) ReadStaff(ctx context.Context, req *staffpb.ReadStaffRequest) (*staffpb.ReadStaffResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read staff request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated staff)
	if staff, exists := r.staffs[req.Data.Id]; exists {
		return &staffpb.ReadStaffResponse{
			Data:    []*staffpb.Staff{staff},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("staff with ID '%s' not found", req.Data.Id)
}

// UpdateStaff updates an existing staff in stateful storage
func (r *MockStaffRepository) UpdateStaff(ctx context.Context, req *staffpb.UpdateStaffRequest) (*staffpb.UpdateStaffResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update staff request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify staff exists
	existingStaff, exists := r.staffs[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("staff with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedStaff := &staffpb.Staff{
		Id:                 req.Data.Id,
		UserId:             req.Data.UserId,
		User:               req.Data.User, // Preserve/update User object
		DateCreated:        existingStaff.DateCreated,       // Preserve original
		DateCreatedString:  existingStaff.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],         // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}
	
	// If User object exists, ensure it has an ID and sync UserId
	if updatedStaff.User != nil {
		if updatedStaff.User.Id == "" && existingStaff.User != nil {
			updatedStaff.User.Id = existingStaff.User.Id // Preserve existing User ID
		}
		if updatedStaff.User.Id != "" {
			updatedStaff.UserId = updatedStaff.User.Id // Keep UserId in sync
		}
	} else if existingStaff.User != nil {
		updatedStaff.User = existingStaff.User // Preserve existing User if none provided
	}

	// Update in persistent store
	r.staffs[req.Data.Id] = updatedStaff

	return &staffpb.UpdateStaffResponse{
		Data:    []*staffpb.Staff{updatedStaff},
		Success: true,
	}, nil
}

// DeleteStaff deletes a staff from stateful storage
func (r *MockStaffRepository) DeleteStaff(ctx context.Context, req *staffpb.DeleteStaffRequest) (*staffpb.DeleteStaffResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete staff request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("staff ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify staff exists before deletion
	if _, exists := r.staffs[req.Data.Id]; !exists {
		return nil, fmt.Errorf("staff with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.staffs, req.Data.Id)

	return &staffpb.DeleteStaffResponse{
		Success: true,
	}, nil
}

// ListStaffs retrieves all staffs from stateful storage
func (r *MockStaffRepository) ListStaffs(ctx context.Context, req *staffpb.ListStaffsRequest) (*staffpb.ListStaffsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of staffs
	staffs := make([]*staffpb.Staff, 0, len(r.staffs))
	for _, staff := range r.staffs {
		staffs = append(staffs, staff)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		staffs,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process staff list data: %w", err)
	}

	// Convert processed items back to staff protobuf format
	processedStaffs := make([]*staffpb.Staff, len(result.Items))
	for i, item := range result.Items {
		if staff, ok := item.(*staffpb.Staff); ok {
			processedStaffs[i] = staff
		} else {
			return nil, fmt.Errorf("failed to convert item to staff type")
		}
	}

	return &staffpb.ListStaffsResponse{
		Data:    processedStaffs,
		Success: true,
	}, nil
}

// GetStaffListPageData retrieves staffs with advanced filtering, sorting, searching, and pagination
func (r *MockStaffRepository) GetStaffListPageData(
	ctx context.Context,
	req *staffpb.GetStaffListPageDataRequest,
) (*staffpb.GetStaffListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get staff list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of staffs
	staffs := make([]*staffpb.Staff, 0, len(r.staffs))
	for _, staff := range r.staffs {
		staffs = append(staffs, staff)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		staffs,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process staff list data: %w", err)
	}

	// Convert processed items back to staff protobuf format
	processedStaffs := make([]*staffpb.Staff, len(result.Items))
	for i, item := range result.Items {
		if staff, ok := item.(*staffpb.Staff); ok {
			processedStaffs[i] = staff
		} else {
			return nil, fmt.Errorf("failed to convert item to staff type")
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

	return &staffpb.GetStaffListPageDataResponse{
		StaffList:     processedStaffs,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetStaffItemPageData retrieves a single staff with enhanced item page data
func (r *MockStaffRepository) GetStaffItemPageData(
	ctx context.Context,
	req *staffpb.GetStaffItemPageDataRequest,
) (*staffpb.GetStaffItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get staff item page data request is required")
	}
	if req.StaffId == "" {
		return nil, fmt.Errorf("staff ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	staff, exists := r.staffs[req.StaffId]
	if !exists {
		return nil, fmt.Errorf("staff with ID '%s' not found", req.StaffId)
	}

	// In a real implementation, you might:
	// 1. Load related data (user details, role details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &staffpb.GetStaffItemPageDataResponse{
		Staff:   staff,
		Success: true,
	}, nil
}

func (r *MockStaffRepository) mapToProtobuf(raw map[string]any) (*staffpb.Staff, error) {
	staff := &staffpb.Staff{}
	if id, ok := raw["id"].(string); ok {
		staff.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}
	if userId, ok := raw["userId"].(string); ok {
		staff.UserId = userId
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

			staff.User = &userpb.User{
				Id:           staff.UserId,
				FirstName:    firstName,
				LastName:     lastName,
				EmailAddress: email,
				Active:       true,
			}

			// Set user timestamps if available
			if dateCreated, ok := raw["dateCreated"].(string); ok {
				staff.User.DateCreatedString = &dateCreated
				if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
					staff.User.DateCreated = &timestamp
				}
			}

			if dateModified, ok := raw["dateModified"].(string); ok {
				staff.User.DateModifiedString = &dateModified
				if timestamp, err := r.parseTimestamp(dateModified); err == nil {
					staff.User.DateModified = &timestamp
				}
			}
		}
	}
	if dateCreated, ok := raw["dateCreated"].(string); ok {
		staff.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			staff.DateCreated = &timestamp
		}
	}
	if dateModified, ok := raw["dateModified"].(string); ok {
		staff.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			staff.DateModified = &timestamp
		}
	}
	if active, ok := raw["active"].(bool); ok {
		staff.Active = active
	}
	return staff, nil
}

func (r *MockStaffRepository) parseTimestamp(timestampStr string) (int64, error) {
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}
	formats := []string{"2006-01-02T15:04:05Z", "2006-01-02 15:04:05", "2006-01-02T15:04:05.000Z"}
	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// NewStaffRepository creates a new staff repository - Provider interface compatibility
func NewStaffRepository(businessType string) staffpb.StaffDomainServiceServer {
	return NewMockStaffRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "staff", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockStaffRepository(businessType), nil
	})
}
