package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/usecases/event"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	// Protobuf imports
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventAttendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
	eventOccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
	eventProductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	eventRecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
	eventResourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// ConfigureEventDomain configures routes for the Event domain with use cases injected directly
func ConfigureEventDomain(eventUseCases *event.EventUseCases) contracts.DomainRouteConfiguration {
	if eventUseCases == nil {
		fmt.Printf("⚠️  Event use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "event",
			Prefix:  "/event",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	fmt.Printf("✅ Event use cases are properly initialized!\n")

	routes := []contracts.RouteConfiguration{}

	// Event core routes
	if eventUseCases.Event != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event/create", Handler: contracts.NewGenericHandler(eventUseCases.Event.CreateEvent, &eventpb.CreateEventRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event/read", Handler: contracts.NewGenericHandler(eventUseCases.Event.ReadEvent, &eventpb.ReadEventRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event/update", Handler: contracts.NewGenericHandler(eventUseCases.Event.UpdateEvent, &eventpb.UpdateEventRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event/delete", Handler: contracts.NewGenericHandler(eventUseCases.Event.DeleteEvent, &eventpb.DeleteEventRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event/list", Handler: contracts.NewGenericHandler(eventUseCases.Event.ListEvents, &eventpb.ListEventsRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.Event.GetEventListPageData, &eventpb.GetEventListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.Event.GetEventItemPageData, &eventpb.GetEventItemPageDataRequest{})},
		)
	}

	// EventAttendee routes
	if eventUseCases.EventAttendee != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attendee/create", Handler: contracts.NewGenericHandler(eventUseCases.EventAttendee.CreateEventAttendee, &eventAttendeepb.CreateEventAttendeeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attendee/read", Handler: contracts.NewGenericHandler(eventUseCases.EventAttendee.ReadEventAttendee, &eventAttendeepb.ReadEventAttendeeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attendee/update", Handler: contracts.NewGenericHandler(eventUseCases.EventAttendee.UpdateEventAttendee, &eventAttendeepb.UpdateEventAttendeeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attendee/delete", Handler: contracts.NewGenericHandler(eventUseCases.EventAttendee.DeleteEventAttendee, &eventAttendeepb.DeleteEventAttendeeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attendee/list", Handler: contracts.NewGenericHandler(eventUseCases.EventAttendee.ListEventAttendees, &eventAttendeepb.ListEventAttendeesRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attendee/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventAttendee.GetEventAttendeeListPageData, &eventAttendeepb.GetEventAttendeeListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attendee/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventAttendee.GetEventAttendeeItemPageData, &eventAttendeepb.GetEventAttendeeItemPageDataRequest{})},
		)
	}

	// EventAttribute routes
	if eventUseCases.EventAttribute != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attribute/create", Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.CreateEventAttribute, &eventattributepb.CreateEventAttributeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attribute/read", Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.ReadEventAttribute, &eventattributepb.ReadEventAttributeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attribute/update", Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.UpdateEventAttribute, &eventattributepb.UpdateEventAttributeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attribute/delete", Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.DeleteEventAttribute, &eventattributepb.DeleteEventAttributeRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attribute/list", Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.ListEventAttributes, &eventattributepb.ListEventAttributesRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attribute/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.GetEventAttributeListPageData, &eventattributepb.GetEventAttributeListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-attribute/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventAttribute.GetEventAttributeItemPageData, &eventattributepb.GetEventAttributeItemPageDataRequest{})},
		)
	}

	// EventClient routes
	if eventUseCases.EventClient != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-client/create", Handler: contracts.NewGenericHandler(eventUseCases.EventClient.CreateEventClient, &eventclientpb.CreateEventClientRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-client/read", Handler: contracts.NewGenericHandler(eventUseCases.EventClient.ReadEventClient, &eventclientpb.ReadEventClientRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-client/update", Handler: contracts.NewGenericHandler(eventUseCases.EventClient.UpdateEventClient, &eventclientpb.UpdateEventClientRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-client/delete", Handler: contracts.NewGenericHandler(eventUseCases.EventClient.DeleteEventClient, &eventclientpb.DeleteEventClientRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-client/list", Handler: contracts.NewGenericHandler(eventUseCases.EventClient.ListEventClients, &eventclientpb.ListEventClientsRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-client/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventClient.GetEventClientListPageData, &eventclientpb.GetEventClientListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-client/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventClient.GetEventClientItemPageData, &eventclientpb.GetEventClientItemPageDataRequest{})},
		)
	}

	// EventOccurrence routes (read-only — no create/update/delete)
	if eventUseCases.EventOccurrence != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-occurrence/list", Handler: contracts.NewGenericHandler(eventUseCases.EventOccurrence.ListEventOccurrences, &eventOccurrencepb.ListEventOccurrencesRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-occurrence/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventOccurrence.GetEventOccurrenceListPageData, &eventOccurrencepb.GetEventOccurrenceListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-occurrence/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventOccurrence.GetEventOccurrenceItemPageData, &eventOccurrencepb.GetEventOccurrenceItemPageDataRequest{})},
		)
	}

	// EventProduct routes (CRUD, no PageData)
	if eventUseCases.EventProduct != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-product/create", Handler: contracts.NewGenericHandler(eventUseCases.EventProduct.CreateEventProduct, &eventProductpb.CreateEventProductRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-product/read", Handler: contracts.NewGenericHandler(eventUseCases.EventProduct.ReadEventProduct, &eventProductpb.ReadEventProductRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-product/update", Handler: contracts.NewGenericHandler(eventUseCases.EventProduct.UpdateEventProduct, &eventProductpb.UpdateEventProductRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-product/delete", Handler: contracts.NewGenericHandler(eventUseCases.EventProduct.DeleteEventProduct, &eventProductpb.DeleteEventProductRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-product/list", Handler: contracts.NewGenericHandler(eventUseCases.EventProduct.ListEventProducts, &eventProductpb.ListEventProductsRequest{})},
		)
	}

	// EventRecurrence routes (CRUD + PageData)
	if eventUseCases.EventRecurrence != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-recurrence/create", Handler: contracts.NewGenericHandler(eventUseCases.EventRecurrence.CreateEventRecurrence, &eventRecurrencepb.CreateEventRecurrenceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-recurrence/read", Handler: contracts.NewGenericHandler(eventUseCases.EventRecurrence.ReadEventRecurrence, &eventRecurrencepb.ReadEventRecurrenceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-recurrence/update", Handler: contracts.NewGenericHandler(eventUseCases.EventRecurrence.UpdateEventRecurrence, &eventRecurrencepb.UpdateEventRecurrenceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-recurrence/delete", Handler: contracts.NewGenericHandler(eventUseCases.EventRecurrence.DeleteEventRecurrence, &eventRecurrencepb.DeleteEventRecurrenceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-recurrence/list", Handler: contracts.NewGenericHandler(eventUseCases.EventRecurrence.ListEventRecurrences, &eventRecurrencepb.ListEventRecurrencesRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-recurrence/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventRecurrence.GetEventRecurrenceListPageData, &eventRecurrencepb.GetEventRecurrenceListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-recurrence/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventRecurrence.GetEventRecurrenceItemPageData, &eventRecurrencepb.GetEventRecurrenceItemPageDataRequest{})},
		)
	}

	// EventResource routes (CRUD + PageData)
	if eventUseCases.EventResource != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-resource/create", Handler: contracts.NewGenericHandler(eventUseCases.EventResource.CreateEventResource, &eventResourcepb.CreateEventResourceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-resource/read", Handler: contracts.NewGenericHandler(eventUseCases.EventResource.ReadEventResource, &eventResourcepb.ReadEventResourceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-resource/update", Handler: contracts.NewGenericHandler(eventUseCases.EventResource.UpdateEventResource, &eventResourcepb.UpdateEventResourceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-resource/delete", Handler: contracts.NewGenericHandler(eventUseCases.EventResource.DeleteEventResource, &eventResourcepb.DeleteEventResourceRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-resource/list", Handler: contracts.NewGenericHandler(eventUseCases.EventResource.ListEventResources, &eventResourcepb.ListEventResourcesRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-resource/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventResource.GetEventResourceListPageData, &eventResourcepb.GetEventResourceListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-resource/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventResource.GetEventResourceItemPageData, &eventResourcepb.GetEventResourceItemPageDataRequest{})},
		)
	}

	// EventTag routes (CRUD + PageData) — master list of tags per workspace.
	if eventUseCases.EventTag != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-tag/create", Handler: contracts.NewGenericHandler(eventUseCases.EventTag.CreateEventTag, &eventtagpb.CreateEventTagRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-tag/read", Handler: contracts.NewGenericHandler(eventUseCases.EventTag.ReadEventTag, &eventtagpb.ReadEventTagRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-tag/update", Handler: contracts.NewGenericHandler(eventUseCases.EventTag.UpdateEventTag, &eventtagpb.UpdateEventTagRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-tag/delete", Handler: contracts.NewGenericHandler(eventUseCases.EventTag.DeleteEventTag, &eventtagpb.DeleteEventTagRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-tag/list", Handler: contracts.NewGenericHandler(eventUseCases.EventTag.ListEventTags, &eventtagpb.ListEventTagsRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-tag/get-list-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventTag.GetEventTagListPageData, &eventtagpb.GetEventTagListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/event/event-tag/get-item-page-data", Handler: contracts.NewGenericHandler(eventUseCases.EventTag.GetEventTagItemPageData, &eventtagpb.GetEventTagItemPageDataRequest{})},
		)
	}

	// NOTE: event_tag_assignment has no standalone HTTP routes — it is driven
	// through the per-event tag picker's Set use case, which is invoked from
	// the Event detail page view (Phase 4) rather than directly via HTTP.

	return contracts.DomainRouteConfiguration{
		Domain:  "event",
		Prefix:  "/event",
		Enabled: true,
		Routes:  routes,
	}
}
