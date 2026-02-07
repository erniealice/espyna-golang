package domain

import (
	"fmt"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	// Protobuf domain services - Entity domain
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"

	// Protobuf domain services - Event domain
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventattributepb "leapfor.xyz/esqyma/golang/v1/domain/event/event_attribute"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// EventRepositories contains all event domain repositories and cross-domain dependencies
// Event domain: Event, EventAttribute, EventClient (3 entities)
// Cross-domain: Client (needed by EventClient use case)
type EventRepositories struct {
	Event          eventpb.EventDomainServiceServer
	EventAttribute eventattributepb.EventAttributeDomainServiceServer
	EventClient    eventclientpb.EventClientDomainServiceServer
	// Cross-domain dependency from Entity domain
	Client clientpb.ClientDomainServiceServer
}

// NewEventRepositories creates and returns a new set of EventRepositories
func NewEventRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*EventRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names directly from dbTableConfig
	eventRepo, err := repoCreator.CreateRepository("event", conn, dbTableConfig.Event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event repository: %w", err)
	}

	eventAttributeRepo, err := repoCreator.CreateRepository("event_attribute", conn, dbTableConfig.EventAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create event_attribute repository: %w", err)
	}

	eventClientRepo, err := repoCreator.CreateRepository("event_client", conn, dbTableConfig.EventClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create event_client repository: %w", err)
	}

	clientRepo, err := repoCreator.CreateRepository("client", conn, dbTableConfig.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	// Type assert each repository to its interface
	return &EventRepositories{
		Event:          eventRepo.(eventpb.EventDomainServiceServer),
		EventAttribute: eventAttributeRepo.(eventattributepb.EventAttributeDomainServiceServer),
		EventClient:    eventClientRepo.(eventclientpb.EventClientDomainServiceServer),
		Client:         clientRepo.(clientpb.ClientDomainServiceServer),
	}, nil
}
