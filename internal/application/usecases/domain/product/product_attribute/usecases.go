package product_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// ProductAttributeRepositories groups all repository dependencies for product attribute use cases
type ProductAttributeRepositories struct {
	ProductAttribute productattributepb.ProductAttributeDomainServiceServer // Primary entity repository
	Product          productpb.ProductDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
}

// ProductAttributeServices groups all business service dependencies for product attribute use cases
type ProductAttributeServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all product attribute-related use cases
type UseCases struct {
	CreateProductAttribute          *CreateProductAttributeUseCase
	ReadProductAttribute            *ReadProductAttributeUseCase
	UpdateProductAttribute          *UpdateProductAttributeUseCase
	DeleteProductAttribute          *DeleteProductAttributeUseCase
	ListProductAttributes           *ListProductAttributesUseCase
	GetProductAttributeListPageData *GetProductAttributeListPageDataUseCase
	GetProductAttributeItemPageData *GetProductAttributeItemPageDataUseCase
}

// NewUseCases creates a new collection of product attribute use cases
func NewUseCases(
	repositories ProductAttributeRepositories,
	services ProductAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductAttributeRepositories{
		ProductAttribute: repositories.ProductAttribute,
		Product:          repositories.Product,
		Attribute:        repositories.Attribute,
	}
	createServices := CreateProductAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductAttributeRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	readServices := ReadProductAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductAttributeRepositories{
		ProductAttribute: repositories.ProductAttribute,
		Product:          repositories.Product,
		Attribute:        repositories.Attribute,
	}
	updateServices := UpdateProductAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductAttributeRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	deleteServices := DeleteProductAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductAttributesRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	listServices := ListProductAttributesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetProductAttributeListPageDataRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	listPageDataServices := GetProductAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetProductAttributeItemPageDataRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	itemPageDataServices := GetProductAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductAttribute:          NewCreateProductAttributeUseCase(createRepos, createServices),
		ReadProductAttribute:            NewReadProductAttributeUseCase(readRepos, readServices),
		UpdateProductAttribute:          NewUpdateProductAttributeUseCase(updateRepos, updateServices),
		DeleteProductAttribute:          NewDeleteProductAttributeUseCase(deleteRepos, deleteServices),
		ListProductAttributes:           NewListProductAttributesUseCase(listRepos, listServices),
		GetProductAttributeListPageData: NewGetProductAttributeListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetProductAttributeItemPageData: NewGetProductAttributeItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
