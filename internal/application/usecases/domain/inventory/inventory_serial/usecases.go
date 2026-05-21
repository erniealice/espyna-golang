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
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInventorySerialRepositories(repositories)
	readServices := ReadInventorySerialServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInventorySerialRepositories(repositories)
	updateServices := UpdateInventorySerialServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInventorySerialRepositories(repositories)
	deleteServices := DeleteInventorySerialServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInventorySerialsRepositories(repositories)
	listServices := ListInventorySerialsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInventorySerial: NewCreateInventorySerialUseCase(createRepos, createServices),
		ReadInventorySerial:   NewReadInventorySerialUseCase(readRepos, readServices),
		UpdateInventorySerial: NewUpdateInventorySerialUseCase(updateRepos, updateServices),
		DeleteInventorySerial: NewDeleteInventorySerialUseCase(deleteRepos, deleteServices),
		ListInventorySerials:  NewListInventorySerialsUseCase(listRepos, listServices),
	}
}
