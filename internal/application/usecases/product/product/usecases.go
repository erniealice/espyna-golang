package product

import (
	"leapfor.xyz/espyna/internal/application/ports"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
)

// ProductRepositories groups all repository dependencies for product use cases
type ProductRepositories struct {
	Product productpb.ProductDomainServiceServer // Primary entity repository
}

// ProductServices groups all business service dependencies for product use cases
type ProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductRepositories(repositories)
	readServices := ReadProductServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductRepositories(repositories)
	updateServices := UpdateProductServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductRepositories(repositories)
	deleteServices := DeleteProductServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductsRepositories(repositories)
	listServices := ListProductsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateProduct: NewCreateProductUseCase(createRepos, createServices),
		ReadProduct:   NewReadProductUseCase(readRepos, readServices),
		UpdateProduct: NewUpdateProductUseCase(updateRepos, updateServices),
		DeleteProduct: NewDeleteProductUseCase(deleteRepos, deleteServices),
		ListProducts:  NewListProductsUseCase(listRepos, listServices),
	}
}
