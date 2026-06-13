package eventproduct

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// UseCases contains all event product-related use cases
type UseCases struct {
	CreateEventProduct *CreateEventProductUseCase
	ReadEventProduct   *ReadEventProductUseCase
	UpdateEventProduct *UpdateEventProductUseCase
	DeleteEventProduct *DeleteEventProductUseCase
	ListEventProducts  *ListEventProductsUseCase
}

// EventProductRepositories groups all repository dependencies for event product use cases
type EventProductRepositories struct {
	EventProduct eventproductpb.EventProductDomainServiceServer // Primary entity repository
	Event        eventpb.EventDomainServiceServer               // Entity reference validation
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// EventProductServices groups all business service dependencies for event product use cases
type EventProductServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of event product use cases
func NewUseCases(
	repositories EventProductRepositories,
	services EventProductServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventProductRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	createServices := CreateEventProductServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventProductRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	readServices := ReadEventProductServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateEventProductRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	updateServices := UpdateEventProductServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteEventProductRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	deleteServices := DeleteEventProductServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListEventProductsRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	listServices := ListEventProductsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateEventProduct: NewCreateEventProductUseCase(createRepos, createServices),
		ReadEventProduct:   NewReadEventProductUseCase(readRepos, readServices),
		UpdateEventProduct: NewUpdateEventProductUseCase(updateRepos, updateServices),
		DeleteEventProduct: NewDeleteEventProductUseCase(deleteRepos, deleteServices),
		ListEventProducts:  NewListEventProductsUseCase(listRepos, listServices),
	}
}

// NewUseCasesUngrouped creates a new collection of event product use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	eventProductRepo eventproductpb.EventProductDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
	productRepo productpb.ProductDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventProductRepositories{
		EventProduct: eventProductRepo,
		Event:        eventRepo,
		Product:      productRepo,
	}

	services := EventProductServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
