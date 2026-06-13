package product_option

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
)

// ProductOptionRepositories groups all repository dependencies for product option use cases
type ProductOptionRepositories struct {
	ProductOption productoptionpb.ProductOptionDomainServiceServer // Primary entity repository
}

// ProductOptionServices groups all business service dependencies for product option use cases
type ProductOptionServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductOptionRepositories(repositories)
	readServices := ReadProductOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductOptionRepositories(repositories)
	updateServices := UpdateProductOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductOptionRepositories(repositories)
	deleteServices := DeleteProductOptionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductOptionsRepositories(repositories)
	listServices := ListProductOptionsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductOption: NewCreateProductOptionUseCase(createRepos, createServices),
		ReadProductOption:   NewReadProductOptionUseCase(readRepos, readServices),
		UpdateProductOption: NewUpdateProductOptionUseCase(updateRepos, updateServices),
		DeleteProductOption: NewDeleteProductOptionUseCase(deleteRepos, deleteServices),
		ListProductOptions:  NewListProductOptionsUseCase(listRepos, listServices),
	}
}
