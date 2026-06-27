package product

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// ProductRepositories groups all repository dependencies for product use cases
type ProductRepositories struct {
	Product productpb.ProductDomainServiceServer // Primary entity repository
}

// ProductServices groups all business service dependencies for product use cases
type ProductServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all product-related use cases
type UseCases struct {
	CreateProduct *CreateProductUseCase
	ReadProduct   *ReadProductUseCase
	UpdateProduct *UpdateProductUseCase
	DeleteProduct *DeleteProductUseCase
	ListProducts  *ListProductsUseCase
}

// NewUseCases creates a new collection of product use cases
func NewUseCases(
	repositories ProductRepositories,
	services ProductServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductRepositories(repositories)
	createServices := CreateProductServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductRepositories(repositories)
	readServices := ReadProductServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductRepositories(repositories)
	updateServices := UpdateProductServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductRepositories(repositories)
	deleteServices := DeleteProductServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductsRepositories(repositories)
	listServices := ListProductsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProduct: NewCreateProductUseCase(createRepos, createServices),
		ReadProduct:   NewReadProductUseCase(readRepos, readServices),
		UpdateProduct: NewUpdateProductUseCase(updateRepos, updateServices),
		DeleteProduct: NewDeleteProductUseCase(deleteRepos, deleteServices),
		ListProducts:  NewListProductsUseCase(listRepos, listServices),
	}
}
