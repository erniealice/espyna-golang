package inventory_item

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
)

// InventoryItemRepositories groups all repository dependencies for inventory item use cases
type InventoryItemRepositories struct {
	InventoryItem inventoryitempb.InventoryItemDomainServiceServer // Primary entity repository
}

// InventoryItemServices groups all business service dependencies for inventory item use cases
type InventoryItemServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all inventory item-related use cases
type UseCases struct {
	CreateInventoryItem *CreateInventoryItemUseCase
	ReadInventoryItem   *ReadInventoryItemUseCase
	UpdateInventoryItem *UpdateInventoryItemUseCase
	DeleteInventoryItem *DeleteInventoryItemUseCase
	ListInventoryItems  *ListInventoryItemsUseCase
}

// NewUseCases creates a new collection of inventory item use cases
func NewUseCases(
	repositories InventoryItemRepositories,
	services InventoryItemServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateInventoryItemRepositories(repositories)
	createServices := CreateInventoryItemServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInventoryItemRepositories(repositories)
	readServices := ReadInventoryItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInventoryItemRepositories(repositories)
	updateServices := UpdateInventoryItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInventoryItemRepositories(repositories)
	deleteServices := DeleteInventoryItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInventoryItemsRepositories(repositories)
	listServices := ListInventoryItemsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInventoryItem: NewCreateInventoryItemUseCase(createRepos, createServices),
		ReadInventoryItem:   NewReadInventoryItemUseCase(readRepos, readServices),
		UpdateInventoryItem: NewUpdateInventoryItemUseCase(updateRepos, updateServices),
		DeleteInventoryItem: NewDeleteInventoryItemUseCase(deleteRepos, deleteServices),
		ListInventoryItems:  NewListInventoryItemsUseCase(listRepos, listServices),
	}
}
