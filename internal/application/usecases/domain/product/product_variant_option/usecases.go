package product_variant_option

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
)

// ProductVariantOptionRepositories groups all repository dependencies for product variant option use cases
type ProductVariantOptionRepositories struct {
	ProductVariantOption productvariantoptionpb.ProductVariantOptionDomainServiceServer // Primary entity repository
}

// ProductVariantOptionServices groups all business service dependencies for product variant option use cases
type ProductVariantOptionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all product variant option-related use cases
type UseCases struct {
	CreateProductVariantOption *CreateProductVariantOptionUseCase
	ReadProductVariantOption   *ReadProductVariantOptionUseCase
	UpdateProductVariantOption *UpdateProductVariantOptionUseCase
	DeleteProductVariantOption *DeleteProductVariantOptionUseCase
	ListProductVariantOptions  *ListProductVariantOptionsUseCase
}

// NewUseCases creates a new collection of product variant option use cases
func NewUseCases(
	repositories ProductVariantOptionRepositories,
	services ProductVariantOptionServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductVariantOptionRepositories(repositories)
	createServices := CreateProductVariantOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductVariantOptionRepositories(repositories)
	readServices := ReadProductVariantOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductVariantOptionRepositories(repositories)
	updateServices := UpdateProductVariantOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductVariantOptionRepositories(repositories)
	deleteServices := DeleteProductVariantOptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductVariantOptionsRepositories(repositories)
	listServices := ListProductVariantOptionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateProductVariantOption: NewCreateProductVariantOptionUseCase(createRepos, createServices),
		ReadProductVariantOption:   NewReadProductVariantOptionUseCase(readRepos, readServices),
		UpdateProductVariantOption: NewUpdateProductVariantOptionUseCase(updateRepos, updateServices),
		DeleteProductVariantOption: NewDeleteProductVariantOptionUseCase(deleteRepos, deleteServices),
		ListProductVariantOptions:  NewListProductVariantOptionsUseCase(listRepos, listServices),
	}
}
