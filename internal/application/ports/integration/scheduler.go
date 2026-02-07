package integration

import (
	"context"

	schedulerpb "leapfor.xyz/esqyma/golang/v1/integration/scheduler"
)

// SchedulerProvider defines the contract for scheduler providers
// This interface abstracts scheduling services like Calendly, Google Calendar, etc.
// following the hexagonal architecture pattern established for PaymentProvider and EmailProvider.
type SchedulerProvider interface {
	// Name returns the name of the scheduler provider (e.g., "calendly", "google_calendar", "mock")
	Name() string

	// Initialize sets up the scheduler provider with the given configuration
	Initialize(config *schedulerpb.SchedulerProviderConfig) error

	// CreateSchedule creates a new scheduled event (books an appointment)
	// Returns a schedule with all details including provider-specific IDs
	CreateSchedule(ctx context.Context, req *schedulerpb.CreateScheduleRequest) (*schedulerpb.CreateScheduleResponse, error)

	// CancelSchedule cancels an existing scheduled event
	CancelSchedule(ctx context.Context, req *schedulerpb.CancelScheduleRequest) (*schedulerpb.CancelScheduleResponse, error)

	// GetSchedule retrieves schedule details by ID
	GetSchedule(ctx context.Context, req *schedulerpb.GetScheduleRequest) (*schedulerpb.GetScheduleResponse, error)

	// ListSchedules lists scheduled events with filtering
	ListSchedules(ctx context.Context, req *schedulerpb.ListSchedulesRequest) (*schedulerpb.ListSchedulesResponse, error)

	// CheckAvailability checks available time slots for booking
	CheckAvailability(ctx context.Context, req *schedulerpb.CheckAvailabilityRequest) (*schedulerpb.CheckAvailabilityResponse, error)

	// ProcessWebhook processes incoming webhook/callback from the scheduler provider
	// Parses and validates webhook data, returns schedule details
	ProcessWebhook(ctx context.Context, req *schedulerpb.ProcessSchedulerWebhookRequest) (*schedulerpb.ProcessSchedulerWebhookResponse, error)

	// ListEventTypes lists available event types from the provider
	ListEventTypes(ctx context.Context, req *schedulerpb.ListEventTypesRequest) (*schedulerpb.ListEventTypesResponse, error)

	// GetEventType retrieves event type details
	GetEventType(ctx context.Context, req *schedulerpb.GetEventTypeRequest) (*schedulerpb.GetEventTypeResponse, error)

	// IsHealthy checks if the scheduler service is available
	IsHealthy(ctx context.Context) error

	// Close cleans up scheduler provider resources
	Close() error

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool

	// GetCapabilities returns the capabilities supported by this provider
	GetCapabilities() []schedulerpb.SchedulerCapability
}

// ScheduleWebhookResult represents the result of processing a scheduler webhook
// This is a convenience type for use cases that need to act on webhook results
type ScheduleWebhookResult struct {
	// Success indicates if webhook processing was successful
	Success bool

	// EventType is the webhook event type (e.g., "invitee.created", "invitee.canceled")
	EventType string

	// Schedule contains the parsed schedule details
	Schedule *schedulerpb.Schedule

	// Action describes what action was taken (created, cancelled, rescheduled, etc.)
	Action string

	// IsReschedule indicates if this was a reschedule event
	IsReschedule bool

	// OldScheduleID is the previous schedule ID (for reschedules)
	OldScheduleID string

	// Error contains any error that occurred during processing
	Error error
}

// CreateScheduleParams provides a simplified parameter structure for creating schedules
// Consumer adapters can use this for convenience instead of the full protobuf request
type CreateScheduleParams struct {
	// EventTypeID is the provider event type to book
	EventTypeID string

	// StartDate in format YYYY-MM-DD
	StartDate string

	// StartTime in format HH:mm (24-hour)
	StartTime string

	// EndDate in format YYYY-MM-DD (optional, defaults to StartDate)
	EndDate string

	// EndTime in format HH:mm (24-hour, optional)
	EndTime string

	// Invitee details
	InviteeName     string
	InviteeEmail    string
	InviteePhone    string
	InviteeTimezone string

	// Location preference
	LocationType string
	LocationURL  string

	// Associated entity references
	ClientID       string
	SubscriptionID string
	PaymentID      string

	// Custom metadata
	Metadata map[string]string
}

// ToProtoRequest converts CreateScheduleParams to the protobuf request type
func (p *CreateScheduleParams) ToProtoRequest() *schedulerpb.CreateScheduleRequest {
	data := &schedulerpb.ScheduleCreateData{
		EventTypeId:    p.EventTypeID,
		StartDate:      p.StartDate,
		StartTime:      p.StartTime,
		EndDate:        p.EndDate,
		EndTime:        p.EndTime,
		ClientId:       p.ClientID,
		SubscriptionId: p.SubscriptionID,
		PaymentId:      p.PaymentID,
		Metadata:       p.Metadata,
		Invitee: &schedulerpb.InviteeInfo{
			Name:     p.InviteeName,
			Email:    p.InviteeEmail,
			Phone:    p.InviteePhone,
			Timezone: p.InviteeTimezone,
		},
	}

	if p.LocationType != "" {
		data.Location = &schedulerpb.ScheduleLocation{
			Type:    p.LocationType,
			JoinUrl: p.LocationURL,
		}
	}

	return &schedulerpb.CreateScheduleRequest{
		Data: data,
	}
}

// CheckAvailabilityParams provides a simplified parameter structure for checking availability
type CheckAvailabilityParams struct {
	// EventTypeID is the event type to check availability for
	EventTypeID string

	// StartDate in format YYYY-MM-DD
	StartDate string

	// StartTime in format HH:mm (24-hour, optional)
	StartTime string

	// EndDate in format YYYY-MM-DD
	EndDate string

	// EndTime in format HH:mm (24-hour, optional)
	EndTime string

	// Timezone in IANA format (e.g., "Asia/Singapore")
	Timezone string
}

// ToProtoRequest converts CheckAvailabilityParams to the protobuf request type
func (p *CheckAvailabilityParams) ToProtoRequest() *schedulerpb.CheckAvailabilityRequest {
	return &schedulerpb.CheckAvailabilityRequest{
		Data: &schedulerpb.AvailabilityCheckData{
			EventTypeId: p.EventTypeID,
			StartDate:   p.StartDate,
			StartTime:   p.StartTime,
			EndDate:     p.EndDate,
			EndTime:     p.EndTime,
			Timezone:    p.Timezone,
		},
	}
}
