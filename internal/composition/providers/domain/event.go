package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Entity domain
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"

	// Protobuf domain services - Product domain
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"

	// Protobuf domain services - Event domain
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventAttendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
	eventOccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
	eventProductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	eventRecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
	eventResourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
)

// EventRepositories contains all event domain repositories and cross-domain dependencies
// Event domain: Event, EventAttendee, EventAttribute, EventClient, EventOccurrence, EventProduct, EventRecurrence, EventResource (8 entities)
// Cross-domain: Client (needed by EventClient), Product (needed by EventProduct)
type EventRepositories struct {
	Event           eventpb.EventDomainServiceServer
	EventAttendee   eventAttendeepb.EventAttendeeDomainServiceServer
	EventAttribute  eventattributepb.EventAttributeDomainServiceServer
	EventClient     eventclientpb.EventClientDomainServiceServer
	EventOccurrence eventOccurrencepb.EventOccurrenceDomainServiceServer
	EventProduct    eventProductpb.EventProductDomainServiceServer
	EventRecurrence eventRecurrencepb.EventRecurrenceDomainServiceServer
	EventResource   eventResourcepb.EventResourceDomainServiceServer
	// Cross-domain dependencies
	Client  clientpb.ClientDomainServiceServer
	Product productpb.ProductDomainServiceServer
}

// NewEventRepositories creates and returns a new set of EventRepositories
func NewEventRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*EventRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Event domain repositories
	eventRepo, err := repoCreator.CreateRepository(entityid.Event, conn, tableConfig.TableName(entityid.Event))
	if err != nil {
		return nil, fmt.Errorf("failed to create event repository: %w", err)
	}

	eventAttendeeRepo, err := repoCreator.CreateRepository(entityid.EventAttendee, conn, tableConfig.TableName(entityid.EventAttendee))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_attendee repository: %w", err)
	}

	eventAttributeRepo, err := repoCreator.CreateRepository(entityid.EventAttribute, conn, tableConfig.TableName(entityid.EventAttribute))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_attribute repository: %w", err)
	}

	eventClientRepo, err := repoCreator.CreateRepository(entityid.EventClient, conn, tableConfig.TableName(entityid.EventClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_client repository: %w", err)
	}

	eventOccurrenceRepo, err := repoCreator.CreateRepository(entityid.EventOccurrence, conn, tableConfig.TableName(entityid.EventOccurrence))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_occurrence repository: %w", err)
	}

	eventProductRepo, err := repoCreator.CreateRepository(entityid.EventProduct, conn, tableConfig.TableName(entityid.EventProduct))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_product repository: %w", err)
	}

	eventRecurrenceRepo, err := repoCreator.CreateRepository(entityid.EventRecurrence, conn, tableConfig.TableName(entityid.EventRecurrence))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_recurrence repository: %w", err)
	}

	eventResourceRepo, err := repoCreator.CreateRepository(entityid.EventResource, conn, tableConfig.TableName(entityid.EventResource))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_resource repository: %w", err)
	}

	// Cross-domain repositories
	clientRepo, err := repoCreator.CreateRepository(entityid.Client, conn, tableConfig.TableName(entityid.Client))
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	productRepo, err := repoCreator.CreateRepository(entityid.Product, conn, tableConfig.TableName(entityid.Product))
	if err != nil {
		return nil, fmt.Errorf("failed to create product repository: %w", err)
	}

	return &EventRepositories{
		Event:           eventRepo.(eventpb.EventDomainServiceServer),
		EventAttendee:   eventAttendeeRepo.(eventAttendeepb.EventAttendeeDomainServiceServer),
		EventAttribute:  eventAttributeRepo.(eventattributepb.EventAttributeDomainServiceServer),
		EventClient:     eventClientRepo.(eventclientpb.EventClientDomainServiceServer),
		EventOccurrence: eventOccurrenceRepo.(eventOccurrencepb.EventOccurrenceDomainServiceServer),
		EventProduct:    eventProductRepo.(eventProductpb.EventProductDomainServiceServer),
		EventRecurrence: eventRecurrenceRepo.(eventRecurrencepb.EventRecurrenceDomainServiceServer),
		EventResource:   eventResourceRepo.(eventResourcepb.EventResourceDomainServiceServer),
		Client:          clientRepo.(clientpb.ClientDomainServiceServer),
		Product:         productRepo.(productpb.ProductDomainServiceServer),
	}, nil
}
