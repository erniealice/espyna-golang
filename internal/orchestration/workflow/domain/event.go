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
	registerEventClientUseCases(useCases, register)
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
