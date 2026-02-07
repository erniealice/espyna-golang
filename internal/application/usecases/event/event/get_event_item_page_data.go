package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
)

type GetEventItemPageDataRepositories struct {
	Event eventpb.EventDomainServiceServer
}

type GetEventItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetEventItemPageDataUseCase handles the business logic for getting event item page data
// with specialized scheduling and calendar context
type GetEventItemPageDataUseCase struct {
	repositories GetEventItemPageDataRepositories
	services     GetEventItemPageDataServices
}

// NewGetEventItemPageDataUseCase creates a new GetEventItemPageDataUseCase
func NewGetEventItemPageDataUseCase(
	repositories GetEventItemPageDataRepositories,
	services GetEventItemPageDataServices,
) *GetEventItemPageDataUseCase {
	return &GetEventItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event item page data operation with scheduling context
func (uc *GetEventItemPageDataUseCase) Execute(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) (*eventpb.GetEventItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.EventId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event item page data retrieval within a transaction
func (uc *GetEventItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) (*eventpb.GetEventItemPageDataResponse, error) {
	var result *eventpb.GetEventItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"event.errors.item_page_data_failed",
				"event item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting event item page data
func (uc *GetEventItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) (*eventpb.GetEventItemPageDataResponse, error) {
	// Create read request for the event
	readReq := &eventpb.ReadEventRequest{
		Data: &eventpb.Event{
			Id: req.EventId,
		},
	}

	// Retrieve the event
	readResp, err := uc.repositories.Event.ReadEvent(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.errors.read_failed",
			"failed to retrieve event: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.errors.not_found",
			"event not found",
		))
	}

	// Get the event (should be only one)
	event := readResp.Data[0]

	// Validate that we got the expected event
	if event.Id != req.EventId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.errors.id_mismatch",
			"retrieved event ID does not match requested ID",
		))
	}

	// Apply event-specific enhancements for scheduling context
	enhancedEvent, err := uc.enhanceEventWithSchedulingData(ctx, event)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.errors.enhancement_failed",
			"failed to enhance event with scheduling data: %w",
		), err)
	}

	return &eventpb.GetEventItemPageDataResponse{
		Event:   enhancedEvent,
		Success: true,
	}, nil
}

// enhanceEventWithSchedulingData enriches the event with additional scheduling context
func (uc *GetEventItemPageDataUseCase) enhanceEventWithSchedulingData(
	ctx context.Context,
	event *eventpb.Event,
) (*eventpb.Event, error) {
	// Create a copy to avoid modifying the original and prevent lock copying
	enhancedEvent := proto.Clone(event).(*eventpb.Event)

	// Convert timestamps for easier manipulation
	startTime := time.Unix(event.StartDateTimeUtc, 0)
	endTime := time.Unix(event.EndDateTimeUtc, 0)

	// Load timezone if specified
	var eventTimezone *time.Location
	if event.Timezone != "" {
		loc, err := time.LoadLocation(event.Timezone)
		if err != nil {
			// Log warning but don't fail - fallback to UTC
			eventTimezone = time.UTC
		} else {
			eventTimezone = loc
		}
	} else {
		eventTimezone = time.UTC
	}

	// Apply timezone to start and end times for display
	startTimeInTZ := startTime.In(eventTimezone)
	endTimeInTZ := endTime.In(eventTimezone)

	// Update string representations with timezone-aware formatting
	startTimeStr := startTimeInTZ.Format(time.RFC3339)
	endTimeStr := endTimeInTZ.Format(time.RFC3339)
	enhancedEvent.StartDateTimeUtcString = &startTimeStr
	enhancedEvent.EndDateTimeUtcString = &endTimeStr

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (attendees, location details, recurring pattern info)
	// 2. Check for scheduling conflicts with other events
	// 3. Calculate derived fields (duration, is_past, is_future, etc.)
	// 4. Apply business rules for event visibility/access control
	// 5. Add calendar integration metadata
	// 6. Load instructor/facilitator information
	// 7. Check capacity and enrollment status
	// 8. Add audit logging for event access

	return enhancedEvent, nil
}

// validateInput validates the input request
func (uc *GetEventItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *eventpb.GetEventItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.request_required",
			"request is required",
		))
	}

	if req.EventId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.id_required",
			"event ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading event item page data
func (uc *GetEventItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	eventId string,
) error {
	// Validate event ID format
	if len(eventId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event.validation.id_too_short",
			"event ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this event
	// - Validate event belongs to the current user's organization
	// - Check if event is in a state that allows viewing (not cancelled/deleted)
	// - Rate limiting for event access
	// - Audit logging requirements
	// - Time-based access restrictions (e.g., historical events may have different permissions)

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedSchedulingData loads related entities like attendees, location, etc.
// This would be called from executeCore if needed
func (uc *GetEventItemPageDataUseCase) loadRelatedSchedulingData(
	ctx context.Context,
	event *eventpb.Event,
) error {
	// TODO: Implement loading of related scheduling data
	// This could involve calls to various repositories to populate:
	// - Attendee/participant information
	// - Location/room details and availability
	// - Instructor/facilitator information
	// - Recurring event pattern details
	// - Related events (series, dependencies)
	// - Calendar integration metadata

	// Example implementation would be:
	// if event.LocationId != "" {
	//     // Load location details and check availability
	// }
	// if event.InstructorId != "" {
	//     // Load instructor information
	// }
	// if event.RecurrencePattern != nil {
	//     // Load recurring pattern and related occurrences
	// }

	return nil
}

// applySchedulingTransformation applies event-specific data transformations
func (uc *GetEventItemPageDataUseCase) applySchedulingTransformation(
	ctx context.Context,
	event *eventpb.Event,
) *eventpb.Event {
	// TODO: Apply transformations needed for optimal frontend scheduling consumption
	// This could include:
	// - Converting timestamps to user's preferred timezone
	// - Computing derived fields (duration, time until start, etc.)
	// - Formatting display strings for different calendar views
	// - Adding calendar export links/data
	// - Computing conflict detection results
	// - Adding availability windows
	// - Applying localization for date/time display

	return event
}

// checkSchedulingPermissions validates user has permission to access this event's scheduling data
func (uc *GetEventItemPageDataUseCase) checkSchedulingPermissions(
	ctx context.Context,
	eventId string,
) error {
	// TODO: Implement proper access control for scheduling data
	// This could involve:
	// - Checking user role/permissions for event access
	// - Validating event belongs to user's organization/workspace
	// - Applying time-based access controls (e.g., future vs. past events)
	// - Checking instructor/facilitator permissions
	// - Applying participant/attendee visibility rules
	// - Multi-tenant access controls for educational institutions

	return nil
}

// detectSchedulingConflicts checks for conflicts with other events
func (uc *GetEventItemPageDataUseCase) detectSchedulingConflicts(
	ctx context.Context,
	event *eventpb.Event,
) ([]*eventpb.Event, error) {
	// TODO: Implement scheduling conflict detection
	// This would involve:
	// - Finding overlapping events for the same resources (room, instructor, etc.)
	// - Checking participant availability conflicts
	// - Validating against business rules (maximum daily hours, break requirements)
	// - Checking against institutional calendars and holidays
	// - Identifying potential scheduling optimization opportunities

	return nil, nil
}

// calculateSchedulingMetrics computes useful metrics for the event
func (uc *GetEventItemPageDataUseCase) calculateSchedulingMetrics(
	ctx context.Context,
	event *eventpb.Event,
) map[string]interface{} {
	// TODO: Calculate scheduling-specific metrics
	// This could include:
	// - Duration calculations in various units
	// - Time until start/end
	// - Utilization metrics for resources
	// - Attendance/capacity ratios
	// - Historical timing analysis
	// - Calendar density metrics

	return make(map[string]interface{})
}
