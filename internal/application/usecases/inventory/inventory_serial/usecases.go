package inventory_serial

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
)

// InventorySerialRepositories groups all repository dependencies for inventory serial use cases
type InventorySerialRepositories struct {
	InventorySerial inventoryserialpb.InventorySerialDomainServiceServer // Primary entity repository
}

// InventorySerialServices groups all business service dependencies for inventory serial use cases
type InventorySerialServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all inventory serial-related use cases
type UseCases struct {
	CreateInventorySerial *CreateInventorySerialUseCase
	ReadInventorySerial   *ReadInventorySerialUseCase
	UpdateInventorySerial *UpdateInventorySerialUseCase
	DeleteInventorySerial *DeleteInventorySerialUseCase
	ListInventorySerials  *ListInventorySerialsUseCase
}

// NewUseCases creates a new collection of inventory serial use cases
func NewUseCases(
	repositories InventorySerialRepositories,
	services InventorySerialServices,
) *UseCases {
	createRepos := CreateInventorySerialRepositories(repositories)
	createServices := CreateInventorySerialServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInventorySerialRepositories(repositories)
	readServices := ReadInventorySerialServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateInventorySerialRepositories(repositories)
	updateServices := UpdateInventorySerialServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteInventorySerialRepositories(repositories)
	deleteServices := DeleteInventorySerialServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListInventorySerialsRepositories(repositories)
	listServices := ListInventorySerialsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateInventorySerial: NewCreateInventorySerialUseCase(createRepos, createServices),
		ReadInventorySerial:   NewReadInventorySerialUseCase(readRepos, readServices),
		UpdateInventorySerial: NewUpdateInventorySerialUseCase(updateRepos, updateServices),
		DeleteInventorySerial: NewDeleteInventorySerialUseCase(deleteRepos, deleteServices),
		ListInventorySerials:  NewListInventorySerialsUseCase(listRepos, listServices),
	}
}
