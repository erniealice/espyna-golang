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
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "activity", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockActivityRepository(businessType), nil
	})
}

// MockActivityRepository implements activity.ActivityRepository using stateful mock data
type MockActivityRepository struct {
	activitypb.UnimplementedActivityDomainServiceServer
	businessType string
	activities   map[string]*activitypb.Activity // Persistent in-memory store
	mutex        sync.RWMutex                     // Thread-safe concurrent access
	initialized  bool                             // Prevent double initialization
	processor    *listdata.ListDataProcessor      // List data processing utilities
}

// ActivityRepositoryOption allows configuration of repository behavior
type ActivityRepositoryOption func(*MockActivityRepository)

// WithActivityTestOptimizations enables test-specific optimizations
func WithActivityTestOptimizations(enabled bool) ActivityRepositoryOption {
	return func(r *MockActivityRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockActivityRepository creates a new mock activity repository
func NewMockActivityRepository(businessType string, options ...ActivityRepositoryOption) activitypb.ActivityDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockActivityRepository{
		businessType: businessType,
		activities:   make(map[string]*activitypb.Activity),
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
func (r *MockActivityRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawActivities, err := datamock.LoadBusinessTypeModule(r.businessType, "activity")
	if err != nil {
		return fmt.Errorf("failed to load initial activities: %w", err)
	}

	// Convert and store each activity
	for _, rawActivity := range rawActivities {
		if activity, err := r.mapToProtobufActivity(rawActivity); err == nil {
			r.activities[activity.Id] = activity
		}
	}

	r.initialized = true
	return nil
}

// CreateActivity creates a new activity with stateful storage
func (r *MockActivityRepository) CreateActivity(ctx context.Context, req *activitypb.CreateActivityRequest) (*activitypb.CreateActivityResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create activity request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("activity data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("activity name is required")
	}
	if req.Data.StageId == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	activityID := fmt.Sprintf("activity-%d-%d", now.UnixNano(), len(r.activities))

	// Create new activity with proper timestamps and defaults
	newActivity := &activitypb.Activity{
		Id:                 activityID,
		StageId:            req.Data.StageId,
		ActivityTemplateId: req.Data.ActivityTemplateId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		// Status:             activitypb.ActivityStatus_ACTIVITY_STATUS_PENDING, // Default to pending - REMOVED
		Priority:           req.Data.Priority,
		AssignedTo:         req.Data.AssignedTo,
		DateDue:            req.Data.DateDue,
		DateDueString:      req.Data.DateDueString,
		InputDataJson:      req.Data.InputDataJson,
		OutputDataJson:     req.Data.OutputDataJson,
	}

	// Store in persistent map
	r.activities[activityID] = newActivity

	return &activitypb.CreateActivityResponse{
		Data:    []*activitypb.Activity{newActivity},
		Success: true,
	}, nil
}

// ReadActivity retrieves an activity by ID from stateful storage
func (r *MockActivityRepository) ReadActivity(ctx context.Context, req *activitypb.ReadActivityRequest) (*activitypb.ReadActivityResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read activity request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated activities)
	if activity, exists := r.activities[req.Data.Id]; exists {
		return &activitypb.ReadActivityResponse{
			Data:    []*activitypb.Activity{activity},
			Success: true,
		}, nil
	}

	return &activitypb.ReadActivityResponse{
		Data:    []*activitypb.Activity{},
		Success: false,
	}, nil
}

// UpdateActivity updates an existing activity in stateful storage
func (r *MockActivityRepository) UpdateActivity(ctx context.Context, req *activitypb.UpdateActivityRequest) (*activitypb.UpdateActivityResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update activity request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required for update")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("activity name is required")
	}
	if req.Data.StageId == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify activity exists
	_, exists := r.activities[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("activity with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	updatedActivity := &activitypb.Activity{
		Id:                 req.Data.Id,
		StageId:            req.Data.StageId,
		ActivityTemplateId: req.Data.ActivityTemplateId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Status:             req.Data.Status,
		Priority:           req.Data.Priority,
		AssignedTo:         req.Data.AssignedTo,
		CompletedBy:        req.Data.CompletedBy,
		DateAssigned:       req.Data.DateAssigned,
		DateAssignedString: req.Data.DateAssignedString,
		DateStarted:        req.Data.DateStarted,
		DateStartedString:  req.Data.DateStartedString,
		DateCompleted:      req.Data.DateCompleted,
		DateCompletedString: req.Data.DateCompletedString,
		DateDue:            req.Data.DateDue,
		DateDueString:      req.Data.DateDueString,
		InputDataJson:      req.Data.InputDataJson,
		OutputDataJson:     req.Data.OutputDataJson,
	}

	// Update in persistent store
	r.activities[req.Data.Id] = updatedActivity

	return &activitypb.UpdateActivityResponse{
		Data:    []*activitypb.Activity{updatedActivity},
		Success: true,
	}, nil
}

// DeleteActivity deletes an activity from stateful storage
func (r *MockActivityRepository) DeleteActivity(ctx context.Context, req *activitypb.DeleteActivityRequest) (*activitypb.DeleteActivityResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete activity request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("activity ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify activity exists before deletion
	if _, exists := r.activities[req.Data.Id]; !exists {
		return nil, fmt.Errorf("activity with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.activities, req.Data.Id)

	return &activitypb.DeleteActivityResponse{
		Success: true,
	}, nil
}

// ListActivities retrieves all activities from stateful storage
func (r *MockActivityRepository) ListActivities(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of activities
	activities := make([]*activitypb.Activity, 0, len(r.activities))
	for _, activity := range r.activities {
		activities = append(activities, activity)
	}

	return &activitypb.ListActivitiesResponse{
		Data:    activities,
		Success: true,
	}, nil
}

// GetActivityListPageData retrieves activities with advanced filtering, sorting, searching, and pagination
func (r *MockActivityRepository) GetActivityListPageData(
	ctx context.Context,
	req *activitypb.GetActivityListPageDataRequest,
) (*activitypb.GetActivityListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get activity list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of activities
	activities := make([]*activitypb.Activity, 0, len(r.activities))
	for _, activity := range r.activities {
		activities = append(activities, activity)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		activities,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process activity list data: %w", err)
	}

	// Convert processed items back to activity protobuf format
	processedActivities := make([]*activitypb.Activity, len(result.Items))
	for i, item := range result.Items {
		if activity, ok := item.(*activitypb.Activity); ok {
			processedActivities[i] = activity
		} else {
			return nil, fmt.Errorf("failed to convert item to activity type")
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

	return &activitypb.GetActivityListPageDataResponse{
		ActivityList:  processedActivities,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetActivityItemPageData retrieves a single activity with enhanced item page data
func (r *MockActivityRepository) GetActivityItemPageData(
	ctx context.Context,
	req *activitypb.GetActivityItemPageDataRequest,
) (*activitypb.GetActivityItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get activity item page data request is required")
	}
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	activity, exists := r.activities[req.ActivityId]
	if !exists {
		return nil, fmt.Errorf("activity with ID '%s' not found", req.ActivityId)
	}

	// In a real implementation, you might:
	// 1. Load related data (stage details, template information, submissions)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &activitypb.GetActivityItemPageDataResponse{
		Activity: activity,
		Success:  true,
	}, nil
}

// GetActivitiesByStage retrieves activities by stage ID
func (r *MockActivityRepository) GetActivitiesByStage(ctx context.Context, req *activitypb.GetActivitiesByStageRequest) (*activitypb.GetActivitiesByStageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get activities by stage request is required")
	}
	if req.StageId == "" {
		return nil, fmt.Errorf("stage ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Filter activities by stage ID
	var activities []*activitypb.Activity
	for _, activity := range r.activities {
		if activity.StageId == req.StageId {
			activities = append(activities, activity)
		}
	}

	return &activitypb.GetActivitiesByStageResponse{
		Activities: activities,
		Success:    true,
	}, nil
}

// mapToProtobufActivity converts raw mock data to protobuf Activity
func (r *MockActivityRepository) mapToProtobufActivity(rawActivity map[string]any) (*activitypb.Activity, error) {
	activity := &activitypb.Activity{}

	// Map required fields
	if id, ok := rawActivity["id"].(string); ok {
		activity.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawActivity["name"].(string); ok {
		activity.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	if stageId, ok := rawActivity["stageId"].(string); ok {
		activity.StageId = stageId
	} else {
		return nil, fmt.Errorf("missing or invalid stageId field")
	}

	// Map optional fields
	if description, ok := rawActivity["description"].(string); ok {
		activity.Description = &description
	}

	if activityTemplateId, ok := rawActivity["activityTemplateId"].(string); ok {
		activity.ActivityTemplateId = activityTemplateId
	}

	// Map status field - REMOVED (enums no longer exist)
	// if statusStr, ok := rawActivity["status"].(string); ok {
	// 	switch statusStr {
	// 	case "pending":
	// 		activity.Status = activitypb.ActivityStatus_ACTIVITY_STATUS_PENDING
	// 	case "in_progress":
	// 		activity.Status = activitypb.ActivityStatus_ACTIVITY_STATUS_IN_PROGRESS
	// 	case "completed":
	// 		activity.Status = activitypb.ActivityStatus_ACTIVITY_STATUS_COMPLETED
	// 	case "cancelled":
	// 		activity.Status = activitypb.ActivityStatus_ACTIVITY_STATUS_CANCELLED
	// 	default:
	// 		activity.Status = activitypb.ActivityStatus_ACTIVITY_STATUS_UNSPECIFIED
	// 	}
	// }

	// Map priority field - REMOVED (enums no longer exist)
	// if priorityStr, ok := rawActivity["priority"].(string); ok {
	// 	switch priorityStr {
	// 	case "low":
	// 		activity.Priority = activitypb.ActivityPriority_ACTIVITY_PRIORITY_MEDIUM
	// 	case "medium":
	// 		activity.Priority = activitypb.ActivityPriority_ACTIVITY_PRIORITY_MEDIUM
	// 	case "high":
	// 		activity.Priority = activitypb.ActivityPriority_ACTIVITY_PRIORITY_MEDIUM
	// 	case "urgent":
	// 		activity.Priority = activitypb.ActivityPriority_ACTIVITY_PRIORITY_MEDIUM
	// 	default:
	// 		activity.Priority = activitypb.ActivityPriority_ACTIVITY_PRIORITY_UNSPECIFIED
	// 	}
	// }

	if assignedTo, ok := rawActivity["assignedTo"].(string); ok {
		activity.AssignedTo = &assignedTo
	}

	if createdBy, ok := rawActivity["createdBy"].(string); ok {
		activity.CreatedBy = &createdBy
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawActivity["dateCreated"].(string); ok {
		activity.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			activity.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawActivity["dateModified"].(string); ok {
		activity.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			activity.DateModified = &timestamp
		}
	}

	if dueDate, ok := rawActivity["dueDate"].(string); ok {
		activity.DateDueString = &dueDate
		if timestamp, err := r.parseTimestamp(dueDate); err == nil {
			activity.DateDue = &timestamp
		}
	}

	
	return activity, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockActivityRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewActivityRepository creates a new activity repository - Provider interface compatibility
func NewActivityRepository(businessType string) activitypb.ActivityDomainServiceServer {
	return NewMockActivityRepository(businessType)
}