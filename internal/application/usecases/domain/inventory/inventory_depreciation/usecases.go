package inventory_depreciation

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
)

// InventoryDepreciationRepositories groups all repository dependencies for inventory depreciation use cases
type InventoryDepreciationRepositories struct {
	InventoryDepreciation inventorydepreciationpb.InventoryDepreciationDomainServiceServer // Primary entity repository
}

// InventoryDepreciationServices groups all business service dependencies for inventory depreciation use cases
type InventoryDepreciationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInventoryDepreciationRepositories(repositories)
	readServices := ReadInventoryDepreciationServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInventoryDepreciationRepositories(repositories)
	updateServices := UpdateInventoryDepreciationServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInventoryDepreciationRepositories(repositories)
	deleteServices := DeleteInventoryDepreciationServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInventoryDepreciationsRepositories(repositories)
	listServices := ListInventoryDepreciationsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInventoryDepreciation: NewCreateInventoryDepreciationUseCase(createRepos, createServices),
		ReadInventoryDepreciation:   NewReadInventoryDepreciationUseCase(readRepos, readServices),
		UpdateInventoryDepreciation: NewUpdateInventoryDepreciationUseCase(updateRepos, updateServices),
		DeleteInventoryDepreciation: NewDeleteInventoryDepreciationUseCase(deleteRepos, deleteServices),
		ListInventoryDepreciations:  NewListInventoryDepreciationsUseCase(listRepos, listServices),
	}
}
