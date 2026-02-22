package product_variant

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

// ProductVariantRepositories groups all repository dependencies for product variant use cases
type ProductVariantRepositories struct {
	ProductVariant productvariantpb.ProductVariantDomainServiceServer // Primary entity repository
}

// ProductVariantServices groups all business service dependencies for product variant use cases
type ProductVariantServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all product variant-related use cases
type UseCases struct {
	CreateProductVariant *CreateProductVariantUseCase
	ReadProductVariant   *ReadProductVariantUseCase
	UpdateProductVariant *UpdateProductVariantUseCase
	DeleteProductVariant *DeleteProductVariantUseCase
	ListProductVariants  *ListProductVariantsUseCase
}

// NewUseCases creates a new collection of product variant use cases
func NewUseCases(
	repositories ProductVariantRepositories,
	services ProductVariantServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductVariantRepositories(repositories)
	createServices := CreateProductVariantServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductVariantRepositories(repositories)
	readServices := ReadProductVariantServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductVariantRepositories(repositories)
	updateServices := UpdateProductVariantServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductVariantRepositories(repositories)
	deleteServices := DeleteProductVariantServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductVariantsRepositories(repositories)
	listServices := ListProductVariantsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateProductVariant: NewCreateProductVariantUseCase(createRepos, createServices),
		ReadProductVariant:   NewReadProductVariantUseCase(readRepos, readServices),
		UpdateProductVariant: NewUpdateProductVariantUseCase(updateRepos, updateServices),
		DeleteProductVariant: NewDeleteProductVariantUseCase(deleteRepos, deleteServices),
		ListProductVariants:  NewListProductVariantsUseCase(listRepos, listServices),
	}
}
