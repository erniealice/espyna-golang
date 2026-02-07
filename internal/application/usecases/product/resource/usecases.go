package resource

import (
	"leapfor.xyz/espyna/internal/application/ports"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
)

// ResourceRepositories groups all repository dependencies for resource use cases
type ResourceRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
	Product  productpb.ProductDomainServiceServer   // Entity reference: resource.product_id -> product.id
}

// ResourceServices groups all business service dependencies for resource use cases
type ResourceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadResourceRepositories{
		Resource: repositories.Resource,
	}
	readServices := ReadResourceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateResourceRepositories(repositories)
	updateServices := UpdateResourceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteResourceRepositories{
		Resource: repositories.Resource,
	}
	deleteServices := DeleteResourceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListResourcesRepositories{
		Resource: repositories.Resource,
	}
	listServices := ListResourcesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateResource: NewCreateResourceUseCase(createRepos, createServices),
		ReadResource:   NewReadResourceUseCase(readRepos, readServices),
		UpdateResource: NewUpdateResourceUseCase(updateRepos, updateServices),
		DeleteResource: NewDeleteResourceUseCase(deleteRepos, deleteServices),
		ListResources:  NewListResourcesUseCase(listRepos, listServices),
	}
}
