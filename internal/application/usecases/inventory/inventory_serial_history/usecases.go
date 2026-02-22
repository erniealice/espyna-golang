package inventory_serial_history

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

// InventorySerialHistoryRepositories groups all repository dependencies for inventory serial history use cases
type InventorySerialHistoryRepositories struct {
	InventorySerialHistory serialhistorypb.InventorySerialHistoryDomainServiceServer // Primary entity repository
}

// InventorySerialHistoryServices groups all business service dependencies for inventory serial history use cases
type InventorySerialHistoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInventorySerialHistoryRepositories(repositories)
	readServices := ReadInventorySerialHistoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteInventorySerialHistoryRepositories(repositories)
	deleteServices := DeleteInventorySerialHistoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListInventorySerialHistoryRepositories(repositories)
	listServices := ListInventorySerialHistoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateInventorySerialHistory: NewCreateInventorySerialHistoryUseCase(createRepos, createServices),
		ReadInventorySerialHistory:   NewReadInventorySerialHistoryUseCase(readRepos, readServices),
		DeleteInventorySerialHistory: NewDeleteInventorySerialHistoryUseCase(deleteRepos, deleteServices),
		ListInventorySerialHistory:   NewListInventorySerialHistoryUseCase(listRepos, listServices),
	}
}
