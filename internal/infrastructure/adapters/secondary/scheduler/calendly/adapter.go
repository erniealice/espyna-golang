//go:build calendly

package calendly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	schedulerpb "leapfor.xyz/esqyma/golang/v1/integration/scheduler"
)

func init() {
	// Register with the global registry
	registry.RegisterSchedulerBuildFromEnv("calendly", func() (ports.SchedulerProvider, error) {
		adapter := NewCalendlyAdapterFromEnv()
		if adapter == nil || !adapter.IsEnabled() {
			return nil, fmt.Errorf("failed to create Calendly adapter from environment")
		}
		return adapter, nil
	})
	log.Printf("[CalendlyAdapter] Registered with scheduler registry")
}

const (
	DefaultAPIBaseURL = "https://api.calendly.com"
	DefaultTimeout    = 30 * time.Second
)

// CalendlyAdapter implements the SchedulerProvider interface for Calendly
type CalendlyAdapter struct {
	config      *schedulerpb.SchedulerProviderConfig
	httpClient  *http.Client
	accessToken string
	userURI     string
	orgURI      string
	enabled     bool
}

// NewCalendlyAdapter creates a new Calendly adapter
func NewCalendlyAdapter() *CalendlyAdapter {
	return &CalendlyAdapter{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		enabled:    false,
	}
}

// NewCalendlyAdapterFromEnv creates a new Calendly adapter from environment variables
func NewCalendlyAdapterFromEnv() *CalendlyAdapter {
	adapter := NewCalendlyAdapter()

	accessToken := os.Getenv("CALENDLY_PERSONAL_ACCESS_TOKEN")
	if accessToken == "" {
		log.Printf("[CalendlyAdapter] CALENDLY_PERSONAL_ACCESS_TOKEN not set, adapter will be disabled")
		return adapter
	}

	config := &schedulerpb.SchedulerProviderConfig{
		ProviderName:       "calendly",
		AccessToken:        accessToken,
		ApiBaseUrl:         os.Getenv("CALENDLY_API_BASE_URL"),
		DefaultEventTypeId: os.Getenv("CALENDLY_DEFAULT_EVENT_TYPE_ID"),
		UserUri:            os.Getenv("CALENDLY_USER_URI"),
		OrganizationUri:    os.Getenv("CALENDLY_ORGANIZATION_URI"),
		WebhookSecret:      os.Getenv("CALENDLY_WEBHOOK_SECRET"),
	}

	if err := adapter.Initialize(config); err != nil {
		log.Printf("[CalendlyAdapter] Failed to initialize: %v", err)
		return adapter
	}

	return adapter
}

// Name returns the name of the scheduler provider
func (a *CalendlyAdapter) Name() string {
	return "calendly"
}

// Initialize sets up the Calendly adapter with the given configuration
func (a *CalendlyAdapter) Initialize(config *schedulerpb.SchedulerProviderConfig) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	if config.AccessToken == "" {
		return fmt.Errorf("access token is required")
	}

	a.config = config
	a.accessToken = config.AccessToken
	a.userURI = config.UserUri
	a.orgURI = config.OrganizationUri

	// If user URI not provided, fetch it from the API
	if a.userURI == "" {
		userURI, err := a.fetchCurrentUserURI()
		if err != nil {
			log.Printf("[CalendlyAdapter] Warning: failed to fetch user URI: %v", err)
		} else {
			a.userURI = userURI
		}
	}

	a.enabled = true
	log.Printf("[CalendlyAdapter] Initialized successfully")
	log.Printf("  User URI: %s", a.userURI)

	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (a *CalendlyAdapter) IsEnabled() bool {
	return a.enabled
}

// IsHealthy checks if the Calendly service is available
func (a *CalendlyAdapter) IsHealthy(ctx context.Context) error {
	if !a.enabled {
		return fmt.Errorf("Calendly adapter is disabled")
	}

	// Simple health check - fetch current user
	_, err := a.fetchCurrentUserURI()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close cleans up adapter resources
func (a *CalendlyAdapter) Close() error {
	a.enabled = false
	return nil
}

// GetCapabilities returns the capabilities supported by Calendly
func (a *CalendlyAdapter) GetCapabilities() []schedulerpb.SchedulerCapability {
	return []schedulerpb.SchedulerCapability{
		schedulerpb.SchedulerCapability_SCHEDULER_CAPABILITY_CANCEL_EVENT,
		schedulerpb.SchedulerCapability_SCHEDULER_CAPABILITY_CHECK_AVAILABILITY,
		schedulerpb.SchedulerCapability_SCHEDULER_CAPABILITY_WEBHOOKS,
		schedulerpb.SchedulerCapability_SCHEDULER_CAPABILITY_INVITEE_MANAGEMENT,
		schedulerpb.SchedulerCapability_SCHEDULER_CAPABILITY_CUSTOM_QUESTIONS,
	}
}

// CreateSchedule creates a new scheduled event
// Note: Calendly doesn't support programmatic event creation - events are created when
// invitees book via the scheduling link. This method returns an error.
func (a *CalendlyAdapter) CreateSchedule(ctx context.Context, req *schedulerpb.CreateScheduleRequest) (*schedulerpb.CreateScheduleResponse, error) {
	return &schedulerpb.CreateScheduleResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:    "UNSUPPORTED_OPERATION",
			Message: "Calendly does not support programmatic event creation. Invitees must book via the scheduling link.",
		},
	}, nil
}

// CancelSchedule cancels an existing scheduled event
func (a *CalendlyAdapter) CancelSchedule(ctx context.Context, req *schedulerpb.CancelScheduleRequest) (*schedulerpb.CancelScheduleResponse, error) {
	if !a.enabled {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Calendly adapter is disabled",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	eventID := req.Data.ProviderScheduleId
	if eventID == "" {
		eventID = req.Data.ScheduleId
	}

	if eventID == "" {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Schedule ID is required",
			},
		}, nil
	}

	// Build cancellation URL
	cancelURL := fmt.Sprintf("%s/scheduled_events/%s/cancellation", DefaultAPIBaseURL, eventID)

	cancelBody := map[string]string{}
	if req.Data.Reason != "" {
		cancelBody["reason"] = req.Data.Reason
	}
	bodyBytes, _ := json.Marshal(cancelBody)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", cancelURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "REQUEST_FAILED",
				Message: fmt.Sprintf("Failed to create request: %v", err),
			},
		}, nil
	}

	httpReq.Header.Set("Authorization", "Bearer "+a.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Failed to cancel event: %v", err),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 201 Created = success, 403 = already cancelled, 404 = not found
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Calendly API returned status %d: %s", resp.StatusCode, string(body)),
			},
		}, nil
	}

	return &schedulerpb.CancelScheduleResponse{
		Success: true,
		Data: []*schedulerpb.ScheduleCancelResult{
			{
				Status:  schedulerpb.ScheduleStatus_SCHEDULE_STATUS_CANCELLED,
				Message: "Event cancelled successfully",
			},
		},
	}, nil
}

// GetSchedule retrieves schedule details
func (a *CalendlyAdapter) GetSchedule(ctx context.Context, req *schedulerpb.GetScheduleRequest) (*schedulerpb.GetScheduleResponse, error) {
	if !a.enabled {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Calendly adapter is disabled",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	eventID := req.Data.ProviderScheduleId
	if eventID == "" {
		eventID = req.Data.ScheduleId
	}

	if eventID == "" {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Schedule ID is required",
			},
		}, nil
	}

	eventURL := fmt.Sprintf("%s/scheduled_events/%s", DefaultAPIBaseURL, eventID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", eventURL, nil)
	if err != nil {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "REQUEST_FAILED",
				Message: fmt.Sprintf("Failed to create request: %v", err),
			},
		}, nil
	}

	httpReq.Header.Set("Authorization", "Bearer "+a.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Failed to get event: %v", err),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Calendly API returned status %d: %s", resp.StatusCode, string(body)),
			},
		}, nil
	}

	// Parse response
	var eventResp CalendlyEventResponse
	if err := json.Unmarshal(body, &eventResp); err != nil {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PARSE_ERROR",
				Message: fmt.Sprintf("Failed to parse response: %v", err),
			},
		}, nil
	}

	schedule := a.convertEventToSchedule(&eventResp.Resource)

	return &schedulerpb.GetScheduleResponse{
		Success: true,
		Data:    []*schedulerpb.Schedule{schedule},
	}, nil
}

// ListSchedules lists scheduled events
func (a *CalendlyAdapter) ListSchedules(ctx context.Context, req *schedulerpb.ListSchedulesRequest) (*schedulerpb.ListSchedulesResponse, error) {
	if !a.enabled {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Calendly adapter is disabled",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Build URL with query parameters
	url := fmt.Sprintf("%s/scheduled_events?user=%s", DefaultAPIBaseURL, a.userURI)

	// Add date filters
	if req.Data.FromDate != "" {
		// Convert YYYY-MM-DD to RFC3339
		minStartTime, _ := time.Parse("2006-01-02", req.Data.FromDate)
		url += fmt.Sprintf("&min_start_time=%s", minStartTime.Format(time.RFC3339))
	} else {
		// Default to now
		url += fmt.Sprintf("&min_start_time=%s", time.Now().Format(time.RFC3339))
	}

	if req.Data.ToDate != "" {
		maxStartTime, _ := time.Parse("2006-01-02", req.Data.ToDate)
		url += fmt.Sprintf("&max_start_time=%s", maxStartTime.Format(time.RFC3339))
	} else {
		// Default to 30 days ahead
		url += fmt.Sprintf("&max_start_time=%s", time.Now().Add(30*24*time.Hour).Format(time.RFC3339))
	}

	if req.Data.Status != "" {
		url += fmt.Sprintf("&status=%s", req.Data.Status)
	}

	if req.Data.Limit > 0 {
		url += fmt.Sprintf("&count=%d", req.Data.Limit)
	}

	if req.Data.PageToken != "" {
		url += fmt.Sprintf("&page_token=%s", req.Data.PageToken)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "REQUEST_FAILED",
				Message: fmt.Sprintf("Failed to create request: %v", err),
			},
		}, nil
	}

	httpReq.Header.Set("Authorization", "Bearer "+a.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Failed to list events: %v", err),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Calendly API returned status %d: %s", resp.StatusCode, string(body)),
			},
		}, nil
	}

	var listResp CalendlyListEventsResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PARSE_ERROR",
				Message: fmt.Sprintf("Failed to parse response: %v", err),
			},
		}, nil
	}

	schedules := make([]*schedulerpb.Schedule, 0, len(listResp.Collection))
	for _, event := range listResp.Collection {
		schedules = append(schedules, a.convertEventToSchedule(&event))
	}

	return &schedulerpb.ListSchedulesResponse{
		Success:       true,
		Data:          schedules,
		NextPageToken: listResp.Pagination.NextPageToken,
		TotalCount:    int32(len(schedules)),
	}, nil
}

// CheckAvailability checks available time slots
func (a *CalendlyAdapter) CheckAvailability(ctx context.Context, req *schedulerpb.CheckAvailabilityRequest) (*schedulerpb.CheckAvailabilityResponse, error) {
	if !a.enabled {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Calendly adapter is disabled",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	if req.Data.EventTypeId == "" {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Event type ID is required",
			},
		}, nil
	}

	// Build availability URL
	url := fmt.Sprintf("%s/event_type_available_times", DefaultAPIBaseURL)

	// Parse dates to RFC3339
	startTime, err := time.Parse("2006-01-02", req.Data.StartDate)
	if err != nil {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_DATE",
				Message: fmt.Sprintf("Invalid start date format: %v", err),
			},
		}, nil
	}

	endTime, err := time.Parse("2006-01-02", req.Data.EndDate)
	if err != nil {
		endTime = startTime.Add(7 * 24 * time.Hour) // Default to 7 days
	}

	// Add time if provided
	if req.Data.StartTime != "" {
		parsed, err := time.Parse("15:04", req.Data.StartTime)
		if err == nil {
			startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
				parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
		}
	}

	if req.Data.EndTime != "" {
		parsed, err := time.Parse("15:04", req.Data.EndTime)
		if err == nil {
			endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(),
				parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
		}
	}

	url += fmt.Sprintf("?event_type=%s", req.Data.EventTypeId)
	url += fmt.Sprintf("&start_time=%s", startTime.Format(time.RFC3339))
	url += fmt.Sprintf("&end_time=%s", endTime.Format(time.RFC3339))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "REQUEST_FAILED",
				Message: fmt.Sprintf("Failed to create request: %v", err),
			},
		}, nil
	}

	httpReq.Header.Set("Authorization", "Bearer "+a.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Failed to check availability: %v", err),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Calendly API returned status %d: %s", resp.StatusCode, string(body)),
			},
		}, nil
	}

	var availResp CalendlyAvailabilityResponse
	if err := json.Unmarshal(body, &availResp); err != nil {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PARSE_ERROR",
				Message: fmt.Sprintf("Failed to parse response: %v", err),
			},
		}, nil
	}

	slots := make([]*schedulerpb.TimeSlot, 0, len(availResp.Collection))
	for _, slot := range availResp.Collection {
		slotStart, _ := time.Parse(time.RFC3339, slot.StartTime)
		slots = append(slots, &schedulerpb.TimeSlot{
			StartDate:         slotStart.Format("2006-01-02"),
			StartTime:         slotStart.Format("15:04"),
			EndDate:           slotStart.Format("2006-01-02"),
			EndTime:           slotStart.Add(time.Duration(slot.Duration) * time.Minute).Format("15:04"),
			IsAvailable:       slot.Status == "available",
			SchedulingLink:    slot.SchedulingURL,
			InviteesRemaining: int32(slot.InviteesRemaining),
			StartTimeIso:      slot.StartTime,
		})
	}

	return &schedulerpb.CheckAvailabilityResponse{
		Success: true,
		Data:    slots,
	}, nil
}

// ProcessWebhook processes incoming Calendly webhook
func (a *CalendlyAdapter) ProcessWebhook(ctx context.Context, req *schedulerpb.ProcessSchedulerWebhookRequest) (*schedulerpb.ProcessSchedulerWebhookResponse, error) {
	if !a.enabled {
		return &schedulerpb.ProcessSchedulerWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Calendly adapter is disabled",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.ProcessSchedulerWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Parse webhook payload
	var webhook CalendlyWebhookPayload
	if err := json.Unmarshal(req.Data.Payload, &webhook); err != nil {
		return &schedulerpb.ProcessSchedulerWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PARSE_ERROR",
				Message: fmt.Sprintf("Failed to parse webhook payload: %v", err),
			},
		}, nil
	}

	log.Printf("[CalendlyAdapter] Processing webhook: %s", webhook.Event)

	// Determine action based on event type
	var action string
	var status schedulerpb.ScheduleStatus
	isReschedule := false
	var oldScheduleID string

	switch webhook.Event {
	case "invitee.created":
		action = "created"
		status = schedulerpb.ScheduleStatus_SCHEDULE_STATUS_ACTIVE
		// Check if this is a reschedule
		if webhook.Payload.OldInvitee != "" {
			isReschedule = true
			action = "rescheduled"
			status = schedulerpb.ScheduleStatus_SCHEDULE_STATUS_RESCHEDULED
			// Extract old event UUID from old_invitee URL
			oldScheduleID = extractEventUUID(webhook.Payload.OldInvitee)
		}
	case "invitee.canceled":
		action = "cancelled"
		status = schedulerpb.ScheduleStatus_SCHEDULE_STATUS_CANCELLED
	default:
		action = "no_action"
		status = schedulerpb.ScheduleStatus_SCHEDULE_STATUS_UNSPECIFIED
	}

	// Build schedule from webhook
	schedule := a.convertWebhookToSchedule(&webhook)
	schedule.Status = status

	return &schedulerpb.ProcessSchedulerWebhookResponse{
		Success: true,
		Data: []*schedulerpb.SchedulerWebhookResult{
			{
				EventType:     webhook.Event,
				Schedule:      schedule,
				Action:        action,
				IsReschedule:  isReschedule,
				OldScheduleId: oldScheduleID,
			},
		},
	}, nil
}

// ListEventTypes lists available event types
func (a *CalendlyAdapter) ListEventTypes(ctx context.Context, req *schedulerpb.ListEventTypesRequest) (*schedulerpb.ListEventTypesResponse, error) {
	if !a.enabled {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Calendly adapter is disabled",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	url := fmt.Sprintf("%s/event_types?user=%s", DefaultAPIBaseURL, a.userURI)
	if req.Data.ActiveOnly {
		url += "&active=true"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "REQUEST_FAILED",
				Message: fmt.Sprintf("Failed to create request: %v", err),
			},
		}, nil
	}

	httpReq.Header.Set("Authorization", "Bearer "+a.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Failed to list event types: %v", err),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Calendly API returned status %d: %s", resp.StatusCode, string(body)),
			},
		}, nil
	}

	var listResp CalendlyEventTypesResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PARSE_ERROR",
				Message: fmt.Sprintf("Failed to parse response: %v", err),
			},
		}, nil
	}

	eventTypes := make([]*schedulerpb.EventType, 0, len(listResp.Collection))
	for _, et := range listResp.Collection {
		eventTypes = append(eventTypes, &schedulerpb.EventType{
			Uri:             et.URI,
			Name:            et.Name,
			Active:          et.Active,
			Slug:            et.Slug,
			DurationMinutes: int32(et.Duration),
			SchedulingUrl:   et.SchedulingURL,
			Description:     et.Description,
			Color:           et.Color,
			Secret:          et.Secret,
			Type:            et.Type,
		})
	}

	return &schedulerpb.ListEventTypesResponse{
		Success: true,
		Data:    eventTypes,
	}, nil
}

// GetEventType retrieves event type details
func (a *CalendlyAdapter) GetEventType(ctx context.Context, req *schedulerpb.GetEventTypeRequest) (*schedulerpb.GetEventTypeResponse, error) {
	if !a.enabled {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Calendly adapter is disabled",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	eventTypeURI := req.Data.EventTypeId
	if !strings.HasPrefix(eventTypeURI, "https://") {
		eventTypeURI = fmt.Sprintf("%s/event_types/%s", DefaultAPIBaseURL, req.Data.EventTypeId)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", eventTypeURI, nil)
	if err != nil {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "REQUEST_FAILED",
				Message: fmt.Sprintf("Failed to create request: %v", err),
			},
		}, nil
	}

	httpReq.Header.Set("Authorization", "Bearer "+a.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Failed to get event type: %v", err),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "API_ERROR",
				Message: fmt.Sprintf("Calendly API returned status %d: %s", resp.StatusCode, string(body)),
			},
		}, nil
	}

	var etResp CalendlyEventTypeResponse
	if err := json.Unmarshal(body, &etResp); err != nil {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PARSE_ERROR",
				Message: fmt.Sprintf("Failed to parse response: %v", err),
			},
		}, nil
	}

	return &schedulerpb.GetEventTypeResponse{
		Success: true,
		Data: []*schedulerpb.EventType{
			{
				Uri:             etResp.Resource.URI,
				Name:            etResp.Resource.Name,
				Active:          etResp.Resource.Active,
				Slug:            etResp.Resource.Slug,
				DurationMinutes: int32(etResp.Resource.Duration),
				SchedulingUrl:   etResp.Resource.SchedulingURL,
				Description:     etResp.Resource.Description,
				Color:           etResp.Resource.Color,
				Secret:          etResp.Resource.Secret,
				Type:            etResp.Resource.Type,
			},
		},
	}, nil
}

// Helper methods

func (a *CalendlyAdapter) fetchCurrentUserURI() (string, error) {
	req, err := http.NewRequest("GET", DefaultAPIBaseURL+"/users/me", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Calendly API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Resource struct {
			URI string `json:"uri"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Resource.URI, nil
}

func (a *CalendlyAdapter) convertEventToSchedule(event *CalendlyEvent) *schedulerpb.Schedule {
	// Parse times
	startTime, _ := time.Parse(time.RFC3339, event.StartTime)
	endTime, _ := time.Parse(time.RFC3339, event.EndTime)
	createdAt, _ := time.Parse(time.RFC3339, event.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, event.UpdatedAt)

	// Extract event ID from URI
	eventID := extractEventUUID(event.URI)

	// Convert status
	var status schedulerpb.ScheduleStatus
	switch event.Status {
	case "active":
		status = schedulerpb.ScheduleStatus_SCHEDULE_STATUS_ACTIVE
	case "canceled":
		status = schedulerpb.ScheduleStatus_SCHEDULE_STATUS_CANCELLED
	default:
		status = schedulerpb.ScheduleStatus_SCHEDULE_STATUS_UNSPECIFIED
	}

	return &schedulerpb.Schedule{
		ProviderScheduleId: eventID,
		ProviderId:         "calendly",
		ProviderType:       schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_CALENDLY,
		Name:               event.Name,
		Status:             status,
		StartDate:          startTime.Format("2006-01-02"),
		StartTime:          startTime.Format("15:04"),
		EndDate:            endTime.Format("2006-01-02"),
		EndTime:            endTime.Format("15:04"),
		Timezone:           event.Timezone,
		DurationMinutes:    int32(endTime.Sub(startTime).Minutes()),
		CancelUrl:          event.CancellationURL,
		RescheduleUrl:      event.RescheduleURL,
		CreatedAt:          timestamppb.New(createdAt),
		UpdatedAt:          timestamppb.New(updatedAt),
		Location:           a.convertLocation(event.Location),
	}
}

func (a *CalendlyAdapter) convertWebhookToSchedule(webhook *CalendlyWebhookPayload) *schedulerpb.Schedule {
	// Extract event ID from event URL
	eventID := extractEventUUID(webhook.Payload.Event)

	// Parse scheduled event times if available
	var startTime, endTime time.Time
	if webhook.Payload.ScheduledEvent != nil {
		startTime, _ = time.Parse(time.RFC3339, webhook.Payload.ScheduledEvent.StartTime)
		endTime, _ = time.Parse(time.RFC3339, webhook.Payload.ScheduledEvent.EndTime)
	}

	// Build invitee info
	invitee := &schedulerpb.InviteeInfo{
		Name:     webhook.Payload.Name,
		Email:    webhook.Payload.Email,
		Timezone: webhook.Payload.Timezone,
		Uri:      webhook.Payload.URI,
	}

	// Parse custom questions
	if len(webhook.Payload.QuestionsAndAnswers) > 0 {
		invitee.CustomAnswers = make(map[string]string)
		for _, qa := range webhook.Payload.QuestionsAndAnswers {
			invitee.CustomAnswers[qa.Question] = qa.Answer
		}
	}

	return &schedulerpb.Schedule{
		ProviderScheduleId: eventID,
		ProviderId:         "calendly",
		ProviderType:       schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_CALENDLY,
		Name:               webhook.Payload.ScheduledEvent.Name,
		StartDate:          startTime.Format("2006-01-02"),
		StartTime:          startTime.Format("15:04"),
		EndDate:            endTime.Format("2006-01-02"),
		EndTime:            endTime.Format("15:04"),
		DurationMinutes:    int32(endTime.Sub(startTime).Minutes()),
		Invitee:            invitee,
		CancelUrl:          webhook.Payload.CancelURL,
		RescheduleUrl:      webhook.Payload.RescheduleURL,
		CreatedAt:          timestamppb.Now(),
	}
}

func (a *CalendlyAdapter) convertLocation(loc *CalendlyLocation) *schedulerpb.ScheduleLocation {
	if loc == nil {
		return nil
	}

	return &schedulerpb.ScheduleLocation{
		Type:     loc.Type,
		Location: loc.Location,
		JoinUrl:  loc.JoinURL,
	}
}

func extractEventUUID(uri string) string {
	if uri == "" {
		return ""
	}
	parts := strings.Split(uri, "/")
	if len(parts) == 0 {
		return ""
	}
	// For invitee URLs, get the scheduled_events UUID
	for i, part := range parts {
		if part == "scheduled_events" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// Otherwise return the last part
	return parts[len(parts)-1]
}
