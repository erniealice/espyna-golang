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
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "stage", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockStageRepository(businessType), nil
	})
}

// MockStageRepository implements stage.StageRepository using stateful mock data
type MockStageRepository struct {
	stagepb.UnimplementedStageDomainServiceServer
	businessType string
	stages       map[string]*stagepb.Stage // Persistent in-memory store
	mutex        sync.RWMutex               // Thread-safe concurrent access
	initialized  bool                       // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing utilities
}

// StageRepositoryOption allows configuration of repository behavior
type StageRepositoryOption func(*MockStageRepository)

// WithStageTestOptimizations enables test-specific optimizations
func WithStageTestOptimizations(enabled bool) StageRepositoryOption {
	return func(r *MockStageRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockStageRepository creates a new mock stage repository
func NewMockStageRepository(businessType string, options ...StageRepositoryOption) stagepb.StageDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockStageRepository{
		businessType: businessType,
		stages:       make(map[string]*stagepb.Stage),
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
func (r *MockStageRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawStages, err := datamock.LoadBusinessTypeModule(r.businessType, "stage")
	if err != nil {
		return fmt.Errorf("failed to load initial stages: %w", err)
	}

	// Convert and store each stage
	for _, rawStage := range rawStages {
		if stage, err := r.mapToProtobufStage(rawStage); err == nil {
			r.stages[stage.Id] = stage
		}
	}

	r.initialized = true
	return nil
}

// CreateStage creates a new stage with stateful storage
func (r *MockStageRepository) CreateStage(ctx context.Context, req *stagepb.CreateStageRequest) (*stagepb.CreateStageResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create stage request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("stage data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("stage name is required")
	}
	if req.Data.WorkflowInstanceId == "" {
		return nil, fmt.Errorf("workflow instance ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	stageID := fmt.Sprintf("stage-%d-%d", now.UnixNano(), len(r.stages))

	// Create new stage with proper timestamps and defaults
	newStage := &stagepb.Stage{
		Id:                 stageID,
		WorkflowInstanceId: req.Data.WorkflowInstanceId,
		StageTemplateId:    req.Data.StageTemplateId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		// Status:             stagepb.StageStatus_STAGE_STATUS_PENDING, // Default to pending - REMOVED
		Priority:           req.Data.Priority,
		AssignedTo:         req.Data.AssignedTo,
		DateDue:            req.Data.DateDue,
		DateDueString:      req.Data.DateDueString,
		ResultJson:         req.Data.ResultJson,
		ErrorMessage:       req.Data.ErrorMessage,
		CreatedBy:          req.Data.CreatedBy,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
	}

	// Store in persistent map
	r.stages[stageID] = newStage

	return &stagepb.CreateStageResponse{
		Data:    []*stagepb.Stage{newStage},
		Success: true,
	}, nil
}

// ReadStage retrieves a stage by ID from stateful storage
func (r *MockStageRepository) ReadStage(ctx context.Context, req *stagepb.ReadStageRequest) (*stagepb.ReadStageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read stage request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated stages)
	if stage, exists := r.stages[req.Data.Id]; exists {
		return &stagepb.ReadStageResponse{
			Data:    []*stagepb.Stage{stage},
			Success: true,
		}, nil
	}

	return &stagepb.ReadStageResponse{
		Data:    []*stagepb.Stage{},
		Success: false,
	}, nil
}

// UpdateStage updates an existing stage in stateful storage
func (r *MockStageRepository) UpdateStage(ctx context.Context, req *stagepb.UpdateStageRequest) (*stagepb.UpdateStageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update stage request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required for update")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("stage name is required")
	}
	if req.Data.WorkflowInstanceId == "" {
		return nil, fmt.Errorf("workflow instance ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify stage exists
	_, exists := r.stages[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("stage with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	updatedStage := &stagepb.Stage{
		Id:                 req.Data.Id,
		WorkflowInstanceId: req.Data.WorkflowInstanceId,
		StageTemplateId:    req.Data.StageTemplateId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Status:             req.Data.Status,
		Priority:           req.Data.Priority,
		AssignedTo:         req.Data.AssignedTo,
		CompletedBy:        req.Data.CompletedBy,
		DateStarted:        req.Data.DateStarted,
		DateStartedString:  req.Data.DateStartedString,
		DateCompleted:      req.Data.DateCompleted,
		DateCompletedString: req.Data.DateCompletedString,
		DateDue:            req.Data.DateDue,
		DateDueString:      req.Data.DateDueString,
		ResultJson:         req.Data.ResultJson,
		ErrorMessage:       req.Data.ErrorMessage,
		CreatedBy:          req.Data.CreatedBy,
		DateCreated:        req.Data.DateCreated,
		DateCreatedString:  req.Data.DateCreatedString,
	}

	// Update in persistent store
	r.stages[req.Data.Id] = updatedStage

	return &stagepb.UpdateStageResponse{
		Data:    []*stagepb.Stage{updatedStage},
		Success: true,
	}, nil
}

// DeleteStage deletes a stage from stateful storage
func (r *MockStageRepository) DeleteStage(ctx context.Context, req *stagepb.DeleteStageRequest) (*stagepb.DeleteStageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete stage request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("stage ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify stage exists before deletion
	if _, exists := r.stages[req.Data.Id]; !exists {
		return nil, fmt.Errorf("stage with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.stages, req.Data.Id)

	return &stagepb.DeleteStageResponse{
		Success: true,
	}, nil
}

// ListStages retrieves all stages from stateful storage
func (r *MockStageRepository) ListStages(ctx context.Context, req *stagepb.ListStagesRequest) (*stagepb.ListStagesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of stages
	stages := make([]*stagepb.Stage, 0, len(r.stages))
	for _, stage := range r.stages {
		stages = append(stages, stage)
	}

	return &stagepb.ListStagesResponse{
		Data:    stages,
		Success: true,
	}, nil
}

// GetStageListPageData retrieves stages with advanced filtering, sorting, searching, and pagination
func (r *MockStageRepository) GetStageListPageData(
	ctx context.Context,
	req *stagepb.GetStageListPageDataRequest,
) (*stagepb.GetStageListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get stage list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of stages
	stages := make([]*stagepb.Stage, 0, len(r.stages))
	for _, stage := range r.stages {
		stages = append(stages, stage)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		stages,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process stage list data: %w", err)
	}

	// Convert processed items back to stage protobuf format
	processedStages := make([]*stagepb.Stage, len(result.Items))
	for i, item := range result.Items {
		if stage, ok := item.(*stagepb.Stage); ok {
			processedStages[i] = stage
		} else {
			return nil, fmt.Errorf("failed to convert item to stage type")
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

	return &stagepb.GetStageListPageDataResponse{
		StageList:    processedStages,
		Pagination:   result.PaginationResponse,
		SearchResults: searchResults,
		Success:      true,
	}, nil
}

// GetStageItemPageData retrieves a single stage with enhanced item page data
func (r *MockStageRepository) GetStageItemPageData(
	ctx context.Context,
	req *stagepb.GetStageItemPageDataRequest,
) (*stagepb.GetStageItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get stage item page data request is required")
	}
	if req.StageId == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	stage, exists := r.stages[req.StageId]
	if !exists {
		return nil, fmt.Errorf("stage with ID '%s' not found", req.StageId)
	}

	// In a real implementation, you might:
	// 1. Load related data (workflow details, template information, activities)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &stagepb.GetStageItemPageDataResponse{
		Stage:   stage,
		Success: true,
	}, nil
}

// GetStagesByWorkflowInstance retrieves stages by workflow instance ID
func (r *MockStageRepository) GetStagesByWorkflowInstance(ctx context.Context, req *stagepb.GetStagesByWorkflowInstanceRequest) (*stagepb.GetStagesByWorkflowInstanceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get stages by workflow instance request is required")
	}
	if req.WorkflowInstanceId == "" {
		return nil, fmt.Errorf("workflow instance ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Filter stages by workflow instance ID
	var stages []*stagepb.Stage
	for _, stage := range r.stages {
		if stage.WorkflowInstanceId == req.WorkflowInstanceId {
			stages = append(stages, stage)
		}
	}

	return &stagepb.GetStagesByWorkflowInstanceResponse{
		Stages:  stages,
		Success: true,
	}, nil
}

// mapToProtobufStage converts raw mock data to protobuf Stage
func (r *MockStageRepository) mapToProtobufStage(rawStage map[string]any) (*stagepb.Stage, error) {
	stage := &stagepb.Stage{}

	// Map required fields
	if id, ok := rawStage["id"].(string); ok {
		stage.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawStage["name"].(string); ok {
		stage.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	if workflowInstanceId, ok := rawStage["workflowInstanceId"].(string); ok {
		stage.WorkflowInstanceId = workflowInstanceId
	} else {
		return nil, fmt.Errorf("missing or invalid workflowInstanceId field")
	}

	// Map optional fields
	if description, ok := rawStage["description"].(string); ok {
		stage.Description = &description
	}

	if stageTemplateId, ok := rawStage["stageTemplateId"].(string); ok {
		stage.StageTemplateId = stageTemplateId
	}

	// Map status field - REMOVED (enums no longer exist)
	// if statusStr, ok := rawStage["status"].(string); ok {
	// 	switch statusStr {
	// 		case "pending":
	// 		stage.Status = stagepb.StageStatus_STAGE_STATUS_PENDING
	// 	case "in_progress":
	// 		stage.Status = stagepb.StageStatus_STAGE_STATUS_IN_PROGRESS
	// 	case "completed":
	// 		stage.Status = stagepb.StageStatus_STAGE_STATUS_COMPLETED
	// 	case "cancelled":
	// 		stage.Status = stagepb.StageStatus_STAGE_STATUS_CANCELLED
	// 	default:
	// 		stage.Status = stagepb.StageStatus_STAGE_STATUS_UNSPECIFIED
	// 	}
	// }

	// Map priority field - REMOVED (enums no longer exist)
	// if priorityStr, ok := rawStage["priority"].(string); ok {
	// 	switch priorityStr {
	// 		case "low":
	// 		stage.Priority = stagepb.StagePriority_STAGE_PRIORITY_LOW
	// 	case "medium":
	// 		stage.Priority = stagepb.StagePriority_STAGE_PRIORITY_MEDIUM
	// 	case "high":
	// 		stage.Priority = stagepb.StagePriority_STAGE_PRIORITY_HIGH
	// 	case "urgent":
	// 		stage.Priority = stagepb.StagePriority_STAGE_PRIORITY_URGENT
	// 	default:
	// 		stage.Priority = stagepb.StagePriority_STAGE_PRIORITY_UNSPECIFIED
	// 	}
	// }

	
	if assignedTo, ok := rawStage["assignedTo"].(string); ok {
		stage.AssignedTo = &assignedTo
	}

	if createdBy, ok := rawStage["createdBy"].(string); ok {
		stage.CreatedBy = &createdBy
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawStage["dateCreated"].(string); ok {
		stage.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			stage.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawStage["dateModified"].(string); ok {
		stage.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			stage.DateModified = &timestamp
		}
	}

	if startDate, ok := rawStage["startDate"].(string); ok {
		stage.DateStartedString = &startDate
		if timestamp, err := r.parseTimestamp(startDate); err == nil {
			stage.DateStarted = &timestamp
		}
	}

	if endDate, ok := rawStage["endDate"].(string); ok {
		stage.DateCompletedString = &endDate
		if timestamp, err := r.parseTimestamp(endDate); err == nil {
			stage.DateCompleted = &timestamp
		}
	}

	
	return stage, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockStageRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewStageRepository creates a new stage repository - Provider interface compatibility
func NewStageRepository(businessType string) stagepb.StageDomainServiceServer {
	return NewMockStageRepository(businessType)
}