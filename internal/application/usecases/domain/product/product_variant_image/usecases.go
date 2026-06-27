package product_variant_image

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

// ProductVariantImageRepositories groups all repository dependencies for product variant image use cases
type ProductVariantImageRepositories struct {
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer // Primary entity repository
}

// ProductVariantImageServices groups all business service dependencies for product variant image use cases
type ProductVariantImageServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductVariantImageRepositories(repositories)
	readServices := ReadProductVariantImageServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductVariantImageRepositories(repositories)
	updateServices := UpdateProductVariantImageServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductVariantImageRepositories(repositories)
	deleteServices := DeleteProductVariantImageServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductVariantImagesRepositories(repositories)
	listServices := ListProductVariantImagesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductVariantImage: NewCreateProductVariantImageUseCase(createRepos, createServices),
		ReadProductVariantImage:   NewReadProductVariantImageUseCase(readRepos, readServices),
		UpdateProductVariantImage: NewUpdateProductVariantImageUseCase(updateRepos, updateServices),
		DeleteProductVariantImage: NewDeleteProductVariantImageUseCase(deleteRepos, deleteServices),
		ListProductVariantImages:  NewListProductVariantImagesUseCase(listRepos, listServices),
	}
}
