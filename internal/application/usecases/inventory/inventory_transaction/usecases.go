package inventory_transaction

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
)

// InventoryTransactionRepositories groups all repository dependencies for inventory transaction use cases
type InventoryTransactionRepositories struct {
	InventoryTransaction inventorytransactionpb.InventoryTransactionDomainServiceServer // Primary entity repository
}

// InventoryTransactionServices groups all business service dependencies for inventory transaction use cases
type InventoryTransactionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all inventory transaction-related use cases
type UseCases struct {
	CreateInventoryTransaction *CreateInventoryTransactionUseCase
	ReadInventoryTransaction   *ReadInventoryTransactionUseCase
	UpdateInventoryTransaction *UpdateInventoryTransactionUseCase
	DeleteInventoryTransaction *DeleteInventoryTransactionUseCase
	ListInventoryTransactions  *ListInventoryTransactionsUseCase
}

// NewUseCases creates a new collection of inventory transaction use cases
func NewUseCases(
	repositories InventoryTransactionRepositories,
	services InventoryTransactionServices,
) *UseCases {
	createRepos := CreateInventoryTransactionRepositories(repositories)
	createServices := CreateInventoryTransactionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInventoryTransactionRepositories(repositories)
	readServices := ReadInventoryTransactionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateInventoryTransactionRepositories(repositories)
	updateServices := UpdateInventoryTransactionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteInventoryTransactionRepositories(repositories)
	deleteServices := DeleteInventoryTransactionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListInventoryTransactionsRepositories(repositories)
	listServices := ListInventoryTransactionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateInventoryTransaction: NewCreateInventoryTransactionUseCase(createRepos, createServices),
		ReadInventoryTransaction:   NewReadInventoryTransactionUseCase(readRepos, readServices),
		UpdateInventoryTransaction: NewUpdateInventoryTransactionUseCase(updateRepos, updateServices),
		DeleteInventoryTransaction: NewDeleteInventoryTransactionUseCase(deleteRepos, deleteServices),
		ListInventoryTransactions:  NewListInventoryTransactionsUseCase(listRepos, listServices),
	}
}
