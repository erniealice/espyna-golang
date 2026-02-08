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
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// MockAdminRepository implements entity.AdminRepository using stateful mock data
type MockAdminRepository struct {
	adminpb.UnimplementedAdminDomainServiceServer
	businessType string
	admins       map[string]*adminpb.Admin // Persistent in-memory store
	mutex        sync.RWMutex              // Thread-safe concurrent access
	initialized  bool                      // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing utilities
}

// AdminRepositoryOption allows configuration of repository behavior
type AdminRepositoryOption func(*MockAdminRepository)

// WithAdminTestOptimizations enables test-specific optimizations
func WithAdminTestOptimizations(enabled bool) AdminRepositoryOption {
	return func(r *MockAdminRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockAdminRepository creates a new mock admin repository
func NewMockAdminRepository(businessType string, options ...AdminRepositoryOption) adminpb.AdminDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockAdminRepository{
		businessType: businessType,
		admins:       make(map[string]*adminpb.Admin),
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
func (r *MockAdminRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawAdmins, err := datamock.LoadAdmins(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial admins: %w", err)
	}

	// Convert and store each admin
	for _, rawAdmin := range rawAdmins {
		if admin, err := r.mapToProtobufAdmin(rawAdmin); err == nil {
			r.admins[admin.Id] = admin
		}
	}

	r.initialized = true
	return nil
}

// CreateAdmin creates a new admin with stateful storage
func (r *MockAdminRepository) CreateAdmin(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create admin request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("admin data is required")
	}
	// Note: UserId may be empty before enrichment, but User data should be present
	if req.Data.User == nil {
		return nil, fmt.Errorf("admin user data is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	adminID := fmt.Sprintf("admin-%d-%d", now.UnixNano(), len(r.admins))

	// Create new admin with proper timestamps and defaults
	newAdmin := &adminpb.Admin{
		Id:                 adminID,
		UserId:             req.Data.UserId,
		User:               req.Data.User, // Copy the entire User object
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.admins[adminID] = newAdmin

	return &adminpb.CreateAdminResponse{
		Data:    []*adminpb.Admin{newAdmin},
		Success: true,
	}, nil
}

// ReadAdmin retrieves an admin by ID from stateful storage
func (r *MockAdminRepository) ReadAdmin(ctx context.Context, req *adminpb.ReadAdminRequest) (*adminpb.ReadAdminResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read admin request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated admins)
	if admin, exists := r.admins[req.Data.Id]; exists {
		return &adminpb.ReadAdminResponse{
			Data:    []*adminpb.Admin{admin},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("admin with ID '%s' not found", req.Data.Id)
}

// UpdateAdmin updates an existing admin in stateful storage
func (r *MockAdminRepository) UpdateAdmin(ctx context.Context, req *adminpb.UpdateAdminRequest) (*adminpb.UpdateAdminResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update admin request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify admin exists
	existingAdmin, exists := r.admins[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("admin with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedAdmin := &adminpb.Admin{
		Id:                 req.Data.Id,
		UserId:             req.Data.UserId,
		User:               req.Data.User, // Copy the entire User object
		DateCreated:        existingAdmin.DateCreated,       // Preserve original
		DateCreatedString:  existingAdmin.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],    // Use millisecond precision for better timestamp granularity
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.admins[req.Data.Id] = updatedAdmin

	return &adminpb.UpdateAdminResponse{
		Data:    []*adminpb.Admin{updatedAdmin},
		Success: true,
	}, nil
}

// DeleteAdmin deletes an admin from stateful storage
func (r *MockAdminRepository) DeleteAdmin(ctx context.Context, req *adminpb.DeleteAdminRequest) (*adminpb.DeleteAdminResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete admin request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("admin ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify admin exists before deletion
	if _, exists := r.admins[req.Data.Id]; !exists {
		return nil, fmt.Errorf("admin with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.admins, req.Data.Id)

	return &adminpb.DeleteAdminResponse{
		Success: true,
	}, nil
}

// ListAdmins retrieves all admins from stateful storage
func (r *MockAdminRepository) ListAdmins(ctx context.Context, req *adminpb.ListAdminsRequest) (*adminpb.ListAdminsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of admins
	admins := make([]*adminpb.Admin, 0, len(r.admins))
	for _, admin := range r.admins {
		admins = append(admins, admin)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		admins,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process admin list data: %w", err)
	}

	// Convert processed items back to admin protobuf format
	processedAdmins := make([]*adminpb.Admin, len(result.Items))
	for i, item := range result.Items {
		if admin, ok := item.(*adminpb.Admin); ok {
			processedAdmins[i] = admin
		} else {
			return nil, fmt.Errorf("failed to convert item to admin type")
		}
	}

	return &adminpb.ListAdminsResponse{
		Data:    processedAdmins,
		Success: true,
	}, nil
}

// GetAdminListPageData retrieves admins with advanced filtering, sorting, searching, and pagination
func (r *MockAdminRepository) GetAdminListPageData(
	ctx context.Context,
	req *adminpb.GetAdminListPageDataRequest,
) (*adminpb.GetAdminListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get admin list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of admins
	admins := make([]*adminpb.Admin, 0, len(r.admins))
	for _, admin := range r.admins {
		admins = append(admins, admin)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		admins,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process admin list data: %w", err)
	}

	// Convert processed items back to admin protobuf format
	processedAdmins := make([]*adminpb.Admin, len(result.Items))
	for i, item := range result.Items {
		if admin, ok := item.(*adminpb.Admin); ok {
			processedAdmins[i] = admin
		} else {
			return nil, fmt.Errorf("failed to convert item to admin type")
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

	return &adminpb.GetAdminListPageDataResponse{
		AdminList:     processedAdmins,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetAdminItemPageData retrieves a single admin with enhanced item page data
func (r *MockAdminRepository) GetAdminItemPageData(
	ctx context.Context,
	req *adminpb.GetAdminItemPageDataRequest,
) (*adminpb.GetAdminItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get admin item page data request is required")
	}
	if req.AdminId == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	admin, exists := r.admins[req.AdminId]
	if !exists {
		return nil, fmt.Errorf("admin with ID '%s' not found", req.AdminId)
	}

	// In a real implementation, you might:
	// 1. Load related data (role details, user details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &adminpb.GetAdminItemPageDataResponse{
		Admin:   admin,
		Success: true,
	}, nil
}

// mapToProtobufAdmin converts raw mock data to protobuf Admin
func (r *MockAdminRepository) mapToProtobufAdmin(rawAdmin map[string]any) (*adminpb.Admin, error) {
	admin := &adminpb.Admin{}

	// Map required fields
	if id, ok := rawAdmin["id"].(string); ok {
		admin.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if userId, ok := rawAdmin["userId"].(string); ok {
		admin.UserId = userId
	} else {
		return nil, fmt.Errorf("missing or invalid userId field")
	}

	// Create User object from mock data fields
	if name, hasName := rawAdmin["name"].(string); hasName {
		if email, hasEmail := rawAdmin["email"].(string); hasEmail {
			// Parse full name into first and last name
			nameParts := strings.Fields(strings.TrimSpace(name))
			firstName := nameParts[0]
			lastName := ""
			if len(nameParts) > 1 {
				lastName = strings.Join(nameParts[1:], " ")
			}

			admin.User = &userpb.User{
				Id:           admin.UserId,
				FirstName:    firstName,
				LastName:     lastName,
				EmailAddress: email,
				Active:       true,
			}

			// Set user timestamps if available
			if dateCreated, ok := rawAdmin["dateCreated"].(string); ok {
				admin.User.DateCreatedString = &dateCreated
				if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
					admin.User.DateCreated = &timestamp
				}
			}

			if dateModified, ok := rawAdmin["dateModified"].(string); ok {
				admin.User.DateModifiedString = &dateModified
				if timestamp, err := r.parseTimestamp(dateModified); err == nil {
					admin.User.DateModified = &timestamp
				}
			}
		}
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawAdmin["dateCreated"].(string); ok {
		admin.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			admin.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawAdmin["dateModified"].(string); ok {
		admin.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			admin.DateModified = &timestamp
		}
	}

	if active, ok := rawAdmin["active"].(bool); ok {
		admin.Active = active
	}

	return admin, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockAdminRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewAdminRepository creates a new admin repository - Provider interface compatibility
func NewAdminRepository(businessType string) adminpb.AdminDomainServiceServer {
	return NewMockAdminRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "admin", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockAdminRepository(businessType), nil
	})
}
