package inventory_transaction

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
)

// InventoryTransactionRepositories groups all repository dependencies for inventory transaction use cases
type InventoryTransactionRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer // Primary entity repository
}

// InventoryTransactionServices groups all business service dependencies for inventory transaction use cases
type InventoryTransactionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all inventory transaction-related use cases
type UseCases struct {
	CreateInventoryTransaction        *CreateInventoryTransactionUseCase
	ReadInventoryTransaction          *ReadInventoryTransactionUseCase
	UpdateInventoryTransaction        *UpdateInventoryTransactionUseCase
	DeleteInventoryTransaction        *DeleteInventoryTransactionUseCase
	ListInventoryTransactions         *ListInventoryTransactionsUseCase
	GetInventoryMovementsListPageData *GetInventoryMovementsListPageDataUseCase
}

// NewUseCases creates a new collection of inventory transaction use cases
func NewUseCases(
	repositories InventoryTransactionRepositories,
	services InventoryTransactionServices,
) *UseCases {
	createRepos := CreateInventoryTransactionRepositories(repositories)
	createServices := CreateInventoryTransactionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInventoryTransactionRepositories(repositories)
	readServices := ReadInventoryTransactionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInventoryTransactionRepositories(repositories)
	updateServices := UpdateInventoryTransactionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInventoryTransactionRepositories(repositories)
	deleteServices := DeleteInventoryTransactionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInventoryTransactionsRepositories(repositories)
	listServices := ListInventoryTransactionsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	movementsRepos := GetInventoryMovementsListPageDataRepositories(repositories)
	movementsSvcs := GetInventoryMovementsListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInventoryTransaction:        NewCreateInventoryTransactionUseCase(createRepos, createServices),
		ReadInventoryTransaction:          NewReadInventoryTransactionUseCase(readRepos, readServices),
		UpdateInventoryTransaction:        NewUpdateInventoryTransactionUseCase(updateRepos, updateServices),
		DeleteInventoryTransaction:        NewDeleteInventoryTransactionUseCase(deleteRepos, deleteServices),
		ListInventoryTransactions:         NewListInventoryTransactionsUseCase(listRepos, listServices),
		GetInventoryMovementsListPageData: NewGetInventoryMovementsListPageDataUseCase(movementsRepos, movementsSvcs),
	}
}
