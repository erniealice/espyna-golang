//go:build mock_db

package subscription

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	plansettingspb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_settings"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "plan_settings", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPlanSettingsRepository(businessType), nil
	})
}

// MockPlanSettingsRepository implements subscription.PlanSettingsRepository using stateful mock data
type MockPlanSettingsRepository struct {
	plansettingspb.UnimplementedPlanSettingsDomainServiceServer
	businessType string
	planSettings map[string]*plansettingspb.PlanSettings // Persistent in-memory store
	mutex        sync.RWMutex                            // Thread-safe concurrent access
	initialized  bool                                    // Prevent double initialization
}

// PlanSettingsRepositoryOption allows configuration of repository behavior
type PlanSettingsRepositoryOption func(*MockPlanSettingsRepository)

// WithPlanSettingsTestOptimizations enables test-specific optimizations
func WithPlanSettingsTestOptimizations(enabled bool) PlanSettingsRepositoryOption {
	return func(r *MockPlanSettingsRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPlanSettingsRepository creates a new mock plan settings repository
func NewMockPlanSettingsRepository(businessType string, options ...PlanSettingsRepositoryOption) plansettingspb.PlanSettingsDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPlanSettingsRepository{
		businessType: businessType,
		planSettings: make(map[string]*plansettingspb.PlanSettings),
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
func (r *MockPlanSettingsRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPlanSettings, err := datamock.LoadBusinessTypeModule(r.businessType, "plan-settings")
	if err != nil {
		return fmt.Errorf("failed to load initial plan settings: %w", err)
	}

	// Convert and store each plan settings
	for _, rawPlanSetting := range rawPlanSettings {
		if planSetting, err := r.mapToProtobufPlanSettings(rawPlanSetting); err == nil {
			r.planSettings[planSetting.Id] = planSetting
		}
	}

	r.initialized = true
	return nil
}

// CreatePlanSettings creates a new plan settings with stateful storage
func (r *MockPlanSettingsRepository) CreatePlanSettings(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) (*plansettingspb.CreatePlanSettingsResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create plan settings request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("plan settings data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("plan settings name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	planSettingsID := fmt.Sprintf("plan-settings-%d-%d", now.UnixNano(), len(r.planSettings))

	// Create new plan settings with proper timestamps and defaults
	newPlanSettings := &plansettingspb.PlanSettings{
		Id:                 planSettingsID,
		PlanId:             req.Data.PlanId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.planSettings[planSettingsID] = newPlanSettings

	return &plansettingspb.CreatePlanSettingsResponse{
		Data:    []*plansettingspb.PlanSettings{newPlanSettings},
		Success: true,
	}, nil
}

// ReadPlanSettings retrieves a plan settings by ID from stateful storage
func (r *MockPlanSettingsRepository) ReadPlanSettings(ctx context.Context, req *plansettingspb.ReadPlanSettingsRequest) (*plansettingspb.ReadPlanSettingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read plan settings request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated plan settings)
	if planSettings, exists := r.planSettings[req.Data.Id]; exists {
		return &plansettingspb.ReadPlanSettingsResponse{
			Data:    []*plansettingspb.PlanSettings{planSettings},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("plan settings with ID '%s' not found", req.Data.Id)
}

// UpdatePlanSettings updates an existing plan settings in stateful storage
func (r *MockPlanSettingsRepository) UpdatePlanSettings(ctx context.Context, req *plansettingspb.UpdatePlanSettingsRequest) (*plansettingspb.UpdatePlanSettingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update plan settings request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify plan settings exists
	existingPlanSettings, exists := r.planSettings[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("plan settings with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPlanSettings := &plansettingspb.PlanSettings{
		Id:                 req.Data.Id,
		PlanId:             req.Data.PlanId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        existingPlanSettings.DateCreated,       // Preserve original
		DateCreatedString:  existingPlanSettings.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],                // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.planSettings[req.Data.Id] = updatedPlanSettings

	return &plansettingspb.UpdatePlanSettingsResponse{
		Data:    []*plansettingspb.PlanSettings{updatedPlanSettings},
		Success: true,
	}, nil
}

// DeletePlanSettings deletes a plan settings from stateful storage
func (r *MockPlanSettingsRepository) DeletePlanSettings(ctx context.Context, req *plansettingspb.DeletePlanSettingsRequest) (*plansettingspb.DeletePlanSettingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete plan settings request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify plan settings exists before deletion
	if _, exists := r.planSettings[req.Data.Id]; !exists {
		return nil, fmt.Errorf("plan settings with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.planSettings, req.Data.Id)

	return &plansettingspb.DeletePlanSettingsResponse{
		Success: true,
	}, nil
}

// ListPlanSettings retrieves all plan settings from stateful storage
func (r *MockPlanSettingsRepository) ListPlanSettings(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) (*plansettingspb.ListPlanSettingsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of plan settings
	planSettings := make([]*plansettingspb.PlanSettings, 0, len(r.planSettings))
	for _, planSetting := range r.planSettings {
		planSettings = append(planSettings, planSetting)
	}

	return &plansettingspb.ListPlanSettingsResponse{
		Data:    planSettings,
		Success: true,
	}, nil
}

// ListPlanSettingsByPlan retrieves all plan settings for a specific plan from stateful storage
func (r *MockPlanSettingsRepository) ListPlanSettingsByPlan(ctx context.Context, req *plansettingspb.ListPlanSettingsByPlanRequest) (*plansettingspb.ListPlanSettingsByPlanResponse, error) {
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Filter plan settings by plan ID
	planSettings := make([]*plansettingspb.PlanSettings, 0)
	for _, planSetting := range r.planSettings {
		if planSetting.PlanId == req.PlanId {
			planSettings = append(planSettings, planSetting)
		}
	}

	return &plansettingspb.ListPlanSettingsByPlanResponse{
		Data:    planSettings,
		Success: true,
	}, nil
}

// mapToProtobufPlanSettings converts raw mock data to protobuf PlanSettings
func (r *MockPlanSettingsRepository) mapToProtobufPlanSettings(rawPlanSettings map[string]any) (*plansettingspb.PlanSettings, error) {
	planSettings := &plansettingspb.PlanSettings{}

	// Map required fields
	if id, ok := rawPlanSettings["id"].(string); ok {
		planSettings.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawPlanSettings["name"].(string); ok {
		planSettings.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if planId, ok := rawPlanSettings["planId"].(string); ok {
		planSettings.PlanId = planId
	}

	if description, ok := rawPlanSettings["description"].(string); ok {
		planSettings.Description = description
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawPlanSettings["dateCreated"].(string); ok {
		planSettings.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			planSettings.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPlanSettings["dateModified"].(string); ok {
		planSettings.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			planSettings.DateModified = &timestamp
		}
	}

	if active, ok := rawPlanSettings["active"].(bool); ok {
		planSettings.Active = active
	}

	return planSettings, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPlanSettingsRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewPlanSettingsRepository creates a new mock plan settings repository (registry constructor)
func NewPlanSettingsRepository(data map[string]*plansettingspb.PlanSettings) plansettingspb.PlanSettingsDomainServiceServer {
	repo := &MockPlanSettingsRepository{
		businessType: "education", // Default business type
		planSettings: data,
		mutex:        sync.RWMutex{},
	}
	if data == nil {
		repo.planSettings = make(map[string]*plansettingspb.PlanSettings)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
