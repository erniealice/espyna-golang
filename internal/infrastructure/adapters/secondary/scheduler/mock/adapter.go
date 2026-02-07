//go:build !calendly

package mock

import (
	"context"
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	schedulerpb "leapfor.xyz/esqyma/golang/v1/integration/scheduler"
)

func init() {
	// Register with the global registry
	registry.RegisterSchedulerBuildFromEnv("mock_scheduler", func() (ports.SchedulerProvider, error) {
		return NewMockSchedulerAdapter(), nil
	})
	log.Printf("[MockSchedulerAdapter] Registered with scheduler registry")
}

// MockSchedulerAdapter provides a mock implementation of SchedulerProvider
type MockSchedulerAdapter struct {
	enabled bool
}

// NewMockSchedulerAdapter creates a new mock scheduler adapter
func NewMockSchedulerAdapter() *MockSchedulerAdapter {
	log.Printf("[MockSchedulerAdapter] Created mock scheduler adapter (Calendly build tag not enabled)")
	return &MockSchedulerAdapter{enabled: true}
}

// Name returns the name of the scheduler provider
func (a *MockSchedulerAdapter) Name() string {
	return "mock_scheduler"
}

// Initialize sets up the mock adapter
func (a *MockSchedulerAdapter) Initialize(config *schedulerpb.SchedulerProviderConfig) error {
	a.enabled = true
	log.Printf("[MockSchedulerAdapter] Initialized")
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (a *MockSchedulerAdapter) IsEnabled() bool {
	return a.enabled
}

// IsHealthy checks if the scheduler service is available
func (a *MockSchedulerAdapter) IsHealthy(ctx context.Context) error {
	return nil
}

// Close cleans up adapter resources
func (a *MockSchedulerAdapter) Close() error {
	a.enabled = false
	return nil
}

// GetCapabilities returns the capabilities supported by this mock provider
func (a *MockSchedulerAdapter) GetCapabilities() []schedulerpb.SchedulerCapability {
	return []schedulerpb.SchedulerCapability{
		schedulerpb.SchedulerCapability_SCHEDULER_CAPABILITY_CHECK_AVAILABILITY,
		schedulerpb.SchedulerCapability_SCHEDULER_CAPABILITY_WEBHOOKS,
	}
}

// CreateSchedule creates a mock scheduled event
func (a *MockSchedulerAdapter) CreateSchedule(ctx context.Context, req *schedulerpb.CreateScheduleRequest) (*schedulerpb.CreateScheduleResponse, error) {
	log.Printf("[MockSchedulerAdapter] CreateSchedule called")

	if req.Data == nil {
		return &schedulerpb.CreateScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	return &schedulerpb.CreateScheduleResponse{
		Success: true,
		Data: []*schedulerpb.Schedule{
			{
				Id:                 "mock-schedule-001",
				ProviderScheduleId: "mock-provider-001",
				ProviderId:         "mock_scheduler",
				ProviderType:       schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_MOCK,
				Name:               "Mock Schedule",
				Status:             schedulerpb.ScheduleStatus_SCHEDULE_STATUS_ACTIVE,
				StartDate:          req.Data.StartDate,
				StartTime:          req.Data.StartTime,
				EndDate:            req.Data.EndDate,
				EndTime:            req.Data.EndTime,
			},
		},
	}, nil
}

// CancelSchedule cancels a mock scheduled event
func (a *MockSchedulerAdapter) CancelSchedule(ctx context.Context, req *schedulerpb.CancelScheduleRequest) (*schedulerpb.CancelScheduleResponse, error) {
	if req.Data == nil {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("[MockSchedulerAdapter] CancelSchedule called for: %s", req.Data.ScheduleId)
	return &schedulerpb.CancelScheduleResponse{
		Success: true,
		Data: []*schedulerpb.ScheduleCancelResult{
			{
				Status:  schedulerpb.ScheduleStatus_SCHEDULE_STATUS_CANCELLED,
				Message: "Mock schedule cancelled",
			},
		},
	}, nil
}

// GetSchedule retrieves mock schedule details
func (a *MockSchedulerAdapter) GetSchedule(ctx context.Context, req *schedulerpb.GetScheduleRequest) (*schedulerpb.GetScheduleResponse, error) {
	if req.Data == nil {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("[MockSchedulerAdapter] GetSchedule called for: %s", req.Data.ScheduleId)
	return &schedulerpb.GetScheduleResponse{
		Success: true,
		Data: []*schedulerpb.Schedule{
			{
				Id:                 req.Data.ScheduleId,
				ProviderScheduleId: req.Data.ProviderScheduleId,
				ProviderId:         "mock_scheduler",
				ProviderType:       schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_MOCK,
				Name:               "Mock Schedule",
				Status:             schedulerpb.ScheduleStatus_SCHEDULE_STATUS_ACTIVE,
			},
		},
	}, nil
}

// ListSchedules lists mock scheduled events
func (a *MockSchedulerAdapter) ListSchedules(ctx context.Context, req *schedulerpb.ListSchedulesRequest) (*schedulerpb.ListSchedulesResponse, error) {
	log.Printf("[MockSchedulerAdapter] ListSchedules called")
	return &schedulerpb.ListSchedulesResponse{
		Success: true,
		Data:    []*schedulerpb.Schedule{},
	}, nil
}

// CheckAvailability checks mock available time slots
func (a *MockSchedulerAdapter) CheckAvailability(ctx context.Context, req *schedulerpb.CheckAvailabilityRequest) (*schedulerpb.CheckAvailabilityResponse, error) {
	if req.Data == nil {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("[MockSchedulerAdapter] CheckAvailability called for: %s", req.Data.EventTypeId)
	return &schedulerpb.CheckAvailabilityResponse{
		Success: true,
		Data: []*schedulerpb.TimeSlot{
			{
				StartDate:   req.Data.StartDate,
				StartTime:   "09:00",
				EndDate:     req.Data.StartDate,
				EndTime:     "09:30",
				IsAvailable: true,
			},
			{
				StartDate:   req.Data.StartDate,
				StartTime:   "10:00",
				EndDate:     req.Data.StartDate,
				EndTime:     "10:30",
				IsAvailable: true,
			},
		},
	}, nil
}

// ProcessWebhook processes mock incoming webhook
func (a *MockSchedulerAdapter) ProcessWebhook(ctx context.Context, req *schedulerpb.ProcessSchedulerWebhookRequest) (*schedulerpb.ProcessSchedulerWebhookResponse, error) {
	if req.Data == nil {
		return &schedulerpb.ProcessSchedulerWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("[MockSchedulerAdapter] ProcessWebhook called")
	return &schedulerpb.ProcessSchedulerWebhookResponse{
		Success: true,
		Data: []*schedulerpb.SchedulerWebhookResult{
			{
				EventType: "mock.event",
				Action:    "created",
				Schedule: &schedulerpb.Schedule{
					Id:           "mock-webhook-001",
					ProviderId:   "mock_scheduler",
					ProviderType: schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_MOCK,
					Status:       schedulerpb.ScheduleStatus_SCHEDULE_STATUS_ACTIVE,
				},
			},
		},
	}, nil
}

// ListEventTypes lists mock event types
func (a *MockSchedulerAdapter) ListEventTypes(ctx context.Context, req *schedulerpb.ListEventTypesRequest) (*schedulerpb.ListEventTypesResponse, error) {
	log.Printf("[MockSchedulerAdapter] ListEventTypes called")
	return &schedulerpb.ListEventTypesResponse{
		Success: true,
		Data: []*schedulerpb.EventType{
			{
				Uri:             "mock://event-types/30min",
				Name:            "30 Minute Meeting",
				Active:          true,
				Slug:            "30min",
				DurationMinutes: 30,
				SchedulingUrl:   "https://mock.calendly.com/30min",
			},
			{
				Uri:             "mock://event-types/60min",
				Name:            "60 Minute Meeting",
				Active:          true,
				Slug:            "60min",
				DurationMinutes: 60,
				SchedulingUrl:   "https://mock.calendly.com/60min",
			},
		},
	}, nil
}

// GetEventType retrieves mock event type details
func (a *MockSchedulerAdapter) GetEventType(ctx context.Context, req *schedulerpb.GetEventTypeRequest) (*schedulerpb.GetEventTypeResponse, error) {
	if req.Data == nil {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("[MockSchedulerAdapter] GetEventType called for: %s", req.Data.EventTypeId)
	return &schedulerpb.GetEventTypeResponse{
		Success: true,
		Data: []*schedulerpb.EventType{
			{
				Uri:             req.Data.EventTypeId,
				Name:            "Mock Event Type",
				Active:          true,
				Slug:            "mock-event",
				DurationMinutes: 30,
				SchedulingUrl:   "https://mock.calendly.com/mock-event",
			},
		},
	}, nil
}
