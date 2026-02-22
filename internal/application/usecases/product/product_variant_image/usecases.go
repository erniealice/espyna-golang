package product_variant_image

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

// ProductVariantImageRepositories groups all repository dependencies for product variant image use cases
type ProductVariantImageRepositories struct {
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer // Primary entity repository
}

// ProductVariantImageServices groups all business service dependencies for product variant image use cases
type ProductVariantImageServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all product variant image-related use cases
type UseCases struct {
	CreateProductVariantImage *CreateProductVariantImageUseCase
	ReadProductVariantImage   *ReadProductVariantImageUseCase
	UpdateProductVariantImage *UpdateProductVariantImageUseCase
	DeleteProductVariantImage *DeleteProductVariantImageUseCase
	ListProductVariantImages  *ListProductVariantImagesUseCase
}

// NewUseCases creates a new collection of product variant image use cases
func NewUseCases(
	repositories ProductVariantImageRepositories,
	services ProductVariantImageServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductVariantImageRepositories(repositories)
	createServices := CreateProductVariantImageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductVariantImageRepositories(repositories)
	readServices := ReadProductVariantImageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductVariantImageRepositories(repositories)
	updateServices := UpdateProductVariantImageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductVariantImageRepositories(repositories)
	deleteServices := DeleteProductVariantImageServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductVariantImagesRepositories(repositories)
	listServices := ListProductVariantImagesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateProductVariantImage: NewCreateProductVariantImageUseCase(createRepos, createServices),
		ReadProductVariantImage:   NewReadProductVariantImageUseCase(readRepos, readServices),
		UpdateProductVariantImage: NewUpdateProductVariantImageUseCase(updateRepos, updateServices),
		DeleteProductVariantImage: NewDeleteProductVariantImageUseCase(deleteRepos, deleteServices),
		ListProductVariantImages:  NewListProductVariantImagesUseCase(listRepos, listServices),
	}
}
