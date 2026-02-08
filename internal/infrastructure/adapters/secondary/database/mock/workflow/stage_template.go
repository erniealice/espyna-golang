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
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "stage_template", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockStageTemplateRepository(businessType), nil
	})
}

// MockStageTemplateRepository implements stage_template.StageTemplateRepository using stateful mock data
type MockStageTemplateRepository struct {
	stageTemplatepb.UnimplementedStageTemplateDomainServiceServer
	businessType   string
	stageTemplates map[string]*stageTemplatepb.StageTemplate // Persistent in-memory store
	mutex          sync.RWMutex                             // Thread-safe concurrent access
	initialized    bool                                     // Prevent double initialization
	processor      *listdata.ListDataProcessor              // List data processing utilities
}

// StageTemplateRepositoryOption allows configuration of repository behavior
type StageTemplateRepositoryOption func(*MockStageTemplateRepository)

// WithStageTemplateTestOptimizations enables test-specific optimizations
func WithStageTemplateTestOptimizations(enabled bool) StageTemplateRepositoryOption {
	return func(r *MockStageTemplateRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockStageTemplateRepository creates a new mock stage template repository
func NewMockStageTemplateRepository(businessType string, options ...StageTemplateRepositoryOption) stageTemplatepb.StageTemplateDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockStageTemplateRepository{
		businessType:   businessType,
		stageTemplates: make(map[string]*stageTemplatepb.StageTemplate),
		processor:      listdata.NewListDataProcessor(),
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
func (r *MockStageTemplateRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawStageTemplates, err := datamock.LoadBusinessTypeModule(r.businessType, "stage_template")
	if err != nil {
		return fmt.Errorf("failed to load initial stage templates: %w", err)
	}

	// Convert and store each stage template
	for _, rawStageTemplate := range rawStageTemplates {
		if stageTemplate, err := r.mapToProtobufStageTemplate(rawStageTemplate); err == nil {
			r.stageTemplates[stageTemplate.Id] = stageTemplate
		}
	}

	r.initialized = true
	return nil
}

// CreateStageTemplate creates a new stage template with stateful storage
func (r *MockStageTemplateRepository) CreateStageTemplate(ctx context.Context, req *stageTemplatepb.CreateStageTemplateRequest) (*stageTemplatepb.CreateStageTemplateResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create stage template request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("stage template data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("stage template name is required")
	}
	if req.Data.WorkflowTemplateId == "" {
		return nil, fmt.Errorf("workflow template ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	stageTemplateID := fmt.Sprintf("stage-template-%d-%d", now.UnixNano(), len(r.stageTemplates))

	// Create new stage template with proper timestamps and defaults
	newStageTemplate := &stageTemplatepb.StageTemplate{
		Id:                 stageTemplateID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		WorkflowTemplateId: req.Data.WorkflowTemplateId,
		// Status:             stageTemplatepb.StageTemplateStatus_STAGE_TEMPLATE_STATUS_DRAFT, // Default to draft - REMOVED
		// StageType:          req.Data.StageType, // REMOVED

		StageType:          req.Data.StageType,
		OrderIndex:         req.Data.OrderIndex,
		IsRequired:         req.Data.IsRequired,
		ConditionExpression: req.Data.ConditionExpression,
		CreatedBy:          req.Data.CreatedBy,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.stageTemplates[stageTemplateID] = newStageTemplate

	return &stageTemplatepb.CreateStageTemplateResponse{
		Data:    []*stageTemplatepb.StageTemplate{newStageTemplate},
		Success: true,
	}, nil
}

// ReadStageTemplate retrieves a stage template by ID from stateful storage
func (r *MockStageTemplateRepository) ReadStageTemplate(ctx context.Context, req *stageTemplatepb.ReadStageTemplateRequest) (*stageTemplatepb.ReadStageTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read stage template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated stage templates)
	if stageTemplate, exists := r.stageTemplates[req.Data.Id]; exists {
		return &stageTemplatepb.ReadStageTemplateResponse{
			Data:    []*stageTemplatepb.StageTemplate{stageTemplate},
			Success: true,
		}, nil
	}

	return &stageTemplatepb.ReadStageTemplateResponse{
		Data:    []*stageTemplatepb.StageTemplate{},
		Success: false,
	}, nil
}

// UpdateStageTemplate updates an existing stage template in stateful storage
func (r *MockStageTemplateRepository) UpdateStageTemplate(ctx context.Context, req *stageTemplatepb.UpdateStageTemplateRequest) (*stageTemplatepb.UpdateStageTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update stage template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage template ID is required for update")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("stage template name is required")
	}
	if req.Data.WorkflowTemplateId == "" {
		return nil, fmt.Errorf("workflow template ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify stage template exists
	existingStageTemplate, exists := r.stageTemplates[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("stage template with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedStageTemplate := &stageTemplatepb.StageTemplate{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		WorkflowTemplateId: req.Data.WorkflowTemplateId,
		Status:             req.Data.Status,
		StageType:          req.Data.StageType,
		OrderIndex:         req.Data.OrderIndex,
		IsRequired:         req.Data.IsRequired,
		ConditionExpression: req.Data.ConditionExpression,
		CreatedBy:          existingStageTemplate.CreatedBy, // Preserve original creator
		DateCreated:        existingStageTemplate.DateCreated,
		DateCreatedString:  existingStageTemplate.DateCreatedString,
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.stageTemplates[req.Data.Id] = updatedStageTemplate

	return &stageTemplatepb.UpdateStageTemplateResponse{
		Data:    []*stageTemplatepb.StageTemplate{updatedStageTemplate},
		Success: true,
	}, nil
}

// DeleteStageTemplate deletes a stage template from stateful storage
func (r *MockStageTemplateRepository) DeleteStageTemplate(ctx context.Context, req *stageTemplatepb.DeleteStageTemplateRequest) (*stageTemplatepb.DeleteStageTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete stage template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage template ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify stage template exists before deletion
	if _, exists := r.stageTemplates[req.Data.Id]; !exists {
		return nil, fmt.Errorf("stage template with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.stageTemplates, req.Data.Id)

	return &stageTemplatepb.DeleteStageTemplateResponse{
		Success: true,
	}, nil
}

// ListStageTemplates retrieves all stage templates from stateful storage
func (r *MockStageTemplateRepository) ListStageTemplates(ctx context.Context, req *stageTemplatepb.ListStageTemplatesRequest) (*stageTemplatepb.ListStageTemplatesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of stage templates
	stageTemplates := make([]*stageTemplatepb.StageTemplate, 0, len(r.stageTemplates))
	for _, stageTemplate := range r.stageTemplates {
		stageTemplates = append(stageTemplates, stageTemplate)
	}

	return &stageTemplatepb.ListStageTemplatesResponse{
		Data:    stageTemplates,
		Success: true,
	}, nil
}

// GetStageTemplateListPageData retrieves stage templates with advanced filtering, sorting, searching, and pagination
func (r *MockStageTemplateRepository) GetStageTemplateListPageData(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateListPageDataRequest,
) (*stageTemplatepb.GetStageTemplateListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get stage template list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of stage templates
	stageTemplates := make([]*stageTemplatepb.StageTemplate, 0, len(r.stageTemplates))
	for _, stageTemplate := range r.stageTemplates {
		stageTemplates = append(stageTemplates, stageTemplate)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		stageTemplates,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process stage template list data: %w", err)
	}

	// Convert processed items back to stage template protobuf format
	processedStageTemplates := make([]*stageTemplatepb.StageTemplate, len(result.Items))
	for i, item := range result.Items {
		if stageTemplate, ok := item.(*stageTemplatepb.StageTemplate); ok {
			processedStageTemplates[i] = stageTemplate
		} else {
			return nil, fmt.Errorf("failed to convert item to stage template type")
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

	return &stageTemplatepb.GetStageTemplateListPageDataResponse{
		StageTemplateList: processedStageTemplates,
		Pagination:        result.PaginationResponse,
		SearchResults:     searchResults,
		Success:           true,
	}, nil
}

// GetStageTemplateItemPageData retrieves a single stage template with enhanced item page data
func (r *MockStageTemplateRepository) GetStageTemplateItemPageData(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateItemPageDataRequest,
) (*stageTemplatepb.GetStageTemplateItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get stage template item page data request is required")
	}
	if req.StageTemplateId == "" {
		return nil, fmt.Errorf("stage template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	stageTemplate, exists := r.stageTemplates[req.StageTemplateId]
	if !exists {
		return nil, fmt.Errorf("stage template with ID '%s' not found", req.StageTemplateId)
	}

	// In a real implementation, you might:
	// 1. Load related data (activity templates, workflow details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &stageTemplatepb.GetStageTemplateItemPageDataResponse{
		StageTemplate: stageTemplate,
		Success:       true,
	}, nil
}

// GetStageTemplatesByWorkflowTemplate retrieves stage templates by workflow template ID
func (r *MockStageTemplateRepository) GetStageTemplatesByWorkflowTemplate(ctx context.Context, req *stageTemplatepb.GetStageTemplatesByWorkflowTemplateRequest) (*stageTemplatepb.GetStageTemplatesByWorkflowTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get stage templates by workflow template request is required")
	}
	if req.WorkflowTemplateId == "" {
		return nil, fmt.Errorf("workflow template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Filter stage templates by workflow template ID
	var stageTemplates []*stageTemplatepb.StageTemplate
	for _, stageTemplate := range r.stageTemplates {
		if stageTemplate.WorkflowTemplateId == req.WorkflowTemplateId {
			stageTemplates = append(stageTemplates, stageTemplate)
		}
	}

	return &stageTemplatepb.GetStageTemplatesByWorkflowTemplateResponse{
		StageTemplates: stageTemplates,
		Success:        true,
	}, nil
}

// mapToProtobufStageTemplate converts raw mock data to protobuf StageTemplate
func (r *MockStageTemplateRepository) mapToProtobufStageTemplate(rawStageTemplate map[string]any) (*stageTemplatepb.StageTemplate, error) {
	stageTemplate := &stageTemplatepb.StageTemplate{}

	// Map required fields
	if id, ok := rawStageTemplate["id"].(string); ok {
		stageTemplate.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawStageTemplate["name"].(string); ok {
		stageTemplate.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	if workflowTemplateId, ok := rawStageTemplate["workflow_template_id"].(string); ok {
		stageTemplate.WorkflowTemplateId = workflowTemplateId
	} else if workflowId, ok := rawStageTemplate["workflowId"].(string); ok {
		// Fallback for backwards compatibility
		stageTemplate.WorkflowTemplateId = workflowId
	} else {
		return nil, fmt.Errorf("missing or invalid workflow_template_id field")
	}

	// Map optional fields
	if description, ok := rawStageTemplate["description"].(string); ok {
		stageTemplate.Description = &description
	}

	if orderIndex, ok := rawStageTemplate["orderIndex"].(float64); ok {
		stageTemplate.OrderIndex = &[]int32{int32(orderIndex)}[0]
	}

	if isRequired, ok := rawStageTemplate["isRequired"].(bool); ok {
		stageTemplate.IsRequired = &isRequired
	}

	if conditionExpression, ok := rawStageTemplate["conditionExpression"].(string); ok {
		stageTemplate.ConditionExpression = &conditionExpression
	}

	if createdBy, ok := rawStageTemplate["createdBy"].(string); ok {
		stageTemplate.CreatedBy = &createdBy
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawStageTemplate["dateCreated"].(string); ok {
		stageTemplate.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			stageTemplate.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawStageTemplate["dateModified"].(string); ok {
		stageTemplate.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			stageTemplate.DateModified = &timestamp
		}
	}

	if active, ok := rawStageTemplate["active"].(bool); ok {
		stageTemplate.Active = active
	}

	return stageTemplate, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockStageTemplateRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewStageTemplateRepository creates a new stage template repository - Provider interface compatibility
func NewStageTemplateRepository(businessType string) stageTemplatepb.StageTemplateDomainServiceServer {
	return NewMockStageTemplateRepository(businessType)
}