package product_attribute

import (
	"leapfor.xyz/espyna/internal/application/ports"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
	productattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/product_attribute"
)

// ProductAttributeRepositories groups all repository dependencies for product attribute use cases
type ProductAttributeRepositories struct {
	ProductAttribute productattributepb.ProductAttributeDomainServiceServer // Primary entity repository
	Product          productpb.ProductDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
}

// ProductAttributeServices groups all business service dependencies for product attribute use cases
type ProductAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductAttributeRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	readServices := ReadProductAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductAttributeRepositories{
		ProductAttribute: repositories.ProductAttribute,
		Product:          repositories.Product,
		Attribute:        repositories.Attribute,
	}
	updateServices := UpdateProductAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductAttributeRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	deleteServices := DeleteProductAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductAttributesRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	listServices := ListProductAttributesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetProductAttributeListPageDataRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	listPageDataServices := GetProductAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetProductAttributeItemPageDataRepositories{
		ProductAttribute: repositories.ProductAttribute,
	}
	itemPageDataServices := GetProductAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
