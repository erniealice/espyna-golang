package inventory_serial

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInventorySerialRepositories(repositories)
	readServices := ReadInventorySerialServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInventorySerialRepositories(repositories)
	updateServices := UpdateInventorySerialServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInventorySerialRepositories(repositories)
	deleteServices := DeleteInventorySerialServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInventorySerialsRepositories(repositories)
	listServices := ListInventorySerialsServices{
		ActionGatekeeper: services.ActionGatekeeper,
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
