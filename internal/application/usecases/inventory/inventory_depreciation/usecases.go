package inventory_depreciation

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
)

// InventoryDepreciationRepositories groups all repository dependencies for inventory depreciation use cases
type InventoryDepreciationRepositories struct {
	InventoryDepreciation inventorydepreciationpb.InventoryDepreciationDomainServiceServer // Primary entity repository
}

// InventoryDepreciationServices groups all business service dependencies for inventory depreciation use cases
type InventoryDepreciationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all inventory depreciation-related use cases
type UseCases struct {
	CreateInventoryDepreciation *CreateInventoryDepreciationUseCase
	ReadInventoryDepreciation   *ReadInventoryDepreciationUseCase
	UpdateInventoryDepreciation *UpdateInventoryDepreciationUseCase
	DeleteInventoryDepreciation *DeleteInventoryDepreciationUseCase
	ListInventoryDepreciations  *ListInventoryDepreciationsUseCase
}

// NewUseCases creates a new collection of inventory depreciation use cases
func NewUseCases(
	repositories InventoryDepreciationRepositories,
	services InventoryDepreciationServices,
) *UseCases {
	createRepos := CreateInventoryDepreciationRepositories(repositories)
	createServices := CreateInventoryDepreciationServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInventoryDepreciationRepositories(repositories)
	readServices := ReadInventoryDepreciationServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateInventoryDepreciationRepositories(repositories)
	updateServices := UpdateInventoryDepreciationServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteInventoryDepreciationRepositories(repositories)
	deleteServices := DeleteInventoryDepreciationServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListInventoryDepreciationsRepositories(repositories)
	listServices := ListInventoryDepreciationsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateInventoryDepreciation: NewCreateInventoryDepreciationUseCase(createRepos, createServices),
		ReadInventoryDepreciation:   NewReadInventoryDepreciationUseCase(readRepos, readServices),
		UpdateInventoryDepreciation: NewUpdateInventoryDepreciationUseCase(updateRepos, updateServices),
		DeleteInventoryDepreciation: NewDeleteInventoryDepreciationUseCase(deleteRepos, deleteServices),
		ListInventoryDepreciations:  NewListInventoryDepreciationsUseCase(listRepos, listServices),
	}
}
