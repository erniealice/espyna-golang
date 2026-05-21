package eventproduct

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventProductRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	readServices := ReadEventProductServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventProductRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	updateServices := UpdateEventProductServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventProductRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	deleteServices := DeleteEventProductServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventProductsRepositories{
		EventProduct: repositories.EventProduct,
		Event:        repositories.Event,
		Product:      repositories.Product,
	}
	listServices := ListEventProductsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventProductRepositories{
		EventProduct: eventProductRepo,
		Event:        eventRepo,
		Product:      productRepo,
	}

	services := EventProductServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
