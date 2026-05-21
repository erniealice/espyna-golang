package product_line

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	linepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
)

// ProductLineRepositories groups all repository dependencies for product line use cases
type ProductLineRepositories struct {
	ProductLine productlinepb.ProductLineDomainServiceServer // Primary entity repository
	Product     productpb.ProductDomainServiceServer
	Line        linepb.LineDomainServiceServer
}

// ProductLineServices groups all business service dependencies for product line use cases
type ProductLineServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all product line-related use cases
type UseCases struct {
	CreateProductLine          *CreateProductLineUseCase
	ReadProductLine            *ReadProductLineUseCase
	UpdateProductLine          *UpdateProductLineUseCase
	DeleteProductLine          *DeleteProductLineUseCase
	ListProductLines           *ListProductLinesUseCase
	GetProductLineListPageData *GetProductLineListPageDataUseCase
	GetProductLineItemPageData *GetProductLineItemPageDataUseCase
}

// NewUseCases creates a new line of product line use cases
func NewUseCases(
	repositories ProductLineRepositories,
	services ProductLineServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductLineRepositories{
		ProductLine: repositories.ProductLine,
		Product:     repositories.Product,
		Line:        repositories.Line,
	}
	createServices := CreateProductLineServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductLineRepositories{
		ProductLine: repositories.ProductLine,
	}
	readServices := ReadProductLineServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductLineRepositories{
		ProductLine: repositories.ProductLine,
		Product:     repositories.Product,
		Line:        repositories.Line,
	}
	updateServices := UpdateProductLineServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductLineRepositories{
		ProductLine: repositories.ProductLine,
	}
	deleteServices := DeleteProductLineServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductLinesRepositories{
		ProductLine: repositories.ProductLine,
	}
	listServices := ListProductLinesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetProductLineListPageDataRepositories{
		ProductLine: repositories.ProductLine,
	}
	listPageDataServices := GetProductLineListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetProductLineItemPageDataRepositories{
		ProductLine: repositories.ProductLine,
	}
	itemPageDataServices := GetProductLineItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductLine:          NewCreateProductLineUseCase(createRepos, createServices),
		ReadProductLine:            NewReadProductLineUseCase(readRepos, readServices),
		UpdateProductLine:          NewUpdateProductLineUseCase(updateRepos, updateServices),
		DeleteProductLine:          NewDeleteProductLineUseCase(deleteRepos, deleteServices),
		ListProductLines:           NewListProductLinesUseCase(listRepos, listServices),
		GetProductLineListPageData: NewGetProductLineListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetProductLineItemPageData: NewGetProductLineItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
