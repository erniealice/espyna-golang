package fulfillment

import (
	"errors"
	"fmt"
)

// FulfillmentStatus represents canonical fulfillment statuses.
type FulfillmentStatus string

const (
	StatusPending            FulfillmentStatus = "PENDING"
	StatusReady              FulfillmentStatus = "READY"
	StatusInTransit          FulfillmentStatus = "IN_TRANSIT"
	StatusDelivered          FulfillmentStatus = "DELIVERED"
	StatusPartiallyDelivered FulfillmentStatus = "PARTIALLY_DELIVERED"
	StatusFailed             FulfillmentStatus = "FAILED"
	StatusCancelled          FulfillmentStatus = "CANCELLED"
)

// FulfillmentEvent represents a valid state transition trigger.
type FulfillmentEvent string

const (
	EventMarkReady      FulfillmentEvent = "mark_ready"
	EventDispatch       FulfillmentEvent = "dispatch"
	EventDeliver        FulfillmentEvent = "deliver"
	EventDeliverPartial FulfillmentEvent = "deliver_partial"
	EventMarkFailed     FulfillmentEvent = "mark_failed"
	EventCancel         FulfillmentEvent = "cancel"
	EventRetry          FulfillmentEvent = "retry"
)

// ErrInvalidTransition is returned when an event is not valid for the current status.
var ErrInvalidTransition = errors.New("invalid fulfillment transition")

// transitionTable maps (from status, event) → target status.
var transitionTable = map[FulfillmentStatus]map[FulfillmentEvent]FulfillmentStatus{
	StatusPending: {
		EventMarkReady:  StatusReady,
		EventMarkFailed: StatusFailed,
		EventCancel:     StatusCancelled,
	},
	StatusReady: {
		EventDispatch:       StatusInTransit,
		EventDeliver:        StatusDelivered,
		EventDeliverPartial: StatusPartiallyDelivered,
		EventMarkFailed:     StatusFailed,
		EventCancel:         StatusCancelled,
	},
	StatusInTransit: {
		EventDeliver:        StatusDelivered,
		EventDeliverPartial: StatusPartiallyDelivered,
		EventMarkFailed:     StatusFailed,
		EventCancel:         StatusCancelled,
	},
	StatusPartiallyDelivered: {
		EventDeliver:    StatusDelivered,
		EventMarkFailed: StatusFailed,
		EventCancel:     StatusCancelled,
	},
	StatusFailed: {
		EventRetry:  StatusPending,
		EventCancel: StatusCancelled,
	},
	// Terminal states — no valid transitions.
	StatusDelivered: {},
	StatusCancelled: {},
}

// ValidateTransition returns the target status for the given event, or ErrInvalidTransition
// if the event is not valid from the current status.
func ValidateTransition(from FulfillmentStatus, event FulfillmentEvent) (FulfillmentStatus, error) {
	events, ok := transitionTable[from]
	if !ok {
		return "", fmt.Errorf("%w: unknown status %q", ErrInvalidTransition, from)
	}
	to, ok := events[event]
	if !ok {
		return "", fmt.Errorf("%w: event %q is not allowed from status %q", ErrInvalidTransition, event, from)
	}
	return to, nil
}

// AllowedEvents returns the list of valid events from the given status.
// Returns an empty slice for terminal statuses or unknown statuses.
func AllowedEvents(from FulfillmentStatus) []FulfillmentEvent {
	events, ok := transitionTable[from]
	if !ok {
		return []FulfillmentEvent{}
	}
	result := make([]FulfillmentEvent, 0, len(events))
	for event := range events {
		result = append(result, event)
	}
	return result
}

// IsTerminal reports whether the given status is a terminal state (no further transitions possible).
func IsTerminal(status FulfillmentStatus) bool {
	return status == StatusDelivered || status == StatusCancelled
}
