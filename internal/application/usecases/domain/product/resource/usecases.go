package resource

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// ResourceRepositories groups all repository dependencies for resource use cases
type ResourceRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
	Product  productpb.ProductDomainServiceServer   // Entity reference: resource.product_id -> product.id
}

// ResourceServices groups all business service dependencies for resource use cases
type ResourceServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all resource-related use cases
type UseCases struct {
	CreateResource *CreateResourceUseCase
	ReadResource   *ReadResourceUseCase
	UpdateResource *UpdateResourceUseCase
	DeleteResource *DeleteResourceUseCase
	ListResources  *ListResourcesUseCase
}

// NewUseCases creates a new collection of resource use cases with entity reference dependencies
func NewUseCases(
	repositories ResourceRepositories,
	services ResourceServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateResourceRepositories(repositories)
	createServices := CreateResourceServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadResourceRepositories{
		Resource: repositories.Resource,
	}
	readServices := ReadResourceServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateResourceRepositories(repositories)
	updateServices := UpdateResourceServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteResourceRepositories{
		Resource: repositories.Resource,
	}
	deleteServices := DeleteResourceServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListResourcesRepositories{
		Resource: repositories.Resource,
	}
	listServices := ListResourcesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateResource: NewCreateResourceUseCase(createRepos, createServices),
		ReadResource:   NewReadResourceUseCase(readRepos, readServices),
		UpdateResource: NewUpdateResourceUseCase(updateRepos, updateServices),
		DeleteResource: NewDeleteResourceUseCase(deleteRepos, deleteServices),
		ListResources:  NewListResourcesUseCase(listRepos, listServices),
	}
}
