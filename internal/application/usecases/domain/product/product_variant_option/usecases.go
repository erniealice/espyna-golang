package product_variant_option

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
)

// ProductVariantOptionRepositories groups all repository dependencies for product variant option use cases
type ProductVariantOptionRepositories struct {
	ProductVariantOption productvariantoptionpb.ProductVariantOptionDomainServiceServer // Primary entity repository
}

// ProductVariantOptionServices groups all business service dependencies for product variant option use cases
type ProductVariantOptionServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductVariantOptionRepositories(repositories)
	readServices := ReadProductVariantOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductVariantOptionRepositories(repositories)
	updateServices := UpdateProductVariantOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductVariantOptionRepositories(repositories)
	deleteServices := DeleteProductVariantOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductVariantOptionsRepositories(repositories)
	listServices := ListProductVariantOptionsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductVariantOption: NewCreateProductVariantOptionUseCase(createRepos, createServices),
		ReadProductVariantOption:   NewReadProductVariantOptionUseCase(readRepos, readServices),
		UpdateProductVariantOption: NewUpdateProductVariantOptionUseCase(updateRepos, updateServices),
		DeleteProductVariantOption: NewDeleteProductVariantOptionUseCase(deleteRepos, deleteServices),
		ListProductVariantOptions:  NewListProductVariantOptionsUseCase(listRepos, listServices),
	}
}
