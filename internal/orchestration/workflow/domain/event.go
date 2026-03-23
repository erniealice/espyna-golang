package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/executor"
)

// RegisterEventUseCases registers all event domain use cases with the registry.
func RegisterEventUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event == nil {
		return
	}

	registerEventCoreUseCases(useCases, register)
	registerEventAttendeeUseCases(useCases, register)
	registerEventClientUseCases(useCases, register)
	registerEventOccurrenceUseCases(useCases, register)
	registerEventProductUseCases(useCases, register)
	registerEventRecurrenceUseCases(useCases, register)
	registerEventResourceUseCases(useCases, register)
}

func registerEventCoreUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event.Event == nil {
		return
	}
	if useCases.Event.Event.CreateEvent != nil {
		register("event.event.create", executor.New(useCases.Event.Event.CreateEvent.Execute))
	}
	if useCases.Event.Event.ReadEvent != nil {
		register("event.event.read", executor.New(useCases.Event.Event.ReadEvent.Execute))
	}
	if useCases.Event.Event.UpdateEvent != nil {
		register("event.event.update", executor.New(useCases.Event.Event.UpdateEvent.Execute))
	}
	if useCases.Event.Event.DeleteEvent != nil {
		register("event.event.delete", executor.New(useCases.Event.Event.DeleteEvent.Execute))
	}
	if useCases.Event.Event.ListEvents != nil {
		register("event.event.list", executor.New(useCases.Event.Event.ListEvents.Execute))
	}
}

func registerEventAttendeeUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event.EventAttendee == nil {
		return
	}
	if useCases.Event.EventAttendee.CreateEventAttendee != nil {
		register("event.event_attendee.create", executor.New(useCases.Event.EventAttendee.CreateEventAttendee.Execute))
	}
	if useCases.Event.EventAttendee.ReadEventAttendee != nil {
		register("event.event_attendee.read", executor.New(useCases.Event.EventAttendee.ReadEventAttendee.Execute))
	}
	if useCases.Event.EventAttendee.UpdateEventAttendee != nil {
		register("event.event_attendee.update", executor.New(useCases.Event.EventAttendee.UpdateEventAttendee.Execute))
	}
	if useCases.Event.EventAttendee.DeleteEventAttendee != nil {
		register("event.event_attendee.delete", executor.New(useCases.Event.EventAttendee.DeleteEventAttendee.Execute))
	}
	if useCases.Event.EventAttendee.ListEventAttendees != nil {
		register("event.event_attendee.list", executor.New(useCases.Event.EventAttendee.ListEventAttendees.Execute))
	}
}

func registerEventClientUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event.EventClient == nil {
		return
	}
	if useCases.Event.EventClient.CreateEventClient != nil {
		register("event.event_client.create", executor.New(useCases.Event.EventClient.CreateEventClient.Execute))
	}
	if useCases.Event.EventClient.ReadEventClient != nil {
		register("event.event_client.read", executor.New(useCases.Event.EventClient.ReadEventClient.Execute))
	}
	if useCases.Event.EventClient.UpdateEventClient != nil {
		register("event.event_client.update", executor.New(useCases.Event.EventClient.UpdateEventClient.Execute))
	}
	if useCases.Event.EventClient.DeleteEventClient != nil {
		register("event.event_client.delete", executor.New(useCases.Event.EventClient.DeleteEventClient.Execute))
	}
	if useCases.Event.EventClient.ListEventClients != nil {
		register("event.event_client.list", executor.New(useCases.Event.EventClient.ListEventClients.Execute))
	}
}

func registerEventOccurrenceUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event.EventOccurrence == nil {
		return
	}
	if useCases.Event.EventOccurrence.ListEventOccurrences != nil {
		register("event.event_occurrence.list", executor.New(useCases.Event.EventOccurrence.ListEventOccurrences.Execute))
	}
}

func registerEventProductUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event.EventProduct == nil {
		return
	}
	if useCases.Event.EventProduct.CreateEventProduct != nil {
		register("event.event_product.create", executor.New(useCases.Event.EventProduct.CreateEventProduct.Execute))
	}
	if useCases.Event.EventProduct.ReadEventProduct != nil {
		register("event.event_product.read", executor.New(useCases.Event.EventProduct.ReadEventProduct.Execute))
	}
	if useCases.Event.EventProduct.UpdateEventProduct != nil {
		register("event.event_product.update", executor.New(useCases.Event.EventProduct.UpdateEventProduct.Execute))
	}
	if useCases.Event.EventProduct.DeleteEventProduct != nil {
		register("event.event_product.delete", executor.New(useCases.Event.EventProduct.DeleteEventProduct.Execute))
	}
	if useCases.Event.EventProduct.ListEventProducts != nil {
		register("event.event_product.list", executor.New(useCases.Event.EventProduct.ListEventProducts.Execute))
	}
}

func registerEventRecurrenceUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event.EventRecurrence == nil {
		return
	}
	if useCases.Event.EventRecurrence.CreateEventRecurrence != nil {
		register("event.event_recurrence.create", executor.New(useCases.Event.EventRecurrence.CreateEventRecurrence.Execute))
	}
	if useCases.Event.EventRecurrence.ReadEventRecurrence != nil {
		register("event.event_recurrence.read", executor.New(useCases.Event.EventRecurrence.ReadEventRecurrence.Execute))
	}
	if useCases.Event.EventRecurrence.UpdateEventRecurrence != nil {
		register("event.event_recurrence.update", executor.New(useCases.Event.EventRecurrence.UpdateEventRecurrence.Execute))
	}
	if useCases.Event.EventRecurrence.DeleteEventRecurrence != nil {
		register("event.event_recurrence.delete", executor.New(useCases.Event.EventRecurrence.DeleteEventRecurrence.Execute))
	}
	if useCases.Event.EventRecurrence.ListEventRecurrences != nil {
		register("event.event_recurrence.list", executor.New(useCases.Event.EventRecurrence.ListEventRecurrences.Execute))
	}
}

func registerEventResourceUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Event.EventResource == nil {
		return
	}
	if useCases.Event.EventResource.CreateEventResource != nil {
		register("event.event_resource.create", executor.New(useCases.Event.EventResource.CreateEventResource.Execute))
	}
	if useCases.Event.EventResource.ReadEventResource != nil {
		register("event.event_resource.read", executor.New(useCases.Event.EventResource.ReadEventResource.Execute))
	}
	if useCases.Event.EventResource.UpdateEventResource != nil {
		register("event.event_resource.update", executor.New(useCases.Event.EventResource.UpdateEventResource.Execute))
	}
	if useCases.Event.EventResource.DeleteEventResource != nil {
		register("event.event_resource.delete", executor.New(useCases.Event.EventResource.DeleteEventResource.Execute))
	}
	if useCases.Event.EventResource.ListEventResources != nil {
		register("event.event_resource.list", executor.New(useCases.Event.EventResource.ListEventResources.Execute))
	}
}
