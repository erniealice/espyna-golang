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
	workspacepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace"
)

// MockWorkspaceRepository implements entity.WorkspaceRepository using stateful mock data
type MockWorkspaceRepository struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	businessType string
	workspaces   map[string]*workspacepb.Workspace // Persistent in-memory store
	mutex        sync.RWMutex                      // Thread-safe concurrent access
	initialized  bool                              // Prevent double initialization
	processor    *listdata.ListDataProcessor       // List data processing utilities
}

// WorkspaceRepositoryOption allows configuration of repository behavior
type WorkspaceRepositoryOption func(*MockWorkspaceRepository)

// WithWorkspaceTestOptimizations enables test-specific optimizations
func WithWorkspaceTestOptimizations(enabled bool) WorkspaceRepositoryOption {
	return func(r *MockWorkspaceRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockWorkspaceRepository creates a new mock workspace repository
func NewMockWorkspaceRepository(businessType string, options ...WorkspaceRepositoryOption) workspacepb.WorkspaceDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockWorkspaceRepository{
		businessType: businessType,
		workspaces:   make(map[string]*workspacepb.Workspace),
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
func (r *MockWorkspaceRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawWorkspaces, err := datamock.LoadWorkspaces(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial workspaces: %w", err)
	}

	// Convert and store each workspace
	for _, rawWorkspace := range rawWorkspaces {
		if workspace, err := r.mapToProtobufWorkspace(rawWorkspace); err == nil {
			r.workspaces[workspace.Id] = workspace
		}
	}

	r.initialized = true
	return nil
}

// CreateWorkspace creates a new workspace with stateful storage
func (r *MockWorkspaceRepository) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create workspace request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("workspace data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("workspace name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	workspaceID := fmt.Sprintf("workspace-%d-%d", now.UnixNano(), len(r.workspaces))

	// Create new workspace with proper timestamps and defaults
	newWorkspace := &workspacepb.Workspace{
		Id:                 workspaceID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Private:            req.Data.Private,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.workspaces[workspaceID] = newWorkspace

	return &workspacepb.CreateWorkspaceResponse{
		Data:    []*workspacepb.Workspace{newWorkspace},
		Success: true,
	}, nil
}

// ReadWorkspace retrieves a workspace by ID from stateful storage
func (r *MockWorkspaceRepository) ReadWorkspace(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read workspace request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated workspaces)
	if workspace, exists := r.workspaces[req.Data.Id]; exists {
		return &workspacepb.ReadWorkspaceResponse{
			Data:    []*workspacepb.Workspace{workspace},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("workspace with ID '%s' not found", req.Data.Id)
}

// UpdateWorkspace updates an existing workspace in stateful storage
func (r *MockWorkspaceRepository) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update workspace request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workspace exists
	existingWorkspace, exists := r.workspaces[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("workspace with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedWorkspace := &workspacepb.Workspace{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Private:            req.Data.Private,
		DateCreated:        existingWorkspace.DateCreated,       // Preserve original
		DateCreatedString:  existingWorkspace.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],             // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.workspaces[req.Data.Id] = updatedWorkspace

	return &workspacepb.UpdateWorkspaceResponse{
		Data:    []*workspacepb.Workspace{updatedWorkspace},
		Success: true,
	}, nil
}

// DeleteWorkspace deletes a workspace from stateful storage
func (r *MockWorkspaceRepository) DeleteWorkspace(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete workspace request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workspace exists before deletion
	if _, exists := r.workspaces[req.Data.Id]; !exists {
		return nil, fmt.Errorf("workspace with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.workspaces, req.Data.Id)

	return &workspacepb.DeleteWorkspaceResponse{
		Success: true,
	}, nil
}

// ListWorkspaces retrieves all workspaces from stateful storage
func (r *MockWorkspaceRepository) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workspaces
	items := make([]*workspacepb.Workspace, 0, len(r.workspaces))
	for _, workspace := range r.workspaces {
		items = append(items, workspace)
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
	processed := make([]*workspacepb.Workspace, len(result.Items))
	for i, item := range result.Items {
		if typed, ok := item.(*workspacepb.Workspace); ok {
			processed[i] = typed
		}
	}

	return &workspacepb.ListWorkspacesResponse{
		Data:    processed,
		Success: true,
	}, nil
}

// mapToProtobufWorkspace converts raw mock data to protobuf Workspace
func (r *MockWorkspaceRepository) mapToProtobufWorkspace(rawWorkspace map[string]any) (*workspacepb.Workspace, error) {
	workspace := &workspacepb.Workspace{}

	// Map required fields
	if id, ok := rawWorkspace["id"].(string); ok {
		workspace.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawWorkspace["name"].(string); ok {
		workspace.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if description, ok := rawWorkspace["description"].(string); ok {
		workspace.Description = description
	}

	if private, ok := rawWorkspace["private"].(bool); ok {
		workspace.Private = private
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawWorkspace["dateCreated"].(string); ok {
		workspace.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			workspace.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawWorkspace["dateModified"].(string); ok {
		workspace.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			workspace.DateModified = &timestamp
		}
	}

	if active, ok := rawWorkspace["active"].(bool); ok {
		workspace.Active = active
	}

	return workspace, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockWorkspaceRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// GetWorkspaceListPageData retrieves workspaces with advanced filtering, sorting, searching, and pagination
func (r *MockWorkspaceRepository) GetWorkspaceListPageData(
	ctx context.Context,
	req *workspacepb.GetWorkspaceListPageDataRequest,
) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workspaces
	workspaces := make([]*workspacepb.Workspace, 0, len(r.workspaces))
	for _, workspace := range r.workspaces {
		workspaces = append(workspaces, workspace)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		workspaces,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process workspace list data: %w", err)
	}

	// Convert processed items back to workspace protobuf format
	processedWorkspaces := make([]*workspacepb.Workspace, len(result.Items))
	for i, item := range result.Items {
		if workspace, ok := item.(*workspacepb.Workspace); ok {
			processedWorkspaces[i] = workspace
		} else {
			return nil, fmt.Errorf("failed to convert item to workspace type")
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

	return &workspacepb.GetWorkspaceListPageDataResponse{
		WorkspaceList: processedWorkspaces,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetWorkspaceItemPageData retrieves a single workspace with enhanced item page data
func (r *MockWorkspaceRepository) GetWorkspaceItemPageData(
	ctx context.Context,
	req *workspacepb.GetWorkspaceItemPageDataRequest,
) (*workspacepb.GetWorkspaceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace item page data request is required")
	}
	if req.WorkspaceId == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	workspace, exists := r.workspaces[req.WorkspaceId]
	if !exists {
		return nil, fmt.Errorf("workspace with ID '%s' not found", req.WorkspaceId)
	}

	// In a real implementation, you might:
	// 1. Load related data (member details, project details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &workspacepb.GetWorkspaceItemPageDataResponse{
		Workspace: workspace,
		Success:   true,
	}, nil
}

// NewWorkspaceRepository creates a new workspace repository - Provider interface compatibility
func NewWorkspaceRepository(businessType string) workspacepb.WorkspaceDomainServiceServer {
	return NewMockWorkspaceRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "workspace", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockWorkspaceRepository(businessType), nil
	})
}
