package product_option

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
)

// ProductOptionRepositories groups all repository dependencies for product option use cases
type ProductOptionRepositories struct {
	ProductOption productoptionpb.ProductOptionDomainServiceServer // Primary entity repository
}

// ProductOptionServices groups all business service dependencies for product option use cases
type ProductOptionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all product option-related use cases
type UseCases struct {
	CreateProductOption *CreateProductOptionUseCase
	ReadProductOption   *ReadProductOptionUseCase
	UpdateProductOption *UpdateProductOptionUseCase
	DeleteProductOption *DeleteProductOptionUseCase
	ListProductOptions  *ListProductOptionsUseCase
}

// NewUseCases creates a new collection of product option use cases
func NewUseCases(
	repositories ProductOptionRepositories,
	services ProductOptionServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductOptionRepositories(repositories)
	createServices := CreateProductOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductOptionRepositories(repositories)
	readServices := ReadProductOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductOptionRepositories(repositories)
	updateServices := UpdateProductOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductOptionRepositories(repositories)
	deleteServices := DeleteProductOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductOptionsRepositories(repositories)
	listServices := ListProductOptionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateProductOption: NewCreateProductOptionUseCase(createRepos, createServices),
		ReadProductOption:   NewReadProductOptionUseCase(readRepos, readServices),
		UpdateProductOption: NewUpdateProductOptionUseCase(updateRepos, updateServices),
		DeleteProductOption: NewDeleteProductOptionUseCase(deleteRepos, deleteServices),
		ListProductOptions:  NewListProductOptionsUseCase(listRepos, listServices),
	}
}
