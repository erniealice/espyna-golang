package inventory_item

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
)

// InventoryItemRepositories groups all repository dependencies for inventory item use cases
type InventoryItemRepositories struct {
	InventoryItem inventoryitempb.InventoryItemDomainServiceServer // Primary entity repository
}

// InventoryItemServices groups all business service dependencies for inventory item use cases
type InventoryItemServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInventoryItemRepositories(repositories)
	readServices := ReadInventoryItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateInventoryItemRepositories(repositories)
	updateServices := UpdateInventoryItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteInventoryItemRepositories(repositories)
	deleteServices := DeleteInventoryItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListInventoryItemsRepositories(repositories)
	listServices := ListInventoryItemsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateInventoryItem: NewCreateInventoryItemUseCase(createRepos, createServices),
		ReadInventoryItem:   NewReadInventoryItemUseCase(readRepos, readServices),
		UpdateInventoryItem: NewUpdateInventoryItemUseCase(updateRepos, updateServices),
		DeleteInventoryItem: NewDeleteInventoryItemUseCase(deleteRepos, deleteServices),
		ListInventoryItems:  NewListInventoryItemsUseCase(listRepos, listServices),
	}
}
