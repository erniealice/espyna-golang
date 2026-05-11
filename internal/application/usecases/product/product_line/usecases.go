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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductLineRepositories{
		ProductLine: repositories.ProductLine,
	}
	readServices := ReadProductLineServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductLineRepositories{
		ProductLine: repositories.ProductLine,
		Product:     repositories.Product,
		Line:        repositories.Line,
	}
	updateServices := UpdateProductLineServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductLineRepositories{
		ProductLine: repositories.ProductLine,
	}
	deleteServices := DeleteProductLineServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductLinesRepositories{
		ProductLine: repositories.ProductLine,
	}
	listServices := ListProductLinesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetProductLineListPageDataRepositories{
		ProductLine: repositories.ProductLine,
	}
	listPageDataServices := GetProductLineListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetProductLineItemPageDataRepositories{
		ProductLine: repositories.ProductLine,
	}
	itemPageDataServices := GetProductLineItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
