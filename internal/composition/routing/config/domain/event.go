package domain

import (
	"fmt"

	"leapfor.xyz/espyna/internal/application/usecases/event"
	"leapfor.xyz/espyna/internal/composition/contracts"

	// Protobuf imports with module naming pattern
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventattributepb "leapfor.xyz/esqyma/golang/v1/domain/event/event_attribute"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// ConfigureEventDomain configures routes for the Event domain with use cases injected directly
func ConfigureEventDomain(eventUseCases *event.EventUseCases) contracts.DomainRouteConfiguration {
	// Handle nil use cases gracefully for backward compatibility
	if eventUseCases == nil {
		fmt.Printf("⚠️  Event use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "event",
			Prefix:  "/event",
			Enabled: false,                            // Disable until use cases are properly initialized
			Routes:  []contracts.RouteConfiguration{}, // No routes without use cases
		}
	}

	fmt.Printf("✅ Event use cases are properly initialized!\n")

	// Initialize routes array
	routes := []contracts.RouteConfiguration{}

	// Event module routes
	if eventUseCases.Event != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event/create",
			Handler: contracts.NewGenericHandler(eventUseCases.Event.CreateEvent, &eventpb.CreateEventRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event/read",
			Handler: contracts.NewGenericHandler(eventUseCases.Event.ReadEvent, &eventpb.ReadEventRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event/update",
			Handler: contracts.NewGenericHandler(eventUseCases.Event.UpdateEvent, &eventpb.UpdateEventRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event/delete",
			Handler: contracts.NewGenericHandler(eventUseCases.Event.DeleteEvent, &eventpb.DeleteEventRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event/list",
			Handler: contracts.NewGenericHandler(eventUseCases.Event.ListEvents, &eventpb.ListEventsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event/get-list-page-data",
			Handler: contracts.NewGenericHandler(eventUseCases.Event.GetEventListPageData, &eventpb.GetEventListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event/get-item-page-data",
			Handler: contracts.NewGenericHandler(eventUseCases.Event.GetEventItemPageData, &eventpb.GetEventItemPageDataRequest{}),
		})
	}

	// EventAttribute module routes
	if eventUseCases.EventAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-attribute/create",
			Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.CreateEventAttribute, &eventattributepb.CreateEventAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-attribute/read",
			Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.ReadEventAttribute, &eventattributepb.ReadEventAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-attribute/update",
			Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.UpdateEventAttribute, &eventattributepb.UpdateEventAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-attribute/delete",
			Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.DeleteEventAttribute, &eventattributepb.DeleteEventAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-attribute/list",
			Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.ListEventAttributes, &eventattributepb.ListEventAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.GetEventAttributeListPageData, &eventattributepb.GetEventAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.GetEventAttributeItemPageData, &eventattributepb.GetEventAttributeItemPageDataRequest{}),
		})
	}

	// EventClient module routes
	if eventUseCases.EventClient != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-client/create",
			Handler: contracts.NewGenericHandler(eventUseCases.EventClient.CreateEventClient, &eventclientpb.CreateEventClientRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-client/read",
			Handler: contracts.NewGenericHandler(eventUseCases.EventClient.ReadEventClient, &eventclientpb.ReadEventClientRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-client/update",
			Handler: contracts.NewGenericHandler(eventUseCases.EventClient.UpdateEventClient, &eventclientpb.UpdateEventClientRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-client/delete",
			Handler: contracts.NewGenericHandler(eventUseCases.EventClient.DeleteEventClient, &eventclientpb.DeleteEventClientRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-client/list",
			Handler: contracts.NewGenericHandler(eventUseCases.EventClient.ListEventClients, &eventclientpb.ListEventClientsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-client/get-list-page-data",
			Handler: contracts.NewGenericHandler(eventUseCases.EventClient.GetEventClientListPageData, &eventclientpb.GetEventClientListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/event/event-client/get-item-page-data",
			Handler: contracts.NewGenericHandler(eventUseCases.EventClient.GetEventClientItemPageData, &eventclientpb.GetEventClientItemPageDataRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "event",
		Prefix:  "/event",
		Enabled: true,
		Routes:  routes,
	}
}
