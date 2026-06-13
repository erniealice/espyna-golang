package product_variant

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

// ProductVariantRepositories groups all repository dependencies for product variant use cases
type ProductVariantRepositories struct {
	ProductVariant productvariantpb.ProductVariantDomainServiceServer // Primary entity repository
}

// ProductVariantServices groups all business service dependencies for product variant use cases
type ProductVariantServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductVariantRepositories(repositories)
	readServices := ReadProductVariantServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductVariantRepositories(repositories)
	updateServices := UpdateProductVariantServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductVariantRepositories(repositories)
	deleteServices := DeleteProductVariantServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductVariantsRepositories(repositories)
	listServices := ListProductVariantsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductVariant: NewCreateProductVariantUseCase(createRepos, createServices),
		ReadProductVariant:   NewReadProductVariantUseCase(readRepos, readServices),
		UpdateProductVariant: NewUpdateProductVariantUseCase(updateRepos, updateServices),
		DeleteProductVariant: NewDeleteProductVariantUseCase(deleteRepos, deleteServices),
		ListProductVariants:  NewListProductVariantsUseCase(listRepos, listServices),
	}
}
