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
	activitytemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "activity_template", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockActivityTemplateRepository(businessType), nil
	})
}

// MockActivityTemplateRepository implements activity_template.ActivityTemplateRepository using stateful mock data
type MockActivityTemplateRepository struct {
	activitytemplatepb.UnimplementedActivityTemplateDomainServiceServer
	businessType      string
	activityTemplates map[string]*activitytemplatepb.ActivityTemplate // Persistent in-memory store
	mutex             sync.RWMutex                                  // Thread-safe concurrent access
	initialized       bool                                          // Prevent double initialization
	processor         *listdata.ListDataProcessor                   // List data processing utilities
}

// ActivityTemplateRepositoryOption allows configuration of repository behavior
type ActivityTemplateRepositoryOption func(*MockActivityTemplateRepository)

// WithActivityTemplateTestOptimizations enables test-specific optimizations
func WithActivityTemplateTestOptimizations(enabled bool) ActivityTemplateRepositoryOption {
	return func(r *MockActivityTemplateRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockActivityTemplateRepository creates a new mock activity template repository
func NewMockActivityTemplateRepository(businessType string, options ...ActivityTemplateRepositoryOption) activitytemplatepb.ActivityTemplateDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockActivityTemplateRepository{
		businessType:      businessType,
		activityTemplates: make(map[string]*activitytemplatepb.ActivityTemplate),
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
func (r *MockActivityTemplateRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawActivityTemplates, err := datamock.LoadBusinessTypeModule(r.businessType, "activity_template")
	if err != nil {
		return fmt.Errorf("failed to load initial activity templates: %w", err)
	}

	// Convert and store each activity template
	for _, rawActivityTemplate := range rawActivityTemplates {
		if activityTemplate, err := r.mapToProtobufActivityTemplate(rawActivityTemplate); err == nil {
			r.activityTemplates[activityTemplate.Id] = activityTemplate
		}
	}

	r.initialized = true
	return nil
}

// CreateActivityTemplate creates a new activity template with stateful storage
func (r *MockActivityTemplateRepository) CreateActivityTemplate(ctx context.Context, req *activitytemplatepb.CreateActivityTemplateRequest) (*activitytemplatepb.CreateActivityTemplateResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create activity template request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("activity template data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("activity template name is required")
	}
	if req.Data.StageTemplateId == "" {
		return nil, fmt.Errorf("stage template ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	activityTemplateID := fmt.Sprintf("activity-template-%d-%d", now.UnixNano(), len(r.activityTemplates))

	// Create new activity template with proper timestamps and defaults
	newActivityTemplate := &activitytemplatepb.ActivityTemplate{
		Id:                activityTemplateID,
		Name:              req.Data.Name,
		Description:       req.Data.Description,
		StageTemplateId:   req.Data.StageTemplateId,
		// Status:            activitytemplatepb.ActivityTemplateStatus_ACTIVITY_TEMPLATE_STATUS_DRAFT, // Default to draft - REMOVED
		ActivityType:      req.Data.ActivityType,
		OrderIndex:        req.Data.OrderIndex,
		IsRequired:        req.Data.IsRequired,
		ConditionExpression: req.Data.ConditionExpression,
		CreatedBy:         req.Data.CreatedBy,
		DateCreated:       &[]int64{now.UnixMilli()}[0],
		DateCreatedString: &[]string{now.Format(time.RFC3339)}[0],
		DateModified:      &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:            true, // Default to active
	}

	// Store in persistent map
	r.activityTemplates[activityTemplateID] = newActivityTemplate

	return &activitytemplatepb.CreateActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{newActivityTemplate},
		Success: true,
	}, nil
}

// ReadActivityTemplate retrieves an activity template by ID from stateful storage
func (r *MockActivityTemplateRepository) ReadActivityTemplate(ctx context.Context, req *activitytemplatepb.ReadActivityTemplateRequest) (*activitytemplatepb.ReadActivityTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read activity template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated activity templates)
	if activityTemplate, exists := r.activityTemplates[req.Data.Id]; exists {
		return &activitytemplatepb.ReadActivityTemplateResponse{
			Data:    []*activitytemplatepb.ActivityTemplate{activityTemplate},
			Success: true,
		}, nil
	}

	return &activitytemplatepb.ReadActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{},
		Success: false,
	}, nil
}

// UpdateActivityTemplate updates an existing activity template in stateful storage
func (r *MockActivityTemplateRepository) UpdateActivityTemplate(ctx context.Context, req *activitytemplatepb.UpdateActivityTemplateRequest) (*activitytemplatepb.UpdateActivityTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update activity template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity template ID is required for update")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("activity template name is required")
	}
	if req.Data.StageTemplateId == "" {
		return nil, fmt.Errorf("stage template ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify activity template exists
	existingActivityTemplate, exists := r.activityTemplates[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("activity template with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedActivityTemplate := &activitytemplatepb.ActivityTemplate{
		Id:                req.Data.Id,
		Name:              req.Data.Name,
		Description:       req.Data.Description,
		StageTemplateId:   req.Data.StageTemplateId,
		Status:            req.Data.Status,
		ActivityType:      req.Data.ActivityType,
		OrderIndex:        req.Data.OrderIndex,
		IsRequired:        req.Data.IsRequired,
		ConditionExpression: req.Data.ConditionExpression,
		CreatedBy:         existingActivityTemplate.CreatedBy, // Preserve original creator
		DateCreated:       existingActivityTemplate.DateCreated,
		DateCreatedString: existingActivityTemplate.DateCreatedString,
		DateModified:      &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:            req.Data.Active,
	}

	// Update in persistent store
	r.activityTemplates[req.Data.Id] = updatedActivityTemplate

	return &activitytemplatepb.UpdateActivityTemplateResponse{
		Data:    []*activitytemplatepb.ActivityTemplate{updatedActivityTemplate},
		Success: true,
	}, nil
}

// DeleteActivityTemplate deletes an activity template from stateful storage
func (r *MockActivityTemplateRepository) DeleteActivityTemplate(ctx context.Context, req *activitytemplatepb.DeleteActivityTemplateRequest) (*activitytemplatepb.DeleteActivityTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete activity template request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity template ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify activity template exists before deletion
	if _, exists := r.activityTemplates[req.Data.Id]; !exists {
		return nil, fmt.Errorf("activity template with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.activityTemplates, req.Data.Id)

	return &activitytemplatepb.DeleteActivityTemplateResponse{
		Success: true,
	}, nil
}

// ListActivityTemplates retrieves all activity templates from stateful storage
func (r *MockActivityTemplateRepository) ListActivityTemplates(ctx context.Context, req *activitytemplatepb.ListActivityTemplatesRequest) (*activitytemplatepb.ListActivityTemplatesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of activity templates
	activityTemplates := make([]*activitytemplatepb.ActivityTemplate, 0, len(r.activityTemplates))
	for _, activityTemplate := range r.activityTemplates {
		activityTemplates = append(activityTemplates, activityTemplate)
	}

	return &activitytemplatepb.ListActivityTemplatesResponse{
		Data:    activityTemplates,
		Success: true,
	}, nil
}

// GetActivityTemplateListPageData retrieves activity templates with advanced filtering, sorting, searching, and pagination
func (r *MockActivityTemplateRepository) GetActivityTemplateListPageData(
	ctx context.Context,
	req *activitytemplatepb.GetActivityTemplateListPageDataRequest,
) (*activitytemplatepb.GetActivityTemplateListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get activity template list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of activity templates
	activityTemplates := make([]*activitytemplatepb.ActivityTemplate, 0, len(r.activityTemplates))
	for _, activityTemplate := range r.activityTemplates {
		activityTemplates = append(activityTemplates, activityTemplate)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		activityTemplates,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process activity template list data: %w", err)
	}

	// Convert processed items back to activity template protobuf format
	processedActivityTemplates := make([]*activitytemplatepb.ActivityTemplate, len(result.Items))
	for i, item := range result.Items {
		if activityTemplate, ok := item.(*activitytemplatepb.ActivityTemplate); ok {
			processedActivityTemplates[i] = activityTemplate
		} else {
			return nil, fmt.Errorf("failed to convert item to activity template type")
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

	return &activitytemplatepb.GetActivityTemplateListPageDataResponse{
		ActivityTemplateList: processedActivityTemplates,
		Pagination:           result.PaginationResponse,
		SearchResults:        searchResults,
		Success:              true,
	}, nil
}

// GetActivityTemplateItemPageData retrieves a single activity template with enhanced item page data
func (r *MockActivityTemplateRepository) GetActivityTemplateItemPageData(
	ctx context.Context,
	req *activitytemplatepb.GetActivityTemplateItemPageDataRequest,
) (*activitytemplatepb.GetActivityTemplateItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get activity template item page data request is required")
	}
	if req.ActivityTemplateId == "" {
		return nil, fmt.Errorf("activity template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	activityTemplate, exists := r.activityTemplates[req.ActivityTemplateId]
	if !exists {
		return nil, fmt.Errorf("activity template with ID '%s' not found", req.ActivityTemplateId)
	}

	// In a real implementation, you might:
	// 1. Load related data (stage template details, usage statistics)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &activitytemplatepb.GetActivityTemplateItemPageDataResponse{
		ActivityTemplate: activityTemplate,
		Success:          true,
	}, nil
}

// GetActivityTemplatesByStageTemplate retrieves activity templates by stage template ID
func (r *MockActivityTemplateRepository) GetActivityTemplatesByStageTemplate(ctx context.Context, req *activitytemplatepb.GetActivityTemplatesByStageTemplateRequest) (*activitytemplatepb.GetActivityTemplatesByStageTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get activity templates by stage template request is required")
	}
	if req.StageTemplateId == "" {
		return nil, fmt.Errorf("stage template ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Filter activity templates by stage template ID
	var activityTemplates []*activitytemplatepb.ActivityTemplate
	for _, activityTemplate := range r.activityTemplates {
		if activityTemplate.StageTemplateId == req.StageTemplateId {
			activityTemplates = append(activityTemplates, activityTemplate)
		}
	}

	return &activitytemplatepb.GetActivityTemplatesByStageTemplateResponse{
		ActivityTemplates: activityTemplates,
		Success:           true,
	}, nil
}

// mapToProtobufActivityTemplate converts raw mock data to protobuf ActivityTemplate
func (r *MockActivityTemplateRepository) mapToProtobufActivityTemplate(rawActivityTemplate map[string]any) (*activitytemplatepb.ActivityTemplate, error) {
	activityTemplate := &activitytemplatepb.ActivityTemplate{}

	// Map required fields
	if id, ok := rawActivityTemplate["id"].(string); ok {
		activityTemplate.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawActivityTemplate["name"].(string); ok {
		activityTemplate.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	if stageTemplateId, ok := rawActivityTemplate["stageTemplateId"].(string); ok {
		activityTemplate.StageTemplateId = stageTemplateId
	} else {
		return nil, fmt.Errorf("missing or invalid stageTemplateId field")
	}

	// Map optional fields
	if description, ok := rawActivityTemplate["description"].(string); ok {
		activityTemplate.Description = &description
	}

	if orderIndex, ok := rawActivityTemplate["orderIndex"].(float64); ok {
		activityTemplate.OrderIndex = &[]int32{int32(orderIndex)}[0]
	}

	if isRequired, ok := rawActivityTemplate["isRequired"].(bool); ok {
		activityTemplate.IsRequired = &isRequired
	}

	if conditionExpression, ok := rawActivityTemplate["conditionExpression"].(string); ok {
		activityTemplate.ConditionExpression = &conditionExpression
	}

	if createdBy, ok := rawActivityTemplate["createdBy"].(string); ok {
		activityTemplate.CreatedBy = &createdBy
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawActivityTemplate["dateCreated"].(string); ok {
		activityTemplate.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			activityTemplate.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawActivityTemplate["dateModified"].(string); ok {
		activityTemplate.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			activityTemplate.DateModified = &timestamp
		}
	}

	if active, ok := rawActivityTemplate["active"].(bool); ok {
		activityTemplate.Active = active
	}

	return activityTemplate, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockActivityTemplateRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewActivityTemplateRepository creates a new activity template repository - Provider interface compatibility
func NewActivityTemplateRepository(businessType string) activitytemplatepb.ActivityTemplateDomainServiceServer {
	return NewMockActivityTemplateRepository(businessType)
}