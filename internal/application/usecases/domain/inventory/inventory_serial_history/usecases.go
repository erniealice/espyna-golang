package inventory_serial_history

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// InventorySerialHistoryRepositories groups all repository dependencies for inventory serial history use cases
type InventorySerialHistoryRepositories struct {
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer // Primary entity repository
}

// InventorySerialHistoryServices groups all business service dependencies for inventory serial history use cases
type InventorySerialHistoryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all inventory serial history-related use cases
// NOTE: No UpdateInventorySerialHistory — immutable audit trail
type UseCases struct {
	CreateInventorySerialHistory *CreateInventorySerialHistoryUseCase
	ReadInventorySerialHistory   *ReadInventorySerialHistoryUseCase
	DeleteInventorySerialHistory *DeleteInventorySerialHistoryUseCase
	ListInventorySerialHistory   *ListInventorySerialHistoryUseCase
}

// NewUseCases creates a new collection of inventory serial history use cases
func NewUseCases(
	repositories InventorySerialHistoryRepositories,
	services InventorySerialHistoryServices,
) *UseCases {
	createRepos := CreateInventorySerialHistoryRepositories(repositories)
	createServices := CreateInventorySerialHistoryServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInventorySerialHistoryRepositories(repositories)
	readServices := ReadInventorySerialHistoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInventorySerialHistoryRepositories(repositories)
	deleteServices := DeleteInventorySerialHistoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInventorySerialHistoryRepositories(repositories)
	listServices := ListInventorySerialHistoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInventorySerialHistory: NewCreateInventorySerialHistoryUseCase(createRepos, createServices),
		ReadInventorySerialHistory:   NewReadInventorySerialHistoryUseCase(readRepos, readServices),
		DeleteInventorySerialHistory: NewDeleteInventorySerialHistoryUseCase(deleteRepos, deleteServices),
		ListInventorySerialHistory:   NewListInventorySerialHistoryUseCase(listRepos, listServices),
	}
}
