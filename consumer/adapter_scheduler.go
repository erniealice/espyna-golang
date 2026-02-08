package consumer

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Scheduler Adapter

Provides direct access to scheduler operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY scheduler provider (Calendly, Google Calendar, etc.)
based on your CONFIG_SCHEDULER_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewSchedulerAdapterFromContainer(container)

	// Check availability
	slots, err := adapter.CheckAvailability(ctx, consumer.CheckAvailabilityParams{
	    EventTypeID: "https://api.calendly.com/event_types/abc123",
	    StartDate:   "2025-12-23",
	    EndDate:     "2025-12-30",
	    Timezone:    "Asia/Singapore",
	})

	// Process webhook
	result, err := adapter.ProcessWebhook(ctx, payload, "application/json", headers)

	// Cancel schedule
	err := adapter.CancelSchedule(ctx, "schedule-id", "No longer needed")
*/

// SchedulerAdapter provides technology-agnostic access to scheduling services.
// It wraps the SchedulerProvider interface and works with Calendly, Google Calendar, etc.
type SchedulerAdapter struct {
	provider  ports.SchedulerProvider
	container *Container
}

// NewSchedulerAdapterFromContainer creates a SchedulerAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's provider.
func NewSchedulerAdapterFromContainer(container *Container) *SchedulerAdapter {
	if container == nil {
		return nil
	}

	provider := container.GetSchedulerProvider()
	if provider == nil {
		return nil
	}

	return &SchedulerAdapter{
		provider:  provider,
		container: container,
	}
}

// Close closes the scheduler adapter.
// Note: If created from container, this does NOT close the container.
func (a *SchedulerAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetProvider returns the underlying SchedulerProvider for advanced operations.
func (a *SchedulerAdapter) GetProvider() ports.SchedulerProvider {
	return a.provider
}

// Name returns the name of the underlying scheduler provider (e.g., "calendly", "google_calendar", "mock")
func (a *SchedulerAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the scheduler provider is enabled
func (a *SchedulerAdapter) IsEnabled() bool {
	return a.provider != nil && a.provider.IsEnabled()
}

// --- Scheduler Operations ---

// CreateScheduleRequest creates a new scheduled event using proto request/response types.
// This is the recommended method for handlers as it uses domain types directly.
// The adapter handles transformation to provider-specific formats.
func (a *SchedulerAdapter) CreateScheduleRequest(ctx context.Context, req *schedulerpb.CreateScheduleRequest) (*schedulerpb.CreateScheduleResponse, error) {
	if a.provider == nil {
		return &schedulerpb.CreateScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_NOT_INITIALIZED",
				Message: "scheduler provider not initialized",
			},
		}, nil
	}

	resp, err := a.provider.CreateSchedule(ctx, req)
	if err != nil {
		return &schedulerpb.CreateScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_ERROR",
				Message: err.Error(),
			},
		}, nil
	}
	return resp, nil
}

// CreateSchedule creates a new scheduled event using simplified params.
// Note: Prefer CreateScheduleRequest for handler use. This method is for convenience.
func (a *SchedulerAdapter) CreateSchedule(ctx context.Context, params CreateScheduleParams) (*schedulerpb.Schedule, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("scheduler provider not initialized")
	}

	req := params.ToProtoRequest()
	resp, err := a.provider.CreateSchedule(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return nil, fmt.Errorf("failed to create schedule: %s", errMsg)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no schedule data returned")
	}
	return resp.Data[0], nil
}

// CancelSchedule cancels an existing scheduled event.
func (a *SchedulerAdapter) CancelSchedule(ctx context.Context, scheduleID string, reason string) error {
	if a.provider == nil {
		return fmt.Errorf("scheduler provider not initialized")
	}

	req := &schedulerpb.CancelScheduleRequest{
		Data: &schedulerpb.ScheduleCancelData{
			ProviderScheduleId: scheduleID,
			Reason:             reason,
			NotifyInvitee:      true,
		},
	}

	resp, err := a.provider.CancelSchedule(ctx, req)
	if err != nil {
		return err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return fmt.Errorf("failed to cancel schedule: %s", errMsg)
	}
	return nil
}

// GetSchedule retrieves schedule details by ID.
func (a *SchedulerAdapter) GetSchedule(ctx context.Context, scheduleID string) (*schedulerpb.Schedule, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("scheduler provider not initialized")
	}

	req := &schedulerpb.GetScheduleRequest{
		Data: &schedulerpb.ScheduleLookup{
			ProviderScheduleId: scheduleID,
		},
	}

	resp, err := a.provider.GetSchedule(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return nil, fmt.Errorf("failed to get schedule: %s", errMsg)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no schedule data returned")
	}
	return resp.Data[0], nil
}

// ListSchedules lists scheduled events with optional filtering.
func (a *SchedulerAdapter) ListSchedules(ctx context.Context, fromDate, toDate, status string, limit int32) ([]*schedulerpb.Schedule, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("scheduler provider not initialized")
	}

	req := &schedulerpb.ListSchedulesRequest{
		Data: &schedulerpb.ScheduleListFilter{
			FromDate: fromDate,
			ToDate:   toDate,
			Status:   status,
			Limit:    limit,
		},
	}

	resp, err := a.provider.ListSchedules(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// CheckAvailability checks available time slots for the given event type and date range.
func (a *SchedulerAdapter) CheckAvailability(ctx context.Context, params CheckAvailabilityParams) ([]*schedulerpb.TimeSlot, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("scheduler provider not initialized")
	}

	req := params.ToProtoRequest()
	resp, err := a.provider.CheckAvailability(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return nil, fmt.Errorf("failed to check availability: %s", errMsg)
	}
	return resp.Data, nil
}

// ProcessWebhook processes an incoming scheduler webhook.
func (a *SchedulerAdapter) ProcessWebhook(ctx context.Context, payload []byte, contentType string, headers map[string]string) (*ScheduleWebhookResult, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("scheduler provider not initialized")
	}

	req := &schedulerpb.ProcessSchedulerWebhookRequest{
		Data: &schedulerpb.SchedulerWebhookData{
			ProviderId:  a.provider.Name(),
			Payload:     payload,
			Headers:     headers,
			ContentType: contentType,
		},
	}

	resp, err := a.provider.ProcessWebhook(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return &ScheduleWebhookResult{
			Success: false,
			Action:  "error",
			Error:   fmt.Errorf("%s", errMsg),
		}, nil
	}

	if len(resp.Data) == 0 {
		return &ScheduleWebhookResult{
			Success: false,
			Action:  "error",
			Error:   fmt.Errorf("no webhook data returned"),
		}, nil
	}

	result := resp.Data[0]
	return &ScheduleWebhookResult{
		Success:       true,
		EventType:     result.EventType,
		Schedule:      result.Schedule,
		Action:        result.Action,
		IsReschedule:  result.IsReschedule,
		OldScheduleID: result.OldScheduleId,
	}, nil
}

// ListEventTypes lists available event types from the provider.
func (a *SchedulerAdapter) ListEventTypes(ctx context.Context, activeOnly bool) ([]*schedulerpb.EventType, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("scheduler provider not initialized")
	}

	req := &schedulerpb.ListEventTypesRequest{
		Data: &schedulerpb.EventTypeListFilter{
			ActiveOnly: activeOnly,
		},
	}

	resp, err := a.provider.ListEventTypes(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return nil, fmt.Errorf("failed to list event types: %s", errMsg)
	}
	return resp.Data, nil
}

// GetEventType retrieves event type details by ID.
func (a *SchedulerAdapter) GetEventType(ctx context.Context, eventTypeID string) (*schedulerpb.EventType, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("scheduler provider not initialized")
	}

	req := &schedulerpb.GetEventTypeRequest{
		Data: &schedulerpb.EventTypeLookup{
			EventTypeId: eventTypeID,
		},
	}

	resp, err := a.provider.GetEventType(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
		}
		return nil, fmt.Errorf("failed to get event type: %s", errMsg)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no event type data returned")
	}
	return resp.Data[0], nil
}

// IsHealthy checks if the scheduler provider is healthy and available.
func (a *SchedulerAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("scheduler provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// GetCapabilities returns the capabilities supported by the scheduler provider.
func (a *SchedulerAdapter) GetCapabilities() []schedulerpb.SchedulerCapability {
	if a.provider == nil {
		return nil
	}
	return a.provider.GetCapabilities()
}

// --- Re-export types for consumer convenience ---

// CreateScheduleParams re-exports the CreateScheduleParams type for consumer convenience
type CreateScheduleParams = ports.CreateScheduleParams

// CheckAvailabilityParams re-exports the CheckAvailabilityParams type for consumer convenience
type CheckAvailabilityParams = ports.CheckAvailabilityParams

// ScheduleWebhookResult re-exports the ScheduleWebhookResult type for consumer convenience
type ScheduleWebhookResult = ports.ScheduleWebhookResult
