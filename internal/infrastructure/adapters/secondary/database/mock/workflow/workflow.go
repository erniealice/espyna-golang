//go:build mock_db

package workflow

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
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "workflow", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockWorkflowRepository(businessType), nil
	})
}

// MockWorkflowRepository implements workflow.WorkflowRepository using stateful mock data
type MockWorkflowRepository struct {
	workflowpb.UnimplementedWorkflowDomainServiceServer
	businessType string
	workflows    map[string]*workflowpb.Workflow // Persistent in-memory store
	mutex        sync.RWMutex                     // Thread-safe concurrent access
	initialized  bool                             // Prevent double initialization
	processor    *listdata.ListDataProcessor      // List data processing utilities
}

// WorkflowRepositoryOption allows configuration of repository behavior
type WorkflowRepositoryOption func(*MockWorkflowRepository)

// WithWorkflowTestOptimizations enables test-specific optimizations
func WithWorkflowTestOptimizations(enabled bool) WorkflowRepositoryOption {
	return func(r *MockWorkflowRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockWorkflowRepository creates a new mock workflow repository
func NewMockWorkflowRepository(businessType string, options ...WorkflowRepositoryOption) workflowpb.WorkflowDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockWorkflowRepository{
		businessType: businessType,
		workflows:    make(map[string]*workflowpb.Workflow),
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
func (r *MockWorkflowRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawWorkflows, err := datamock.LoadBusinessTypeModule(r.businessType, "workflow")
	if err != nil {
		return fmt.Errorf("failed to load initial workflows: %w", err)
	}

	// Convert and store each workflow
	for _, rawWorkflow := range rawWorkflows {
		if workflow, err := r.mapToProtobufWorkflow(rawWorkflow); err == nil {
			r.workflows[workflow.Id] = workflow
		}
	}

	r.initialized = true
	return nil
}

// CreateWorkflow creates a new workflow with stateful storage
func (r *MockWorkflowRepository) CreateWorkflow(ctx context.Context, req *workflowpb.CreateWorkflowRequest) (*workflowpb.CreateWorkflowResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create workflow request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("workflow data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	workflowID := fmt.Sprintf("workflow-%d-%d", now.UnixNano(), len(r.workflows))

	// Create new workflow with proper timestamps and defaults
	newWorkflow := &workflowpb.Workflow{
		Id:                 workflowID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		// Status:             workflowpb.WorkflowStatus_WORKFLOW_STATUS_DRAFT, // Default to draft - REMOVED
		WorkspaceId:        req.Data.WorkspaceId,
		CreatedBy:          req.Data.CreatedBy,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
		Version:            &[]int32{1}[0], // Start with version 1
	}

	// Store in persistent map
	r.workflows[workflowID] = newWorkflow

	return &workflowpb.CreateWorkflowResponse{
		Data:    []*workflowpb.Workflow{newWorkflow},
		Success: true,
	}, nil
}

// ReadWorkflow retrieves a workflow by ID from stateful storage
func (r *MockWorkflowRepository) ReadWorkflow(ctx context.Context, req *workflowpb.ReadWorkflowRequest) (*workflowpb.ReadWorkflowResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read workflow request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated workflows)
	if workflow, exists := r.workflows[req.Data.Id]; exists {
		return &workflowpb.ReadWorkflowResponse{
			Data:    []*workflowpb.Workflow{workflow},
			Success: true,
		}, nil
	}

	return &workflowpb.ReadWorkflowResponse{
		Data:    []*workflowpb.Workflow{},
		Success: false,
	}, nil
}

// UpdateWorkflow updates an existing workflow in stateful storage
func (r *MockWorkflowRepository) UpdateWorkflow(ctx context.Context, req *workflowpb.UpdateWorkflowRequest) (*workflowpb.UpdateWorkflowResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update workflow request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required for update")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workflow exists
	existingWorkflow, exists := r.workflows[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("workflow with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedWorkflow := &workflowpb.Workflow{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Status:             req.Data.Status,
		WorkspaceId:        req.Data.WorkspaceId,
		CreatedBy:          existingWorkflow.CreatedBy, // Preserve original creator
		DateCreated:        existingWorkflow.DateCreated,
		DateCreatedString:  existingWorkflow.DateCreatedString,
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
		Version:            req.Data.Version,
	}

	// Increment version if not specified
	if updatedWorkflow.Version == nil {
		newVersion := int32(1)
		if existingWorkflow.Version != nil {
			newVersion = *existingWorkflow.Version + 1
		}
		updatedWorkflow.Version = &newVersion
	}

	// Update in persistent store
	r.workflows[req.Data.Id] = updatedWorkflow

	return &workflowpb.UpdateWorkflowResponse{
		Data:    []*workflowpb.Workflow{updatedWorkflow},
		Success: true,
	}, nil
}

// DeleteWorkflow deletes a workflow from stateful storage
func (r *MockWorkflowRepository) DeleteWorkflow(ctx context.Context, req *workflowpb.DeleteWorkflowRequest) (*workflowpb.DeleteWorkflowResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete workflow request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workflow exists before deletion
	if _, exists := r.workflows[req.Data.Id]; !exists {
		return nil, fmt.Errorf("workflow with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.workflows, req.Data.Id)

	return &workflowpb.DeleteWorkflowResponse{
		Success: true,
	}, nil
}

// ListWorkflows retrieves all workflows from stateful storage
func (r *MockWorkflowRepository) ListWorkflows(ctx context.Context, req *workflowpb.ListWorkflowsRequest) (*workflowpb.ListWorkflowsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workflows
	workflows := make([]*workflowpb.Workflow, 0, len(r.workflows))
	for _, workflow := range r.workflows {
		workflows = append(workflows, workflow)
	}

	return &workflowpb.ListWorkflowsResponse{
		Data:    workflows,
		Success: true,
	}, nil
}

// GetWorkflowListPageData retrieves workflows with advanced filtering, sorting, searching, and pagination
func (r *MockWorkflowRepository) GetWorkflowListPageData(
	ctx context.Context,
	req *workflowpb.GetWorkflowListPageDataRequest,
) (*workflowpb.GetWorkflowListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workflow list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workflows
	workflows := make([]*workflowpb.Workflow, 0, len(r.workflows))
	for _, workflow := range r.workflows {
		workflows = append(workflows, workflow)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		workflows,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process workflow list data: %w", err)
	}

	// Convert processed items back to workflow protobuf format
	processedWorkflows := make([]*workflowpb.Workflow, len(result.Items))
	for i, item := range result.Items {
		if workflow, ok := item.(*workflowpb.Workflow); ok {
			processedWorkflows[i] = workflow
		} else {
			return nil, fmt.Errorf("failed to convert item to workflow type")
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

	return &workflowpb.GetWorkflowListPageDataResponse{
		WorkflowList:  processedWorkflows,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetWorkflowItemPageData retrieves a single workflow with enhanced item page data
func (r *MockWorkflowRepository) GetWorkflowItemPageData(
	ctx context.Context,
	req *workflowpb.GetWorkflowItemPageDataRequest,
) (*workflowpb.GetWorkflowItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workflow item page data request is required")
	}
	if req.WorkflowId == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	workflow, exists := r.workflows[req.WorkflowId]
	if !exists {
		return nil, fmt.Errorf("workflow with ID '%s' not found", req.WorkflowId)
	}

	// In a real implementation, you might:
	// 1. Load related data (stage templates, usage statistics)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &workflowpb.GetWorkflowItemPageDataResponse{
		Workflow: workflow,
		Success:  true,
	}, nil
}

// mapToProtobufWorkflow converts raw mock data to protobuf Workflow
func (r *MockWorkflowRepository) mapToProtobufWorkflow(rawWorkflow map[string]any) (*workflowpb.Workflow, error) {
	workflow := &workflowpb.Workflow{}

	// Map required fields
	if id, ok := rawWorkflow["id"].(string); ok {
		workflow.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawWorkflow["name"].(string); ok {
		workflow.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if description, ok := rawWorkflow["description"].(string); ok {
		workflow.Description = &description
	}

	// Map status field
	// Status mapping - REMOVED (enums no longer exist)
	// if statusStr, ok := rawWorkflow["status"].(string); ok {
	// 	switch statusStr {
	// 		case "draft":
	// 		workflow.Status = workflowpb.WorkflowStatus_WORKFLOW_STATUS_DRAFT
	// 	case "active":
	// 		workflow.Status = workflowpb.WorkflowStatus_WORKFLOW_STATUS_ACTIVE
	// 	case "inactive":
	// 		workflow.Status = workflowpb.WorkflowStatus_WORKFLOW_STATUS_INACTIVE
	// 	case "archived":
	// 		workflow.Status = workflowpb.WorkflowStatus_WORKFLOW_STATUS_ARCHIVED
	// 	default:
	// 		workflow.Status = workflowpb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
	// 	}
	// }

	if workspaceId, ok := rawWorkflow["workspaceId"].(string); ok {
		workflow.WorkspaceId = &workspaceId
	}

	if createdBy, ok := rawWorkflow["createdBy"].(string); ok {
		workflow.CreatedBy = &createdBy
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawWorkflow["dateCreated"].(string); ok {
		workflow.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			workflow.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawWorkflow["dateModified"].(string); ok {
		workflow.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			workflow.DateModified = &timestamp
		}
	}

	if active, ok := rawWorkflow["active"].(bool); ok {
		workflow.Active = active
	}

	if version, ok := rawWorkflow["version"].(float64); ok { // JSON numbers are float64
		workflowVersion := int32(version)
		workflow.Version = &workflowVersion
	}

	return workflow, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockWorkflowRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewWorkflowRepository creates a new workflow repository - Provider interface compatibility
func NewWorkflowRepository(businessType string) workflowpb.WorkflowDomainServiceServer {
	return NewMockWorkflowRepository(businessType)
}