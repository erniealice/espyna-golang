package inventory_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

// InventoryAttributeRepositories groups all repository dependencies for inventory attribute use cases
type InventoryAttributeRepositories struct {
	InventoryAttribute inventoryattributepb.InventoryAttributeDomainServiceServer // Primary entity repository
}

// InventoryAttributeServices groups all business service dependencies for inventory attribute use cases
type InventoryAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all inventory attribute-related use cases
type UseCases struct {
	CreateInventoryAttribute *CreateInventoryAttributeUseCase
	ReadInventoryAttribute   *ReadInventoryAttributeUseCase
	UpdateInventoryAttribute *UpdateInventoryAttributeUseCase
	DeleteInventoryAttribute *DeleteInventoryAttributeUseCase
	ListInventoryAttributes  *ListInventoryAttributesUseCase
}

// NewUseCases creates a new collection of inventory attribute use cases
func NewUseCases(
	repositories InventoryAttributeRepositories,
	services InventoryAttributeServices,
) *UseCases {
	createRepos := CreateInventoryAttributeRepositories(repositories)
	createServices := CreateInventoryAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInventoryAttributeRepositories(repositories)
	readServices := ReadInventoryAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInventoryAttributeRepositories(repositories)
	updateServices := UpdateInventoryAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInventoryAttributeRepositories(repositories)
	deleteServices := DeleteInventoryAttributeServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInventoryAttributesRepositories(repositories)
	listServices := ListInventoryAttributesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInventoryAttribute: NewCreateInventoryAttributeUseCase(createRepos, createServices),
		ReadInventoryAttribute:   NewReadInventoryAttributeUseCase(readRepos, readServices),
		UpdateInventoryAttribute: NewUpdateInventoryAttributeUseCase(updateRepos, updateServices),
		DeleteInventoryAttribute: NewDeleteInventoryAttributeUseCase(deleteRepos, deleteServices),
		ListInventoryAttributes:  NewListInventoryAttributesUseCase(listRepos, listServices),
	}
}
