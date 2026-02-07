//go:build mock_db

package workflow

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
	workflowtemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "workflow_template", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockWorkflowTemplateRepository(businessType), nil
	})
}

// MockWorkflowTemplateRepository implements workflow_template.WorkflowTemplateDomainServiceServer using stateful mock data
type MockWorkflowTemplateRepository struct {
	workflowtemplatepb.UnimplementedWorkflowTemplateDomainServiceServer
	businessType      string
	workflowTemplates map[string]*workflowtemplatepb.WorkflowTemplate // Persistent in-memory store
	mutex             sync.RWMutex                                     // Thread-safe concurrent access
	initialized       bool                                             // Prevent double initialization
	processor         *listdata.ListDataProcessor                      // List data processing utilities
}

// WorkflowTemplateRepositoryOption allows configuration of repository behavior
type WorkflowTemplateRepositoryOption func(*MockWorkflowTemplateRepository)

// WithWorkflowTemplateTestOptimizations enables test-specific optimizations
func WithWorkflowTemplateTestOptimizations(enabled bool) WorkflowTemplateRepositoryOption {
	return func(r *MockWorkflowTemplateRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockWorkflowTemplateRepository creates a new mock workflow template repository
func NewMockWorkflowTemplateRepository(businessType string, options ...WorkflowTemplateRepositoryOption) workflowtemplatepb.WorkflowTemplateDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockWorkflowTemplateRepository{
		businessType:      businessType,
		workflowTemplates: make(map[string]*workflowtemplatepb.WorkflowTemplate),
		processor:         listdata.NewListDataProcessor(),
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
func (r *MockWorkflowTemplateRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawWorkflowTemplates, err := datamock.LoadBusinessTypeModule(r.businessType, "workflow_template")
	if err != nil {
		return fmt.Errorf("failed to load initial workflow templates: %w", err)
	}

	// Convert and store each workflow template
	for _, rawWorkflowTemplate := range rawWorkflowTemplates {
		if workflowTemplate, err := r.mapToProtobufWorkflowTemplate(rawWorkflowTemplate); err == nil {
			r.workflowTemplates[workflowTemplate.Id] = workflowTemplate
		}
	}

	r.initialized = true
	return nil
}

// CreateWorkflowTemplate creates a new workflow template with stateful storage
func (r *MockWorkflowTemplateRepository) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.CreateWorkflowTemplateRequest) (*workflowtemplatepb.CreateWorkflowTemplateResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create workflow template request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("workflow template data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("workflow template name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	workflowTemplateID := fmt.Sprintf("workflow-template-%d-%d", now.UnixNano(), len(r.workflowTemplates))

	// Create new workflow template with proper timestamps and defaults
	newWorkflowTemplate := &workflowtemplatepb.WorkflowTemplate{
		Id:                 workflowTemplateID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		WorkspaceId:        req.Data.WorkspaceId,
		Status:             "draft", // Default to draft
		BusinessType:       r.businessType, // Use repository's business type
		ConfigurationJson:  req.Data.ConfigurationJson,
		CreatedBy:          req.Data.CreatedBy,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
		Version:            &[]int32{1}[0], // Start with version 1
	}

	// Store in persistent map
	r.workflowTemplates[workflowTemplateID] = newWorkflowTemplate

	return &workflowtemplatepb.CreateWorkflowTemplateResponse{
		Data:    []*workflowtemplatepb.WorkflowTemplate{newWorkflowTemplate},
		Success: true,
	}, nil
}

// ReadWorkflowTemplate retrieves a workflow template by ID from stateful storage
func (r *MockWorkflowTemplateRepository) ReadWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.ReadWorkflowTemplateRequest) (*workflowtemplatepb.ReadWorkflowTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read workflow template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated workflow templates)
	if workflowTemplate, exists := r.workflowTemplates[req.Data.Id]; exists {
		return &workflowtemplatepb.ReadWorkflowTemplateResponse{
			Data:    []*workflowtemplatepb.WorkflowTemplate{workflowTemplate},
			Success: true,
		}, nil
	}

	return &workflowtemplatepb.ReadWorkflowTemplateResponse{
		Data:    []*workflowtemplatepb.WorkflowTemplate{},
		Success: false,
	}, nil
}

// UpdateWorkflowTemplate updates an existing workflow template in stateful storage
func (r *MockWorkflowTemplateRepository) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.UpdateWorkflowTemplateRequest) (*workflowtemplatepb.UpdateWorkflowTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update workflow template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow template ID is required for update")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("workflow template name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workflow template exists
	existingWorkflowTemplate, exists := r.workflowTemplates[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("workflow template with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedWorkflowTemplate := &workflowtemplatepb.WorkflowTemplate{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		WorkspaceId:        req.Data.WorkspaceId,
		Status:             req.Data.Status,
		BusinessType:       req.Data.BusinessType,
		ConfigurationJson:  req.Data.ConfigurationJson,
		CreatedBy:          existingWorkflowTemplate.CreatedBy, // Preserve original creator
		DateCreated:        existingWorkflowTemplate.DateCreated,
		DateCreatedString:  existingWorkflowTemplate.DateCreatedString,
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
		Version:            req.Data.Version,
	}

	// Ensure business type is set
	if updatedWorkflowTemplate.BusinessType == "" {
		updatedWorkflowTemplate.BusinessType = existingWorkflowTemplate.BusinessType
		if updatedWorkflowTemplate.BusinessType == "" {
			updatedWorkflowTemplate.BusinessType = r.businessType
		}
	}

	// Increment version if not specified
	if updatedWorkflowTemplate.Version == nil {
		newVersion := int32(1)
		if existingWorkflowTemplate.Version != nil {
			newVersion = *existingWorkflowTemplate.Version + 1
		}
		updatedWorkflowTemplate.Version = &newVersion
	}

	// Update in persistent store
	r.workflowTemplates[req.Data.Id] = updatedWorkflowTemplate

	return &workflowtemplatepb.UpdateWorkflowTemplateResponse{
		Data:    []*workflowtemplatepb.WorkflowTemplate{updatedWorkflowTemplate},
		Success: true,
	}, nil
}

// DeleteWorkflowTemplate deletes a workflow template from stateful storage
func (r *MockWorkflowTemplateRepository) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepb.DeleteWorkflowTemplateRequest) (*workflowtemplatepb.DeleteWorkflowTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete workflow template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workflow template ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workflow template exists before deletion
	if _, exists := r.workflowTemplates[req.Data.Id]; !exists {
		return nil, fmt.Errorf("workflow template with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.workflowTemplates, req.Data.Id)

	return &workflowtemplatepb.DeleteWorkflowTemplateResponse{
		Success: true,
	}, nil
}

// ListWorkflowTemplates retrieves all workflow templates from stateful storage
func (r *MockWorkflowTemplateRepository) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepb.ListWorkflowTemplatesRequest) (*workflowtemplatepb.ListWorkflowTemplatesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workflow templates
	workflowTemplates := make([]*workflowtemplatepb.WorkflowTemplate, 0, len(r.workflowTemplates))
	for _, workflowTemplate := range r.workflowTemplates {
		workflowTemplates = append(workflowTemplates, workflowTemplate)
	}

	return &workflowtemplatepb.ListWorkflowTemplatesResponse{
		Data:    workflowTemplates,
		Success: true,
	}, nil
}

// GetWorkflowTemplateListPageData retrieves workflow templates with advanced filtering, sorting, searching, and pagination
func (r *MockWorkflowTemplateRepository) GetWorkflowTemplateListPageData(
	ctx context.Context,
	req *workflowtemplatepb.GetWorkflowTemplateListPageDataRequest,
) (*workflowtemplatepb.GetWorkflowTemplateListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workflow template list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workflow templates
	workflowTemplates := make([]*workflowtemplatepb.WorkflowTemplate, 0, len(r.workflowTemplates))
	for _, workflowTemplate := range r.workflowTemplates {
		workflowTemplates = append(workflowTemplates, workflowTemplate)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		workflowTemplates,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process workflow template list data: %w", err)
	}

	// Convert processed items back to workflow template protobuf format
	processedWorkflowTemplates := make([]*workflowtemplatepb.WorkflowTemplate, len(result.Items))
	for i, item := range result.Items {
		if workflowTemplate, ok := item.(*workflowtemplatepb.WorkflowTemplate); ok {
			processedWorkflowTemplates[i] = workflowTemplate
		} else {
			return nil, fmt.Errorf("failed to convert item to workflow template type")
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

	return &workflowtemplatepb.GetWorkflowTemplateListPageDataResponse{
		WorkflowTemplateList: processedWorkflowTemplates,
		Pagination:           result.PaginationResponse,
		SearchResults:        searchResults,
		Success:              true,
	}, nil
}

// GetWorkflowTemplateItemPageData retrieves a single workflow template with enhanced item page data
func (r *MockWorkflowTemplateRepository) GetWorkflowTemplateItemPageData(
	ctx context.Context,
	req *workflowtemplatepb.GetWorkflowTemplateItemPageDataRequest,
) (*workflowtemplatepb.GetWorkflowTemplateItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workflow template item page data request is required")
	}
	if req.WorkflowTemplateId == "" {
		return nil, fmt.Errorf("workflow template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	workflowTemplate, exists := r.workflowTemplates[req.WorkflowTemplateId]
	if !exists {
		return nil, fmt.Errorf("workflow template with ID '%s' not found", req.WorkflowTemplateId)
	}

	// In a real implementation, you might:
	// 1. Load related data (stage templates, usage statistics)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &workflowtemplatepb.GetWorkflowTemplateItemPageDataResponse{
		WorkflowTemplate: workflowTemplate,
		Success:          true,
	}, nil
}

// mapToProtobufWorkflowTemplate converts raw mock data to protobuf WorkflowTemplate
func (r *MockWorkflowTemplateRepository) mapToProtobufWorkflowTemplate(rawWorkflowTemplate map[string]any) (*workflowtemplatepb.WorkflowTemplate, error) {
	workflowTemplate := &workflowtemplatepb.WorkflowTemplate{}

	// Map required fields
	if id, ok := rawWorkflowTemplate["id"].(string); ok {
		workflowTemplate.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawWorkflowTemplate["name"].(string); ok {
		workflowTemplate.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if description, ok := rawWorkflowTemplate["description"].(string); ok {
		workflowTemplate.Description = &description
	}

	if workspaceId, ok := rawWorkflowTemplate["workspaceId"].(string); ok {
		workflowTemplate.WorkspaceId = &workspaceId
	}

	// Map status field
	if status, ok := rawWorkflowTemplate["status"].(string); ok {
		workflowTemplate.Status = status
	} else {
		workflowTemplate.Status = "draft" // Default status
	}

	// Map business type field
	if businessType, ok := rawWorkflowTemplate["businessType"].(string); ok {
		workflowTemplate.BusinessType = businessType
	} else {
		// Use repository's business type as fallback
		workflowTemplate.BusinessType = r.businessType
	}

	// Map configuration JSON field
	if configurationJson, ok := rawWorkflowTemplate["configurationJson"].(string); ok {
		workflowTemplate.ConfigurationJson = &configurationJson
	}

	if createdBy, ok := rawWorkflowTemplate["createdBy"].(string); ok {
		workflowTemplate.CreatedBy = &createdBy
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawWorkflowTemplate["dateCreated"].(string); ok {
		workflowTemplate.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			workflowTemplate.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawWorkflowTemplate["dateModified"].(string); ok {
		workflowTemplate.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			workflowTemplate.DateModified = &timestamp
		}
	}

	if active, ok := rawWorkflowTemplate["active"].(bool); ok {
		workflowTemplate.Active = active
	} else {
		workflowTemplate.Active = true // Default to active
	}

	if version, ok := rawWorkflowTemplate["version"].(float64); ok { // JSON numbers are float64
		workflowTemplateVersion := int32(version)
		workflowTemplate.Version = &workflowTemplateVersion
	} else {
		defaultVersion := int32(1)
		workflowTemplate.Version = &defaultVersion
	}

	return workflowTemplate, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockWorkflowTemplateRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewWorkflowTemplateRepository creates a new workflow template repository - Provider interface compatibility
func NewWorkflowTemplateRepository(businessType string) workflowtemplatepb.WorkflowTemplateDomainServiceServer {
	return NewMockWorkflowTemplateRepository(businessType)
}