package inventory_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

// InventoryAttributeRepositories groups all repository dependencies for inventory attribute use cases
type InventoryAttributeRepositories struct {
	InventoryAttribute inventoryattributepb.InventoryAttributeDomainServiceServer // Primary entity repository
}

// InventoryAttributeServices groups all business service dependencies for inventory attribute use cases
type InventoryAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all inventory attribute-related use cases
type UseCases struct {
	CreateInventoryAttribute *CreateInventoryAttributeUseCase
	ReadInventoryAttribute   *ReadInventoryAttributeUseCase
	UpdateInventoryAttribute *UpdateInventoryAttributeUseCase
	DeleteInventoryAttribute *DeleteInventoryAttributeUseCase
	ListInventoryAttributes  *ListInventoryAttributesUseCase
}

// NewUseCases creates a new collection of inventory attribute use cases
func NewUseCases(
	repositories InventoryAttributeRepositories,
	services InventoryAttributeServices,
) *UseCases {
	createRepos := CreateInventoryAttributeRepositories(repositories)
	createServices := CreateInventoryAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInventoryAttributeRepositories(repositories)
	readServices := ReadInventoryAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateInventoryAttributeRepositories(repositories)
	updateServices := UpdateInventoryAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteInventoryAttributeRepositories(repositories)
	deleteServices := DeleteInventoryAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListInventoryAttributesRepositories(repositories)
	listServices := ListInventoryAttributesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateInventoryAttribute: NewCreateInventoryAttributeUseCase(createRepos, createServices),
		ReadInventoryAttribute:   NewReadInventoryAttributeUseCase(readRepos, readServices),
		UpdateInventoryAttribute: NewUpdateInventoryAttributeUseCase(updateRepos, updateServices),
		DeleteInventoryAttribute: NewDeleteInventoryAttributeUseCase(deleteRepos, deleteServices),
		ListInventoryAttributes:  NewListInventoryAttributesUseCase(listRepos, listServices),
	}
}
